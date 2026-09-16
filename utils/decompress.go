package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractResult holds the result of decompressing a single archive.
type ExtractResult struct {
	ArchivePath string   `json:"archivePath"`
	Status      string   `json:"status"` // "success", "failed"
	Files       []string `json:"files,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// VerifyExtractedFiles checks if all extracted files exist and are complete (non-zero size).
func VerifyExtractedFiles(files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files were extracted")
	}
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return fmt.Errorf("extracted file %s does not exist: %w", f, err)
		}
		if !info.IsDir() && info.Size() == 0 {
			return fmt.Errorf("extracted file %s is incomplete (0 bytes)", f)
		}
	}
	return nil
}

// DecompressAndCleanup checks if the file at filePath is a compressed archive.
// If it is, it decompresses the archive to its containing directory, verifies file completeness,
// deletes the original archive only if complete, and returns a list of paths of the decompressed files.
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
	} else if strings.HasSuffix(lowerPath, ".7z") || strings.HasSuffix(lowerPath, ".rar") {
		isArchive = true
		extractedFiles, err = extract7z(filePath, destDir)
	}

	if !isArchive {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	// Verify extracted files completeness before deleting original archive
	if err := VerifyExtractedFiles(extractedFiles); err != nil {
		return extractedFiles, fmt.Errorf("completeness check failed for extracted files: %w", err)
	}

	// Decompress and completeness check succeeded, delete original compressed archive
	if err := os.Remove(filePath); err != nil {
		return extractedFiles, fmt.Errorf("failed to delete original compressed file: %w", err)
	}

	return extractedFiles, nil
}

// ExtractRomsInDir scans dir recursively for archives, extracts them, checks completeness, and deletes original archives.
func ExtractRomsInDir(dir string) ([]ExtractResult, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", dir)
	}

	var results []ExtractResult
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		lower := strings.ToLower(path)
		isArchive := strings.HasSuffix(lower, ".zip") ||
			strings.HasSuffix(lower, ".7z") ||
			strings.HasSuffix(lower, ".rar") ||
			strings.HasSuffix(lower, ".tar.gz") ||
			strings.HasSuffix(lower, ".tgz") ||
			strings.HasSuffix(lower, ".gz")

		if isArchive {
			extracted, decErr := DecompressAndCleanup(path)
			relPath, _ := filepath.Rel(dir, path)
			if relPath == "" {
				relPath = path
			}

			if decErr != nil {
				results = append(results, ExtractResult{
					ArchivePath: relPath,
					Status:      "failed",
					Error:       decErr.Error(),
				})
			} else if len(extracted) > 0 {
				var relExtracted []string
				for _, f := range extracted {
					r, _ := filepath.Rel(dir, f)
					if r != "" {
						relExtracted = append(relExtracted, r)
					} else {
						relExtracted = append(relExtracted, f)
					}
				}
				results = append(results, ExtractResult{
					ArchivePath: relPath,
					Status:      "success",
					Files:       relExtracted,
				})
			}
		}
		return nil
	})

	return results, err
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

func extract7z(archivePath, destDir string) ([]string, error) {
	cmdName := "7z"
	if _, err := exec.LookPath("7z"); err != nil {
		if _, err2 := exec.LookPath("7za"); err2 == nil {
			cmdName = "7za"
		} else {
			return nil, fmt.Errorf("neither 7z nor 7za command line tool is installed")
		}
	}

	listCmd := exec.Command(cmdName, "l", archivePath)
	listOut, err := listCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list archive contents with %s: %w", cmdName, err)
	}

	expectedNames := parse7zListing(string(listOut))

	extractCmd := exec.Command(cmdName, "x", "-y", "-o"+destDir, archivePath)
	extractOut, err := extractCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s extraction failed: %w, output: %s", cmdName, err, string(extractOut))
	}

	var extractedFiles []string
	for _, name := range expectedNames {
		fullPath := filepath.Clean(filepath.Join(destDir, name))
		if !strings.HasPrefix(fullPath, filepath.Clean(destDir)) {
			return nil, fmt.Errorf("illegal file path in 7z archive: %s", name)
		}
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			extractedFiles = append(extractedFiles, fullPath)
		}
	}

	return extractedFiles, nil
}

func parse7zListing(stdout string) []string {
	lines := strings.Split(stdout, "\n")
	tableLines := []string{}
	inTable := false
	for _, line := range lines {
		lineClean := strings.TrimRight(line, "\r")
		if strings.HasPrefix(lineClean, "------------------- ----- ------------ ------------") {
			if !inTable {
				inTable = true
			} else {
				inTable = false
			}
			continue
		}
		if inTable {
			tableLines = append(tableLines, lineClean)
		}
	}

	var names []string
	for _, line := range tableLines {
		if len(line) < 54 {
			continue
		}
		attr := line[20:25]
		if strings.Contains(attr, "D") {
			continue
		}
		name := strings.TrimSpace(line[53:])
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
