package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestDecompressAndCleanup_Zip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_decompress_zip")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zw := zip.NewWriter(zipFile)
	fileNames := []string{"file1.txt", "sub/file2.txt"}
	fileContents := []string{"content1", "content2"}

	for i, name := range fileNames {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create entry in zip: %v", err)
		}
		if _, err := w.Write([]byte(fileContents[i])); err != nil {
			t.Fatalf("failed to write to zip entry: %v", err)
		}
	}
	zw.Close()
	zipFile.Close()

	// Decompress and verify
	extracted, err := DecompressAndCleanup(zipPath)
	if err != nil {
		t.Fatalf("DecompressAndCleanup failed: %v", err)
	}

	if len(extracted) != 2 {
		t.Errorf("expected 2 extracted files, got %d", len(extracted))
	}

	// The zip file should be deleted
	if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
		t.Errorf("expected zip file to be deleted, but it exists")
	}

	// Verify contents of extracted files
	for i, name := range fileNames {
		extractedPath := filepath.Join(tempDir, name)
		if _, err := os.Stat(extractedPath); err != nil {
			t.Errorf("expected extracted file to exist at %s, but got err: %v", extractedPath, err)
			continue
		}
		data, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("failed to read extracted file %s: %v", extractedPath, err)
			continue
		}
		if string(data) != fileContents[i] {
			t.Errorf("expected content %s, got %s", fileContents[i], string(data))
		}
	}
}

func TestDecompressAndCleanup_TarGz(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_decompress_targz")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tarGzPath := filepath.Join(tempDir, "test.tar.gz")
	tarGzFile, err := os.Create(tarGzPath)
	if err != nil {
		t.Fatalf("failed to create tar.gz file: %v", err)
	}

	gw := gzip.NewWriter(tarGzFile)
	tw := tar.NewWriter(gw)

	fileNames := []string{"tfile1.txt", "tsub/tfile2.txt"}
	fileContents := []string{"tarcontent1", "tarcontent2"}

	for i, name := range fileNames {
		body := []byte(fileContents[i])
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write header to tar: %v", err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatalf("failed to write content to tar: %v", err)
		}
	}
	tw.Close()
	gw.Close()
	tarGzFile.Close()

	// Decompress and verify
	extracted, err := DecompressAndCleanup(tarGzPath)
	if err != nil {
		t.Fatalf("DecompressAndCleanup failed: %v", err)
	}

	if len(extracted) != 2 {
		t.Errorf("expected 2 extracted files, got %d", len(extracted))
	}

	// Original tar.gz should be deleted
	if _, err := os.Stat(tarGzPath); !os.IsNotExist(err) {
		t.Errorf("expected tar.gz file to be deleted, but it exists")
	}

	// Verify contents of extracted files
	for i, name := range fileNames {
		extractedPath := filepath.Join(tempDir, name)
		data, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("failed to read extracted file %s: %v", extractedPath, err)
			continue
		}
		if string(data) != fileContents[i] {
			t.Errorf("expected content %s, got %s", fileContents[i], string(data))
		}
	}
}

func TestDecompressAndCleanup_Gz(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_decompress_gz")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gzPath := filepath.Join(tempDir, "test.nes.gz")
	gzFile, err := os.Create(gzPath)
	if err != nil {
		t.Fatalf("failed to create gz file: %v", err)
	}

	gw := gzip.NewWriter(gzFile)
	content := []byte("nesromdata")
	if _, err := gw.Write(content); err != nil {
		t.Fatalf("failed to write content to gz: %v", err)
	}
	gw.Close()
	gzFile.Close()

	// Decompress and verify
	extracted, err := DecompressAndCleanup(gzPath)
	if err != nil {
		t.Fatalf("DecompressAndCleanup failed: %v", err)
	}

	if len(extracted) != 1 {
		t.Errorf("expected 1 extracted file, got %d", len(extracted))
	}

	expectedExtractedPath := filepath.Join(tempDir, "test.nes")
	if extracted[0] != expectedExtractedPath {
		t.Errorf("expected extracted path to be %s, got %s", expectedExtractedPath, extracted[0])
	}

	// Original gz should be deleted
	if _, err := os.Stat(gzPath); !os.IsNotExist(err) {
		t.Errorf("expected gz file to be deleted, but it exists")
	}

	// Verify content
	data, err := os.ReadFile(expectedExtractedPath)
	if err != nil {
		t.Fatalf("failed to read extracted file %s: %v", expectedExtractedPath, err)
	}
	if string(data) != string(content) {
		t.Errorf("expected content %s, got %s", string(content), string(data))
	}
}

func TestDecompressAndCleanup_NonArchive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_non_archive")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "game.nes")
	content := []byte("nesromdata")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to create dummy file: %v", err)
	}

	extracted, err := DecompressAndCleanup(filePath)
	if err != nil {
		t.Fatalf("DecompressAndCleanup failed: %v", err)
	}

	if extracted != nil {
		t.Errorf("expected nil extracted files for non-archive, got %v", extracted)
	}

	// File should NOT be deleted
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("expected file to still exist, but got err: %v", err)
	}

	// Verify content is unchanged
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("expected content %s, got %s", string(content), string(data))
	}
}

func TestDecompressAndCleanup_ZipSlip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_zipslip")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "slip.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zw := zip.NewWriter(zipFile)
	// Entry with path traversal
	w, err := zw.Create("../malicious.txt")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := w.Write([]byte("malicious content")); err != nil {
		t.Fatalf("failed to write to zip entry: %v", err)
	}
	zw.Close()
	zipFile.Close()

	// Decompress should fail due to path traversal check
	_, err = DecompressAndCleanup(zipPath)
	if err == nil {
		t.Fatalf("expected DecompressAndCleanup to fail with path traversal, but succeeded")
	}

	// Zip file should not be deleted because it failed
	if _, err := os.Stat(zipPath); err != nil {
		t.Errorf("expected zip file to remain on failure, but got err: %v", err)
	}
}
