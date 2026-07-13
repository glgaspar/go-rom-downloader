package sources

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alcmoraes/go-rom-downloader/domains"
	"github.com/gocolly/colly"
)

type EmuparadiseSource struct {
	Endpoint  string
	UserAgent string
	LookupURL string
	c         *colly.Collector
}

func (self *EmuparadiseSource) newCollector() *colly.Collector {
	return colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Linux; Android 6.0; SAMSUNG SM-G930F Build/MMB29K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/4.0 Chrome/44.0.2403.133 Mobile Safari/537.36"),
	)
}

func (self *EmuparadiseSource) Lookup(name string) []domains.Rom {

	roms := []domains.Rom{}
	c := self.newCollector()

	// Find and visit all links
	c.OnHTML("#content .roms", func(e *colly.HTMLElement) {
		roms = append(roms, *domains.CreateRom(
			e.ChildText("a[data-filter]"),
			e.ChildText("a.sysname"),
			e.ChildAttr("a[data-filter]", "href"),
			"",
		))
	})

	// Do the first query
	c.Visit(fmt.Sprintf(self.Endpoint+self.LookupURL, name))

	return roms

}

func (self *EmuparadiseSource) GetDownloadLink(rom *domains.Rom) string {
	trimmed := strings.Trim(rom.URL, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return ""
	}
	gameID := parts[len(parts)-1]

	url := fmt.Sprintf("%sroms/get-download.php?gid=%s&test=true", self.Endpoint, gameID)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			return ""
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; SAMSUNG SM-G930F Build/MMB29K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/4.0 Chrome/44.0.2403.133 Mobile Safari/537.36")
		req.Header.Set("Referer", self.Endpoint)

		resp, err := client.Do(req)
		if err != nil {
			return ""
		}
		resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				return ""
			}
			if strings.Contains(loc, "invalid_referer") {
				return ""
			}
			if !strings.HasPrefix(loc, "http://") && !strings.HasPrefix(loc, "https://") {
				loc = self.Endpoint + strings.TrimPrefix(loc, "/")
			}
			url = loc
		} else if resp.StatusCode == http.StatusOK {
			break
		} else {
			return ""
		}
	}

	if !strings.Contains(url, ".php") {
		rom.SetDownloadURL(url)
		return url
	}

	return ""
}

func NewEmuparadiseSource() *EmuparadiseSource {
	return &EmuparadiseSource{
		Endpoint:  "https://m.emuparadise.me/",
		LookupURL: "roms/search.php?query=%s",
		c: colly.NewCollector(
			colly.UserAgent("Mozilla/5.0 (Linux; Android 6.0; SAMSUNG SM-G930F Build/MMB29K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/4.0 Chrome/44.0.2403.133 Mobile Safari/537.36"),
		),
	}
}

