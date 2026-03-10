package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dhamidi/htmlc"
)

func pageData() map[string]any {
	return map[string]any{
		"title": "My Blog",
		"links": []any{
			map[string]any{"url": "/", "label": "Home"},
			map[string]any{"url": "/about", "label": "About"},
		},
		"posts": []any{
			map[string]any{"title": "First Post", "body": "Hello world!"},
			map[string]any{"title": "Second Post", "body": "Another entry."},
		},
	}
}

func newEngine() (*htmlc.Engine, error) {
	return htmlc.New(htmlc.Options{ComponentDir: "components"})
}

func main() {
	engine, err := newEngine()
	if err != nil {
		log.Fatalf("init engine: %v", err)
	}

	if len(os.Args) > 1 && os.Args[1] == "print" {
		out, err := engine.RenderFragmentString("HomePage", pageData())
		if err != nil {
			log.Fatalf("render error: %v", err)
		}
		fmt.Println(out)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		out, err := engine.RenderFragmentString("HomePage", pageData())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<!DOCTYPE html><html><head></head><body>%s</body></html>", out)
	})

	addr := ":8080"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
