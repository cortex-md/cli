package dev

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 200 * time.Millisecond

var ignoredDirectories = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
}

func Watch(ctx context.Context, root string, relevant func(path string) bool, onChange func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	if err := addTree(watcher, root); err != nil {
		return err
	}

	var timer *time.Timer
	changes := make(chan struct{}, 1)
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = addTree(watcher, event.Name)
				}
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 || !relevant(event.Name) {
				continue
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDelay, func() {
				select {
				case changes <- struct{}{}:
				default:
				}
			})

		case <-changes:
			onChange()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("file watcher failed: %w", err)
		}
	}
}

func addTree(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil
		}
		if path != root && ignoredDirectories[entry.Name()] {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}
