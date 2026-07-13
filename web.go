package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/alcmoraes/go-rom-downloader/sources"
)

//go:embed static/*
var staticFS embed.FS

// DownloadReq represents the JSON payload to trigger background ROM downloads.
type DownloadReq struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Console string `json:"console"`
	URL     string `json:"url"`
}

func handleSources(w http.ResponseWriter, r *http.Request) {
	sourcesMap := make([]string, 0, len(sources.RomSources))
	for k := range sources.RomSources {
		sourcesMap = append(sourcesMap, k)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sourcesMap)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	sourceName := r.URL.Query().Get("source")
	query := r.URL.Query().Get("query")

	if sourceName == "" || query == "" {
		http.Error(w, `{"error":"source and query parameters are required"}`, http.StatusBadRequest)
		return
	}

	source := sources.LoadSource(sourceName, nil)
	if source == nil {
		http.Error(w, fmt.Sprintf(`{"error":"source '%s' not found"}`, sourceName), http.StatusBadRequest)
		return
	}

	escapedQuery := url.QueryEscape(query)
	roms := source.Lookup(escapedQuery)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roms)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req DownloadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Source == "" || req.Name == "" || req.URL == "" {
		http.Error(w, `{"error":"source, name, and url are required"}`, http.StatusBadRequest)
		return
	}

	taskID := fmt.Sprintf("dl_%d", time.Now().UnixNano())
	task := &DownloadTask{
		ID:       taskID,
		Name:     req.Name,
		Console:  req.Console,
		Status:   "queued",
		AddedAt:  time.Now(),
	}

	downloadsMu.Lock()
	downloads = append(downloads, task)
	downloadsMu.Unlock()

	go startBackgroundDownload(task, req.Source, req.URL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "started",
		"downloadId": taskID,
	})
}

func handleDownloads(w http.ResponseWriter, r *http.Request) {
	downloadsMu.RLock()
	defer downloadsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(downloads)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"downloadsDir": downloadsDir,
		"port":         serverPort,
	})
}

func handleOrganize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Run post-processing script with no arguments to scan downloadsDir
	scriptPath := "/app/post_process.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "./post_process.py"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			http.Error(w, `{"error":"Post-processing script not found"}`, http.StatusInternalServerError)
			return
		}
	}

	cmd := exec.Command("python3", scriptPath)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Manual organization failed: %v, Output: %s", err, string(output))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   fmt.Sprintf("Failed to run organization: %v", err),
			"details": string(output),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Loose files organized successfully.",
		"output":  string(output),
	})
}

// runWebServer bootstraps the net/http multiplexer and starts listening.
func runWebServer(port string) {
	mux := http.NewServeMux()

	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := staticFS.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	mux.HandleFunc("GET /api/sources", handleSources)
	mux.HandleFunc("GET /api/search", handleSearch)
	mux.HandleFunc("POST /api/download", handleDownload)
	mux.HandleFunc("GET /api/downloads", handleDownloads)
	mux.HandleFunc("GET /api/config", handleConfig)
	mux.HandleFunc("POST /api/organize", handleOrganize)

	addr := ":" + port
	log.Printf("== GO ROM DOWNLOADER WEB SERVER ==")
	log.Printf("Listening on %s", addr)
	log.Printf("Downloads directory: %s", downloadsDir)
	log.Printf("Open http://localhost:%s in your browser", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
