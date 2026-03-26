package main

import (
	"bytes"
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
	engine, err := htmlc.New(htmlc.Options{
		FS:           os.DirFS("components"),
		ComponentDir: ".",
	})
	if err != nil {
		log.Fatalf("init engine: %v", err)
	}

	collector, err := engine.CollectCustomElements()
	if err != nil {
		log.Fatalf("collect custom elements: %v", err)
	}

	indexJS := collector.IndexJS("/scripts/")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := engine.RenderPageWithCollector(r.Context(), &buf, "DashboardPage", nil, collector); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		buf.WriteTo(w)
	})

	scriptHandler := http.StripPrefix("/scripts/", htmlc.NewScriptFSServer(collector))
	http.Handle("/scripts/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.URL.Path, "/scripts/") == "index.js" {
			w.Header().Set("Content-Type", "text/javascript")
			io.WriteString(w, indexJS)
			return
		}
		scriptHandler.ServeHTTP(w, r)
	}))

	http.HandleFunc("/api/shapes/stream", streamShapes)

	addr := ":8081"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
