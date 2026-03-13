package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// buildBinary compiles the binary into a temp dir and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := t.TempDir() + "/v-syntax-highlight"
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build binary: %v", err)
	}
	return bin
}

func defaultFormatter() *chromahtml.Formatter {
	return chromahtml.New(chromahtml.WithClasses(true))
}

// TestHighlightsGoCode checks that the created hook highlights Go code and
// populates inner_html and the correct class attribute.
func TestHighlightsGoCode(t *testing.T) {
	style := styles.Get("monokai")
	formatter := defaultFormatter()

	req := request{
		Hook: "created",
		ID:   "1",
		Tag:  "pre",
		Text: "func main(){}",
	}
	req.Binding.Value = "go"

	resp := processRequest(req, style, formatter)

	if resp.ID != "1" {
		t.Errorf("id: got %q, want %q", resp.ID, "1")
	}
	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
	if !strings.Contains(resp.InnerHTML, "<span") {
		t.Errorf("inner_html should contain <span, got: %s", resp.InnerHTML)
	}
	if !strings.Contains(resp.Attrs["class"], "language-go") {
		t.Errorf("class should contain language-go, got: %q", resp.Attrs["class"])
	}
}

// TestFallbackForUnknownLanguage checks that an unknown language falls back
// gracefully and returns non-empty output without error.
func TestFallbackForUnknownLanguage(t *testing.T) {
	style := styles.Get("monokai")
	formatter := defaultFormatter()

	req := request{
		Hook: "created",
		ID:   "2",
		Tag:  "pre",
		Text: "hello",
	}
	req.Binding.Value = "brainfuck42"

	resp := processRequest(req, style, formatter)

	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
	if resp.InnerHTML == "" {
		t.Error("inner_html should be non-empty for fallback language")
	}
}

// TestMountedHookReturnsEmptyHTML checks that the mounted hook responds with
// an empty html field.
func TestMountedHookReturnsEmptyHTML(t *testing.T) {
	style := styles.Get("monokai")
	formatter := defaultFormatter()

	req := request{
		Hook: "mounted",
		ID:   "3",
		Tag:  "pre",
	}
	req.Binding.Value = "go"

	resp := processRequest(req, style, formatter)

	if resp.ID != "3" {
		t.Errorf("id: got %q, want %q", resp.ID, "3")
	}
	if resp.HTML != "" {
		t.Errorf("html should be empty, got: %q", resp.HTML)
	}
}

// TestPrintCSSFlag verifies that -print-css writes non-empty CSS and exits 0.
func TestPrintCSSFlag(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "-print-css", "-style", "monokai")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("-print-css exited with error: %v\nstderr: %s", err, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("-print-css produced no output")
	}
	if !strings.Contains(stdout.String(), ".chroma") {
		t.Errorf("-print-css output should contain .chroma class, got: %.200s", stdout.String())
	}
}

// TestHighlightsHTMLWithMustacheSyntax verifies that HTML containing mustache
// template syntax is highlighted correctly and that {{ expressions }} are
// preserved as literal text rather than evaluated.
func TestHighlightsHTMLWithMustacheSyntax(t *testing.T) {
	style := styles.Get("monokai")
	formatter := defaultFormatter()

	req := request{
		Hook: "created",
		ID:   "10",
		Tag:  "pre",
		Text: "<span>{{ items.length }}</span>",
	}
	req.Binding.Value = "html"

	resp := processRequest(req, style, formatter)

	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
	if resp.InnerHTML == "" {
		t.Error("inner_html should be non-empty")
	}
	if !strings.Contains(resp.InnerHTML, "{{ items.length }}") {
		t.Errorf("inner_html should preserve mustache text, got: %s", resp.InnerHTML)
	}
}

// TestRequestParsesInnerHTML verifies that the inner_html field is correctly
// unmarshalled from a request JSON object into the InnerHTML field.
func TestRequestParsesInnerHTML(t *testing.T) {
	input := `{"hook":"created","id":"5","tag":"pre","text":"func main(){}","inner_html":"<code>func main(){}</code>","binding":{"value":"go"}}`
	var req request
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.InnerHTML != "<code>func main(){}</code>" {
		t.Errorf("InnerHTML = %q, want %q", req.InnerHTML, "<code>func main(){}</code>")
	}
	if req.Text != "func main(){}" {
		t.Errorf("Text = %q, want %q", req.Text, "func main(){}")
	}
}

// TestMalformedInputContinues verifies that a bad JSON line is skipped with a
// warning on stderr and that the process continues to handle valid lines.
func TestMalformedInputContinues(t *testing.T) {
	bin := buildBinary(t)
	input := "NOT JSON\n" +
		`{"hook":"mounted","id":"99","tag":"pre","binding":{"value":"go"}}` + "\n"

	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("process exited with error: %v\nstderr: %s", err, stderr.String())
	}

	// stderr should mention the bad request
	if !strings.Contains(stderr.String(), "bad request") {
		t.Errorf("expected 'bad request' in stderr, got: %s", stderr.String())
	}

	// stdout should contain exactly one valid JSON response for id=99
	scanner := bufio.NewScanner(&stdout)
	var responses []map[string]any
	for scanner.Scan() {
		var m map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &m); err == nil {
			responses = append(responses, m)
		}
	}
	if len(responses) != 1 {
		t.Errorf("expected 1 response, got %d", len(responses))
	} else if responses[0]["id"] != "99" {
		t.Errorf("expected id=99, got %v", responses[0]["id"])
	}
}
