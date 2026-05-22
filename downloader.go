package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alcmoraes/go-rom-downloader/domains"
	"github.com/alcmoraes/go-rom-downloader/sources"
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

	downloadsMu.Lock()
	defer downloadsMu.Unlock()

	if err := resp.Err(); err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		log.Printf("Download task %s failed: %v", task.ID, err)
	} else {
		task.Status = "completed"
		task.Progress = 100.0
		task.Filename = filepath.Base(resp.Filename)
		log.Printf("Download task %s completed: %s", task.ID, task.Filename)
	}
}
