package main

import (
	"log"
	"net/http"
	"os"
)

// Config holds runtime configuration read from environment variables.
type Config struct {
	Port          string
	AdminUsername string
	AdminPassword string
	SiteTitle     string
	LogFile       string
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	cfg := Config{
		Port:          getEnv("PORT", "8080"),
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "password"),
		SiteTitle:     getEnv("SITE_TITLE", "My Blog"),
		LogFile:       getEnv("LOG_FILE", "blog.jsonl"),
	}

	store, err := NewStore(cfg.LogFile)
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	srv, err := NewServer(store, cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	addr := ":" + cfg.Port
	log.Printf("listening on http://localhost%s", addr)
	log.Printf("admin: http://localhost%s/admin (user: %s)", addr, cfg.AdminUsername)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
