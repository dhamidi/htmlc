package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dhamidi/htmlc"
)

func newEngine() (*htmlc.Engine, error) {
	return htmlc.New(htmlc.Options{
		FS:           os.DirFS("components"),
		ComponentDir: ".",
	})
}

// collectScripts does a warm render to collect all custom element scripts.
func collectScripts(engine *htmlc.Engine) (*htmlc.CustomElementCollector, error) {
	collector := htmlc.NewCustomElementCollector()
	var buf strings.Builder
	if err := engine.RenderPageWithCollector(context.Background(), &buf, "DashboardPage", nil, collector); err != nil {
		return nil, err
	}
	return collector, nil
}

// buildIndexJS generates an ES module that imports each collected script.
func buildIndexJS(collector *htmlc.CustomElementCollector, urlPrefix string) string {
	raw := collector.ImportMapJSON(urlPrefix)
	var importMap struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(raw), &importMap); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, url := range importMap.Imports {
		fmt.Fprintf(&sb, "import %q\n", url)
	}
	return sb.String()
}

func renderDashboard(w http.ResponseWriter, r *http.Request, engine *htmlc.Engine, collector *htmlc.CustomElementCollector) {
	var buf bytes.Buffer
	c := htmlc.NewCustomElementCollector()
	if err := engine.RenderPageWithCollector(r.Context(), &buf, "DashboardPage", nil, c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	html := buf.String()
	if collector.Len() > 0 {
		importMapHTML := `<script type="importmap">` + collector.ImportMapJSON("/scripts/") + `</script>` +
			"\n" + `<script type="module" src="/scripts/index.js"></script>`
		html = strings.Replace(html, "</head>", importMapHTML+"\n</head>", 1)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, html)
}

var colors = []string{
	"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6",
	"#1abc9c", "#e67e22", "#34495e", "#e91e63", "#00bcd4",
}

func randomColor() string {
	return colors[rand.Intn(len(colors))]
}

type shape struct {
	Type  string `json:"type"`
	Color string `json:"color,omitempty"`
	X     int    `json:"x,omitempty"`
	Y     int    `json:"y,omitempty"`
	W     int    `json:"w,omitempty"`
	H     int    `json:"h,omitempty"`
	R     int    `json:"r,omitempty"`
}

func streamShapes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			var s shape
			if count > 0 && count%30 == 0 {
				s = shape{Type: "clear"}
			} else {
				shapeType := "rect"
				if rand.Intn(2) == 0 {
					shapeType = "circle"
				}
				s = shape{
					Type:  shapeType,
					Color: randomColor(),
					X:     rand.Intn(380),
					Y:     rand.Intn(280),
					W:     rand.Intn(80) + 20,
					H:     rand.Intn(80) + 20,
					R:     rand.Intn(40) + 10,
				}
			}
			data, _ := json.Marshal(s)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			count++
		}
	}
}

func main() {
	engine, err := newEngine()
	if err != nil {
		log.Fatalf("init engine: %v", err)
	}

	collector, err := collectScripts(engine)
	if err != nil {
		log.Fatalf("collect scripts: %v", err)
	}

	scriptServer := htmlc.NewScriptFSServer(collector)
	indexJS := buildIndexJS(collector, "/scripts/")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderDashboard(w, r, engine, collector)
	})

	http.Handle("/scripts/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/scripts/")
		if name == "index.js" {
			w.Header().Set("Content-Type", "text/javascript")
			io.WriteString(w, indexJS)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + name
		scriptServer.ServeHTTP(w, r2)
	}))

	http.HandleFunc("/api/shapes/stream", streamShapes)

	addr := ":8081"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
