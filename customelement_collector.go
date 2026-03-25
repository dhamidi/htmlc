package htmlc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"testing/fstest"
)

// CustomElementCollector accumulates custom element scripts encountered
// during a render pass. It is safe for use by a single goroutine only.
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

// ScriptsFS returns an fs.FS where each file is named "<hash>.js" and
// contains the corresponding script source.
func (c *CustomElementCollector) ScriptsFS() fs.FS {
	mapFS := make(fstest.MapFS, len(c.scripts))
	for hash, src := range c.scripts {
		mapFS[hash+".js"] = &fstest.MapFile{
			Data: []byte(src),
		}
	}
	return mapFS
}

// ImportMapJSON returns a JSON string suitable for embedding in
// <script type="importmap">…</script>.
// Each entry maps the tag name to the script URL: urlPrefix + "<hash>.js".
// The urlPrefix should end with "/" (e.g. "/scripts/").
func (c *CustomElementCollector) ImportMapJSON(urlPrefix string) string {
	imports := make(map[string]string, len(c.order))
	for _, e := range c.order {
		imports[e.Tag] = urlPrefix + e.Hash + ".js"
	}
	data, _ := json.Marshal(map[string]any{"imports": imports})
	return string(data)
}

// NewScriptFSServer returns an http.Handler that serves the scripts in
// collector at paths of the form "/<hash>.js" with Content-Type: text/javascript.
func NewScriptFSServer(collector *CustomElementCollector) http.Handler {
	return http.FileServerFS(collector.ScriptsFS())
}
