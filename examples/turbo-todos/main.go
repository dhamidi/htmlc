// Command turbo-todos is a small, standalone example application proving
// out RFC 014 §6 Example 5 end-to-end: a real HTTP server whose POST
// /todos handler writes two <turbo-stream> actions into a single response
// via github.com/dhamidi/htmlc/hypermedia/turbo — appending a new TodoItem
// and updating the TodoCount summary — with no special framing between
// the two, exactly matching the RFC's own handleTodoCreate.
//
// GET / always renders the full HomePage document, listing whatever
// todos already exist in the in-memory store. POST /todos creates a new
// todo from form data and branches on turbo.WantsStream: a Turbo-aware
// client (one that sends "Accept: text/vnd.turbo-stream.html", which Turbo
// Drive does automatically for form submissions it intercepts) gets the
// two-action Turbo Stream response; any other client — a JS-disabled
// browser doing a plain synchronous form POST, for instance — gets a
// standard 303 redirect back to "/", showing the same new todo via a full
// page reload instead. This demonstrates both the request-side detection
// helper (WantsStream) and the response-side writer (WriteStream), not
// just the latter.
//
// This demo deliberately has no "print" CLI mode (unlike
// examples/slot-style-demo and examples/radix-demo): its whole point is a
// stateful todo list mutated by repeated POSTs against one shared,
// in-memory server; there is no single static render worth dumping to
// stdout the way a one-shot page-print mode implies.
package main

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/dhamidi/htmlc"
	"github.com/dhamidi/htmlc/hypermedia/turbo"
)

// Todo is one entry in the in-memory todo list. Fields are exported so
// component templates can read them directly (e.g. "todo.Text").
type Todo struct {
	ID   int
	Text string
	Done bool
}

// server holds the demo's process-lifetime state: the todo list and the
// counter used to assign each new todo's ID.
//
// Unlike examples/htmx-counter, this demo has no shared *htmlc.RenderSession
// to protect — every render below goes through engine.RenderFragmentString
// or engine.RenderPage directly, and *htmlc.Engine is documented (engine.go)
// as "safe for concurrent use. All render methods may be called from
// multiple goroutines simultaneously." mu therefore only needs to guard
// the todo slice and the ID counter themselves, not any render call.
type server struct {
	engine *htmlc.Engine

	mu     sync.Mutex
	todos  []Todo
	nextID int
}

func newEngine() (*htmlc.Engine, error) {
	return htmlc.New(htmlc.Options{ComponentDir: "components"})
}

func newServer() (*server, error) {
	engine, err := newEngine()
	if err != nil {
		return nil, err
	}
	s := &server{engine: engine}
	// Seed a couple of todos so GET / has something to show on first load.
	s.todos = []Todo{
		{ID: s.nextTodoID(), Text: "Buy milk"},
		{ID: s.nextTodoID(), Text: "Walk the dog", Done: true},
	}
	return s, nil
}

// nextTodoID must only be called while s.mu is held, or (as in newServer)
// before the server is reachable by any handler.
func (s *server) nextTodoID() int {
	s.nextID++
	return s.nextID
}

// snapshot returns a defensive copy of the current todo list and its
// length, taken under s.mu. Callers render from the copy after releasing
// the lock, so a render (which never needs to hold mu) cannot block
// concurrent POST /todos requests, and the returned slice is never
// aliased with s.todos.
func (s *server) snapshot() ([]Todo, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	todos := make([]Todo, len(s.todos))
	copy(todos, s.todos)
	return todos, len(todos)
}

// handleIndex serves GET /: always a full page, regardless of any header.
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	todos, count := s.snapshot()
	data := map[string]any{"title": "turbo-todos", "todos": todos, "count": count}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.engine.RenderPage(r.Context(), w, "HomePage", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleTodoCreate serves POST /todos. It always creates the todo from
// form data first, then branches on turbo.WantsStream — mirroring
// htmx-counter's handleIncrement branching on htmx.IsHTMXRequest, and RFC
// 014 §6 Example 5's own handleTodoCreate for the Turbo Stream branch
// itself.
//
// Turbo-aware branch: sets Content-Type to turbo.StreamContentType, then
// writes exactly two <turbo-stream> elements to w with no framing between
// them — first "append" the newly rendered TodoItem onto #todo-list, then
// "update" #todo-count with a freshly rendered TodoCount reflecting the
// new total. Both fragments are rendered via engine.RenderFragmentString,
// matching the RFC's own code (not a *htmlc.RenderSession — this demo has
// no cross-call style-dedup need, unlike htmx-counter's shared session).
//
// Non-Turbo fallback: a plain 303 redirect back to "/", so a normal
// <form method="post"> submission from a client that never asked for the
// Turbo Stream format still ends in a correct, reloaded page rather than a
// browser trying to render a text/vnd.turbo-stream.html body directly, and
// without the form-resubmission-on-refresh risk a direct 200 render would
// have.
func (s *server) handleTodoCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		http.Error(w, "text must not be empty", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	todo := Todo{ID: s.nextTodoID(), Text: text}
	s.todos = append(s.todos, todo)
	count := len(s.todos)
	s.mu.Unlock()

	if !turbo.WantsStream(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", turbo.StreamContentType)

	itemHTML, err := s.engine.RenderFragmentString(r.Context(), "TodoItem", map[string]any{"todo": todo})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := turbo.WriteStream(w, "append", "todo-list", itemHTML); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	countHTML, err := s.engine.RenderFragmentString(r.Context(), "TodoCount", map[string]any{"count": count})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := turbo.WriteStream(w, "update", "todo-count", countHTML); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("POST /todos", s.handleTodoCreate)
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
