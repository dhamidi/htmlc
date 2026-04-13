package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// renderMarkdown converts Markdown source to HTML using goldmark with GFM and syntax highlighting.
func renderMarkdown(src string) string {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
			),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return src
	}
	return buf.String()
}

// renderExcerptHTML returns rendered HTML for the excerpt of a Markdown document.
// If the source contains <!--more-->, everything before it is rendered.
// Otherwise, the first 300 runes of the source are rendered.
func renderExcerptHTML(markdown string) string {
	if idx := strings.Index(markdown, "<!--more-->"); idx >= 0 {
		return renderMarkdown(markdown[:idx])
	}
	runes := []rune(markdown)
	if len(runes) > 300 {
		runes = runes[:300]
	}
	return renderMarkdown(string(runes))
}

// readingTime estimates the reading time of a Markdown document in minutes.
func readingTime(markdown string) int {
	words := len(strings.Fields(markdown))
	mins := words / 200
	if mins < 1 {
		return 1
	}
	return mins
}

// slugify converts a title to a URL-safe slug.
func slugify(title string) string {
	s := strings.ToLower(title)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// uniqueSlug returns a slug that doesn't already exist, appending a numeric
// suffix if necessary.
func uniqueSlug(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !exists(candidate) {
			return candidate
		}
	}
}
