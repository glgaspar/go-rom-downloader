package sources

import (
	"fmt"
	"strings"

	"github.com/alcmoraes/go-rom-downloader/domains"
	"github.com/gocolly/colly"
)

type ConsoleromsSource struct {
	Endpoint  string
	UserAgent string
	LookupURL string
	c         *colly.Collector
}

func (self *ConsoleromsSource) newCollector() *colly.Collector {
	return colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
}

func (self *ConsoleromsSource) Lookup(name string) []domains.Rom {
	roms := []domains.Rom{}
	c := self.newCollector()

	// Find and visit all links
	c.OnHTML(".thumbnail-home", func(e *colly.HTMLElement) {
		title := e.ChildText(".infoBox strong")
		console := e.ChildText(".tagBtns a.emulator")
		href := e.ChildAttr(".imgCon a", "href")
		if title != "" && href != "" {
			roms = append(roms, *domains.CreateRom(
				title,
				console,
				href,
				"",
			))
		}
	})

	// Do the first query
	c.Visit(fmt.Sprintf(self.Endpoint+self.LookupURL, name))

	return roms
}

func (self *ConsoleromsSource) GetDownloadLink(rom *domains.Rom) string {
	c := self.newCollector()

	var downloadURL string

	// Step 1: Visit the game page (e.g. /roms/sega-genesis/sonic-the-hedgehog)
	// Find the download page link: a[itemprop="downloadUrl"]
	c.OnHTML("a[itemprop='downloadUrl']", func(e *colly.HTMLElement) {
		downloadPageURL := e.Attr("href")
		if downloadPageURL != "" {
			// Visit the download sub-page
			e.Request.Visit(downloadPageURL)
		}
	})

	// Step 2: On the download sub-page, find the rel="nofollow" click here link
	c.OnHTML("a[rel='nofollow']", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if strings.HasSuffix(href, "/download") {
			return
		}
		href = strings.ReplaceAll(href, " ", "%20")
		if strings.HasPrefix(href, "http") {
			downloadURL = href
		} else if href != "" {
			downloadURL = self.Endpoint + strings.TrimPrefix(href, "/")
		}
	})

	c.Visit(self.Endpoint + strings.TrimPrefix(rom.URL, "/"))

	if downloadURL != "" {
		rom.SetDownloadURL(downloadURL)
	}

	return downloadURL
}

func NewConsoleromsSource() *ConsoleromsSource {
	return &ConsoleromsSource{
		Endpoint:  "https://www.consoleroms.com/",
		LookupURL: "search?search_term_string=%s",
		c: colly.NewCollector(
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		),
	}
}
