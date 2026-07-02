package webapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultOpenCodeBaseURL = "https://opencode.ai/zen/v1"
	defaultOpenCodeModel   = "mimo-v2.5-free"
)

var defaultOpenCodeFallbackModels = []string{
	defaultOpenCodeModel,
	"deepseek-v4-flash-free",
	"north-mini-code-free",
	"nemotron-3-ultra-free",
	"big-pickle",
}

type openCodeAPIError struct {
	Status  int
	Message string
	Code    string
}

func (e openCodeAPIError) Error() string {
	if e.Message != "" {
		return "OpenCode request failed: " + e.Message
	}
	if e.Status != 0 {
		return fmt.Sprintf("OpenCode request failed with status %d", e.Status)
	}
	return "OpenCode request failed"
}

type openCodeChatRequest struct {
	Model       string                `json:"model"`
	Messages    []openCodeChatMessage `json:"messages"`
	Temperature float64               `json:"temperature,omitempty"`
}

type openCodeChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openCodeChatResponse struct {
	Choices []struct {
		Message openCodeChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

func GenerateOpenCodePrompt(ctx context.Context, llm LLMRequest, controlledInput string) (string, error) {
	return generateOpenCodePrompt(ctx, llm, controlledInput, &http.Client{Timeout: 60 * time.Second})
}

func TestOpenCodeConnection(ctx context.Context, llm LLMRequest) (string, error) {
	return GenerateOpenCodePrompt(ctx, llm, "只回复 pong")
}

func LLMRequestFromConfig(cfg Config) LLMRequest {
	return LLMRequest{
		Provider: "opencode",
		Model:    defaultOpenCodeModel,
		APIKey:   cfg.LLMAPIKey,
		BaseURL:  defaultOpenCodeBaseURL,
	}
}

func MergeLLMRequest(saved Config, input LLMRequest) LLMRequest {
	if strings.TrimSpace(input.Provider) == "" {
		input.Provider = "opencode"
	}
	if strings.TrimSpace(input.Model) == "" {
		input.Model = defaultOpenCodeModel
	}
	if strings.TrimSpace(input.APIKey) == "" {
		input.APIKey = saved.LLMAPIKey
	}
	if strings.TrimSpace(input.BaseURL) == "" {
		input.BaseURL = defaultOpenCodeBaseURL
	}
	return input
}

func generateOpenCodePrompt(ctx context.Context, llm LLMRequest, controlledInput string, client *http.Client) (string, error) {
	apiKey := strings.TrimSpace(llm.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENCODE_API_KEY"))
	}
	if apiKey == "" {
		return "", fmt.Errorf("OpenCode API Key is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(llm.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultOpenCodeBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	models := openCodeCandidateModels(llm.Model)
	var lastErr error
	for _, model := range models {
		prompt, err := requestOpenCodePrompt(ctx, client, baseURL, apiKey, model, controlledInput)
		if err == nil {
			return prompt, nil
		}
		lastErr = err
		if !isRetryableOpenCodeModelError(err) {
			break
		}
	}
	if len(models) > 1 && isRetryableOpenCodeModelError(lastErr) {
		return "", fmt.Errorf("%w; tried free models: %s", lastErr, strings.Join(models, ", "))
	}
	return "", lastErr
}

func openCodeCandidateModels(model string) []string {
	model = strings.TrimSpace(model)
	if model != "" && model != defaultOpenCodeModel {
		return []string{model}
	}
	candidates := make([]string, 0, len(defaultOpenCodeFallbackModels))
	seen := map[string]bool{}
	for _, candidate := range defaultOpenCodeFallbackModels {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}
	return candidates
}

func requestOpenCodePrompt(ctx context.Context, client *http.Client, baseURL, apiKey, model, controlledInput string) (string, error) {
	body, err := json.Marshal(openCodeChatRequest{
		Model: model,
		Messages: []openCodeChatMessage{
			{
				Role:    "system",
				Content: "You convert structured local requirement context into one bounded OpenClaw implementation prompt. Never request local paths, localhost URLs, or browser blob URLs.",
			},
			{
				Role:    "user",
				Content: controlledInput,
			},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var output openCodeChatResponse
	if err := json.NewDecoder(response.Body).Decode(&output); err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if output.Error != nil && output.Error.Message != "" {
			return "", openCodeAPIError{Status: response.StatusCode, Message: output.Error.Message, Code: output.Error.Code}
		}
		return "", openCodeAPIError{Status: response.StatusCode}
	}
	if output.Error != nil && output.Error.Message != "" {
		return "", openCodeAPIError{Status: response.StatusCode, Message: output.Error.Message, Code: output.Error.Code}
	}
	if len(output.Choices) == 0 {
		return "", fmt.Errorf("OpenCode response did not include choices")
	}
	prompt := strings.TrimSpace(output.Choices[0].Message.Content)
	if prompt == "" {
		return "", fmt.Errorf("OpenCode response was empty")
	}
	return prompt, nil
}

func isRetryableOpenCodeModelError(err error) bool {
	var apiErr openCodeAPIError
	if !errors.As(err, &apiErr) {
		return false
	}
	message := strings.ToLower(apiErr.Message)
	return strings.Contains(message, "model is disabled") ||
		strings.Contains(message, "model disabled") ||
		strings.Contains(message, "model not found") ||
		strings.Contains(message, "model_not_found")
}
