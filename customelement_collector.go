package htmlc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing/fstest"
)

// CustomElementCollector accumulates custom element scripts encountered during
// rendering. It is managed automatically by the engine; callers rarely need to
// construct one directly.
//
// Key methods:
//
//   - ScriptsFS() – returns an in-memory fs.FS of content-hashed .js files,
//     one file per unique script collected (file name: "<tag>.<hash>.js").
//
//   - IndexJS() – returns a side-effecting ES-module entry point that imports
//     all collected scripts using relative paths ("./<tag>.<hash>.js"). Returns
//     an empty string when no scripts have been collected.
//
//   - ImportMapJSON(urlPrefix) – returns the JSON value suitable for embedding
//     in a <script type="importmap"> tag. Each entry maps the tag name to
//     urlPrefix + "<tag>.<hash>.js".
//
// It is safe for use by a single goroutine only.
type CustomElementCollector struct {
	// maps content-hash (hex) → script source
	scripts map[string]string
	// ordered list of (tag, hash) pairs in encounter order (for importmap)
	order []ceEntry
}

type ceEntry struct {
	Tag  string // e.g. "ui-date-picker"
	Hash string // hex content hash (first 16 chars of sha256)
}

// NewCustomElementCollector creates an empty CustomElementCollector.
func NewCustomElementCollector() *CustomElementCollector {
	return &CustomElementCollector{
		scripts: make(map[string]string),
	}
}

// contentHash returns the first 16 hex characters of the SHA-256 hash of s.
func contentHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:8]) // 8 bytes = 16 hex chars
}

// Add records a custom element script if not already seen (by content hash).
// If the same script content is added again (even under a different tag), it
// is deduplicated. If a new tag maps to already-seen content, only the first
// (tag, hash) entry in order is kept.
func (c *CustomElementCollector) Add(tag, script string) {
	hash := contentHash(script)
	if _, exists := c.scripts[hash]; exists {
		// Already recorded; check if this tag is already in order.
		for _, e := range c.order {
			if e.Tag == tag {
				return
			}
		}
		// New tag pointing to same script — add order entry for import map.
		c.order = append(c.order, ceEntry{Tag: tag, Hash: hash})
		return
	}
	c.scripts[hash] = script
	c.order = append(c.order, ceEntry{Tag: tag, Hash: hash})
}

// Len returns the number of unique scripts collected.
func (c *CustomElementCollector) Len() int {
	return len(c.scripts)
}

// scriptFilename returns the filename for a script given its tag and hash.
// The format is "<tag>.<hash>.js".
func scriptFilename(tag, hash string) string {
	return tag + "." + hash + ".js"
}

// hashToFirstTag builds a map from content hash to the first tag name
// encountered for that hash, based on the encounter order.
func (c *CustomElementCollector) hashToFirstTag() map[string]string {
	m := make(map[string]string, len(c.scripts))
	for _, e := range c.order {
		if _, exists := m[e.Hash]; !exists {
			m[e.Hash] = e.Tag
		}
	}
	return m
}

// ScriptsFS returns an fs.FS where each file is named "<tag>.<hash>.js" and
// contains the corresponding script source. When the same content is
// registered under multiple tags, the first tag encountered is used in the
// filename.
func (c *CustomElementCollector) ScriptsFS() fs.FS {
	hashToTag := c.hashToFirstTag()
	mapFS := make(fstest.MapFS, len(c.scripts))
	for hash, src := range c.scripts {
		name := scriptFilename(hashToTag[hash], hash)
		mapFS[name] = &fstest.MapFile{
			Data: []byte(src),
		}
	}
	return mapFS
}

// ImportMapJSON returns a JSON string suitable for embedding in
// <script type="importmap">…</script>.
// Each entry maps the tag name to the script URL: urlPrefix + "<tag>.<hash>.js".
// When the same content is registered under multiple tags, the first tag
// encountered determines the filename prefix.
// The urlPrefix should end with "/" (e.g. "/scripts/").
func (c *CustomElementCollector) ImportMapJSON(urlPrefix string) string {
	hashToTag := c.hashToFirstTag()
	imports := make(map[string]string, len(c.order))
	for _, e := range c.order {
		imports[e.Tag] = urlPrefix + scriptFilename(hashToTag[e.Hash], e.Hash)
	}
	data, _ := json.Marshal(map[string]any{"imports": imports})
	return string(data)
}

// IndexJS returns an ES module string with one import statement per collected
// script, in stable (encounter) order. Each line has the form:
//
//	import "./<tag>.<hash>.js"
//
// Duplicate hashes (same content added under different tags) are emitted only
// once. Returns an empty string when no scripts have been collected.
func (c *CustomElementCollector) IndexJS() string {
	if len(c.order) == 0 {
		return ""
	}
	hashToTag := c.hashToFirstTag()
	seen := make(map[string]bool, len(c.order))
	var sb strings.Builder
	for _, e := range c.order {
		if seen[e.Hash] {
			continue
		}
		seen[e.Hash] = true
		sb.WriteString(`import "./`)
		sb.WriteString(scriptFilename(hashToTag[e.Hash], e.Hash))
		sb.WriteString("\"\n")
	}
	return sb.String()
}

// NewScriptFSServer returns an http.Handler that serves the scripts in
// collector.
//
// Hashed .js files (e.g. "/abcd1234.js") are served from collector.ScriptsFS()
// via http.FileServerFS.
//
// When the request path (after stripping the leading "/") equals "index.js",
// the handler responds with collector.IndexJS() — a side-effecting ES module
// that imports all collected scripts using relative paths. index.js is
// regenerated on each request and carries no long-lived cache header.
//
// Typical mount:
//
//	http.Handle("/scripts/", http.StripPrefix("/scripts/", NewScriptFSServer(collector)))
func NewScriptFSServer(collector *CustomElementCollector) http.Handler {
	fileServer := http.FileServerFS(collector.ScriptsFS())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "index.js" {
			w.Header().Set("Content-Type", "application/javascript")
			io.WriteString(w, collector.IndexJS())
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
