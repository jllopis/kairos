// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package config handles configuration loading with environment layering.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	App struct {
		Name     string `yaml:"name"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"app"`

	LLM struct {
		Provider string `yaml:"provider"`
		Model    string `yaml:"model"`
		BaseURL  string `yaml:"base_url"`
	} `yaml:"llm"`

	Memory struct {
		Backend string `yaml:"backend"`
	} `yaml:"memory"`

	Governance struct {
		Enable   bool     `yaml:"enable"`
		Policies []string `yaml:"policies"`
	} `yaml:"governance"`

	Telemetry struct {
		Exporter    string `yaml:"exporter"`
		Endpoint    string `yaml:"endpoint"`
		ServiceName string `yaml:"service_name"`
	} `yaml:"telemetry"`

	MCP struct {
		Enable  bool        `yaml:"enable"`
		Servers []MCPServer `yaml:"servers"`
	} `yaml:"mcp"`
}

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	Name    string   `yaml:"name"`
	Command []string `yaml:"command"`
}

// Load reads configuration from base file and optional environment override.
// Environment files are named config.<env>.yaml in the same directory.
func Load(basePath string, env string) (*Config, error) {
	cfg := &Config{}

	// Load base config
	if err := loadYAML(basePath, cfg); err != nil {
		return nil, fmt.Errorf("loading base config: %w", err)
	}

	// Load environment override if specified
	if env != "" {
		dir := filepath.Dir(basePath)
		envPath := filepath.Join(dir, fmt.Sprintf("config.%s.yaml", env))
		if _, err := os.Stat(envPath); err == nil {
			if err := loadYAML(envPath, cfg); err != nil {
				return nil, fmt.Errorf("loading %s config: %w", env, err)
			}
		}
	}

	return cfg, nil
}

func loadYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}
