// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherDetectsChanges(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initial := `llm:
  provider: ollama
  model: test-model
`
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	// Create watcher
	watcher, err := NewWatcher([]string{configPath}, WithWatchInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	// Track changes
	changes := make(chan *Config, 1)
	watcher.OnChange(func(cfg *Config) {
		changes <- cfg
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx)
	defer watcher.Stop()

	// Verify initial config
	cfg := watcher.Config()
	if cfg.LLM.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", cfg.LLM.Model)
	}

	// Wait a bit to ensure watcher is running
	time.Sleep(100 * time.Millisecond)

	// Modify config
	updated := `llm:
  provider: ollama
  model: updated-model
`
	if err := os.WriteFile(configPath, []byte(updated), 0644); err != nil {
		t.Fatalf("failed to write updated config: %v", err)
	}

	// Wait for change notification
	select {
	case newCfg := <-changes:
		if newCfg.LLM.Model != "updated-model" {
			t.Errorf("expected model 'updated-model', got %q", newCfg.LLM.Model)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for config change notification")
	}
}

func TestWatcherMultipleListeners(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initial := `llm:
  model: v1
`
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	watcher, err := NewWatcher([]string{configPath}, WithWatchInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	// Multiple listeners
	count1 := 0
	count2 := 0
	watcher.OnChange(func(*Config) { count1++ })
	watcher.OnChange(func(*Config) { count2++ })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx)
	defer watcher.Stop()

	time.Sleep(100 * time.Millisecond)

	// Trigger change
	if err := os.WriteFile(configPath, []byte(`llm:
  model: v2
`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if count1 != 1 || count2 != 1 {
		t.Errorf("expected both listeners called once, got count1=%d, count2=%d", count1, count2)
	}
}

func TestWatcherStops(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(`llm: {}`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	watcher, err := NewWatcher([]string{configPath}, WithWatchInterval(10*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	ctx := context.Background()
	watcher.Start(ctx)

	// Stop should complete quickly
	done := make(chan struct{})
	go func() {
		watcher.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good
	case <-time.After(1 * time.Second):
		t.Error("watcher.Stop() did not complete in time")
	}
}

func TestReloadableConfig(t *testing.T) {
	cfg1 := &Config{
		LLM: LLMConfig{Model: "model-1"},
	}
	cfg2 := &Config{
		LLM: LLMConfig{Model: "model-2"},
	}

	rc := NewReloadableConfig(cfg1)

	// Initial value
	if rc.LLM().Model != "model-1" {
		t.Errorf("expected model-1, got %q", rc.LLM().Model)
	}

	// Update
	rc.Update(cfg2)

	// New value
	if rc.LLM().Model != "model-2" {
		t.Errorf("expected model-2, got %q", rc.LLM().Model)
	}

	// Get full config
	if rc.Get().LLM.Model != "model-2" {
		t.Errorf("expected model-2 from Get(), got %q", rc.Get().LLM.Model)
	}
}

func TestWatchConfigWithProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create base config
	basePath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(basePath, []byte(`llm:
  model: base
`), 0644); err != nil {
		t.Fatalf("failed to write base config: %v", err)
	}

	// Create dev profile
	devPath := filepath.Join(tmpDir, "config.dev.yaml")
	if err := os.WriteFile(devPath, []byte(`llm:
  model: dev
`), 0644); err != nil {
		t.Fatalf("failed to write dev config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher, cfg, err := WatchConfig(ctx, basePath, WithWatchInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to watch config: %v", err)
	}
	defer watcher.Stop()

	// Verify initial config (base, since we didn't specify profile)
	if cfg.LLM.Model != "base" {
		t.Errorf("expected model 'base', got %q", cfg.LLM.Model)
	}
}
