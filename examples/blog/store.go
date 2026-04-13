package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Post represents a blog post with its current state.
type Post struct {
	ID          int
	Title       string
	Slug        string
	Tags        []string
	Body        string // raw Markdown
	Published   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt time.Time
	Impressions int
	ReadingTime int // minutes, computed at load time
}

// ArchiveGroup groups published posts by year and month.
type ArchiveGroup struct {
	Year  int
	Month time.Month
	Label string // e.g. "April 2025"
	Posts []*Post
}

// Event type constants.
const (
	evPostCreated     = "PostCreated"
	evPostUpdated     = "PostUpdated"
	evPostPublished   = "PostPublished"
	evPostUnpublished = "PostUnpublished"
	evPostDeleted     = "PostDeleted"
	evImpression      = "ImpressionRecorded"
)

// event is a single entry in the JSONL log.
type event struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// event payload structs
type postCreatedData struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Tags      []string  `json:"tags"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type postUpdatedData struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Tags      []string  `json:"tags"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type postPublishedData struct {
	ID          int       `json:"id"`
	PublishedAt time.Time `json:"publishedAt"`
}

type postIDData struct {
	ID int `json:"id"`
}

type impressionData struct {
	PostID int `json:"postID"`
}

// command types sent to the store goroutine.
type cmdCreate struct {
	title, body, slug string
	tags              []string
	resp              chan *Post
}

type cmdUpdate struct {
	id                int
	title, body, slug string
	tags              []string
	resp              chan bool
}

type cmdPublish struct {
	id   int
	resp chan bool
}

type cmdUnpublish struct {
	id   int
	resp chan bool
}

type cmdDelete struct {
	id   int
	resp chan bool
}

type cmdImpression struct {
	id   int
	resp chan bool
}

type cmdList struct {
	publishedOnly bool
	resp          chan []*Post
}

type cmdGet struct {
	id   int
	resp chan *Post
}

type cmdGetBySlug struct {
	slug string
	resp chan *Post
}

type cmdListByTag struct {
	tag  string
	resp chan []*Post
}

type cmdListArchive struct {
	resp chan []ArchiveGroup
}

// Store is an event-sourced store backed by a JSONL log file.
// All mutations are processed by a single goroutine to ensure consistency.
type Store struct {
	ch   chan any
	done chan struct{}
}

// NewStore opens (or creates) the JSONL log at logPath, replays all events to
// rebuild in-memory state, and starts the store goroutine.
func NewStore(logPath string) (*Store, error) {
	posts, nextID, err := replayLog(logPath)
	if err != nil {
		return nil, fmt.Errorf("store: replay: %w", err)
	}

	s := &Store{
		ch:   make(chan any, 64),
		done: make(chan struct{}),
	}
	go s.run(posts, nextID, logPath)
	return s, nil
}

// Close shuts down the store goroutine and waits for it to finish.
func (s *Store) Close() {
	close(s.ch)
	<-s.done
}

// replayLog reads all events from the JSONL file and returns the rebuilt state.
func replayLog(path string) (map[int]*Post, int, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return make(map[int]*Post), 1, nil
	}
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	posts := make(map[int]*Post)
	nextID := 1

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue // skip malformed lines
		}
		applyEvent(posts, &nextID, ev)
	}
	return posts, nextID, scanner.Err()
}

// applyEvent updates in-memory state for a single event.
func applyEvent(posts map[int]*Post, nextID *int, ev event) {
	switch ev.Type {
	case evPostCreated:
		var d postCreatedData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return
		}
		posts[d.ID] = &Post{
			ID:          d.ID,
			Title:       d.Title,
			Slug:        d.Slug,
			Tags:        d.Tags,
			Body:        d.Body,
			CreatedAt:   d.CreatedAt,
			UpdatedAt:   d.CreatedAt,
			ReadingTime: readingTime(d.Body),
		}
		if d.ID >= *nextID {
			*nextID = d.ID + 1
		}

	case evPostUpdated:
		var d postUpdatedData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return
		}
		if p, ok := posts[d.ID]; ok {
			p.Title = d.Title
			p.Slug = d.Slug
			p.Tags = d.Tags
			p.Body = d.Body
			p.UpdatedAt = d.UpdatedAt
			p.ReadingTime = readingTime(d.Body)
		}

	case evPostPublished:
		var d postPublishedData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return
		}
		if p, ok := posts[d.ID]; ok {
			p.Published = true
			p.PublishedAt = d.PublishedAt
		}

	case evPostUnpublished:
		var d postIDData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return
		}
		if p, ok := posts[d.ID]; ok {
			p.Published = false
		}

	case evPostDeleted:
		var d postIDData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return
		}
		delete(posts, d.ID)

	case evImpression:
		var d impressionData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return
		}
		if p, ok := posts[d.PostID]; ok {
			p.Impressions++
		}
	}
}

