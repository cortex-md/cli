package dev

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cortex/cli/internal/plugin"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
	"github.com/fsnotify/fsnotify"
)

type DevOptions struct {
	SkipInitialBuild bool
	SkipLink         bool
}

type DevSession struct {
	pluginDir     string
	manifest      *manifest.PluginManifest
	watcher       *fsnotify.Watcher
	ctx           context.Context
	cancel        context.CancelFunc
	rebuildNeeded chan struct{}
}

func Start(pluginDir string, opts DevOptions) error {
	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	ux.Info("Starting development mode for %s v%s", m.Name, m.Version)

	if !opts.SkipLink {
		if err := Link(absDir); err != nil {
			return fmt.Errorf("failed to link plugin: %w", err)
		}
	}

	if !opts.SkipInitialBuild {
		ux.Step("Running initial build...")
		if err := plugin.Build(absDir, plugin.BuildOptions{}); err != nil {
			ux.Warning("Initial build failed: %v", err)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	session := &DevSession{
		pluginDir:     absDir,
		manifest:      m,
		watcher:       watcher,
		ctx:           ctx,
		cancel:        cancel,
		rebuildNeeded: make(chan struct{}, 1),
	}

	defer session.cleanup()

	if err := session.setupWatcher(); err != nil {
		return fmt.Errorf("failed to setup watcher: %w", err)
	}

	ux.Success("Development mode started")
	ux.Info("Watching for changes in src/ and manifest.json")
	ux.Info("Press Ctrl+C to stop")
	fmt.Println()

	return session.run()
}

func (s *DevSession) setupWatcher() error {
	srcDir := filepath.Join(s.pluginDir, "src")
	if err := s.addDirRecursive(srcDir); err != nil {
		return err
	}

	manifestPath := filepath.Join(s.pluginDir, "manifest.json")
	if err := s.watcher.Add(manifestPath); err != nil {
		ux.Warning("Could not watch manifest.json: %v", err)
	}

	return nil
}

func (s *DevSession) addDirRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "dist" {
				return filepath.SkipDir
			}
			return s.watcher.Add(path)
		}

		return nil
	})
}

func (s *DevSession) run() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var debounceTimer *time.Timer
	debounceDuration := 300 * time.Millisecond

	for {
		select {
		case <-s.ctx.Done():
			return nil

		case sig := <-sigChan:
			ux.Info("Received %v, shutting down...", sig)
			return nil

		case event, ok := <-s.watcher.Events:
			if !ok {
				return nil
			}

			if !isRelevantChange(event) {
				continue
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDuration, func() {
				select {
				case s.rebuildNeeded <- struct{}{}:
				default:
				}
			})

		case <-s.rebuildNeeded:
			s.rebuild()

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return nil
			}
			ux.Warning("Watcher error: %v", err)
		}
	}
}

func (s *DevSession) rebuild() {
	ux.Step("Change detected, rebuilding...")

	startTime := time.Now()

	if err := plugin.Build(s.pluginDir, plugin.BuildOptions{}); err != nil {
		ux.Error("Build failed: %v", err)
		return
	}

	duration := time.Since(startTime)
	ux.Success("Rebuilt in %dms", duration.Milliseconds())

	s.notifyReload()
}

func (s *DevSession) notifyReload() {
	ux.Info("Plugin rebuilt - reload Cortex to see changes")
}

func (s *DevSession) cleanup() {
	s.cancel()
	if s.watcher != nil {
		s.watcher.Close()
	}
}

func isRelevantChange(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}

	ext := filepath.Ext(event.Name)
	relevantExtensions := map[string]bool{
		".ts":   true,
		".tsx":  true,
		".js":   true,
		".jsx":  true,
		".json": true,
		".css":  true,
	}

	return relevantExtensions[ext]
}
