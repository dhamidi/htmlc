package main

import (
	"bufio"
	"encoding/json"
	"os"
)

type request struct {
	ID        string `json:"id"`
	InnerHTML string `json:"inner_html"`
}

type response struct {
	ID        string `json:"id"`
	InnerHTML string `json:"inner_html"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		enc.Encode(response{ID: req.ID, InnerHTML: req.InnerHTML})
	}
}
