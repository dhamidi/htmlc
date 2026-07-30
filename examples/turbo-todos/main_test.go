package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/dhamidi/htmlc/hypermedia/turbo"
)

// turboStreamRE matches one complete <turbo-stream> element as written by
// turbo.WriteStream: action and target as quoted attributes, and the
// wrapped fragment inside <template>...</template>. Used to parse the
// response body precisely rather than relying on loose substring checks —
// see TestPostTodos_TurboStreamShape's doc comment for why this matters.
var turboStreamRE = regexp.MustCompile(`(?s)<turbo-stream action="([^"]*)" target="([^"]*)"><template>(.*?)</template></turbo-stream>`)

// newTestServer builds a *server with an empty todo list (unlike
// newServer, which seeds two todos for a nicer GET / demo on first run) so
// tests that assert exact counts and IDs ("the count reflects 2 total
// todos") don't have to account for seed data.
func newTestServer(t *testing.T) *server {
	t.Helper()
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("newEngine: %v", err)
	}
	s := &server{engine: engine}
	if errs := s.engine.ValidateAll(); len(errs) != 0 {
		t.Fatalf("ValidateAll() = %v, want zero errors", errs)
	}
	return s
}

// postTodo issues a POST /todos with the given text as form-encoded body.
// If wantStream is true, it sets the Accept header Turbo Drive itself
// sends so turbo.WantsStream(r) is true inside the handler; otherwise it
// sends a plain browser-form Accept header so the non-Turbo fallback
// branch fires.
func postTodo(mux *http.ServeMux, text string, wantStream bool) *httptest.ResponseRecorder {
	form := url.Values{"text": {text}}
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if wantStream {
		req.Header.Set("Accept", turbo.StreamContentType+", text/html")
	} else {
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// TestGetIndex_ShowsInitialTodosAndCount proves GET / renders a full page
// showing the seeded todos and their count.
func TestGetIndex_ShowsInitialTodosAndCount(t *testing.T) {
	s, err := newServer()
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	if errs := s.engine.ValidateAll(); len(errs) != 0 {
		t.Fatalf("ValidateAll() = %v, want zero errors", errs)
	}
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
	if !strings.Contains(body, "Buy milk") {
		t.Errorf("expected the seeded todo %q, got: %s", "Buy milk", body)
	}
	if !strings.Contains(body, "Walk the dog") {
		t.Errorf("expected the seeded todo %q, got: %s", "Walk the dog", body)
	}
	if !strings.Contains(body, `id="todo-count"`) {
		t.Errorf("expected the #todo-count wrapper, got: %s", body)
	}
	if !strings.Contains(body, "2 todos") {
		t.Errorf("expected the initial count of 2 todos, got: %s", body)
	}
	if !strings.Contains(body, `id="todo-list"`) {
		t.Errorf("expected the #todo-list wrapper, got: %s", body)
	}
}

// TestPostTodos_TurboStreamShape is the central assertion of this commit:
// a single POST /todos response, when the request declares it wants a
// Turbo Stream (RFC 014 §6 Example 5), contains exactly two well-formed
// <turbo-stream> elements concatenated with no framing between them — one
// "append" onto #todo-list carrying the new TodoItem, one "update" onto
// #todo-count carrying the refreshed TodoCount.
//
// This parses the body with turboStreamRE rather than testing
// strings.Contains("turbo-stream") anywhere in it, per this commit's
// self-adversarial review requirement: a loose substring check could pass
// even if, say, the two actions were malformed, reordered incorrectly, or
// merged into one element. Asserting on the parsed action/target/content
// triples confirms the real shape RFC Example 5 describes.
func TestPostTodos_TurboStreamShape(t *testing.T) {
	s := newTestServer(t)
	mux := s.routes()

	rec := postTodo(mux, "Write RFC test", true)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != turbo.StreamContentType {
		t.Fatalf("Content-Type = %q, want exactly %q", got, turbo.StreamContentType)
	}

	body := rec.Body.String()
	t.Logf("POST /todos turbo-stream body:\n%s", body)

	matches := turboStreamRE.FindAllStringSubmatch(body, -1)
	if len(matches) != 2 {
		t.Fatalf("found %d <turbo-stream> elements, want exactly 2; body: %s", len(matches), body)
	}

	appendMatch, updateMatch := matches[0], matches[1]

	if action, target := appendMatch[1], appendMatch[2]; action != "append" || target != "todo-list" {
		t.Errorf("first turbo-stream: action=%q target=%q, want action=%q target=%q", action, target, "append", "todo-list")
	}
	if content := appendMatch[3]; !strings.Contains(content, "Write RFC test") {
		t.Errorf("first turbo-stream template does not contain the new todo's text, got: %s", content)
	}
	if content := appendMatch[3]; !strings.Contains(content, `id="todo-1"`) {
		t.Errorf("first turbo-stream template does not carry the new todo's own id (todo-1), got: %s", content)
	}

	if action, target := updateMatch[1], updateMatch[2]; action != "update" || target != "todo-count" {
		t.Errorf("second turbo-stream: action=%q target=%q, want action=%q target=%q", action, target, "update", "todo-count")
	}
	if content := updateMatch[3]; !strings.Contains(content, "1 todos") {
		t.Errorf("second turbo-stream template does not contain the updated count, got: %s", content)
	}

	// Confirm the two elements are truly back-to-back with no separator or
	// extra markup between them, matching WriteStream's documented
	// no-framing-required contract.
	firstEnd := strings.Index(body, "</turbo-stream>") + len("</turbo-stream>")
	secondStart := strings.Index(body[firstEnd:], "<turbo-stream")
	if secondStart != 0 {
		t.Errorf("expected the second <turbo-stream> to start immediately after the first with no framing, found %d bytes of separator: %q", secondStart, body[firstEnd:firstEnd+secondStart])
	}
}

// TestPostTodos_AccumulatesAcrossSequentialRequests confirms the shared
// store actually accumulates every created todo, not just the most recent
// one: two sequential POST /todos calls must leave the second response's
// TodoCount fragment reading "2 todos", and a follow-up GET / must show
// both todos' text, not just the second.
func TestPostTodos_AccumulatesAcrossSequentialRequests(t *testing.T) {
	s := newTestServer(t)
	mux := s.routes()

	first := postTodo(mux, "Task A", true)
	if first.Code != http.StatusOK {
		t.Fatalf("first POST status = %d, want 200; body: %s", first.Code, first.Body.String())
	}
	firstMatches := turboStreamRE.FindAllStringSubmatch(first.Body.String(), -1)
	if len(firstMatches) != 2 {
		t.Fatalf("first POST: found %d turbo-stream elements, want 2; body: %s", len(firstMatches), first.Body.String())
	}
	if content := firstMatches[1][3]; !strings.Contains(content, "1 todos") {
		t.Errorf("first POST: expected count 1 todos, got: %s", content)
	}

	second := postTodo(mux, "Task B", true)
	if second.Code != http.StatusOK {
		t.Fatalf("second POST status = %d, want 200; body: %s", second.Code, second.Body.String())
	}
	secondMatches := turboStreamRE.FindAllStringSubmatch(second.Body.String(), -1)
	if len(secondMatches) != 2 {
		t.Fatalf("second POST: found %d turbo-stream elements, want 2; body: %s", len(secondMatches), second.Body.String())
	}
	if action, target := secondMatches[0][1], secondMatches[0][2]; action != "append" || target != "todo-list" {
		t.Errorf("second POST first stream: action=%q target=%q, want append/todo-list", action, target)
	}
	if content := secondMatches[0][3]; !strings.Contains(content, "Task B") {
		t.Errorf("second POST: expected the appended item to contain %q, got: %s", "Task B", content)
	}
	if content := secondMatches[1][3]; !strings.Contains(content, "2 todos") {
		t.Errorf("second POST: expected count to reflect 2 total todos, got: %s", content)
	}

	// Confirm the store itself accumulated both, not just the last write:
	// a follow-up GET / must show both todos' text.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Task A") {
		t.Errorf("expected GET / to still show the first todo %q, got: %s", "Task A", body)
	}
	if !strings.Contains(body, "Task B") {
		t.Errorf("expected GET / to show the second todo %q, got: %s", "Task B", body)
	}
	if !strings.Contains(body, "2 todos") {
		t.Errorf("expected GET / to show the final count of 2 todos, got: %s", body)
	}
}

// TestPostTodos_NonTurboFallback confirms a client that does not declare
// it wants a Turbo Stream response (turbo.WantsStream(r) == false) gets
// the plain-POST fallback — a 303 redirect back to "/" — rather than a
// text/vnd.turbo-stream.html body it cannot use, and that the todo was
// still created despite taking the fallback branch.
func TestPostTodos_NonTurboFallback(t *testing.T) {
	s := newTestServer(t)
	mux := s.routes()

	rec := postTodo(mux, "Fallback task", false)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/" {
		t.Errorf("Location = %q, want %q", got, "/")
	}
	if got := rec.Header().Get("Content-Type"); got == turbo.StreamContentType {
		t.Errorf("Content-Type = %q, want NOT the Turbo Stream type on the fallback branch", got)
	}

	// The todo must have been created despite the fallback branch.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, req)
	body := getRec.Body.String()
	if !strings.Contains(body, "Fallback task") {
		t.Errorf("expected the todo created via the fallback branch to appear on GET /, got: %s", body)
	}
	if !strings.Contains(body, "1 todos") {
		t.Errorf("expected the count to reflect the one todo created via the fallback branch, got: %s", body)
	}
}

