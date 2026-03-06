package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	store := NewStore()
	srv, err := NewServer(store, "templates")
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv, srv.Routes()
}

func TestServer_ListEmpty(t *testing.T) {
	_, handler := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No posts yet.") {
		t.Errorf("expected empty-state message in body: %s", rec.Body.String())
	}
}

func TestServer_ListWithPosts(t *testing.T) {
	srv, handler := newTestServer(t)
	srv.store.Create("Alpha Post", "Body A")
	srv.store.Create("Beta Post", "Body B")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Alpha Post") {
		t.Errorf("expected 'Alpha Post' in body: %s", body)
	}
	if !strings.Contains(body, "Beta Post") {
		t.Errorf("expected 'Beta Post' in body: %s", body)
	}
}

func TestServer_NewForm(t *testing.T) {
	_, handler := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/posts/new", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<form") {
		t.Errorf("expected <form element in body: %s", rec.Body.String())
	}
}

func TestServer_CreatePost(t *testing.T) {
	_, handler := newTestServer(t)

	form := url.Values{"title": {"New Post"}, "body": {"New Body"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/posts/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec2, req2)

	if !strings.Contains(rec2.Body.String(), "New Post") {
		t.Errorf("expected new post title in list: %s", rec2.Body.String())
	}
}

func TestServer_GetPost_Found(t *testing.T) {
	srv, handler := newTestServer(t)
	srv.store.Create("Hello World", "This is the body.")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/posts/1", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello World") {
		t.Errorf("expected post title in body: %s", body)
	}
	if !strings.Contains(body, "This is the body.") {
		t.Errorf("expected post body in body: %s", body)
	}
	if !strings.Contains(body, `href="/"`) {
		t.Errorf("expected back link to / in body: %s", body)
	}
	if !strings.Contains(body, `href="/posts/1/edit"`) {
		t.Errorf("expected edit link in body: %s", body)
	}
}

func TestServer_GetPost_NotFound(t *testing.T) {
	_, handler := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/posts/999", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestServer_EditForm_Found(t *testing.T) {
	srv, handler := newTestServer(t)
	srv.store.Create("Edit Me", "Some Body")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/posts/1/edit", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Edit Me") {
		t.Errorf("expected post title pre-filled in form: %s", rec.Body.String())
	}
}

func TestServer_EditForm_NotFound(t *testing.T) {
	_, handler := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/posts/999/edit", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestServer_UpdatePost(t *testing.T) {
	srv, handler := newTestServer(t)
	srv.store.Create("Original", "Body")

	form := url.Values{"title": {"Updated Title"}, "body": {"Updated Body"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/posts/1/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec2, req2)

	if !strings.Contains(rec2.Body.String(), "Updated Title") {
		t.Errorf("expected updated title in list: %s", rec2.Body.String())
	}
}

func TestServer_UpdatePost_NotFound(t *testing.T) {
	_, handler := newTestServer(t)

	form := url.Values{"title": {"X"}, "body": {"Y"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/posts/999/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
