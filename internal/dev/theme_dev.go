package dev

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
	"github.com/fsnotify/fsnotify"
)

type ThemeDevOptions struct {
	SkipLink  bool
	VaultPath string
}

type ThemeDevSession struct {
	themeDir     string
	manifest     *manifest.ThemeManifest
	watcher      *fsnotify.Watcher
	ctx          context.Context
	cancel       context.CancelFunc
	reloadNeeded chan struct{}
	themeID      string
	linkPath     string
	reloadDir    string
	linkTarget   string
}

func StartTheme(themeDir string, opts ThemeDevOptions) error {
	absDir, err := filepath.Abs(themeDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadTheme(absDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	ux.Info("Starting theme development mode for %s v%s", m.DisplayName, m.Version)

	var linkResult *LinkResult
	if !opts.SkipLink {
		linkResult, err = LinkTheme(absDir, LinkOptions{VaultPath: opts.VaultPath})
		if err != nil {
			return fmt.Errorf("failed to link theme: %w", err)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &ThemeDevSession{
		themeDir:     absDir,
		manifest:     m,
		watcher:      watcher,
		ctx:          ctx,
		cancel:       cancel,
		reloadNeeded: make(chan struct{}, 1),
	}
	if linkResult != nil {
		session.themeID = linkResult.ID
		session.linkPath = linkResult.LinkPath
		session.reloadDir = linkResult.ParentDir
		session.linkTarget = linkResult.Target
	}

	defer session.cleanup()

	if err := session.setupWatcher(); err != nil {
		return fmt.Errorf("failed to setup watcher: %w", err)
	}

	ux.Success("Theme development mode started")
	ux.Info("Watching for changes in manifest.json and CSS files")
	if session.linkPath != "" {
		ux.Info("Cortex will reload from: %s", session.linkPath)
	}
	ux.Info("Press Ctrl+C to stop")
	fmt.Println()

	return session.run()
}

func (s *ThemeDevSession) setupWatcher() error {
	return s.watcher.Add(s.themeDir)
}

func (s *ThemeDevSession) run() error {
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

			if !isRelevantThemeChange(event) {
				continue
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDuration, func() {
				select {
				case s.reloadNeeded <- struct{}{}:
				default:
				}
			})

		case <-s.reloadNeeded:
			s.notifyReload()

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return nil
			}
			ux.Warning("Watcher error: %v", err)
		}
	}
}

func (s *ThemeDevSession) notifyReload() {
	if s.linkPath == "" {
		ux.Info("Theme changed - link it to a vault with cortex theme link --vault /path/to/vault")
		return
	}
	if err := TouchDevReloadMarker(s.reloadDir, s.themeID); err != nil {
		ux.Warning("Theme changed, but reload notification failed: %v", err)
		return
	}
	ux.Info("Theme changed - Cortex will reload it when the vault is open")
}

func (s *ThemeDevSession) cleanup() {
	s.cancel()
	if s.watcher != nil {
		s.watcher.Close()
	}
	if s.linkPath != "" {
		if err := RemoveDevLink(s.linkPath, s.linkTarget, s.reloadDir, s.themeID); err != nil {
			ux.Warning("Could not remove dev theme link: %v", err)
			return
		}
		ux.Info("Removed dev theme link: %s", s.linkPath)
	}
}

func isRelevantThemeChange(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}

	ext := filepath.Ext(event.Name)
	return ext == ".css" || ext == ".json"
}
