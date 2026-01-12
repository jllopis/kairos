package core

// RoleManifest captures semantic role metadata for an agent.
type RoleManifest struct {
	Role           string
	Responsibility string
	Inputs         []string
	Outputs        []string
	Constraints    map[string]any
}

// RoleManifestProvider exposes role metadata for an agent.
type RoleManifestProvider interface {
	RoleManifest() RoleManifest
}