// run is the store goroutine. It processes commands and appends events to the log.
func (s *Store) run(posts map[int]*Post, nextID int, logPath string) {
	defer close(s.done)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		f = nil
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	appendEvent := func(evType string, data any) {
		if f == nil {
			return
		}
		raw, _ := json.Marshal(data)
		ev := event{Type: evType, Timestamp: time.Now(), Data: raw}
		line, _ := json.Marshal(ev)
		f.Write(line)         //nolint:errcheck
		f.Write([]byte("\n")) //nolint:errcheck
	}

	copyPost := func(p *Post) *Post {
		cp := *p
		if p.Tags != nil {
			cp.Tags = make([]string, len(p.Tags))
			copy(cp.Tags, p.Tags)
		}
		return &cp
	}

	// slugExists checks if a slug is already in use.
	slugExists := func(slug string) bool {
		for _, p := range posts {
			if p.Slug == slug {
				return true
			}
		}
		return false
	}

	for cmd := range s.ch {
		switch c := cmd.(type) {
		case cmdCreate:
			id := nextID
			nextID++
			now := time.Now()
			slug := uniqueSlug(c.slug, slugExists)
			p := &Post{
				ID:          id,
				Title:       c.title,
				Slug:        slug,
				Tags:        c.tags,
				Body:        c.body,
				CreatedAt:   now,
				UpdatedAt:   now,
				ReadingTime: readingTime(c.body),
			}
			posts[id] = p
			appendEvent(evPostCreated, postCreatedData{
				ID:        id,
				Title:     c.title,
				Slug:      slug,
				Tags:      c.tags,
				Body:      c.body,
				CreatedAt: now,
			})
			c.resp <- copyPost(p)

		case cmdUpdate:
			p, ok := posts[c.id]
			if !ok {
				c.resp <- false
				continue
			}
			now := time.Now()
			slug := c.slug
			if slug == "" {
				slug = p.Slug
			}
			// If slug changed, ensure uniqueness (excluding current post)
			if slug != p.Slug {
				slugExistsExcluding := func(s string) bool {
					for _, post := range posts {
						if post.ID != c.id && post.Slug == s {
							return true
						}
					}
					return false
				}
				slug = uniqueSlug(slug, slugExistsExcluding)
			}
			p.Title = c.title
			p.Slug = slug
			p.Tags = c.tags
			p.Body = c.body
			p.UpdatedAt = now
			p.ReadingTime = readingTime(c.body)
			appendEvent(evPostUpdated, postUpdatedData{
				ID:        c.id,
				Title:     c.title,
				Slug:      slug,
				Tags:      c.tags,
				Body:      c.body,
				UpdatedAt: now,
			})
			c.resp <- true

		case cmdPublish:
			p, ok := posts[c.id]
			if !ok {
				c.resp <- false
				continue
			}
			now := time.Now()
			p.Published = true
			p.PublishedAt = now
			appendEvent(evPostPublished, postPublishedData{ID: c.id, PublishedAt: now})
			c.resp <- true

		case cmdUnpublish:
			p, ok := posts[c.id]
			if !ok {
				c.resp <- false
				continue
			}
			p.Published = false
			appendEvent(evPostUnpublished, postIDData{ID: c.id})
			c.resp <- true

		case cmdDelete:
			if _, ok := posts[c.id]; !ok {
				c.resp <- false
				continue
			}
			delete(posts, c.id)
			appendEvent(evPostDeleted, postIDData{ID: c.id})
			c.resp <- true

		case cmdImpression:
			p, ok := posts[c.id]
			if !ok {
				c.resp <- false
				continue
			}
			p.Impressions++
			appendEvent(evImpression, impressionData{PostID: c.id})
			c.resp <- true

		case cmdList:
			var result []*Post
			for _, p := range posts {
				if c.publishedOnly && !p.Published {
					continue
				}
				result = append(result, copyPost(p))
			}
			sort.Slice(result, func(i, j int) bool {
				return result[i].ID > result[j].ID
			})
			c.resp <- result

		case cmdGet:
			p, ok := posts[c.id]
			if !ok {
				c.resp <- nil
				continue
			}
			c.resp <- copyPost(p)

		case cmdGetBySlug:
			var found *Post
			for _, p := range posts {
				if p.Slug == c.slug {
					found = copyPost(p)
					break
				}
			}
			c.resp <- found

		case cmdListByTag:
			var result []*Post
			for _, p := range posts {
				if !p.Published {
					continue
				}
				for _, t := range p.Tags {
					if t == c.tag {
						result = append(result, copyPost(p))
						break
					}
				}
			}
			sort.Slice(result, func(i, j int) bool {
				return result[i].PublishedAt.After(result[j].PublishedAt)
			})
			c.resp <- result

		case cmdListArchive:
			type groupKey struct {
				year  int
				month time.Month
			}
			keyOrder := []groupKey{}
			groups := map[groupKey][]*Post{}
			for _, p := range posts {
				if !p.Published {
					continue
				}
				key := groupKey{year: p.PublishedAt.Year(), month: p.PublishedAt.Month()}
				if _, exists := groups[key]; !exists {
					keyOrder = append(keyOrder, key)
				}
				groups[key] = append(groups[key], copyPost(p))
			}
			// Sort groups newest first
			sort.Slice(keyOrder, func(i, j int) bool {
				a, b := keyOrder[i], keyOrder[j]
				if a.year != b.year {
					return a.year > b.year
				}
				return a.month > b.month
			})
			var result []ArchiveGroup
			for _, key := range keyOrder {
				ps := groups[key]
				sort.Slice(ps, func(i, j int) bool {
					return ps[i].PublishedAt.After(ps[j].PublishedAt)
				})
				label := fmt.Sprintf("%s %d", time.Month(key.month).String(), key.year)
				result = append(result, ArchiveGroup{
					Year:  key.year,
					Month: key.month,
					Label: label,
					Posts: ps,
				})
			}
			c.resp <- result
		}
	}
}

