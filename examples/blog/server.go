package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dhamidi/htmlc"
)

// Server wires together a Store, an htmlc engine, and HTTP route handlers.
type Server struct {
	store  *Store
	engine *htmlc.Engine
}

// NewServer creates a Server, loading .vue templates from templateDir.
func NewServer(store *Store, templateDir string) (*Server, error) {
	engine, err := htmlc.New(htmlc.Options{ComponentDir: templateDir})
	if err != nil {
		return nil, fmt.Errorf("server: init engine: %w", err)
	}
	return &Server{store: store, engine: engine}, nil
}

// Routes returns an http.Handler serving all blog routes.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleListPosts)
	mux.HandleFunc("GET /posts/new", s.handleNewPostForm)
	mux.HandleFunc("POST /posts/new", s.handleCreatePost)
	mux.HandleFunc("GET /posts/{id}", s.handleGetPost)
	mux.HandleFunc("GET /posts/{id}/edit", s.handleEditPostForm)
	mux.HandleFunc("POST /posts/{id}/edit", s.handleUpdatePost)
	return mux
}

func (s *Server) renderPage(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.engine.RenderPage(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleListPosts(w http.ResponseWriter, r *http.Request) {
	posts := s.store.List()
	items := make([]any, len(posts))
	for i, p := range posts {
		items[i] = map[string]any{
			"ID":    p.ID,
			"Title": p.Title,
			"Body":  p.Body,
		}
	}
	s.renderPage(w, "PostListPage", map[string]any{"posts": items})
}

func (s *Server) handleNewPostForm(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, "PostFormPage", map[string]any{
		"pageTitle":   "New Post",
		"action":      "/posts/new",
		"submitLabel": "Create",
		"post":        map[string]any{"Title": "", "Body": ""},
	})
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.store.Create(r.FormValue("title"), r.FormValue("body"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleGetPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	post, ok := s.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.renderPage(w, "PostDetailPage", map[string]any{
		"post": map[string]any{"ID": post.ID, "Title": post.Title, "Body": post.Body},
	})
}

func (s *Server) handleEditPostForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	post, ok := s.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.renderPage(w, "PostFormPage", map[string]any{
		"pageTitle":   "Edit Post",
		"action":      fmt.Sprintf("/posts/%d/edit", post.ID),
		"submitLabel": "Update",
		"post":        map[string]any{"Title": post.Title, "Body": post.Body},
	})
}

func (s *Server) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_, ok := s.store.Update(id, r.FormValue("title"), r.FormValue("body"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
