package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// splitEvents splits a recorded SSE body into its individual
// "datastar-patch-elements" events, exactly mirroring
// hypermedia/datastar/datastar_test.go's own splitEvents: every event ends
// with datastar-go's DoubleNewLine ("\n\n") blank-line separator on top of
// the trailing newline already present after its last "data: " line, i.e.
// three consecutive newlines ("\n\n\n") between one event's end and the
// next event's "event: " line (and after the very last event). Splitting on
// "\n\n\n" and dropping the trailing empty tail recovers exactly the
// sequence of raw events written to the wire.
func splitEvents(body string) []string {
	parts := strings.Split(body, "\n\n\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// testServer builds a *server with a fast, deterministic, test-sized tick
// configuration: 3 ticks, 1ms apart — small enough to keep the test fast,
// but still genuinely exercising the bounded loop in handleCounterStream
// (as opposed to, say, tickCount: 0, which would prove nothing about the
// loop actually iterating).
func testServer(t *testing.T) *server {
	t.Helper()
	s, err := newServer()
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	if errs := s.engine.ValidateAll(); len(errs) != 0 {
		t.Fatalf("ValidateAll() = %v, want zero errors", errs)
	}
	s.tickCount = 3
	s.tickDelay = time.Millisecond
	return s
}

// TestHandleIndex_FullPageWithInitialState proves GET / renders a full
// document containing the #ds-counter target div required by RFC 014 §6
// Example 6 (the exact selector handleCounterStream's PatchElementsFragment
// calls target), with the initial count.
func TestHandleIndex_FullPageWithInitialState(t *testing.T) {
	s := testServer(t)
	mux := s.routes()

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
	if !strings.Contains(body, `id="ds-counter"`) {
		t.Errorf(`expected the #ds-counter target div present, got: %s`, body)
	}
	if !strings.Contains(body, "Count: 0") {
		t.Errorf("expected the initial count 0, got: %s", body)
	}
}

// TestHandleCounterStream_HandlerReturnsSynchronously confirms (before the
// main assertions below rely on it) that handleCounterStream genuinely
// returns once its bounded loop completes, rather than hanging — this is
// what determines whether the test below can be a simple, synchronous
// httptest.NewRecorder() call (it can) or needs a goroutine/channel/timeout
// dance instead. This test fails by timing out (the test binary itself would
// hang past `go test`'s default timeout) if handleCounterStream ever blocks
// past its configured ticks, which is a deliberately blunt but effective
// way to catch a regression that turns the handler into an infinite loop.
func TestHandleCounterStream_HandlerReturnsSynchronously(t *testing.T) {
	s := testServer(t)
	mux := s.routes()

	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/counter-stream", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-done:
		// handler returned on its own — exactly the synchronous behavior
		// this demo's design relies on.
	case <-time.After(5 * time.Second):
		t.Fatal("handleCounterStream did not return within 5s; the bounded loop appears to hang")
	}
}

// TestHandleCounterStream_ThreeTicksStyleOnlyOnFirst is the capstone
// assertion of this entire example (RFC 014 §6 Example 6, the real,
// end-to-end version of hypermedia/datastar's own
// TestPatchElementsFragment_StyleDedup_ThreeCalls, now exercised through an
// actual GET /counter-stream HTTP handler instead of calling
// PatchElementsFragment directly): after the handler returns, the recorded
// SSE body must contain exactly 3 "datastar-patch-elements" events; only the
// first carries Counter's <style scoped> block; the second and third carry
// no <style tag at all; and each event's rendered count increments 1, 2, 3
// in order, matching the RFC's own loop variable.
func TestHandleCounterStream_ThreeTicksStyleOnlyOnFirst(t *testing.T) {
	s := testServer(t)
	mux := s.routes()

	req := httptest.NewRequest(http.MethodGet, "/counter-stream", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
	}

	body := rec.Body.String()
	// Self-adversarial review: actually look at the raw SSE wire output,
	// not just strings.Contains assertions on it.
	t.Logf("raw SSE body:\n%s", body)

	events := splitEvents(body)
	if len(events) != 3 {
		t.Fatalf("got %d SSE events, want exactly 3; body=%q", len(events), body)
	}

	for i, ev := range events {
		if !strings.HasPrefix(ev, "event: datastar-patch-elements\n") {
			t.Errorf("event %d: unexpected framing, got %q", i, ev)
		}
		if !strings.Contains(ev, "data: selector #ds-counter\n") {
			t.Errorf("event %d: expected selector data line for #ds-counter, got %q", i, ev)
		}
		wantCount := i + 1
		if !strings.Contains(ev, "Count: "+strconv.Itoa(wantCount)) {
			t.Errorf("event %d: expected rendered count %d, got %q", i, wantCount, ev)
		}
	}

	if !strings.Contains(events[0], "<style") {
		t.Errorf("event 0 (first tick): expected the <style scoped> block present, got %q", events[0])
	}
	for i := 1; i < 3; i++ {
		if strings.Contains(events[i], "<style") {
			t.Errorf("event %d: expected NO <style tag at all (already emitted on tick 1), got %q", i, events[i])
		}
	}
}

// TestHandleCounterStream_GenuineIncrementalDelivery is the bonus test RFC
// 014 §1.3 asks for: proof that GET /counter-stream's events actually arrive
// on the wire incrementally, spaced out over time as each tick fires, rather
// than being buffered by net/http and flushed all at once when the handler
// returns. Unlike the tests above (which read the whole response only after
// ServeHTTP returns, via httptest.NewRecorder — adequate for content
// assertions, but blind to timing since NewRecorder has nothing resembling a
// real streaming socket), this test uses a real httptest.NewServer plus a
// real net/http client reading the response body line-by-line as bytes
// arrive, timestamping the moment each "event: datastar-patch-elements"
// line is observed.
//
// tickDelay is set well above typical scheduling/network jitter (60ms) and
// the assertion threshold (20ms) leaves a generous 3x margin below that —
// chosen after running this exact test 8 consecutive times during
// development (see this commit's report) and observing consistent ~60ms
// gaps every time, with -race enabled too. If this starts flaking in a
// slower or more heavily loaded CI environment, widening tickDelay and/or
// lowering the threshold further preserves the same proof without chasing a
// tighter bound than the property being tested actually needs.
func TestHandleCounterStream_GenuineIncrementalDelivery(t *testing.T) {
	s, err := newServer()
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	if errs := s.engine.ValidateAll(); len(errs) != 0 {
		t.Fatalf("ValidateAll() = %v, want zero errors", errs)
	}
	s.tickCount = 3
	s.tickDelay = 60 * time.Millisecond

	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/counter-stream")
	if err != nil {
		t.Fatalf("GET /counter-stream: %v", err)
	}
	defer resp.Body.Close()

	var arrivals []time.Time
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "event: datastar-patch-elements") {
			arrivals = append(arrivals, time.Now())
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading SSE stream: %v", err)
	}

	if len(arrivals) != 3 {
		t.Fatalf("observed %d event arrivals, want 3", len(arrivals))
	}

	const minGap = 20 * time.Millisecond
	for i := 1; i < len(arrivals); i++ {
		gap := arrivals[i].Sub(arrivals[i-1])
		t.Logf("inter-event gap %d: %v (configured tickDelay=%v)", i, gap, s.tickDelay)
		if gap < minGap {
			t.Errorf("gap between event %d and %d was only %v (< %v); events appear to have arrived buffered together rather than incrementally", i-1, i, gap, minGap)
		}
	}
}
