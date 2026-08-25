package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:embed static/index.html
var indexHTML []byte

//go:embed static/app.js
var appJS []byte

//go:embed static/style.css
var styleCSS []byte

const addr = "127.0.0.1:5039"

type submitResult struct {
	cancelled bool
	query     string
	selected  []string
}

type filterRequest struct {
	Query string `json:"query"`
	Seq   int    `json:"seq"`
}

type filterResponse struct {
	Matches []string `json:"matches"`
	Total   int      `json:"total"`
	Seq     int      `json:"seq"`
}

type submitRequest struct {
	Action   string   `json:"action"`
	Query    string   `json:"query"`
	Selected []string `json:"selected"`
}

// runWeb reads the candidate list from stdin, serves the checkbox UI, and
// prints fzf's expected output (the query line, then selected items) once
// the user pushes or cancels.
func runWeb(realFzf string, exact bool) int {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fzfweb: could not read candidate list: "+err.Error())
		return 1
	}
	items := splitLines(string(data))

	known := make(map[string]bool, len(items))
	for _, item := range items {
		known[item] = true
	}

	resultCh := make(chan submitResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/app.js", handleAppJS)
	mux.HandleFunc("/style.css", handleStyleCSS)
	mux.HandleFunc("/filter", handleFilter(realFzf, items, exact))
	mux.HandleFunc("/submit", handleSubmit(known, resultCh))

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fzfweb: could not bind "+addr+": "+err.Error())
		return 1
	}

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	url := "http://" + addr + "/"
	fmt.Fprintln(os.Stderr, "fzfweb: open "+url)
	openBrowser(url)

	result := <-resultCh

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

	if result.cancelled {
		return 0
	}

	fmt.Println(result.query)
	for _, item := range result.selected {
		fmt.Println(item)
	}
	return 0
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func handleAppJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(appJS)
}

func handleStyleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(styleCSS)
}

func handleFilter(realFzf string, items []string, exact bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		var req filterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request body", http.StatusBadRequest)
			return
		}

		matches, err := filter(realFzf, items, req.Query, exact)
		if err != nil {
			http.Error(w, "filter failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filterResponse{
			Matches: matches,
			Total:   len(items),
			Seq:     req.Seq,
		})
	}
}

func handleSubmit(known map[string]bool, resultCh chan submitResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		var req submitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request body", http.StatusBadRequest)
			return
		}

		var result submitResult
		if req.Action == "cancel" {
			result = submitResult{cancelled: true}
		} else {
			valid := make([]string, 0, len(req.Selected))
			for _, item := range req.Selected {
				if known[item] {
					valid = append(valid, item)
				}
			}
			result = submitResult{query: req.Query, selected: valid}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		select {
		case resultCh <- result:
		default:
			// Already submitted once (e.g. a double click); ignore.
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func openBrowser(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	_ = cmd.Start()
}
