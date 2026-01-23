package providers

import (
	"context"
	"fmt"

	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/providers/gemini"
)

// New devuelve el proveedor LLM configurado.
// Por ahora soporta "mock"  y gemini
func New(ctx context.Context, cfg *config.Config) (llm.Provider, error) {
	switch cfg.LLM.Provider {
	case "mock":
		return llm.NewScriptedMockProvider(cfg.LLM.Model, "Your ar SkyGuide, a travel assistant"), nil
	case "gemini":
		return gemini.New(ctx, gemini.WithModel(cfg.LLM.Model))
	default:
		return nil, fmt.Errorf("proveedor LLM no soportado: %s", cfg.LLM.Provider)
	}
}
