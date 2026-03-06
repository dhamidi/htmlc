package main

import "testing"

func TestStore_CreateAndList(t *testing.T) {
	s := NewStore()
	p1 := s.Create("First", "Body 1")
	p2 := s.Create("Second", "Body 2")

	posts := s.List()
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if posts[0].ID != p1.ID || posts[0].Title != "First" {
		t.Errorf("unexpected first post: %+v", posts[0])
	}
	if posts[1].ID != p2.ID || posts[1].Title != "Second" {
		t.Errorf("unexpected second post: %+v", posts[1])
	}
	if p1.ID >= p2.ID {
		t.Errorf("expected sequential IDs: p1.ID=%d, p2.ID=%d", p1.ID, p2.ID)
	}
}

func TestStore_Get_Found(t *testing.T) {
	s := NewStore()
	created := s.Create("Hello", "World")

	got, ok := s.Get(created.ID)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if got.Title != "Hello" || got.Body != "World" {
		t.Errorf("unexpected post: %+v", got)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	s := NewStore()
	got, ok := s.Get(999)
	if ok || got != nil {
		t.Errorf("expected (nil, false), got (%v, %v)", got, ok)
	}
}

func TestStore_Update_Found(t *testing.T) {
	s := NewStore()
	p := s.Create("Old Title", "Old Body")

	updated, ok := s.Update(p.ID, "New Title", "New Body")
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if updated.Title != "New Title" || updated.Body != "New Body" {
		t.Errorf("unexpected updated post: %+v", updated)
	}

	got, _ := s.Get(p.ID)
	if got.Title != "New Title" {
		t.Errorf("Get after Update returned wrong title: %s", got.Title)
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	s := NewStore()
	got, ok := s.Update(999, "X", "Y")
	if ok || got != nil {
		t.Errorf("expected (nil, false), got (%v, %v)", got, ok)
	}
}
