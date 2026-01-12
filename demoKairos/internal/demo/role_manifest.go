package demo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jllopis/kairos/pkg/core"
	"gopkg.in/yaml.v3"
)

// LoadRoleManifest loads a role manifest from demo docs by filename.
func LoadRoleManifest(filename string) (core.RoleManifest, error) {
	path, err := findRoleManifestPath(filename)
	if err != nil {
		return core.RoleManifest{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return core.RoleManifest{}, fmt.Errorf("read role manifest: %w", err)
	}
	var manifest core.RoleManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return core.RoleManifest{}, fmt.Errorf("parse role manifest: %w", err)
	}
	return manifest, nil
}

func findRoleManifestPath(filename string) (string, error) {
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "docs", filename),
		filepath.Join(cwd, "demoKairos", "docs", filename),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("role manifest %q not found", filename)
}
