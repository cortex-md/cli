package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrPathTraversal  = errors.New("path traversal detected")
	ErrNotInDirectory = errors.New("path is not within allowed directory")
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func SafeJoin(base string, parts ...string) (string, error) {
	joined := filepath.Join(append([]string{base}, parts...)...)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(abs, absBase) {
		return "", ErrPathTraversal
	}

	return abs, nil
}

func CreateSymlink(target, link string) error {
	if Exists(link) {
		if err := os.Remove(link); err != nil {
			return err
		}
	}

	linkDir := filepath.Dir(link)
	if err := EnsureDir(linkDir); err != nil {
		return err
	}

	return os.Symlink(target, link)
}

func RemoveSymlink(link string) error {
	info, err := os.Lstat(link)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return errors.New("not a symlink")
	}

	return os.Remove(link)
}
