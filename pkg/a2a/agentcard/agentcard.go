package agentcard

import a2av1 "github.com/jllopis/kairos/pkg/a2a/types"

// Config describes AgentCard fields that can be derived from runtime settings.
type Config struct {
	ProtocolVersion      string
	Name                 string
	Description          string
	Version              string
	DocumentationURL     string
	IconURL              string
	SupportedInterfaces  []*a2av1.AgentInterface
	Capabilities         *a2av1.AgentCapabilities
	SecuritySchemes      map[string]*a2av1.SecurityScheme
	Security             []*a2av1.Security
	DefaultInputModes    []string
	DefaultOutputModes   []string
	Skills               []*a2av1.AgentSkill
	SupportsExtendedCard bool
	Provider             *a2av1.AgentProvider
	Signatures           []*a2av1.AgentCardSignature
}

// Build assembles an AgentCard from the provided config.
func Build(cfg Config) *a2av1.AgentCard {
	protocolVersion := stringPtr(cfg.ProtocolVersion)
	documentationURL := stringPtr(cfg.DocumentationURL)
	iconURL := stringPtr(cfg.IconURL)
	supportsExtended := boolPtr(cfg.SupportsExtendedCard)

	return &a2av1.AgentCard{
		ProtocolVersion:           protocolVersion,
		Name:                      cfg.Name,
		Description:               cfg.Description,
		Version:                   cfg.Version,
		DocumentationUrl:          documentationURL,
		IconUrl:                   iconURL,
		SupportedInterfaces:       cfg.SupportedInterfaces,
		Capabilities:              cfg.Capabilities,
		SecuritySchemes:           cfg.SecuritySchemes,
		Security:                  cfg.Security,
		DefaultInputModes:         cfg.DefaultInputModes,
		DefaultOutputModes:        cfg.DefaultOutputModes,
		Skills:                    cfg.Skills,
		SupportsExtendedAgentCard: supportsExtended,
		Provider:                  cfg.Provider,
		Signatures:                cfg.Signatures,
	}
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
