// Command datastar-counter-sse is a small, standalone example application
// proving out RFC 014 §6 Example 6 end-to-end: a real HTTP server driving a
// genuine, long-lived text/event-stream connection through
// github.com/dhamidi/htmlc/hypermedia/datastar's PatchElementsFragment, on
// one shared *htmlc.RenderSession per connection, so the Counter component's
// <style scoped> block is only ever sent on the first tick of each stream.
//
// GET / serves the initial full-page load (engine.RenderPage) — a normal,
// static document containing the starting DOM (Counter at count 0) plus a
// <script> tag for the real Datastar client library, exactly how a real
// Datastar app bootstraps before the client-side library takes over via SSE
// (RFC 014 §1.3). GET /counter-stream is the SSE endpoint: it opens one
// *dsdatastar.ServerSentEventGenerator, allocates one *htmlc.RenderSession
// for the connection, and runs a small, explicitly bounded loop —
// s.tickCount ticks, s.tickDelay apart — calling PatchElementsFragment once
// per tick, mirroring the RFC's own handleCounterStream almost verbatim.
//
// tickCount/tickDelay are struct fields, not hardcoded literals, precisely
// so tests can drive the same handler with a tiny tickCount and a
// near-zero tickDelay (fast, deterministic) while production defaults match
// the RFC's own "1 to 3 ticks, 200ms apart" example.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/dhamidi/htmlc"
	htmlcdatastar "github.com/dhamidi/htmlc/hypermedia/datastar"
	dsdatastar "github.com/starfederation/datastar-go/datastar"
)

// server holds the demo's process-lifetime state: the engine used to render
// every page/fragment, and the tick parameters for GET /counter-stream's
// bounded loop.
type server struct {
	engine *htmlc.Engine

	// tickCount is the number of datastar-patch-elements events
	// GET /counter-stream sends before the handler returns and the
	// connection closes. Production default: 3, matching RFC 014 §6
	// Example 6's "for i := 1; i <= 3; i++" loop.
	tickCount int

	// tickDelay is the pause between ticks. Production default: 200ms,
	// matching the RFC's own time.Sleep(200 * time.Millisecond). Tests set
	// this to a much smaller value so the handler returns quickly.
	tickDelay time.Duration
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
		engine:    engine,
		tickCount: 3,
		tickDelay: 200 * time.Millisecond,
	}, nil
}

// handleIndex serves GET /: always a full page, the initial DOM the
// Datastar client library then enhances once its <script> tag loads (see
// components/HomePage.vue). The counter always starts at 0 here — every
// subsequent count the visitor ever sees comes exclusively from
// GET /counter-stream's server-pushed SSE patches, never from this handler.
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{"title": "datastar-counter-sse", "count": 0}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.engine.RenderPage(r.Context(), w, "HomePage", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleCounterStream serves GET /counter-stream, implementing RFC 014 §6
// Example 6's handleCounterStream loop almost verbatim: open one SSE
// generator, allocate one *htmlc.RenderSession for the life of this
// connection, then tick s.tickCount times, sleeping s.tickDelay between
// ticks, calling PatchElementsFragment once per tick with the loop's own 1-
// based counter as the new count. The loop is bounded and this handler is
// fully synchronous — it returns (and the connection closes) as soon as the
// configured number of ticks have been sent, with no background goroutine
// left running.
//
// A PatchElementsFragment error (for example, the client having disconnected
// mid-stream, surfaced through datastar-go's own write-error handling) stops
// the loop early and returns; there is no point continuing to render ticks
// nobody can receive.
func (s *server) handleCounterStream(w http.ResponseWriter, r *http.Request) {
	sse := dsdatastar.NewSSE(w, r)
	sess := s.engine.NewRenderSession()

	for i := 1; i <= s.tickCount; i++ {
		if err := htmlcdatastar.PatchElementsFragment(sse, s.engine, sess, r.Context(),
			"Counter", map[string]any{"count": i}, "#ds-counter"); err != nil {
			log.Printf("counter-stream tick %d: %v", i, err)
			return
		}
		if i < s.tickCount {
			time.Sleep(s.tickDelay)
		}
	}
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /counter-stream", s.handleCounterStream)
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
