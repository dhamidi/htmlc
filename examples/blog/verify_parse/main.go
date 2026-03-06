package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dhamidi/htmlc"
)

func main() {
	dir := "examples/blog/templates"
	files := []string{"Layout.vue", "PostList.vue", "PostCard.vue", "PostForm.vue"}
	ok := true
	for _, f := range files {
		path := filepath.Join(dir, f)
		_, err := htmlc.ParseFile(path, mustRead(path))
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", f, err)
			ok = false
		} else {
			fmt.Printf("PASS %s\n", f)
		}
	}
	if !ok {
		os.Exit(1)
	}
}

func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}
