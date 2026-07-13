package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alcmoraes/go-rom-downloader/domains"
	"github.com/alcmoraes/go-rom-downloader/sources"
	"github.com/alcmoraes/go-rom-downloader/utils"
	"github.com/cavaliergopher/grab/v3"
)

// DownloadTask represents a background download task state.
type DownloadTask struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Console          string    `json:"console"`
	Progress         float64   `json:"progress"`
	BytesTransferred int64     `json:"bytesTransferred"`
	TotalSize        int64     `json:"totalSize"`
	Speed            float64   `json:"speed"`
	Status           string    `json:"status"` // "queued", "downloading", "completed", "failed"
	Error            string    `json:"error"`
	Filename         string    `json:"filename"`
	AddedAt          time.Time `json:"addedAt"`
}

var (
	downloads    = make([]*DownloadTask, 0)
	downloadsMu  sync.RWMutex
	downloadsDir = "./downloads"
	serverPort   = "8080"
)

// startBackgroundDownload fetches the download link and processes it in the background using Grab.
func startBackgroundDownload(task *DownloadTask, sourceName, romURL string) {
	defer func() {
		if r := recover(); r != nil {
			downloadsMu.Lock()
			task.Status = "failed"
			task.Error = fmt.Sprintf("Panic: %v", r)
			downloadsMu.Unlock()
			log.Printf("Panic in background download: %v", r)
		}
	}()

	downloadsMu.Lock()
	task.Status = "downloading"
	downloadsMu.Unlock()

	source := sources.LoadSource(sourceName, nil)
	if source == nil {
		downloadsMu.Lock()
		task.Status = "failed"
		task.Error = fmt.Sprintf("Source '%s' not found", sourceName)
		downloadsMu.Unlock()
		return
	}

	rom := domains.CreateRom(task.Name, task.Console, romURL, "")
	dlLink := source.GetDownloadLink(rom)
	if dlLink == "" {
		downloadsMu.Lock()
		task.Status = "failed"
		task.Error = "Could not resolve download link from source page"
		downloadsMu.Unlock()
		return
	}

	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		downloadsMu.Lock()
		task.Status = "failed"
		task.Error = fmt.Sprintf("Could not create downloads directory: %v", err)
		downloadsMu.Unlock()
		return
	}

	client := grab.NewClient()
	client.UserAgent = "Mozilla/5.0 (Linux; Android 6.0; SAMSUNG SM-G930F Build/MMB29K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/4.0 Chrome/44.0.2403.133 Mobile Safari/537.36"
	
	req, err := grab.NewRequest(downloadsDir, dlLink)
	if err != nil {
		downloadsMu.Lock()
		task.Status = "failed"
		task.Error = fmt.Sprintf("Invalid download request: %v", err)
		downloadsMu.Unlock()
		return
	}

	if strings.Contains(dlLink, "romsgames.net") {
		req.HTTPRequest.Header.Set("Referer", "https://www.romsgames.net/")
	}

	resp := client.Do(req)
	
	downloadsMu.Lock()
	task.Filename = filepath.Base(resp.Filename)
	downloadsMu.Unlock()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

downloadLoop:
	for {
		select {
		case <-ticker.C:
			downloadsMu.Lock()
			task.BytesTransferred = resp.BytesComplete()
			task.TotalSize = resp.Size()
			task.Progress = resp.Progress() * 100
			task.Speed = resp.BytesPerSecond()
			downloadsMu.Unlock()

		case <-resp.Done:
			break downloadLoop
		}
	}

	if err := resp.Err(); err != nil {
		downloadsMu.Lock()
		task.Status = "failed"
		task.Error = err.Error()
		downloadsMu.Unlock()
		log.Printf("Download task %s failed: %v", task.ID, err)
	} else {
		lowerPath := strings.ToLower(resp.Filename)
		isArchive := strings.HasSuffix(lowerPath, ".zip") ||
			strings.HasSuffix(lowerPath, ".tar.gz") ||
			strings.HasSuffix(lowerPath, ".tgz") ||
			strings.HasSuffix(lowerPath, ".gz")

		if isArchive {
			downloadsMu.Lock()
			task.Status = "decompressing"
			task.Filename = filepath.Base(resp.Filename)
			downloadsMu.Unlock()

			log.Printf("Download task %s decompressing...", task.ID)
			extractedFiles, decErr := utils.DecompressAndCleanup(resp.Filename)

			downloadsMu.Lock()
			if decErr != nil {
				task.Status = "failed"
				task.Error = decErr.Error()
				log.Printf("Download task %s decompression failed: %v", task.ID, decErr)
				downloadsMu.Unlock()
			} else {
				task.Status = "completed"
				task.Progress = 100.0
				if len(extractedFiles) > 0 {
					task.Filename = filepath.Base(extractedFiles[0])
				}
				log.Printf("Download task %s completed & decompressed: %s", task.ID, task.Filename)
				downloadsMu.Unlock()

				// Run post-processing
				runPostProcessing(filepath.Join(downloadsDir, task.Filename), task.Console)
			}
		} else {
			downloadsMu.Lock()
			task.Status = "completed"
			task.Progress = 100.0
			task.Filename = filepath.Base(resp.Filename)
			downloadsMu.Unlock()
			log.Printf("Download task %s completed: %s", task.ID, task.Filename)

			// Run post-processing
			runPostProcessing(resp.Filename, task.Console)
		}
	}
}

func runPostProcessing(filePath string, consoleName string) {
	// Find post_process.py
	scriptPath := "/app/post_process.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "./post_process.py"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			log.Printf("Post-processing script not found at /app/post_process.py or ./post_process.py")
			return
		}
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	log.Printf("Executing post-processing script %s for file %s with console %s", scriptPath, absPath, consoleName)
	cmd := exec.Command("python3", scriptPath, absPath, consoleName)
	
	// Inherit system environment variables
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Post-processing failed: %v, Output: %s", err, string(output))
	} else {
		log.Printf("Post-processing completed successfully. Output: %s", string(output))
	}
}