// Create adds a new (unpublished) post and returns it.
func (s *Store) Create(title, body, slug string, tags []string) *Post {
	resp := make(chan *Post, 1)
	s.ch <- cmdCreate{title: title, body: body, slug: slug, tags: tags, resp: resp}
	return <-resp
}

// Update edits an existing post's title, body, slug, and tags. Returns false if not found.
func (s *Store) Update(id int, title, body, slug string, tags []string) bool {
	resp := make(chan bool, 1)
	s.ch <- cmdUpdate{id: id, title: title, body: body, slug: slug, tags: tags, resp: resp}
	return <-resp
}

// Publish marks a post as published. Returns false if not found.
func (s *Store) Publish(id int) bool {
	resp := make(chan bool, 1)
	s.ch <- cmdPublish{id: id, resp: resp}
	return <-resp
}

// Unpublish reverts a post to draft status. Returns false if not found.
func (s *Store) Unpublish(id int) bool {
	resp := make(chan bool, 1)
	s.ch <- cmdUnpublish{id: id, resp: resp}
	return <-resp
}

// Delete removes a post permanently. Returns false if not found.
func (s *Store) Delete(id int) bool {
	resp := make(chan bool, 1)
	s.ch <- cmdDelete{id: id, resp: resp}
	return <-resp
}

// RecordImpression increments the impression counter for a post.
func (s *Store) RecordImpression(id int) bool {
	resp := make(chan bool, 1)
	s.ch <- cmdImpression{id: id, resp: resp}
	return <-resp
}

// ListPublished returns all published posts sorted by ID descending.
func (s *Store) ListPublished() []*Post {
	resp := make(chan []*Post, 1)
	s.ch <- cmdList{publishedOnly: true, resp: resp}
	return <-resp
}

// ListAll returns all posts (published and drafts) sorted by ID descending.
func (s *Store) ListAll() []*Post {
	resp := make(chan []*Post, 1)
	s.ch <- cmdList{publishedOnly: false, resp: resp}
	return <-resp
}

// Get returns a post by ID and an ok flag.
func (s *Store) Get(id int) (*Post, bool) {
	resp := make(chan *Post, 1)
	s.ch <- cmdGet{id: id, resp: resp}
	p := <-resp
	return p, p != nil
}

// GetBySlug returns a post by slug and an ok flag.
func (s *Store) GetBySlug(slug string) (*Post, bool) {
	resp := make(chan *Post, 1)
	s.ch <- cmdGetBySlug{slug: slug, resp: resp}
	p := <-resp
	return p, p != nil
}

// ListByTag returns all published posts with the given tag, sorted by PublishedAt descending.
func (s *Store) ListByTag(tag string) []*Post {
	resp := make(chan []*Post, 1)
	s.ch <- cmdListByTag{tag: tag, resp: resp}
	return <-resp
}

// ListArchive returns published posts grouped by year/month, newest first.
func (s *Store) ListArchive() []ArchiveGroup {
	resp := make(chan []ArchiveGroup, 1)
	s.ch <- cmdListArchive{resp: resp}
	return <-resp
}
