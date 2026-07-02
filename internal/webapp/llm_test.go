package webapp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestGenerateOpenCodePromptUsesZenEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		var input openCodeChatRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if input.Model != "mimo-v2.5-free" {
			t.Fatalf("model = %q, want mimo-v2.5-free", input.Model)
		}
		if len(input.Messages) != 2 {
			t.Fatalf("messages = %d, want 2", len(input.Messages))
		}
		if input.Messages[1].Content != "只回复 pong" {
			t.Fatalf("test prompt = %q, want 只回复 pong", input.Messages[1].Content)
		}
		var body bytes.Buffer
		_ = json.NewEncoder(&body).Encode(openCodeChatResponse{
			Choices: []struct {
				Message openCodeChatMessage `json:"message"`
			}{
				{Message: openCodeChatMessage{Role: "assistant", Content: "OpenClaw prompt"}},
			},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&body),
			Header:     make(http.Header),
		}, nil
	})}

	prompt, err := generateOpenCodePrompt(context.Background(), LLMRequest{
		Model:   "mimo-v2.5-free",
		APIKey:  "test-key",
		BaseURL: "https://opencode.ai/zen/v1",
	}, "只回复 pong", client)
	if err != nil {
		t.Fatalf("GenerateOpenCodePrompt returned error: %v", err)
	}
	if prompt != "OpenClaw prompt" {
		t.Fatalf("prompt = %q, want OpenClaw prompt", prompt)
	}
	if gotPath != "/zen/v1/chat/completions" {
		t.Fatalf("path = %q, want /zen/v1/chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("authorization header = %q, want bearer token", gotAuth)
	}
}

func TestGenerateOpenCodePromptFallsBackWhenDefaultModelDisabled(t *testing.T) {
	var models []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var input openCodeChatRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		models = append(models, input.Model)
		var body bytes.Buffer
		if input.Model == defaultOpenCodeModel {
			_ = json.NewEncoder(&body).Encode(openCodeChatResponse{
				Error: &struct {
					Message string `json:"message"`
					Type    string `json:"type,omitempty"`
					Code    string `json:"code,omitempty"`
				}{Message: "Model is disabled"},
			})
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(&body),
				Header:     make(http.Header),
			}, nil
		}
		_ = json.NewEncoder(&body).Encode(openCodeChatResponse{
			Choices: []struct {
				Message openCodeChatMessage `json:"message"`
			}{
				{Message: openCodeChatMessage{Role: "assistant", Content: "pong"}},
			},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&body),
			Header:     make(http.Header),
		}, nil
	})}

	prompt, err := generateOpenCodePrompt(context.Background(), LLMRequest{
		APIKey:  "test-key",
		BaseURL: "https://opencode.ai/zen/v1",
	}, "只回复 pong", client)
	if err != nil {
		t.Fatalf("GenerateOpenCodePrompt returned error: %v", err)
	}
	if prompt != "pong" {
		t.Fatalf("prompt = %q, want pong", prompt)
	}
	if len(models) != 2 {
		t.Fatalf("models tried = %v, want default plus one fallback", models)
	}
	if models[0] != defaultOpenCodeModel {
		t.Fatalf("first model = %q, want %q", models[0], defaultOpenCodeModel)
	}
	if models[1] == defaultOpenCodeModel || models[1] == "" {
		t.Fatalf("fallback model = %q, want non-default fallback", models[1])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
