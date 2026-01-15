package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// SkillSpec describes a skill as defined by the AgentSkills spec.
type SkillSpec struct {
	Name          string
	Description   string
	License       string
	Compatibility string
	Metadata      map[string]string
	AllowedTools  []string
	Body          string
	Path          string
	Dir           string
}

const (
	maxNameLen        = 64
	maxDescriptionLen = 1024
	maxCompatLen      = 500
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// LoadDir scans a directory for skill subdirectories with SKILL.md.
func LoadDir(root string) ([]SkillSpec, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []SkillSpec
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(root, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			continue
		}
		skill, err := LoadFile(skillPath)
		if err != nil {
			return nil, err
		}
		out = append(out, skill)
	}
	return out, nil
}

// LoadFile parses a single SKILL.md file.
func LoadFile(path string) (SkillSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SkillSpec{}, err
	}
	content := string(data)
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return SkillSpec{}, err
	}
	var parsed frontmatter
	if err := yaml.Unmarshal([]byte(fm), &parsed); err != nil {
		return SkillSpec{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	allowed, err := normalizeAllowedTools(parsed.AllowedTools)
	if err != nil {
		return SkillSpec{}, err
	}
	dir := filepath.Dir(path)
	spec := SkillSpec{
		Name:          parsed.Name,
		Description:   parsed.Description,
		License:       parsed.License,
		Compatibility: parsed.Compatibility,
		Metadata:      parsed.Metadata,
		AllowedTools:  allowed,
		Body:          strings.TrimSpace(body),
		Path:          path,
		Dir:           dir,
	}
	if err := validate(spec); err != nil {
		return SkillSpec{}, err
	}
	return spec, nil
}

type frontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Metadata      map[string]string `yaml:"metadata"`
	AllowedTools  any               `yaml:"allowed-tools"`
}

func splitFrontmatter(content string) (string, string, error) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return "", "", errors.New("missing frontmatter")
	}
	parts := strings.SplitN(trimmed, "---", 3)
	if len(parts) < 3 {
		return "", "", errors.New("invalid frontmatter")
	}
	fm := strings.TrimSpace(parts[1])
	body := strings.TrimSpace(parts[2])
	return fm, body, nil
}

func validate(spec SkillSpec) error {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		return fmt.Errorf("name exceeds %d characters", maxNameLen)
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("name must match %s", namePattern.String())
	}
	dirName := filepath.Base(spec.Dir)
	if dirName != name {
		return fmt.Errorf("name must match directory name (%s)", dirName)
	}
	desc := strings.TrimSpace(spec.Description)
	if desc == "" {
		return errors.New("description is required")
	}
	if utf8.RuneCountInString(desc) > maxDescriptionLen {
		return fmt.Errorf("description exceeds %d characters", maxDescriptionLen)
	}
	compat := strings.TrimSpace(spec.Compatibility)
	if compat != "" && utf8.RuneCountInString(compat) > maxCompatLen {
		return fmt.Errorf("compatibility exceeds %d characters", maxCompatLen)
	}
	return nil
}

func normalizeAllowedTools(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case string:
		return splitAllowed(sanitizeAllowed(v)), nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, errors.New("allowed-tools must be string list")
			}
			out = append(out, sanitizeAllowed(strings.TrimSpace(str)))
		}
		return dedupe(out), nil
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, sanitizeAllowed(strings.TrimSpace(item)))
		}
		return dedupe(out), nil
	default:
		return nil, errors.New("allowed-tools must be string or list")
	}
}

func splitAllowed(input string) []string {
	fields := strings.Fields(input)
	return dedupe(fields)
}

func sanitizeAllowed(input string) string {
	replacer := strings.NewReplacer(
		"( ", "(",
		" )", ")",
		": ", ":",
		" :", ":",
	)
	return replacer.Replace(input)
}

func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
