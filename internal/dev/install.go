package dev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cortex/cli/internal/build"
)

const DevMarker = ".cortex-dev"

var ErrNotDevInstall = errors.New("folder was not installed by cortex dev")

func IsForeignInstall(installDir string) bool {
	info, err := os.Lstat(installDir)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	_, err = os.Stat(filepath.Join(installDir, DevMarker))
	return err != nil
}

func Install(installDir string, files []build.File) error {
	if info, err := os.Lstat(installDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(installDir); err != nil {
			return fmt.Errorf("failed to replace legacy dev symlink: %w", err)
		}
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", installDir, err)
	}

	for _, file := range files {
		target := filepath.Join(installDir, filepath.FromSlash(file.Path))
		if err := writeAtomically(target, file); err != nil {
			return err
		}
	}

	markerPath := filepath.Join(installDir, DevMarker)
	if _, err := os.Stat(markerPath); err != nil {
		if err := os.WriteFile(markerPath, []byte("installed by cortex dev\n"), 0o644); err != nil {
			return fmt.Errorf("failed to write dev marker: %w", err)
		}
	}
	return nil
}

func Uninstall(installDir string) error {
	if _, err := os.Stat(installDir); os.IsNotExist(err) {
		return nil
	}
	if IsForeignInstall(installDir) {
		return fmt.Errorf("%w: %s", ErrNotDevInstall, installDir)
	}
	return os.RemoveAll(installDir)
}

func writeAtomically(target string, file build.File) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	temp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if file.Content != nil {
		_, err = temp.Write(file.Content)
	} else {
		err = copyInto(temp, file.Source)
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", file.Path, err)
	}

	if err := os.Chmod(tempPath, 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, target)
}

func copyInto(target io.Writer, sourcePath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	_, err = io.Copy(target, source)
	return err
}
