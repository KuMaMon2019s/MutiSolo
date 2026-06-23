package controllayer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func StoreArtifact(dir string, artifact Artifact) (string, error) {
	if dir == "" {
		dir = "artifacts"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create artifacts dir: %w", err)
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode artifact: %w", err)
	}
	path := filepath.Join(dir, artifact.ID+".json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, append(data, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write artifact: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return "", fmt.Errorf("replace artifact: %w", err)
	}
	return path, nil
}

func NewArtifactID(prompt string, generation Generation, validation Validation) string {
	slug := strings.ToLower(prompt)
	slug = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, slug)
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 {
		slug = slug[:40]
	}
	if slug == "" {
		slug = "artifact"
	}
	sum := sha256.Sum256([]byte(prompt + "\n" + generation.Content + "\n" + string(validation.Class) + "\n" + string(validation.Status)))
	return fmt.Sprintf("%s-%x", slug, sum[:6])
}
