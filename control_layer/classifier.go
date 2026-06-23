package controllayer

import "strings"

func Classify(content string) CodeClass {
	text := strings.ToLower(content)

	systemSignals := []string{
		"control_layer/",
		"system prompt",
		"self-modifying",
		"recursive generation",
		"architecture",
		"multi-agent",
		"agent expansion",
		"workflow engine",
		"runtime system",
		"go.mod",
		"cmd/opclawctl",
		"internal/coordination",
	}
	if containsAny(text, systemSignals) {
		return ClassSystemDesign
	}

	infraSignals := []string{
		"dockerfile",
		"docker-compose",
		"kubernetes",
		"terraform",
		"github/workflows",
		"ci.yml",
		"deployment",
	}
	if containsAny(text, infraSignals) {
		return ClassInfra
	}

	return ClassSafeModule
}

func containsAny(text string, signals []string) bool {
	for _, signal := range signals {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return false
}
