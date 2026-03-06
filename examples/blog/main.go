package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	store := NewStore()

	// Locate the templates directory relative to this source file so the
	// binary works regardless of the working directory it is run from.
	_, file, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(filepath.Dir(file), "templates")

	srv, err := NewServer(store, templateDir)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
