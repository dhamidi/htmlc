// Command expvar-demo runs a small HTTP server that demonstrates the htmlc
// expvar integration. It exposes:
//
//   - GET /render            — renders the Hello component and returns HTML
//   - GET /admin/debug/on    — enables debug render mode
//   - GET /admin/debug/off   — disables debug render mode
//   - GET /admin/reload/on   — enables hot-reload
//   - GET /admin/reload/off  — disables hot-reload
//   - GET /debug/vars        — standard expvar endpoint (registered by import)
//
// The server listens on :9876. Engine metrics are published under the "htmlc"
// prefix and are visible at /debug/vars.
package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/dhamidi/htmlc"
)

func main() {
	// Locate the templates directory relative to this source file so the
	// demo works when run with "go run ./cmd/expvar-demo/" from any CWD.
	_, srcFile, _, _ := runtime.Caller(0)
	templatesDir := filepath.Join(filepath.Dir(srcFile), "templates")

	engine, err := htmlc.New(htmlc.Options{
		ComponentDir: templatesDir,
	})
	if err != nil {
		log.Fatalf("htmlc.New: %v", err)
	}
	engine.PublishExpvars("htmlc")

	http.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := engine.RenderPage(w, "Hello", nil); err != nil {
			http.Error(w, fmt.Sprintf("render error: %v", err), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/admin/debug/on", func(w http.ResponseWriter, r *http.Request) {
		engine.SetDebug(true)
		fmt.Fprintln(w, "debug enabled")
	})
	http.HandleFunc("/admin/debug/off", func(w http.ResponseWriter, r *http.Request) {
		engine.SetDebug(false)
		fmt.Fprintln(w, "debug disabled")
	})
	http.HandleFunc("/admin/reload/on", func(w http.ResponseWriter, r *http.Request) {
		engine.SetReload(true)
		fmt.Fprintln(w, "reload enabled")
	})
	http.HandleFunc("/admin/reload/off", func(w http.ResponseWriter, r *http.Request) {
		engine.SetReload(false)
		fmt.Fprintln(w, "reload disabled")
	})

	log.Println("expvar-demo listening on :9876")
	log.Println("  /render           — render Hello component")
	log.Println("  /debug/vars       — expvar metrics")
	log.Println("  /admin/debug/on   — enable debug mode")
	log.Println("  /admin/debug/off  — disable debug mode")
	log.Println("  /admin/reload/on  — enable hot-reload")
	log.Println("  /admin/reload/off — disable hot-reload")

	if err := http.ListenAndServe(":9876", nil); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
