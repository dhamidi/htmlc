package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

type request struct {
	ID      string         `json:"id"`
	Hook    string         `json:"hook"`
	Binding map[string]any `json:"binding"`
}

type response struct {
	ID        string         `json:"id"`
	Attrs     map[string]any `json:"attrs,omitempty"`
	InnerHTML string         `json:"inner_html,omitempty"`
	HTML      string         `json:"html,omitempty"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		var resp response
		resp.ID = req.ID
		switch req.Hook {
		case "created":
			resp.Attrs = map[string]any{"data-echo": "true"}
			if val, ok := req.Binding["value"].(string); ok && strings.HasPrefix(val, "inner_html:") {
				resp.InnerHTML = strings.TrimPrefix(val, "inner_html:")
			}
		case "mounted":
			resp.HTML = "<!--mounted-->"
		}
		enc.Encode(resp)
	}
}
