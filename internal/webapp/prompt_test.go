package webapp

import (
	"strings"
	"testing"
)

func TestBuildPromptSegmentsAndStoresArtifact(t *testing.T) {
	project := Project{
		ID:          "project-1",
		Name:        "Console",
		Description: "OpenClaw control console",
		Plan:        "Connect status, manage requirements, emit prompts",
		Docs:        "Keep generated output separated from runtime",
	}
	req := Requirement{
		ID:          "req-1",
		Title:       "Status panel",
		Description: "Show online and offline state",
	}

	result, err := StorePromptArtifact(project, req, BuildPrompt(project, req), t.TempDir())
	if err != nil {
		t.Fatalf("StorePromptArtifact returned error: %v", err)
	}
	if result.ProjectID != project.ID {
		t.Fatalf("project id = %q, want %q", result.ProjectID, project.ID)
	}
	if len(result.Segments) == 0 {
		t.Fatal("prompt was not segmented")
	}
	if result.ArtifactPath == "" {
		t.Fatal("artifact path is empty")
	}
	if result.DiscordText == "" {
		t.Fatal("discord text is empty")
	}
}

func TestBuildDiscordMessageIncludesCommitInstruction(t *testing.T) {
	project := Project{Name: "Console"}
	req := Requirement{ID: "req-1", Title: "Status panel"}

	message := BuildDiscordMessage(project, req, "do the work")

	if !strings.Contains(message, "OpenClaw A task") {
		t.Fatalf("message does not target OpenClaw A: %q", message)
	}
	if !strings.Contains(message, "commit: <sha>") {
		t.Fatalf("message does not include commit instruction: %q", message)
	}
}

func TestBuildDiscordMessageCanMentionBot(t *testing.T) {
	project := Project{Name: "Console"}
	req := Requirement{ID: "req-1", Title: "Status panel"}

	message := BuildDiscordMessageForBot(project, req, "do the work", "1503733248587730996")

	if !strings.Contains(message, "<@1503733248587730996>") {
		t.Fatalf("message does not mention bot: %q", message)
	}
}

func TestBuildDiscordMessageTargetsAssignedAgent(t *testing.T) {
	project := Project{Name: "Console"}
	req := Requirement{ID: "req-1", Title: "Status panel", AgentID: "openclaw-b"}

	message := BuildDiscordMessage(project, req, "do the work")

	if !strings.Contains(message, "OpenClaw B task") {
		t.Fatalf("message does not target assigned agent: %q", message)
	}
}

func TestBuildDiscordMessageTargetsTailscaleAgentName(t *testing.T) {
	project := Project{Name: "Console"}
	req := Requirement{ID: "req-1", Title: "Status panel", AgentID: "panda"}

	message := BuildDiscordMessage(project, req, "do the work")

	if !strings.Contains(message, "panda task") {
		t.Fatalf("message does not target tailscale agent name: %q", message)
	}
}

func TestBuildRequirementEditorPromptKeepsObjectStorageAssets(t *testing.T) {
	prompt := BuildRequirementEditorPrompt(
		"实现登录页",
		[]map[string]any{{"type": "paragraph"}},
		[]RequirementEditorTencentDoc{{
			Type:            "tencent_doc",
			Title:           "需求说明文档",
			URL:             "https://docs.qq.com/example",
			ReadInstruction: "只读取功能需求和接口要求",
		}},
		[]RequirementEditorAttachment{{
			Name:       "flow.png",
			MIMEType:   "image/png",
			Size:       2048,
			Kind:       "image",
			URL:        "http://127.0.0.1:9000/Mutesolo-assets/2026/06/25/flow.png",
			StorageKey: "2026/06/25/flow.png",
			Source:     "minio",
		}},
	)

	for _, want := range []string{
		"Do not request or read localhost paths",
		"https://docs.qq.com/example",
		"只读取功能需求和接口要求",
		"flow.png",
		"Storage key: 2026/06/25/flow.png",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt does not contain %q: %q", want, prompt)
		}
	}
	if strings.Contains(prompt, "http://127.0.0.1:9000") {
		t.Fatalf("prompt leaked local object URL: %q", prompt)
	}
}

func TestBuildLLMPromptInputIncludesControlledRulesAndDetail(t *testing.T) {
	project := Project{Name: "Mutesolo", Description: "Agent coordination"}
	req := Requirement{ID: "req-1", Title: "需求编辑器", Priority: "high", AgentID: "panda"}
	editor := RequirementEditorPromptRequest{
		PlainText: "实现 BlockNote 需求详情编辑",
		Blocks:    []map[string]any{{"type": "paragraph"}},
	}

	prompt := BuildLLMPromptInput(project, req, editor)

	for _, want := range []string{
		"Backend structured rules",
		"Do not introduce self-modifying runtime behavior",
		"Mutesolo",
		"需求编辑器",
		"实现 BlockNote 需求详情编辑",
		"panda",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt does not contain %q: %q", want, prompt)
		}
	}
}
