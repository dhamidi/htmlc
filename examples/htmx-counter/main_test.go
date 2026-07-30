package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestHTMXCounterEndToEnd proves RFC 014 §6 Example 4's actual claims with
// real HTTP round-trips, in the order the demo's counter and shared
// *htmlc.RenderSession accumulate state. State matters here — each step
// depends on the counter value and the render session's dedup bookkeeping
// left behind by the previous step — so this is one function with ordered
// subtests (t.Run calls run sequentially, top to bottom, since none of them
// call t.Parallel) rather than independent, possibly-reordered test
// functions.
func TestHTMXCounterEndToEnd(t *testing.T) {
	s, err := newServer()
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	if errs := s.engine.ValidateAll(); len(errs) != 0 {
		t.Fatalf("ValidateAll() = %v, want zero errors", errs)
	}
	mux := s.routes()

	// Step 1: GET / with no headers is always a full page — the initial
	// load, count 0.
	t.Run("GET slash renders a full page with the initial count", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<!DOCTYPE html>") {
			t.Errorf("expected a full document shell (<!DOCTYPE html>), got: %s", body)
		}
		if !strings.Contains(body, `id="counter"`) {
			t.Errorf("expected the rendered Counter component (id=\"counter\"), got: %s", body)
		}
		if !strings.Contains(body, "Count: 0") {
			t.Errorf("expected initial count 0, got: %s", body)
		}
	})

	// Step 2: POST /increment WITHOUT HX-Request set must take the
	// non-htmx fallback branch — proving the RFC's "if not htmx, render
	// the full page" branch actually fires, not just the happy path. This
	// also advances the counter to 1 for the following steps.
	t.Run("POST increment without HX-Request falls back to a full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/increment", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("HX-Trigger"); got != "" {
			t.Errorf("HX-Trigger = %q, want unset on the non-htmx fallback branch", got)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<!DOCTYPE html>") {
			t.Errorf("expected the non-htmx fallback to render a full document shell, got: %s", body)
		}
		if !strings.Contains(body, "Count: 1") {
			t.Errorf("expected the counter to have advanced to 1, got: %s", body)
		}
	})

	// Step 3: POST /increment WITH HX-Request: true (first htmx call on
	// this shared session) — both components' <style> blocks must be
	// present, since this is the first time either has ever been rendered
	// on s.sess.
	t.Run("first htmx increment emits both components' style blocks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/increment", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("HX-Trigger"); got != "counter-updated" {
			t.Errorf(`HX-Trigger = %q, want "counter-updated"`, got)
		}

		body := rec.Body.String()
		// Self-adversarial review: actually look at the raw output, not
		// just strings.Contains assertions on it.
		t.Logf("first htmx call body:\n%s", body)

		if got := strings.Count(body, "<style>"); got != 2 {
			t.Errorf("expected exactly 2 <style> blocks (Counter + StatusBadge, both first-seen on this session), got %d:\n%s", got, body)
		}
		if !strings.Contains(body, "border: 1px solid #ccc") {
			t.Errorf("expected Counter's own CSS body in the style block, got: %s", body)
		}
		if !strings.Contains(body, "font-style: italic") {
			t.Errorf("expected StatusBadge's own CSS body in the style block, got: %s", body)
		}
		if !strings.Contains(body, `<div id="status-badge" hx-swap-oob="true">`) {
			t.Errorf("expected the hx-swap-oob wrapper around StatusBadge, got: %s", body)
		}
		// The wrapper must actually be closed, with StatusBadge's markup
		// nested inside it, not merely present anywhere in the body.
		wrapIdx := strings.Index(body, `<div id="status-badge" hx-swap-oob="true">`)
		if wrapIdx < 0 || !strings.Contains(body[wrapIdx:], `class="status-badge"`) || !strings.Contains(body[wrapIdx:], "</div>") {
			t.Errorf("expected StatusBadge's markup nested inside the OOB wrapper, got: %s", body)
		}
		if !strings.Contains(body, "Count: 2") {
			t.Errorf("expected the counter to have advanced to 2, got: %s", body)
		}
		if !strings.Contains(body, "Last updated: 2 increments") {
			t.Errorf("expected StatusBadge to reflect count 2, got: %s", body)
		}
	})

	// Step 4: POST /increment WITH HX-Request: true again (second htmx
	// call on the SAME shared session) — this is the single most important
	// assertion in this commit. Without RenderSession, RenderFragment
	// would build a brand-new *StyleCollector on every call and re-emit
	// both <style> blocks every single time (see engine.go's
	// RenderFragmentContext, which always starts from a fresh collector).
	// Because handleIncrement instead threads the one shared s.sess through
	// RenderFragmentSession, and s.sess.styleEmitted already advanced past
	// both components' style contributions on step 3, neither block may
	// appear again here — while the actual rendered content must still be
	// correct.
	t.Run("second htmx increment on the same session has no style blocks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/increment", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("HX-Trigger"); got != "counter-updated" {
			t.Errorf(`HX-Trigger = %q, want "counter-updated"`, got)
		}

		body := rec.Body.String()
		t.Logf("second htmx call body:\n%s", body)

		if strings.Contains(body, "<style>") {
			t.Errorf("expected NO <style> block on the second call on the same session (both components' styles were already emitted on the first call), got: %s", body)
		}
		if !strings.Contains(body, `id="counter"`) {
			t.Errorf("expected the Counter fragment's own markup still present, got: %s", body)
		}
		if !strings.Contains(body, "Count: 3") {
			t.Errorf("expected the counter to have advanced to 3, got: %s", body)
		}
		if !strings.Contains(body, `<div id="status-badge" hx-swap-oob="true">`) {
			t.Errorf("expected the OOB wrapper still present, got: %s", body)
		}
		if !strings.Contains(body, "Last updated: 3 increments") {
			t.Errorf("expected StatusBadge to reflect count 3, got: %s", body)
		}
	})
}

// TestConcurrentIncrements is a concurrency smoke test: it fires several
// concurrent POST /increment requests (with HX-Request: true) via
// goroutines against a running httptest.NewServer (a real network
// round-trip through net/http's own concurrent handler dispatch, not
// direct in-process handler calls, which could mask a genuine data race)
// and confirms the final counter value equals the expected total — no lost
// increments from an unguarded shared counter, and no corruption of the
// shared *htmlc.RenderSession. Run this test with `go test -race`.
func TestConcurrentIncrements(t *testing.T) {
	s, err := newServer()
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	if errs := s.engine.ValidateAll(); len(errs) != 0 {
		t.Fatalf("ValidateAll() = %v, want zero errors", errs)
	}

	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/increment", nil)
			if err != nil {
				errCh <- fmt.Errorf("build request: %w", err)
				return
			}
			req.Header.Set("HX-Request", "true")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errCh <- fmt.Errorf("do request: %w", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errCh <- fmt.Errorf("read body: %w", err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("status = %d, want 200; body: %s", resp.StatusCode, body)
				return
			}
			if resp.Header.Get("HX-Trigger") != "counter-updated" {
				errCh <- fmt.Errorf("HX-Trigger = %q, want %q", resp.Header.Get("HX-Trigger"), "counter-updated")
				return
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent request: %v", err)
	}

	s.mu.Lock()
	got := s.count
	s.mu.Unlock()

	if got != n {
		t.Fatalf("final s.count = %d, want %d (no lost increments across %d concurrent requests)", got, n, n)
	}
}
