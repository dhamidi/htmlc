package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

type request struct {
	ID   string `json:"id"`
	Hook string `json:"hook"`
	Text string `json:"text"`
}

type response struct {
	ID        string `json:"id"`
	InnerHTML string `json:"inner_html,omitempty"`
	HTML      string `json:"html,omitempty"`
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
		if req.Hook == "created" {
			resp.InnerHTML = strings.ToUpper(req.Text)
		} else {
			resp.HTML = ""
		}
		enc.Encode(resp)
	}
}
