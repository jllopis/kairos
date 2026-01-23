package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jllopis/kairos/examples/playbook/shared/agent"
	"github.com/jllopis/kairos/examples/playbook/shared/config"
	"github.com/jllopis/kairos/examples/playbook/shared/observability"
	"github.com/jllopis/kairos/examples/playbook/shared/providers"
	kairosagent "github.com/jllopis/kairos/pkg/agent"
)

func main() {
	ctx := context.Background()

	// 1. Cargar configuración
	cfg := config.MustLoad()

	// 2. Inicializar telemetría
	shutdown, err := observability.Init("skyguide-tools", "0.3.0", cfg)
	if err != nil {
		fmt.Printf("Error initializing telemetry: %v\n", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	// 3. Crear el proveedor LLM
	provider, err := providers.New(ctx, cfg)
	if err != nil {
		fmt.Printf("Error creating LLM provider: %v\n", err)
		os.Exit(1)
	}

	// 4. Crear la herramienta del tiempo
	// weatherTool := &tools.WeatherTool{}

	// 5. Crear el agente con la herramienta
	skyGuide, err := agent.NewSkyGuide("skyguide", provider, cfg,
		//
		// kairosagent.WithTools([]core.Tool{weatherTool}),
		kairosagent.WithMCPServerConfigs(cfg.MCP.Servers),
	)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// 6. Ejecutar el agente
	input := "¿Cuál es el tiempo en Borriana (Castellón)."
	fmt.Printf("\n--- User Message ---\n%s\n", input)

	response, err := skyGuide.Run(ctx, input)
	if err != nil {
		fmt.Printf("Error running agent: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n--- SkyGuide Message ---\n%s\n", response)
}
