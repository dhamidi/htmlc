package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dhamidi/htmlc"
)

//go:embed templates
var templateFS embed.FS

// Server wires together a Store, an htmlc engine, and HTTP route handlers.
type Server struct {
	store    *Store
	engine   *htmlc.Engine
	cfg      Config
	sessions sync.Map // token string -> username string
}

// NewServer creates a Server, loading .vue templates from the embedded FS.
func NewServer(store *Store, cfg Config) (*Server, error) {
	engine, err := htmlc.New(htmlc.Options{
		ComponentDir: "templates",
		FS:           templateFS,
	})
	if err != nil {
		return nil, fmt.Errorf("server: init engine: %w", err)
	}
	return &Server{store: store, engine: engine, cfg: cfg}, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.routes().ServeHTTP(w, r)
}

// routes builds and returns the HTTP mux.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Catch-all for 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.renderNotFound(w, r)
	})

	// Public routes
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /posts/{slug}", s.handlePost)
	mux.HandleFunc("GET /tags/{tag}", s.handleTag)
	mux.HandleFunc("GET /archive", s.handleArchive)
	mux.HandleFunc("GET /about", s.handleAbout)
	mux.HandleFunc("GET /feed.atom", s.handleFeed)

	// Admin auth
	mux.HandleFunc("GET /admin/login", s.handleLoginForm)
	mux.HandleFunc("POST /admin/login", s.handleLogin)
	mux.HandleFunc("POST /admin/logout", s.handleLogout)

	// Admin pages (protected)
	mux.HandleFunc("GET /admin/{$}", s.requireAdmin(s.handleDashboard))
	mux.HandleFunc("GET /admin/drafts", s.requireAdmin(s.handleDrafts))
	mux.HandleFunc("GET /admin/posts/new", s.requireAdmin(s.handleNewPostForm))
	mux.HandleFunc("POST /admin/posts/new", s.requireAdmin(s.handleCreatePost))
	mux.HandleFunc("POST /admin/posts/preview", s.requireAdmin(s.handlePreviewPost))
	mux.HandleFunc("GET /admin/posts/{id}/edit", s.requireAdmin(s.handleEditPostForm))
	mux.HandleFunc("POST /admin/posts/{id}/edit", s.requireAdmin(s.handleUpdatePost))
	mux.HandleFunc("POST /admin/posts/{id}/publish", s.requireAdmin(s.handlePublishPost))
	mux.HandleFunc("POST /admin/posts/{id}/unpublish", s.requireAdmin(s.handleUnpublishPost))
	mux.HandleFunc("POST /admin/posts/{id}/delete", s.requireAdmin(s.handleDeletePost))

	return mux
}

// renderPage renders a full HTML page component with the given status code.
func (s *Server) renderPage(w http.ResponseWriter, status int, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.engine.RenderPage(context.Background(), w, name, data); err != nil {
		// Headers already sent; log the error but don't try to write another status.
		log.Printf("renderPage %s: %v", name, err)
	}
}

// renderNotFound renders the 404 page.
func (s *Server) renderNotFound(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, http.StatusNotFound, "NotFoundPage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
	})
}

// isAuthenticated returns true if the request carries a valid session cookie.
func (s *Server) isAuthenticated(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err != nil {
		return false
	}
	_, ok := s.sessions.Load(c.Value)
	return ok
}

