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
	"path/filepath"
	"strings"
	"time"

	"github.com/alcmoraes/go-rom-downloader/sources"
	"github.com/alcmoraes/go-rom-downloader/utils"
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
		"romsDir":      romsDir,
		"port":         serverPort,
	})
}

func handleOrganize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Run post-processing script with no arguments to scan downloadsDir and romsDir
	scriptPath := "/app/post_process.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "./post_process.py"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			http.Error(w, `{"error":"Post-processing script not found"}`, http.StatusInternalServerError)
			return
		}
	}

	cmd := exec.Command("python3", scriptPath)
	env := os.Environ()
	env = append(env, fmt.Sprintf("DOWNLOADS_DIR=%s", downloadsDir))
	env = append(env, fmt.Sprintf("ROMS_DIR=%s", romsDir))
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	log.Printf("[ORGANIZATION TERMINAL OUTPUT]\n%s", string(output))

	if err != nil {
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

type FileItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
}

type FilesResponse struct {
	DownloadsDir string     `json:"downloadsDir"`
	RomsDir      string     `json:"romsDir"`
	Downloads    []FileItem `json:"downloads"`
	Roms         []FileItem `json:"roms"`
}

func listFolderFiles(dirPath string) []FileItem {
	items := make([]FileItem, 0)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return items
	}

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == dirPath {
			return nil
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			relPath = path
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		items = append(items, FileItem{
			Name:    d.Name(),
			Path:    relPath,
			Size:    info.Size(),
			IsDir:   d.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		return nil
	})

	if err != nil {
		log.Printf("Error walking directory %s: %v", dirPath, err)
	}

	return items
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	downloadsList := listFolderFiles(downloadsDir)
	romsList := listFolderFiles(romsDir)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FilesResponse{
		DownloadsDir: downloadsDir,
		RomsDir:      romsDir,
		Downloads:    downloadsList,
		Roms:         romsList,
	})
}

type DeleteFileReq struct {
	Path string `json:"path"`
}

func handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req DeleteFileReq
	if r.Method == http.MethodDelete {
		req.Path = r.URL.Query().Get("path")
	}
	if req.Path == "" && r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if req.Path == "" {
		http.Error(w, `{"error":"path parameter is required"}`, http.StatusBadRequest)
		return
	}

	absDownloads, err := filepath.Abs(downloadsDir)
	if err != nil {
		absDownloads = downloadsDir
	}

	absRoms, err := filepath.Abs(romsDir)
	if err != nil {
		absRoms = romsDir
	}

	cleanRel := filepath.Clean(req.Path)
	targetPathDl := filepath.Clean(filepath.Join(absDownloads, cleanRel))
	targetPathRom := filepath.Clean(filepath.Join(absRoms, cleanRel))

	var targetPath string
	if _, err := os.Stat(targetPathDl); err == nil {
		targetPath = targetPathDl
	} else if _, err := os.Stat(targetPathRom); err == nil {
		targetPath = targetPathRom
	} else {
		targetPath = targetPathDl
	}

	relDl, errDl := filepath.Rel(absDownloads, targetPath)
	inDl := (errDl == nil && !strings.HasPrefix(relDl, "..") && relDl != "." && targetPath != absDownloads)

	relRom, errRom := filepath.Rel(absRoms, targetPath)
	inRom := (errRom == nil && !strings.HasPrefix(relRom, "..") && relRom != "." && targetPath != absRoms)

	if !inDl && !inRom {
		http.Error(w, `{"error":"Invalid path or access denied outside downloads and roms directory"}`, http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		http.Error(w, `{"error":"File does not exist"}`, http.StatusNotFound)
		return
	}

	log.Printf("[FILE DELETE] Removing file/folder: %s", targetPath)
	if err := os.RemoveAll(targetPath); err != nil {
		log.Printf("[FILE DELETE FAILED] Error removing %s: %v", targetPath, err)
		http.Error(w, fmt.Sprintf(`{"error":"Failed to delete file: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "File deleted successfully",
		"path":    req.Path,
	})
}

func handleExtractRoms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[EXTRACT] Triggered ROM extraction in ROMs directory: %s", romsDir)
	results, err := utils.ExtractRomsInDir(romsDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to extract ROMs: %v", err),
		})
		return
	}

	successCount := 0
	for _, res := range results {
		if res.Status == "success" {
			successCount++
		}
	}

	if successCount > 0 {
		log.Printf("[EXTRACT] Extracted %d archives. Running post-processing organization...", successCount)
		runPostProcessing(romsDir, "")
	}

	msg := fmt.Sprintf("Extracted %d archive(s) successfully and verified completeness.", successCount)
	if len(results) == 0 {
		msg = "No compressed archives found in ROMs directory."
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"message":        msg,
		"extractedCount": successCount,
		"results":        results,
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
	mux.HandleFunc("POST /api/extract", handleExtractRoms)
	mux.HandleFunc("POST /api/roms/extract", handleExtractRoms)
	mux.HandleFunc("GET /api/files", handleFiles)
	mux.HandleFunc("/api/files/delete", handleDeleteFile)
	mux.HandleFunc("DELETE /api/files", handleDeleteFile)
	mux.HandleFunc("POST /api/files/delete", handleDeleteFile)

	addr := ":" + port
	log.Printf("== GO ROM DOWNLOADER WEB SERVER ==")
	log.Printf("Listening on %s", addr)
	log.Printf("Downloads directory: %s", downloadsDir)
	log.Printf("ROMs directory: %s", romsDir)
	log.Printf("Open http://localhost:%s in your browser", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
