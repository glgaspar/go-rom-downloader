package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alcmoraes/go-rom-downloader/domains"
	"github.com/alcmoraes/go-rom-downloader/sources"
	"github.com/alcmoraes/go-rom-downloader/utils"
	"github.com/cavaliergopher/grab/v3"
)

func SourceInput() string {
	sourcesMap := make([]string, len(sources.RomSources))
	i := 0
	for k := range sources.RomSources {
		sourcesMap[i] = k
		i++
	}

source_input:
	fmt.Println("From which source you would like to search?")

	for i, source := range sourcesMap {
		fmt.Printf("[%d] %s\n", i+1, source)
	}

	sourceInputScanner := bufio.NewScanner(os.Stdin)
	var sourceChosen int
	for sourceInputScanner.Scan() {
		sourceChosenVal, err := strconv.ParseInt(sourceInputScanner.Text(), 10, 32)
		sourceChosen = int(sourceChosenVal)
		if err != nil || sourceChosen < 1 || sourceChosen > len(sourcesMap) {
			utils.CallClear()
			fmt.Println("\rInvalid option.")
			goto source_input
		}
		return sourcesMap[sourceChosen-1]
	}
	return sourcesMap[sourceChosen-1]
}

func RomQueryInput() string {
	fmt.Println("Enter the game you would like to search.")
rom_query_input:
	romQueryScanner := bufio.NewScanner(os.Stdin)
	var romQueryChosen string
	for romQueryScanner.Scan() {
		romQueryChosen = romQueryScanner.Text()
		if len(romQueryChosen) == 0 {
			utils.CallClear()
			fmt.Println("\rInvalid query. Type it again.")
			goto rom_query_input
		}
		return romQueryChosen
	}
	return romQueryChosen
}

func ChooseRomInput(roms []domains.Rom) domains.Rom {
	fmt.Println("============================")
	for i, rom := range roms {
		fmt.Printf("[%d] %s (%s)\n", i+1, rom.Name, rom.Console)
	}
	fmt.Printf("============================\n")
	fmt.Printf("Type the number of the game you want do download (eg.: 13) and press enter.\n")
rom_choose_input:
	romChooseScanner := bufio.NewScanner(os.Stdin)
	var romChosen int
	for romChooseScanner.Scan() {
		romChosenVal, err := strconv.ParseInt(romChooseScanner.Text(), 10, 32)
		romChosen = int(romChosenVal)
		if err != nil || romChosen < 1 || romChosen > len(roms) {
			fmt.Println("Invalid option. Try again.")
			goto rom_choose_input
		}
		return roms[romChosen-1]
	}
	return roms[romChosen-1]
}

func cliDownload(rom domains.Rom) {
	client := grab.NewClient()
	client.UserAgent = "Mozilla/5.0 (Linux; Android 6.0; SAMSUNG SM-G930F Build/MMB29K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/4.0 Chrome/44.0.2403.133 Mobile Safari/537.36"
	req, _ := grab.NewRequest(downloadsDir, rom.DownloadURL)
	if strings.Contains(rom.DownloadURL, "romsgames.net") {
		req.HTTPRequest.Header.Set("Referer", "https://www.romsgames.net/")
	}
	fmt.Printf("Downloading %v...\n", rom.Name)

	resp := client.Do(req)
	fmt.Printf("  %v\n", resp.HTTPResponse.Status)
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
Loop:
	for {
		select {
		case <-t.C:
			fmt.Printf("\rTransferred %v / %v bytes (%.2f%%)",
				resp.BytesComplete(),
				resp.Size(),
				100*resp.Progress())

		case <-resp.Done:
			break Loop
		}
	}
	if err := resp.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "\nDownload failed: %v\n", err)
		os.Exit(1)
	}

	extractedFiles, decErr := utils.DecompressAndCleanup(resp.Filename)
	if decErr != nil {
		fmt.Fprintf(os.Stderr, "\nDecompression failed: %v\n", decErr)
		os.Exit(1)
	}

	if len(extractedFiles) > 0 {
		fmt.Printf("\nDownload saved and decompressed: \n")
		for _, f := range extractedFiles {
			fmt.Printf("  %s\n", f)
		}
	} else {
		fmt.Printf("\nDownload saved to %s\n", resp.Filename)
	}
}

func cliMain() {
	utils.CallClear()
	fmt.Println("== GO ROM DOWNLOADER v1.0 ==")
	source := sources.LoadSource(SourceInput(), nil)
start:
	roms := source.Lookup(url.QueryEscape(RomQueryInput()))
	if len(roms) == 0 {
		utils.CallClear()
		fmt.Println("No roms match your query. Try again.")
		goto start
	}
	fmt.Printf("A total of %d roms found.\n", len(roms))
	romChosen := ChooseRomInput(roms)
	source.GetDownloadLink(&romChosen)
	cliDownload(romChosen)
	os.Exit(0)
}
