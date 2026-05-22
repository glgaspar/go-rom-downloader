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
	flag.Parse()

	// Environment variable / flag resolution for downloads folder
	downloadsDir = os.Getenv("DOWNLOADS_DIR")
	if downloadsDir == "" {
		downloadsDir = "./downloads"
	}
	if *dirFlag != "" {
		downloadsDir = *dirFlag
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

	// Ensure the downloads directory exists
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		log.Fatalf("Could not create downloads directory: %v", err)
	}

	// Start Web Server
	runWebServer(serverPort)
}
