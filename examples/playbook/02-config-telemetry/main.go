package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jllopis/kairos/examples/playbook/shared/agent"
	"github.com/jllopis/kairos/examples/playbook/shared/config"
	"github.com/jllopis/kairos/examples/playbook/shared/observability"
	"github.com/jllopis/kairos/examples/playbook/shared/providers"
)

func main() {
	ctx := context.Background()

	// 1. Cargar configuración (soporta flags como --set)
	cfg := config.MustLoad()

	// 2. Inicializar telemetría
	shutdown, err := observability.Init("skyguide", "v0.2.0", cfg)
	if err != nil {
		slog.Error("error inicializando telemetría", "error", err)
		return
	}
	defer shutdown(ctx)

	// 3. Crear el proveedor LLM (Gemini o Mock)
	provider, err := providers.New(ctx, cfg)
	if err != nil {
		slog.Error("error creando el proveedor", "error", err)
		return
	}

	// 4. Crear el agente usando el helper
	skyguide, err := agent.NewSkyGuide("skyguide", provider, cfg)
	if err != nil {
		slog.Error("error creando el agente", "error", err)
		return
	}

	// 5. Ejecutar una interacción
	fmt.Println("\n[SkyGuide está listo]")
	response, err := skyguide.Run(ctx, "Hola, ¿quién eres y cómo puedes ayudarme?")
	if err != nil {
		slog.Error("error ejecutando el agente", "error", err)
		return
	}

	fmt.Printf("\nSKYGUIDE: %s\n", response)
}