// requireAdmin wraps a handler with admin authentication.
func (s *Server) requireAdmin(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthenticated(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// generateToken creates a random 32-character hex session token.
func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

// postToMap converts a Post to a map[string]any for template rendering.
func postToMap(p *Post) map[string]any {
	return map[string]any{
		"ID":          p.ID,
		"Title":       p.Title,
		"Slug":        p.Slug,
		"Tags":        p.Tags,
		"Body":        p.Body,
		"BodyHTML":    renderMarkdown(p.Body),
		"ExcerptHTML": renderExcerptHTML(p.Body),
		"ReadingTime": p.ReadingTime,
		"Published":   p.Published,
		"CreatedAt":   p.CreatedAt.Format("2 Jan 2006"),
		"PublishedAt": p.PublishedAt.Format("2 Jan 2006"),
		"Impressions": p.Impressions,
	}
}

// postsToSlice converts a slice of Posts to []any for template rendering.
func postsToSlice(posts []*Post) []any {
	items := make([]any, len(posts))
	for i, p := range posts {
		items[i] = postToMap(p)
	}
	return items
}

// parseTags splits a comma-separated tags string into a slice.
func parseTags(s string) []string {
	var tags []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// paginate returns the current page, total pages, and the slice for that page.
func paginate(posts []*Post, r *http.Request, size int) (page, totalPages int, result []*Post) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	totalPages = (len(posts) + size - 1) / size
	if totalPages == 0 {
		totalPages = 1
	}
	start := (page - 1) * size
	end := start + size
	if start >= len(posts) {
		return page, totalPages, nil
	}
	if end > len(posts) {
		end = len(posts)
	}
	return page, totalPages, posts[start:end]
}

// copyQuery returns a shallow copy of url.Values.
func copyQuery(q url.Values) url.Values {
	nq := make(url.Values)
	for k, v := range q {
		nq[k] = v
	}
	return nq
}

// buildPagination builds a pagination map for template rendering.
func buildPagination(page, totalPages int, base string, q url.Values) map[string]any {
	nextURL := ""
	prevURL := ""
	if page < totalPages {
		nq := copyQuery(q)
		nq.Set("page", strconv.Itoa(page+1))
		nextURL = base + "?" + nq.Encode()
	}
	if page > 1 {
		pq := copyQuery(q)
		pq.Set("page", strconv.Itoa(page-1))
		prevURL = base + "?" + pq.Encode()
	}
	return map[string]any{
		"Page":       page,
		"TotalPages": totalPages,
		"NextURL":    nextURL,
		"PrevURL":    prevURL,
	}
}

// ── Public handlers ──────────────────────────────────────────────────────────

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	posts := s.store.ListPublished()
	page, totalPages, paginated := paginate(posts, r, 10)
	s.renderPage(w, http.StatusOK, "IndexPage", map[string]any{
		"siteTitle":  s.cfg.SiteTitle,
		"posts":      postsToSlice(paginated),
		"pagination": buildPagination(page, totalPages, "/", r.URL.Query()),
	})
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var post *Post
	var ok bool
	// Try numeric ID first for backwards compat
	if id, err := strconv.Atoi(slug); err == nil {
		post, ok = s.store.Get(id)
		if ok && post.Published && post.Slug != "" {
			http.Redirect(w, r, "/posts/"+post.Slug, http.StatusMovedPermanently)
			return
		}
	}
	post, ok = s.store.GetBySlug(slug)
	if !ok || !post.Published {
		http.NotFound(w, r)
		return
	}
	s.store.RecordImpression(post.ID)
	s.renderPage(w, http.StatusOK, "PostPage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
		"post":      postToMap(post),
	})
}

func (s *Server) handleTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	posts := s.store.ListByTag(tag)
	page, totalPages, paginated := paginate(posts, r, 10)
	s.renderPage(w, http.StatusOK, "TagPage", map[string]any{
		"siteTitle":  s.cfg.SiteTitle,
		"tag":        tag,
		"posts":      postsToSlice(paginated),
		"pagination": buildPagination(page, totalPages, "/tags/"+tag, r.URL.Query()),
	})
}

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	groups := s.store.ListArchive()
	var archiveData []any
	for _, g := range groups {
		archiveData = append(archiveData, map[string]any{
			"Label": g.Label,
			"Posts": postsToSlice(g.Posts),
		})
	}
	s.renderPage(w, http.StatusOK, "ArchivePage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
		"groups":    archiveData,
	})
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	content := s.cfg.AboutHTML
	if s.cfg.AboutFile != "" {
		if data, err := os.ReadFile(s.cfg.AboutFile); err == nil {
			content = string(data)
		}
	}
	if content == "" {
		content = `<p>This blog is powered by <a href="https://github.com/dhamidi/htmlc">htmlc</a>.</p>`
	}
	s.renderPage(w, http.StatusOK, "AboutPage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
		"content":   content,
	})
}

