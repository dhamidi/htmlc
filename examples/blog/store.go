package blog

import "time"

// Post represents a blog post.
type Post struct {
	ID        int
	Title     string
	Body      string
	CreatedAt time.Time
}

// Store is an in-memory store for blog posts.
type Store struct {
	posts  []*Post
	nextID int
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{nextID: 1}
}

// Create assigns the next ID, sets CreatedAt to now, appends to the store, and returns a pointer to the new post.
func (s *Store) Create(title, body string) *Post {
	p := &Post{
		ID:        s.nextID,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}
	s.nextID++
	s.posts = append(s.posts, p)
	return p
}

// List returns all posts in insertion order.
func (s *Store) List() []*Post {
	return s.posts
}

// Get returns a post by ID and an ok flag.
func (s *Store) Get(id int) (*Post, bool) {
	for _, p := range s.posts {
		if p.ID == id {
			return p, true
		}
	}
	return nil, false
}

// Update updates the title and body of an existing post and returns the updated post and ok flag.
// Returns (nil, false) if the post is not found.
func (s *Store) Update(id int, title, body string) (*Post, bool) {
	for _, p := range s.posts {
		if p.ID == id {
			p.Title = title
			p.Body = body
			return p, true
		}
	}
	return nil, false
}
