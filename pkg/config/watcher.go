// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Watcher monitors configuration files for changes and triggers reload.
type Watcher struct {
	mu          sync.RWMutex
	paths       []string
	interval    time.Duration
	lastModTime map[string]time.Time
	config      *Config
	listeners   []func(*Config)
	stopCh      chan struct{}
	doneCh      chan struct{}
	logger      *slog.Logger
}

// WatcherOption configures the watcher.
type WatcherOption func(*Watcher)

// WithWatchInterval sets the polling interval for file changes.
func WithWatchInterval(d time.Duration) WatcherOption {
	return func(w *Watcher) {
		if d > 0 {
			w.interval = d
		}
	}
}

// WithWatchLogger sets the logger for the watcher.
func WithWatchLogger(logger *slog.Logger) WatcherOption {
	return func(w *Watcher) {
		w.logger = logger
	}
}

// NewWatcher creates a new configuration watcher.
// It monitors the given paths for changes and reloads configuration.
func NewWatcher(paths []string, opts ...WatcherOption) (*Watcher, error) {
	w := &Watcher{
		paths:       paths,
		interval:    1 * time.Second,
		lastModTime: make(map[string]time.Time),
		listeners:   make([]func(*Config), 0),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
		logger:      slog.Default(),
	}

	for _, opt := range opts {
		opt(w)
	}

	// Initialize mod times
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			w.lastModTime[path] = info.ModTime()
		}
	}

	// Load initial config
	cfg, err := w.loadConfig()
	if err != nil {
		return nil, err
	}
	w.config = cfg

	return w, nil
}

// OnChange registers a callback to be called when config changes.
func (w *Watcher) OnChange(fn func(*Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.listeners = append(w.listeners, fn)
}

// Config returns the current configuration.
func (w *Watcher) Config() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

// Start begins watching for configuration changes.
func (w *Watcher) Start(ctx context.Context) {
	go w.watch(ctx)
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	close(w.stopCh)
	<-w.doneCh
}

func (w *Watcher) watch(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if w.checkForChanges() {
				w.reload()
			}
		}
	}
}

func (w *Watcher) checkForChanges() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := false
	for _, path := range w.paths {
		info, err := os.Stat(path)
		if err != nil {
			// File might have been deleted or doesn't exist
			continue
		}

		lastMod, exists := w.lastModTime[path]
		if !exists || info.ModTime().After(lastMod) {
			w.lastModTime[path] = info.ModTime()
			changed = true
		}
	}
	return changed
}

func (w *Watcher) reload() {
	w.logger.Info("config file changed, reloading")

	cfg, err := w.loadConfig()
	if err != nil {
		w.logger.Error("failed to reload config", "error", err)
		return
	}

	w.mu.Lock()
	w.config = cfg
	listeners := make([]func(*Config), len(w.listeners))
	copy(listeners, w.listeners)
	w.mu.Unlock()

	w.logger.Info("config reloaded successfully")

	// Notify listeners
	for _, fn := range listeners {
		fn(cfg)
	}
}

func (w *Watcher) loadConfig() (*Config, error) {
	if len(w.paths) == 0 {
		return Load("")
	}
	// Use the first path as the main config
	return Load(w.paths[0])
}

// WatchConfig creates a watcher for the given config path and starts watching.
// It returns the watcher and initial config.
func WatchConfig(ctx context.Context, configPath string, opts ...WatcherOption) (*Watcher, *Config, error) {
	paths := []string{}
	
	if configPath != "" {
		paths = append(paths, configPath)
		
		// Also watch profile-specific files if they exist
		dir := filepath.Dir(configPath)
		ext := filepath.Ext(configPath)
		base := filepath.Base(configPath)
		nameWithoutExt := base[:len(base)-len(ext)]
		
		// Check for common profiles
		for _, profile := range []string{"dev", "prod", "staging", "local"} {
			profilePath := filepath.Join(dir, nameWithoutExt+"."+profile+ext)
			if _, err := os.Stat(profilePath); err == nil {
				paths = append(paths, profilePath)
			}
		}
	}

	watcher, err := NewWatcher(paths, opts...)
	if err != nil {
		return nil, nil, err
	}

	watcher.Start(ctx)
	return watcher, watcher.Config(), nil
}

// ReloadableConfig provides a thread-safe wrapper around Config
// that can be atomically updated.
type ReloadableConfig struct {
	mu     sync.RWMutex
	config *Config
}

// NewReloadableConfig creates a new reloadable config wrapper.
func NewReloadableConfig(cfg *Config) *ReloadableConfig {
	return &ReloadableConfig{config: cfg}
}

// Get returns the current configuration.
func (r *ReloadableConfig) Get() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// Update atomically replaces the configuration.
func (r *ReloadableConfig) Update(cfg *Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

// LLM returns the LLM configuration.
func (r *ReloadableConfig) LLM() LLMConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.LLM
}

// Agent returns the agent configuration.
func (r *ReloadableConfig) Agent() AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.Agent
}

// Telemetry returns the telemetry configuration.
func (r *ReloadableConfig) Telemetry() TelemetryConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.Telemetry
}

// Log returns the log configuration.
func (r *ReloadableConfig) Log() LogConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.Log
}
