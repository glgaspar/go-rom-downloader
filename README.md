# RetroROM Downloader

A modern, self-hosted web application and command-line tool built in Go to easily search, scrape, and download retro gaming ROMs from popular emulator sites.

![Interface Screenshot](https://github.com/alcmoraes/go-rom-downloader/blob/master/res/screenshot.png)

---

## 🚀 Key Features

*   **Premium Web UI**: Responsive dark-mode interface featuring neon styling, a clean search console, and visual results grid.
*   **Live Background Queue**: Track multiple active downloads with live progress bars, exact file sizes, and speeds in real-time.
*   **Docker Volume Mapping**: Volume mount your host folder to have all downloaded ROMs save directly to your media server or local library.
*   **Fully Self-Contained**: The web application and all static frontend assets (HTML, CSS, JS) are embedded directly inside a single static Go binary.
*   **CLI Mode Fallback**: Keep your legacy scripts running; interactive terminal mode is still fully supported.

---

## 🐳 Running with Docker (Recommended)

Run the self-hosted app in seconds using Docker or Docker Compose.

### Method A: Docker Compose (Easiest)

1.  **Start the container**:
    ```bash
    docker compose up -d
    ```
2.  Open **`http://localhost:8080`** in your browser.
3.  All downloaded ROMs will be saved to your host machine's `./downloads` directory (automatically created).

### Method B: Pure Docker CLI

Map your host machine's media path to save ROMs directly to a specific folder:

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/your/roms:/downloads \
  --name rom-downloader \
  --restart unless-stopped \
  go-rom-downloader:latest
```

---

## 🛠️ Local Development & Manual Build

The project is built on **Go Modules** (requires Go 1.16+).

### Prerequisites
*   [Go](https://go.dev/) (1.16 or higher)

### Build Steps

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/alcmoraes/go-rom-downloader.git
    cd go-rom-downloader
    ```
2.  **Tidy dependencies and compile**:
    ```bash
    go mod tidy
    go build -o rom-downloader .
    ```
3.  **Run the Web Server**:
    ```bash
    # Starts server on http://localhost:8080, saving downloads to ./downloads
    ./rom-downloader
    ```

### Command Line Flags & Environment Variables

You can customize the Web Server using command-line flags or environment variables:

| Flag | Environment Variable | Default | Description |
|---|---|---|---|
| `-port` | `PORT` | `8080` | Network port for the web server to listen on. |
| `-dir` | `DOWNLOADS_DIR` | `./downloads` | Local path where downloaded files are saved. |
| `-cli` | *N/A* | `false` | Fall back to the original interactive terminal CLI mode. |

**Example using flags:**
```bash
./rom-downloader -port 9000 -dir /mnt/games/retro
```

**Example using env variables:**
```bash
PORT=9000 DOWNLOADS_DIR=/mnt/games/retro ./rom-downloader
```

---

## 🎮 Running in Legacy CLI Mode

If you prefer to search and download ROMs directly in your terminal, run the application with the `-cli` flag:

```bash
./rom-downloader -cli
```

---

## 📁 Project Structure

*   `main.go`: Application entrypoint, parses CLI flags, and bootstraps CLI or Web Mode.
*   `web.go`: Embedded single-page frontend server and REST API handlers.
*   `downloader.go`: Thread-safe background download scheduler and Grab progress manager.
*   `cli.go`: Original interactive terminal user prompts.
*   `static/`: Elegant Vanilla CSS, HTML5 layouts, and JS state controllers.
*   `sources/`: Custom scrapers for sites like Coolrom and Emuparadise.
*   `domains/`: ROM entity structures.
*   `utils/`: Terminal clearing and system functions.