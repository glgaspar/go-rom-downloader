package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/alcmoraes/go-rom-downloader/domains"
	"github.com/gocolly/colly"
)

type RomsgamesSource struct {
	Endpoint  string
	UserAgent string
	LookupURL string
	c         *colly.Collector
}

func (self *RomsgamesSource) newCollector() *colly.Collector {
	return colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
}

func getConsoleFromURL(romURL string) string {
	parts := strings.Split(strings.TrimPrefix(romURL, "/"), "-rom-")
	if len(parts) > 1 {
		console := parts[0]
		console = strings.ReplaceAll(console, "-", " ")
		return strings.Title(console)
	}
	return "ROM"
}

func (self *RomsgamesSource) Lookup(name string) []domains.Rom {
	roms := []domains.Rom{}
	c := self.newCollector()

	// Parse search result grid elements
	c.OnHTML("div.grid a[href*='-rom-']", func(e *colly.HTMLElement) {
		titleDiv := e.DOM.Find("div").First()
		title := strings.TrimSpace(titleDiv.Text())
		href := e.Attr("href")
		if title != "" && href != "" {
			console := getConsoleFromURL(href)
			roms = append(roms, *domains.CreateRom(
				title,
				console,
				href,
				"",
			))
		}
	})

	// Do the search query
	c.Visit(fmt.Sprintf(self.Endpoint+self.LookupURL, name))

	return roms
}

type RomsgamesJSONResponse struct {
	DownloadUrl  string `json:"downloadUrl"`
	DownloadName string `json:"downloadName"`
}

func (self *RomsgamesSource) GetDownloadLink(rom *domains.Rom) string {
	c := self.newCollector()

	var mediaID string

	// Step 1: Extract data-media-id from button on the game page
	c.OnHTML("button[data-media-id]", func(e *colly.HTMLElement) {
		mediaID = e.Attr("data-media-id")
	})

	c.Visit(self.Endpoint + strings.TrimPrefix(rom.URL, "/"))

	if mediaID == "" {
		return ""
	}

	// Step 2: Make the POST request to Endpoint + ROM_URL + "?download"
	// To get the JSON response containing the download URL and name
	postURL := fmt.Sprintf("%s%s?download", self.Endpoint, strings.TrimPrefix(rom.URL, "/"))
	
	client := &http.Client{}
	
	data := url.Values{}
	data.Set("mediaId", mediaID)

	req, err := http.NewRequest("POST", postURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", self.Endpoint+strings.TrimPrefix(rom.URL, "/"))

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var jsonResp RomsgamesJSONResponse
	if err := json.Unmarshal(bodyBytes, &jsonResp); err != nil {
		return ""
	}

	if jsonResp.DownloadUrl == "" {
		return ""
	}

	// Step 3: Construct the final download URL
	// We need to pass mediaId and attach as query parameters
	finalURL := fmt.Sprintf("%s?mediaId=%s&attach=%s", jsonResp.DownloadUrl, mediaID, jsonResp.DownloadName)
	rom.SetDownloadURL(finalURL)

	return finalURL
}

func NewRomsgamesSource() *RomsgamesSource {
	return &RomsgamesSource{
		Endpoint:  "https://www.romsgames.net/",
		LookupURL: "search/?q=%s",
		c: colly.NewCollector(
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		),
	}
}
