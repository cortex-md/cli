package zipx

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CreateFromDir(outPath string, sourceDir string, basePath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer outFile.Close()

	writer := zip.NewWriter(outFile)
	defer writer.Close()

	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		zipPath := filepath.Join(basePath, relPath)
		zipPath = filepath.ToSlash(zipPath)

		return addFileToZip(writer, path, zipPath)
	})
}

func Create(outPath string, files map[string]string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer outFile.Close()

	writer := zip.NewWriter(outFile)
	defer writer.Close()

	for zipPath, sourcePath := range files {
		if err := addFileToZip(writer, sourcePath, zipPath); err != nil {
			return err
		}
	}

	return nil
}

func CreateFromBytes(outPath string, files map[string][]byte) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer outFile.Close()

	writer := zip.NewWriter(outFile)
	defer writer.Close()

	for zipPath, content := range files {
		zipPath = filepath.ToSlash(zipPath)

		w, err := writer.Create(zipPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry %s: %w", zipPath, err)
		}

		if _, err := w.Write(content); err != nil {
			return fmt.Errorf("failed to write zip entry %s: %w", zipPath, err)
		}
	}

	return nil
}

func AddDir(writer *zip.Writer, sourceDir string, basePath string) error {
	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		zipPath := filepath.Join(basePath, relPath)
		zipPath = filepath.ToSlash(zipPath)

		return addFileToZip(writer, path, zipPath)
	})
}

func addFileToZip(writer *zip.Writer, sourcePath string, zipPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", sourcePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", sourcePath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create zip header for %s: %w", sourcePath, err)
	}

	header.Name = zipPath
	header.Method = zip.Deflate

	w, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", zipPath, err)
	}

	if _, err := io.Copy(w, file); err != nil {
		return fmt.Errorf("failed to write zip entry %s: %w", zipPath, err)
	}

	return nil
}

func GetSize(zipPath string) (int64, error) {
	info, err := os.Stat(zipPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func shouldSkip(name string) bool {
	skipPatterns := []string{
		".git",
		".DS_Store",
		"node_modules",
		".env",
		".env.local",
		"thumbs.db",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	return false
}
