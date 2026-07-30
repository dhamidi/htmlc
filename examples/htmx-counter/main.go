// Command htmx-counter is a small, standalone example application proving
// out RFC 014 §6 Example 4 end-to-end: a real HTTP server combining the
// github.com/dhamidi/htmlc/hypermedia/htmx module with a single, shared
// *htmlc.RenderSession (RFC 014 §4.5) reused across every /increment call
// for the lifetime of the process.
//
// GET / always renders the full HomePage document, regardless of any
// header — this is the initial page load. POST /increment mirrors the
// RFC's own handleIncrement example: a non-htmx request falls back to a
// full-page render of the (now incremented) HomePage; an htmx request sets
// the HX-Trigger response header, renders the Counter fragment, and appends
// an hx-swap-oob="true" StatusBadge fragment — both rendered through the
// one shared *htmlc.RenderSession, so each component's <style> block is
// only ever written once for the life of the server, not once per request.
//
// This demo deliberately has no "print" CLI mode (unlike
// examples/slot-style-demo and examples/radix-demo): its whole point is a
// stateful counter mutated by repeated POSTs against one shared, in-memory
// server; there is no single static render worth dumping to stdout the way
// a one-shot page-print mode implies.
package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/dhamidi/htmlc"
	"github.com/dhamidi/htmlc/hypermedia/htmx"
)

// server holds the demo's process-lifetime state: the increment counter and
// the single *htmlc.RenderSession shared across every /increment call.
//
// *htmlc.RenderSession is documented (engine.go) as "safe for use by a
// single goroutine only" — but net/http runs a handler goroutine per
// in-flight request, so overlapping /increment requests would otherwise
// call RenderFragmentSession on the same session concurrently. mu guards
// both count and sess together: every access to either — including the
// render calls themselves, not just the plain increment — happens while mu
// is held, for the entire remainder of the handler. That is a deliberate,
// conservative choice: sess's internal bookkeeping (styleEmitted) and the
// counter value must both be read/advanced as one atomic step per request,
// or two overlapping requests could interleave their RenderFragmentSession
// calls against the same session and corrupt its style-dedup bookkeeping
// (see the package doc comment above the server struct's mu field... i.e.
// this comment). Holding mu across the response write as well means
// concurrent /increment requests are effectively serialized rather than
// merely counted correctly — acceptable for this demo (correctness over
// throughput), and see the report/README note on RenderSession's API
// ergonomics for HTTP-server use.
type server struct {
	engine *htmlc.Engine

	mu    sync.Mutex
	count int
	sess  *htmlc.RenderSession
}

func newEngine() (*htmlc.Engine, error) {
	return htmlc.New(htmlc.Options{ComponentDir: "components"})
}

func newServer() (*server, error) {
	engine, err := newEngine()
	if err != nil {
		return nil, err
	}
	return &server{
		engine: engine,
		sess:   engine.NewRenderSession(),
	}, nil
}

// handleIndex serves GET /: always a full page, regardless of any header.
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	count := s.count
	s.mu.Unlock()

	data := map[string]any{"title": "htmx-counter", "count": count}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.engine.RenderPage(r.Context(), w, "HomePage", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleIncrement serves POST /increment, mirroring RFC 014 §6 Example 4's
// handleIncrement exactly: the counter is incremented unconditionally, then
// branches on htmx.IsHTMXRequest.
//
// Non-htmx fallback: re-renders the full HomePage directly (matching the
// RFC's own pseudocode verbatim: `engine.RenderPage(r.Context(), w,
// "HomePage", data); return`) rather than issuing an HTTP redirect. This
// keeps the branch a single, direct render — exactly what a plain,
// JS-disabled <form method="post" action="/increment"> submission needs: a
// normal synchronous POST response showing the updated page. (A redirect
// would also work and avoids form-resubmission-on-refresh, but the RFC's
// own example renders directly, and doing the same here keeps this
// commit's behavior a literal, testable proof of that example rather than
// a variant of it.)
func (s *server) handleIncrement(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	data := map[string]any{"count": s.count}

	if !htmx.IsHTMXRequest(r) {
		data["title"] = "htmx-counter"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := s.engine.RenderPage(r.Context(), w, "HomePage", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	htmx.SetTrigger(w, "counter-updated")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := s.engine.RenderFragmentSession(r.Context(), w, "Counter", data, s.sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, `<div id="status-badge" hx-swap-oob="true">`)
	if err := s.engine.RenderFragmentSession(r.Context(), w, "StatusBadge", data, s.sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, `</div>`)
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("POST /increment", s.handleIncrement)
	return mux
}

func main() {
	s, err := newServer()
	if err != nil {
		log.Fatalf("init: %v", err)
	}

	// Validate at startup, per this project's documented convention: a
	// registration problem must fail fast, not surface later as a
	// request-time surprise.
	if errs := s.engine.ValidateAll(); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("validate: %v", e)
		}
		log.Fatalf("engine validation failed with %d error(s)", len(errs))
	}

	addr := ":8080"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}
