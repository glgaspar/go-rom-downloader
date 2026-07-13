package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DecompressAndCleanup checks if the file at filePath is a compressed archive.
// If it is, it decompresses the archive to its containing directory, deletes the
// original archive, and returns a list of relative/absolute paths of the decompressed files.
// If it is not a recognized archive format, it returns nil, nil.
func DecompressAndCleanup(filePath string) ([]string, error) {
	lowerPath := strings.ToLower(filePath)
	destDir := filepath.Dir(filePath)

	var extractedFiles []string
	var err error
	var isArchive bool

	if strings.HasSuffix(lowerPath, ".zip") {
		isArchive = true
		extractedFiles, err = extractZip(filePath, destDir)
	} else if strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz") {
		isArchive = true
		extractedFiles, err = extractTarGz(filePath, destDir)
	} else if strings.HasSuffix(lowerPath, ".gz") {
		isArchive = true
		extractedFiles, err = extractGz(filePath, destDir)
	}

	if !isArchive {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	// Decompress succeeded, delete the original compressed archive
	if err := os.Remove(filePath); err != nil {
		return extractedFiles, fmt.Errorf("failed to delete original compressed file: %w", err)
	}

	return extractedFiles, nil
}

func extractZip(archivePath, destDir string) ([]string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var extractedFiles []string
	for _, f := range r.File {
		err := func() error {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			path := filepath.Join(destDir, f.Name)
			// Prevent Zip Slip vulnerability
			if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(destDir)) {
				return fmt.Errorf("illegal file path in zip: %s", f.Name)
			}

			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(path, f.Mode()); err != nil {
					return err
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					return err
				}
				fOut, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					return err
				}
				defer fOut.Close()

				if _, err = io.Copy(fOut, rc); err != nil {
					return err
				}
				extractedFiles = append(extractedFiles, path)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	return extractedFiles, nil
}

func extractTarGz(archivePath, destDir string) ([]string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var extractedFiles []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		path := filepath.Join(destDir, header.Name)
		// Prevent Zip Slip vulnerability
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(destDir)) {
			return nil, fmt.Errorf("illegal file path in tar: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, header.FileInfo().Mode()); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return nil, err
			}
			err := func() error {
				fOut, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
				if err != nil {
					return err
				}
				defer fOut.Close()

				if _, err := io.Copy(fOut, tr); err != nil {
					return err
				}
				extractedFiles = append(extractedFiles, path)
				return nil
			}()
			if err != nil {
				return nil, err
			}
		}
	}
	return extractedFiles, nil
}

func extractGz(archivePath, destDir string) ([]string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	baseName := filepath.Base(archivePath)
	if !strings.HasSuffix(strings.ToLower(baseName), ".gz") {
		return nil, fmt.Errorf("invalid gz file name: %s", baseName)
	}
	destName := baseName[:len(baseName)-3]
	destPath := filepath.Join(destDir, destName)

	fOut, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer fOut.Close()

	if _, err := io.Copy(fOut, gzr); err != nil {
		return nil, err
	}

	return []string{destPath}, nil
}
