package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	config "github.com/eryalito/multi-wordpress-file-manager/pkg/config"
)

// Load reads and parses the YAML configuration file at path.
func Load(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Watch watches the directory containing path and invokes onChange whenever the
// target file is changed. It debounces rapid sequences of events and reloads the
// config before invoking the callback. The callback receives either the new
// config or an error if reload failed.
func Watch(ctx context.Context, path string, onChange func(*config.Config, error)) (func() error, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}
	dir := filepath.Dir(abs)
	base := filepath.Base(abs)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	// fsnotify is cross-platform. On Linux it uses inotify under the hood.
	if err := w.Add(dir); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("watch dir %s: %w", dir, err)
	}

	// Debounce timer; zero value means inactive.
	const debounce = 200 * time.Millisecond
	var timer *time.Timer
	var timerC <-chan time.Time
	reset := func() {
		if timer == nil {
			timer = time.NewTimer(debounce)
			timerC = timer.C
			return
		}
		if !timer.Stop() {
			// Drain if fired
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(debounce)
	}

	go func() {
		defer w.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-w.Events:
				if ev.Name == "" {
					continue
				}
				// Only react to events for our file
				if !sameFile(ev.Name, abs) {
					continue
				}
				// Interested in writes, creates, renames, removes, chmods
				if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) || ev.Has(fsnotify.Rename) || ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Chmod) {
					reset()
				}
			case err := <-w.Errors:
				// Surface watcher errors via callback
				if onChange != nil {
					onChange(nil, fmt.Errorf("watch error: %w", err))
				}
			case <-timerC:
				// Debounced reload
				cfg, err := Load(abs)
				if onChange != nil {
					onChange(cfg, err)
				}
			}
		}
	}()

	stop := func() error {
		return w.Close()
	}
	_ = base // retained for potential future filtering
	return stop, nil
}

func sameFile(a, b string) bool {
	if a == b {
		return true
	}
	// On some platforms editors may emit different path casings
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
