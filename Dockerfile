# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files first to leverage caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o rom-downloader .

# Run stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata python3 p7zip

WORKDIR /app

# Copy build artifact
COPY --from=builder /app/rom-downloader /app/rom-downloader

# Copy post-processing script
COPY post_process.py /app/post_process.py

# Create default downloads and roms directories and set permissions
RUN mkdir /downloads /roms && chmod 777 /downloads /roms

# Set environment variables
ENV PORT=8080
ENV DOWNLOADS_DIR=/downloads
ENV ROMS_DIR=/roms

# Expose port
EXPOSE 8080

# Volumes for downloads and roms
VOLUME ["/downloads", "/roms"]

# Run the app
ENTRYPOINT ["/app/rom-downloader"]
