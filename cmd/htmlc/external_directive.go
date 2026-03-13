package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dhamidi/htmlc"
	"golang.org/x/net/html"
)

// externalDirective implements htmlc.Directive and htmlc.DirectiveWithContent
// for a directive backed by an external executable.
//
// The executable is spawned once at build start and communicates over
// newline-delimited JSON (NDJSON) on stdin/stdout.
type externalDirective struct {
	name   string    // directive name without "v-" prefix
	path   string    // absolute path to executable
	stderr io.Writer // where to forward subprocess stderr and warnings

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	mu          sync.Mutex // serialise requests
	nextID      uint64
	pendingHTML string // inner_html from last Created call
}

// start spawns the directive executable and wires up its stderr.
func (ed *externalDirective) start() error {
	cmd := exec.Command(ed.path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	// Forward subprocess stderr verbatim.
	cmd.Stderr = ed.stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	ed.cmd = cmd
	ed.stdin = stdin
	ed.stdout = bufio.NewScanner(stdout)
	return nil
}

// stop closes the directive's stdin and waits for it to exit.
// A non-zero exit code is treated as a warning, not an error.
func (ed *externalDirective) stop() {
	if ed.stdin != nil {
		ed.stdin.Close()
	}
	if ed.cmd != nil {
		if err := ed.cmd.Wait(); err != nil {
			fmt.Fprintf(ed.stderr, "htmlc: directive %q exited with error: %v\n", ed.name, err)
		}
	}
}

// request sends one NDJSON request and reads one NDJSON response.
// It is safe to call from multiple goroutines (serialised by mu).
func (ed *externalDirective) request(req map[string]any) (map[string]any, error) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	id := fmt.Sprintf("%d", atomic.AddUint64(&ed.nextID, 1))
	req["id"] = id

	line, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	line = append(line, '\n')

	if _, err := ed.stdin.Write(line); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	if !ed.stdout.Scan() {
		if err := ed.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("directive closed stdout unexpectedly")
	}

	var resp map[string]any
	if err := json.Unmarshal(ed.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// Verify echoed id.
	respID, _ := resp["id"].(string)
	if respID != id {
		return nil, fmt.Errorf("response id mismatch: want %q, got %q", id, respID)
	}

	return resp, nil
}

// extractTextContent recursively concatenates all text node descendants of n.
func extractTextContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c)
	}
	return b.String()
}

// attrsToMap converts a node's attribute slice to a map[string]any.
func attrsToMap(attrs []html.Attribute) map[string]any {
	m := make(map[string]any, len(attrs))
	for _, a := range attrs {
		m[a.Key] = a.Val
	}
	return m
}

// Created implements htmlc.Directive.
func (ed *externalDirective) Created(node *html.Node, binding htmlc.DirectiveBinding, ctx htmlc.DirectiveContext) error {
	modifiers := make(map[string]any, len(binding.Modifiers))
	for k, v := range binding.Modifiers {
		modifiers[k] = v
	}

	req := map[string]any{
		"hook":  "created",
		"tag":   node.Data,
		"attrs": attrsToMap(node.Attr),
		"text":  extractTextContent(node),
		"binding": map[string]any{
			"value":     binding.Value,
			"raw_expr":  binding.RawExpr,
			"arg":       binding.Arg,
			"modifiers": modifiers,
		},
	}

	resp, err := ed.request(req)
	if err != nil {
		fmt.Fprintf(ed.stderr, "htmlc: directive %q Created: %v\n", ed.name, err)
		return nil // treat as no-op, do not abort build
	}

	// Check for directive-level error.
	if errMsg, _ := resp["error"].(string); errMsg != "" {
		return fmt.Errorf("%s", errMsg)
	}

	// Apply tag change.
	if tag, _ := resp["tag"].(string); tag != "" {
		node.Data = tag
	}

	// Apply attrs replacement.
	if attrsRaw, ok := resp["attrs"]; ok && attrsRaw != nil {
		if attrsMap, ok := attrsRaw.(map[string]any); ok {
			var newAttrs []html.Attribute
			for k, v := range attrsMap {
				newAttrs = append(newAttrs, html.Attribute{Key: k, Val: fmt.Sprintf("%v", v)})
			}
			node.Attr = newAttrs
		}
	}

	// Store inner_html for InnerHTML().
	ed.pendingHTML, _ = resp["inner_html"].(string)

	return nil
}

// Mounted implements htmlc.Directive.
func (ed *externalDirective) Mounted(w io.Writer, node *html.Node, binding htmlc.DirectiveBinding, ctx htmlc.DirectiveContext) error {
	modifiers := make(map[string]any, len(binding.Modifiers))
	for k, v := range binding.Modifiers {
		modifiers[k] = v
	}

	req := map[string]any{
		"hook":  "mounted",
		"tag":   node.Data,
		"attrs": attrsToMap(node.Attr),
		"text":  extractTextContent(node),
		"binding": map[string]any{
			"value":     binding.Value,
			"raw_expr":  binding.RawExpr,
			"arg":       binding.Arg,
			"modifiers": modifiers,
		},
	}

	resp, err := ed.request(req)
	if err != nil {
		fmt.Fprintf(ed.stderr, "htmlc: directive %q Mounted: %v\n", ed.name, err)
		return nil // treat as no-op
	}

	// Check for directive-level error.
	if errMsg, _ := resp["error"].(string); errMsg != "" {
		return fmt.Errorf("%s", errMsg)
	}

	// Inject HTML after element.
	if injected, _ := resp["html"].(string); injected != "" {
		_, err := io.WriteString(w, injected)
		return err
	}

	return nil
}

// InnerHTML implements htmlc.DirectiveWithContent.
func (ed *externalDirective) InnerHTML() (string, bool) {
	h := ed.pendingHTML
	ed.pendingHTML = ""
	if h == "" {
		return "", false
	}
	return h, true
}
