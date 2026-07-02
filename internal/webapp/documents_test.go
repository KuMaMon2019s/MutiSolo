package webapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareDocumentInputRejectsPathByDefault(t *testing.T) {
	_, _, err := prepareDocumentInput(context.Background(), DocumentParseRequest{Path: "README.md"})
	if err == nil {
		t.Fatal("prepareDocumentInput accepted a raw path without MUTESOLO_ALLOW_PARSE_PATHS=1")
	}
}

func TestPrepareDocumentInputUsesLocalFallbackStorageKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MUTESOLO_ASSET_FALLBACK_DIR", root)
	key := "2026/06/25/doc.txt"
	fullPath := filepath.Join(root, "2026", "06", "25", "doc.txt")
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	localPath, name, err := prepareDocumentInput(context.Background(), DocumentParseRequest{
		Name:       "doc.txt",
		StorageKey: key,
		Source:     "local_static_fallback",
	})
	if err != nil {
		t.Fatalf("prepareDocumentInput returned error: %v", err)
	}
	if localPath != fullPath {
		t.Fatalf("localPath = %q, want %q", localPath, fullPath)
	}
	if name != "doc.txt" {
		t.Fatalf("name = %q, want doc.txt", name)
	}
}
