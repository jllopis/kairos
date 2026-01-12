package governance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentInstructions holds the contents of an AGENTS.md file.
type AgentInstructions struct {
	Path     string
	Raw      string
	LoadedAt time.Time
}

// LoadAGENTS searches for AGENTS.md starting at startDir and walking upwards.
func LoadAGENTS(startDir string) (*AgentInstructions, error) {
	if strings.TrimSpace(startDir) == "" {
		return nil, errors.New("startDir is required")
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}
	for {
		candidate := filepath.Join(dir, "AGENTS.md")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			raw, err := os.ReadFile(candidate)
			if err != nil {
				return nil, err
			}
			return &AgentInstructions{
				Path:     candidate,
				Raw:      string(raw),
				LoadedAt: time.Now().UTC(),
			}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, nil
}
