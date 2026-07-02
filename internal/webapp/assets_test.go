package webapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetObjectKeyUsesDateAndExtension(t *testing.T) {
	key := assetObjectKey("abc123", "screen.PNG", "image/png")
	if !strings.HasSuffix(key, "/abc123.png") {
		t.Fatalf("asset key = %q, want generated id with lowercase extension", key)
	}
	if len(strings.Split(key, "/")) != 4 {
		t.Fatalf("asset key = %q, want yyyy/mm/dd/id.ext", key)
	}
}

func TestBuildRequirementEditorPromptDoesNotLeakLocalAssetURL(t *testing.T) {
	prompt := BuildRequirementEditorPrompt(
		"上传截图",
		nil,
		nil,
		[]RequirementEditorAttachment{{
			Name:       "screen.png",
			MIMEType:   "image/png",
			Size:       1024,
			Kind:       "image",
			URL:        "http://127.0.0.1:9000/Mutesolo-assets/2026/06/25/screen.png",
			StorageKey: "2026/06/25/screen.png",
			Source:     "minio",
		}},
	)
	if strings.Contains(prompt, "http://127.0.0.1:9000") {
		t.Fatalf("prompt leaked local URL: %q", prompt)
	}
	if !strings.Contains(prompt, "Storage key: 2026/06/25/screen.png") {
		t.Fatalf("prompt did not include storage key: %q", prompt)
	}
}

func TestUploadLocalFallbackWritesStaticAsset(t *testing.T) {
	root := t.TempDir()
	storage := AssetStorage{LocalDir: root}
	result, err := storage.uploadLocalFallback("screen.png", "image/png", "2026/06/25/screen.png", []byte("png"), assertErr("offline"))
	if err != nil {
		t.Fatalf("uploadLocalFallback returned error: %v", err)
	}
	if result.URL != "/assets/2026/06/25/screen.png" {
		t.Fatalf("url = %q, want /assets path", result.URL)
	}
	data, err := os.ReadFile(filepath.Join(root, "2026", "06", "25", "screen.png"))
	if err != nil {
		t.Fatalf("read fallback asset: %v", err)
	}
	if string(data) != "png" {
		t.Fatalf("fallback asset = %q, want png", data)
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
