package observability

import (
	"log/slog"
	"os"
	"strings"

	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/telemetry"
)

// Init inicializa el sistema de telemetría y el logger global, devolviendo una función de cierre
func Init(serviceName, version string, cfg *config.Config) (telemetry.ShutdownFunc, error) {
	// Configurar el nivel del logger global (slog)
	var level slog.Level
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	case "disabled", "none", "off":
		level = 99 // Nivel muy alto para silenciarlo
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.ToLower(cfg.Log.Format) == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))

	return telemetry.InitWithConfig(serviceName, version, telemetry.Config{
		Exporter:           cfg.Telemetry.Exporter,
		OTLPEndpoint:       cfg.Telemetry.OTLPEndpoint,
		OTLPInsecure:       cfg.Telemetry.OTLPInsecure,
		OTLPTimeoutSeconds: cfg.Telemetry.OTLPTimeoutSeconds,
		OTLPHeaders:        cfg.Telemetry.OTLPHeaders,
		OTLPUser:           cfg.Telemetry.OTLPUser,
		OTLPToken:          cfg.Telemetry.OTLPToken,
	})
}
