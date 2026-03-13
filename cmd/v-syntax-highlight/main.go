// v-syntax-highlight is an external directive executable for htmlc that
// syntax-highlights source code using the chroma library.
//
// # Usage
//
//	v-syntax-highlight [flags]
//
// # Flags
//
//	-formatter string   Chroma formatter: "html" (default), "terminal256"
//	-style    string    Chroma style name (default "monokai")
//	-inline             Emit inline <span> tags without wrapping <div class="highlight">
//	-print-css          Print CSS for the chosen style to stdout and exit
//
// When invoked with no flags it reads NDJSON from stdin and writes NDJSON to
// stdout, following the htmlc external directive protocol.
//
// # External directive protocol
//
// Each line of input is a JSON object with a "hook" field ("created" or
// "mounted"). Each line of output is a JSON response with the same "id".
//
// For the "created" hook the response may contain:
//   - "attrs"      — replacement attribute map for the element
//   - "inner_html" — HTML to replace the element's children
//
// For the "mounted" hook the response may contain:
//   - "html" — HTML to inject after the element (empty means nothing injected)
//
// # CSS
//
// The highlighted HTML uses CSS classes emitted by chroma. Generate a
// stylesheet for the chosen style with:
//
//	v-syntax-highlight -print-css -style monokai > assets/highlight.css
//
// Include that stylesheet in your page layout component.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

type request struct {
	Hook      string            `json:"hook"`
	ID        string            `json:"id"`
	Tag       string            `json:"tag"`
	Attrs     map[string]string `json:"attrs"`
	Text      string            `json:"text"`
	InnerHTML string            `json:"inner_html"`
	Binding   struct {
		Value   any    `json:"value"`
		RawExpr string `json:"raw_expr"`
		Arg     string `json:"arg"`
	} `json:"binding"`
}

type response struct {
	ID        string            `json:"id"`
	Tag       string            `json:"tag,omitempty"`
	Attrs     map[string]string `json:"attrs,omitempty"`
	InnerHTML string            `json:"inner_html,omitempty"`
	HTML      string            `json:"html,omitempty"`
	Error     string            `json:"error,omitempty"`
}

func main() {
	styleName := flag.String("style", "monokai", "chroma style name")
	printCSS := flag.Bool("print-css", false, "print CSS for the chosen style and exit")
	flag.String("formatter", "html", "chroma formatter: html or terminal256") // reserved for future use
	flag.Bool("inline", false, "emit inline <span> tags without wrapping <div class=\"highlight\">")
	flag.Parse()

	style := styles.Get(*styleName)
	if style == nil {
		style = styles.Fallback
	}
	formatter := chromahtml.New(chromahtml.WithClasses(true))

	if *printCSS {
		if err := formatter.WriteCSS(os.Stdout, style); err != nil {
			fmt.Fprintf(os.Stderr, "v-syntax-highlight: write css: %v\n", err)
			os.Exit(1)
		}
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			fmt.Fprintf(os.Stderr, "v-syntax-highlight: bad request: %v\n", err)
			continue
		}

		resp := processRequest(req, style, formatter)

		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "v-syntax-highlight: encode error: %v\n", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "v-syntax-highlight: read error: %v\n", err)
		os.Exit(1)
	}
}

func processRequest(req request, style *chroma.Style, formatter *chromahtml.Formatter) response {
	resp := response{ID: req.ID}

	switch req.Hook {
	case "created":
		lang := "text"
		if s, ok := req.Binding.Value.(string); ok && s != "" {
			lang = s
		}

		lexer := lexers.Get(lang)
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)

		var buf bytes.Buffer
		iter, err := lexer.Tokenise(nil, req.Text)
		if err != nil {
			resp.Error = fmt.Sprintf("tokenise: %v", err)
			return resp
		}
		if err := formatter.Format(&buf, style, iter); err != nil {
			resp.Error = fmt.Sprintf("format: %v", err)
			return resp
		}

		// Build updated attrs: preserve existing, add/merge class.
		attrs := make(map[string]string, len(req.Attrs)+1)
		for k, v := range req.Attrs {
			attrs[k] = v
		}
		langClass := "language-" + lang
		existing := attrs["class"]
		if !strings.Contains(existing, langClass) {
			if existing != "" {
				attrs["class"] = existing + " " + langClass
			} else {
				attrs["class"] = langClass
			}
		}

		resp.Attrs = attrs
		resp.InnerHTML = buf.String()

	case "mounted":
		resp.HTML = ""
	}

	return resp
}
