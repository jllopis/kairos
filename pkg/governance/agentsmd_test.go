package governance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAGENTS(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := []byte("test instructions")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	doc, err := LoadAGENTS(nested)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc == nil {
		t.Fatalf("expected document")
	}
	if doc.Raw != string(content) {
		t.Fatalf("unexpected content: %q", doc.Raw)
	}
}