// ── Atom feed ────────────────────────────────────────────────────────────────

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	NS      string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	Link    atomLink    `xml:"link"`
	Updated string      `xml:"updated"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}

type atomEntry struct {
	Title   string   `xml:"title"`
	Link    atomLink `xml:"link"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Summary string   `xml:"summary"`
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	posts := s.store.ListPublished()
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	base := scheme + "://" + r.Host

	updated := time.Now().UTC().Format(time.RFC3339)
	if len(posts) > 0 {
		updated = posts[0].PublishedAt.UTC().Format(time.RFC3339)
	}

	feed := atomFeed{
		NS:      "http://www.w3.org/2005/Atom",
		Title:   s.cfg.SiteTitle,
		Link:    atomLink{Href: base + "/feed.atom", Rel: "self"},
		Updated: updated,
	}
	for _, p := range posts {
		postURL := fmt.Sprintf("%s/posts/%s", base, p.Slug)
		feed.Entries = append(feed.Entries, atomEntry{
			Title:   p.Title,
			Link:    atomLink{Href: postURL},
			ID:      postURL,
			Updated: p.PublishedAt.UTC().Format(time.RFC3339),
			Summary: renderExcerptHTML(p.Body),
		})
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	xml.NewEncoder(w).Encode(feed) //nolint:errcheck
}

// ── Admin auth handlers ───────────────────────────────────────────────────────

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	if s.isAuthenticated(r) {
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
		return
	}
	s.renderPage(w, http.StatusOK, "LoginPage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
		"error":     "",
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username != s.cfg.AdminUsername || password != s.cfg.AdminPassword {
		s.renderPage(w, http.StatusOK, "LoginPage", map[string]any{
			"siteTitle": s.cfg.SiteTitle,
			"error":     "Invalid username or password.",
		})
		return
	}
	token := generateToken()
	s.sessions.Store(token, username)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session"); err == nil {
		s.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// ── Admin page handlers ───────────────────────────────────────────────────────

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	posts := s.store.ListAll()
	s.renderPage(w, http.StatusOK, "DashboardPage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
		"posts":     postsToSlice(posts),
	})
}

func (s *Server) handleDrafts(w http.ResponseWriter, r *http.Request) {
	all := s.store.ListAll()
	var drafts []*Post
	for _, p := range all {
		if !p.Published {
			drafts = append(drafts, p)
		}
	}
	s.renderPage(w, http.StatusOK, "DraftsPage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
		"posts":     postsToSlice(drafts),
	})
}

func (s *Server) handleNewPostForm(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, http.StatusOK, "PostFormPage", map[string]any{
		"siteTitle":   s.cfg.SiteTitle,
		"pageTitle":   "New Post",
		"action":      "/admin/posts/new",
		"submitLabel": "Create Draft",
		"post":        map[string]any{"Title": "", "Slug": "", "Tags": []string{}, "Body": "", "Published": false},
	})
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	body := r.FormValue("body")
	slug := r.FormValue("slug")
	tags := parseTags(r.FormValue("tags"))
	if slug == "" {
		slug = slugify(title)
	}
	post := s.store.Create(title, body, slug, tags)
	if r.FormValue("publish") == "1" {
		s.store.Publish(post.ID)
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (s *Server) handlePreviewPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	body := r.FormValue("body")
	tags := parseTags(r.FormValue("tags"))
	fake := &Post{
		Title:       title,
		Body:        body,
		Tags:        tags,
		ReadingTime: readingTime(body),
		PublishedAt: time.Now(),
	}
	s.renderPage(w, http.StatusOK, "PostPreviewPage", map[string]any{
		"siteTitle": s.cfg.SiteTitle,
		"post":      postToMap(fake),
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
	s.renderPage(w, http.StatusOK, "PostFormPage", map[string]any{
		"siteTitle":   s.cfg.SiteTitle,
		"pageTitle":   "Edit Post",
		"action":      fmt.Sprintf("/admin/posts/%d/edit", post.ID),
		"submitLabel": "Save Changes",
		"post":        postToMap(post),
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
	title := r.FormValue("title")
	body := r.FormValue("body")
	slug := r.FormValue("slug")
	tags := parseTags(r.FormValue("tags"))
	if !s.store.Update(id, title, body, slug, tags) {
		http.NotFound(w, r)
		return
	}
	if r.FormValue("publish") == "1" {
		s.store.Publish(id)
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (s *Server) handlePublishPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.store.Publish(id) {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (s *Server) handleUnpublishPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.store.Unpublish(id) {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (s *Server) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.store.Delete(id) {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}
