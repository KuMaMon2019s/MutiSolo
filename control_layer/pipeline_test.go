package controllayer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAllowsSafeModuleCode(t *testing.T) {
	generation := Generation{Content: "package mathx\nfunc Add(a, b int) int { return a + b }"}

	validation := Validate("write a small helper module", generation, false)

	if validation.Class != ClassSafeModule {
		t.Fatalf("class = %q, want %q", validation.Class, ClassSafeModule)
	}
	if validation.Status != ValidationAllowed {
		t.Fatalf("status = %q, want %q", validation.Status, ValidationAllowed)
	}
}

func TestValidateClassifiesInfrastructureCode(t *testing.T) {
	generation := Generation{Content: "FROM golang:1.26\nRUN go test ./..."}

	validation := Validate("create a Dockerfile", generation, false)

	if validation.Class != ClassInfra {
		t.Fatalf("class = %q, want %q", validation.Class, ClassInfra)
	}
	if validation.Status != ValidationAllowed {
		t.Fatalf("status = %q, want %q", validation.Status, ValidationAllowed)
	}
}

func TestValidateBlocksSystemDesignByDefault(t *testing.T) {
	generation := Generation{Content: "rewrite the architecture and update go.mod"}

	validation := Validate("change system architecture", generation, false)

	if validation.Class != ClassSystemDesign {
		t.Fatalf("class = %q, want %q", validation.Class, ClassSystemDesign)
	}
	if validation.Status != ValidationBlocked {
		t.Fatalf("status = %q, want %q", validation.Status, ValidationBlocked)
	}
}

func TestRunPipelineStoresArtifactOnly(t *testing.T) {
	dir := t.TempDir()

	result, err := RunPipeline(PipelineInput{Prompt: "write a parser helper"}, dir)
	if err != nil {
		t.Fatalf("RunPipeline returned error: %v", err)
	}

	if result.Artifact.Validation.Status != ValidationAllowed {
		t.Fatalf("status = %q, want %q", result.Artifact.Validation.Status, ValidationAllowed)
	}
	if filepath.Dir(result.Path) != dir {
		t.Fatalf("artifact dir = %q, want %q", filepath.Dir(result.Path), dir)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("artifact was not stored: %v", err)
	}
}

func TestRunPipelineProducesDeterministicArtifact(t *testing.T) {
	dir := t.TempDir()

	first, err := RunPipeline(PipelineInput{Prompt: "write a deterministic helper"}, dir)
	if err != nil {
		t.Fatalf("first RunPipeline returned error: %v", err)
	}
	second, err := RunPipeline(PipelineInput{Prompt: "write a deterministic helper"}, dir)
	if err != nil {
		t.Fatalf("second RunPipeline returned error: %v", err)
	}

	if first.Artifact.ID != second.Artifact.ID {
		t.Fatalf("artifact id = %q, want %q", second.Artifact.ID, first.Artifact.ID)
	}
	if first.Path != second.Path {
		t.Fatalf("artifact path = %q, want %q", second.Path, first.Path)
	}
}
