package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type LlmModel string

const (
	ArticleLLM LlmModel = "LM_GEN"
	ImproveLLM LlmModel = "LM_IMP"
	ExpandLLM  LlmModel = "LM_EXP"
)

type llmConfig struct {
	baseURL string
	apiKey  string
	model   string
}

func configFor(model LlmModel) (llmConfig, error) {
	cfg := llmConfig{
		baseURL: os.Getenv(string(model) + "_BASE_URL"),
		apiKey:  os.Getenv(string(model)),
		model:   os.Getenv(string(model) + "_MODEL"),
	}
	if cfg.baseURL == "" || cfg.model == "" || cfg.apiKey == "" {
		slog.Error("Unable to get config for llm model", "Model", model)
		return llmConfig{}, fmt.Errorf("Unable to get config for %s", model)
	}
	return cfg, nil
}

func CallLLM(model LlmModel, userPrompt string, systemPrompt string) (string, error) {
	cfg, err := configFor(model)
	if err != nil {
		return "", err
	}

	return doChatCompletion(cfg, chatCompletionRequest{
		Model: cfg.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
}

func CallLLMStructured(model LlmModel, p prompt, schema string) (string, error) {
	cfg, err := configFor(model)
	if err != nil {
		return "", err
	}

	// DeepSeek's OpenAI-compatible API does not support OpenAI's
	// response_format.json_schema, so we use json_object mode and pass the
	// schema in the system prompt instead. This also works on OpenAI and most
	// compatible servers. (The word "JSON" must appear in the prompt for
	// json_object mode to be allowed.)
	systemPrompt := p.System + "\nRespond only with valid JSON conforming to this JSON schema:\n" + schema

	return doChatCompletion(cfg, chatCompletionRequest{
		Model: cfg.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: p.User},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
	})
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionRequest is the OpenAI-compatible chat completion payload.
type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

// responseFormat asks the model for JSON output.
type responseFormat struct {
	Type string `json:"type"`
}

// chatCompletionResponse is the OpenAI-compatible chat completion response.
type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// doChatCompletion performs the HTTP call and extracts the reply text.
func doChatCompletion(cfg llmConfig, reqBody chatCompletionRequest) (string, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal llm request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call llm: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse llm response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}

	return stripCodeFences(parsed.Choices[0].Message.Content), nil
}

// stripCodeFences removes markdown ```json ... ``` wrappers some providers add.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
