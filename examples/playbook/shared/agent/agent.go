package agent

import (
	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/llm"
)

// NewSkyGuide crea una nueva instancia del agente SkyGuide con la configuración estándar
func NewSkyGuide(id string, provider llm.Provider, cfg *config.Config, opts ...agent.Option) (*agent.Agent, error) {
	// Obtenemos la configuración específica para este ID de agente (si existe en el yaml)
	agentCfg := cfg.AgentConfigFor(id)

	baseOpts := []agent.Option{
		agent.WithRole("You are SkyGuide, a travel assistant. Be helpful, professional and concise."),
		agent.WithModel(cfg.LLM.Model),
		agent.WithDisableActionFallback(agentCfg.DisableActionFallback),
		agent.WithActionFallbackWarning(agentCfg.WarnOnActionFallback),
	}

	// Combinamos las opciones base con las proporcionadas
	allOpts := append(baseOpts, opts...)

	return agent.New(id, provider, allOpts...)
}
