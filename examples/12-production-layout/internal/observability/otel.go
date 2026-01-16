// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package observability sets up OpenTelemetry tracing and metrics.
package observability

import (
	"context"

	"github.com/jllopis/kairos/examples/12-production-layout/internal/config"
	"github.com/jllopis/kairos/pkg/telemetry"
)

// Init initializes OpenTelemetry with the provided configuration.
// Returns a shutdown function that should be called on application exit.
func Init(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
	teleCfg := telemetry.Config{
		Exporter:     cfg.Telemetry.Exporter,
		OTLPEndpoint: cfg.Telemetry.Endpoint,
	}

	return telemetry.InitWithConfig(cfg.Telemetry.ServiceName, "1.0.0", teleCfg)
}