// TestConcurrentPostTodos is a concurrency smoke test: it fires several
// concurrent POST /todos requests (with the Turbo-Stream-accepting Accept
// header) via goroutines against a running httptest.NewServer (a real
// network round-trip through net/http's own concurrent handler dispatch,
// not direct in-process handler calls, which could mask a genuine data
// race) and confirms the final todo count equals the number of requests
// sent — no lost writes to the shared, mutex-guarded store. Run this test
// with `go test -race`.
func TestConcurrentPostTodos(t *testing.T) {
	s := newTestServer(t)

	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			form := url.Values{"text": {fmt.Sprintf("concurrent task %d", i)}}
			req, err := http.NewRequest(http.MethodPost, ts.URL+"/todos", strings.NewReader(form.Encode()))
			if err != nil {
				errCh <- fmt.Errorf("build request: %w", err)
				return
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept", turbo.StreamContentType+", text/html")

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
			if got := resp.Header.Get("Content-Type"); got != turbo.StreamContentType {
				errCh <- fmt.Errorf("Content-Type = %q, want %q", got, turbo.StreamContentType)
				return
			}
			if got := turboStreamRE.FindAllStringSubmatch(string(body), -1); len(got) != 2 {
				errCh <- fmt.Errorf("found %d turbo-stream elements, want 2; body: %s", len(got), body)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent request: %v", err)
	}

	s.mu.Lock()
	got := len(s.todos)
	s.mu.Unlock()

	if got != n {
		t.Fatalf("final len(s.todos) = %d, want %d (no lost writes across %d concurrent requests)", got, n, n)
	}
}
