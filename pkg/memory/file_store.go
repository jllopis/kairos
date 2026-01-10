package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// FileStore persists entries as JSON lines in a file.
type FileStore struct {
	path string
}

// NewFileStore creates a file-backed memory store.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Store appends a JSON-encoded entry to the file.
func (f *FileStore) Store(_ context.Context, data any) error {
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(f.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	return enc.Encode(data)
}

// Retrieve returns the most recent match from the file.
// If query is nil, it returns the last stored entry.
// If query is a func(any) bool, it returns the last matching entry.
func (f *FileStore) Retrieve(_ context.Context, query any) (any, error) {
	file, err := os.Open(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer file.Close()

	var (
		last any
		hit  bool
	)

	var match func(any) bool
	if query == nil {
		match = func(any) bool { return true }
	} else if fn, ok := query.(func(any) bool); ok {
		match = fn
	} else {
		return nil, errors.New("memory: unsupported query type")
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry any
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		if match(entry) {
			last = entry
			hit = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !hit {
		return nil, ErrNotFound
	}
	return last, nil
}
