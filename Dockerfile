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

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy build artifact
COPY --from=builder /app/rom-downloader /app/rom-downloader

# Create default downloads directory and set permissions
RUN mkdir /downloads && chmod 777 /downloads

# Set environment variables
ENV PORT=8080
ENV DOWNLOADS_DIR=/downloads

# Expose port
EXPOSE 8080

# Volume for downloads
VOLUME /downloads

# Run the app
ENTRYPOINT ["/app/rom-downloader"]
