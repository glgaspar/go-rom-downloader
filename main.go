package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	cliFlag := flag.Bool("cli", false, "Run in interactive command-line mode")
	portFlag := flag.String("port", "", "Port to run the web server on (overrides PORT env)")
	dirFlag := flag.String("dir", "", "Directory to save downloads (overrides DOWNLOADS_DIR env)")
	romsDirFlag := flag.String("roms-dir", "", "Directory for organized ROMs in RomM style (overrides ROMS_DIR env)")
	flag.Parse()

	// Environment variable / flag resolution for downloads folder
	downloadsDir = os.Getenv("DOWNLOADS_DIR")
	if downloadsDir == "" {
		downloadsDir = "./downloads"
	}
	if *dirFlag != "" {
		downloadsDir = *dirFlag
	}

	// Environment variable / flag resolution for roms folder
	romsDir = os.Getenv("ROMS_DIR")
	if romsDir == "" {
		romsDir = "./roms"
	}
	if *romsDirFlag != "" {
		romsDir = *romsDirFlag
	}

	// Environment variable / flag resolution for port
	serverPort = os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	if *portFlag != "" {
		serverPort = *portFlag
	}

	if *cliFlag {
		cliMain()
		return
	}

	// Ensure the downloads and roms directories exist
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		log.Fatalf("Could not create downloads directory: %v", err)
	}
	if err := os.MkdirAll(romsDir, 0755); err != nil {
		log.Fatalf("Could not create roms directory: %v", err)
	}

	// Start Web Server
	runWebServer(serverPort)
}
