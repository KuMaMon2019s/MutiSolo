package controllayer

import (
	"fmt"
	"strings"
)

func Generate(prompt string) (Generation, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return Generation{}, fmt.Errorf("prompt is required")
	}

	return Generation{
		Content: fmt.Sprintf(`// Generated artifact only. Do not auto-apply to runtime.
// Prompt: %s

package generated

func Output() string {
	return %q
}
`, sanitizeLine(prompt), prompt),
	}, nil
}

func sanitizeLine(text string) string {
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}
