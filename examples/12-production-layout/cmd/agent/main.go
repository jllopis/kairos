// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package main provides the entrypoint for a production Kairos agent.
// This file should remain minimal - all logic goes in internal/app.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jllopis/kairos/examples/12-production-layout/internal/app"
	"github.com/jllopis/kairos/examples/12-production-layout/internal/config"
)

func main() {
	cfgPath := flag.String("config", "./config/config.yaml", "path to config file")
	envOverride := flag.String("env", "", "environment override (dev, prod)")
	flag.Parse()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load configuration with optional environment override
	cfg, err := config.Load(*cfgPath, *envOverride)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create and run the application
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("app error: %v", err)
	}
}
