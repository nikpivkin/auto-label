package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type labelingAssistant struct {
	model  string
	client *openai.Client
}

func newLabelingAssistant(token string, model string, httpClient *http.Client) *labelingAssistant {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	clientCfg := openai.DefaultConfig(token)
	clientCfg.HTTPClient = httpClient
	client := openai.NewClientWithConfig(clientCfg)
	return &labelingAssistant{client: client, model: model}
}

type getLabelsRequest struct {
	labels  string
	payload string
	details string
}

func (a labelingAssistant) GetLabels(ctx context.Context, request getLabelsRequest) ([]string, error) {
	prompt := buildPrompt(request.payload, request.labels, request.details)
	resp, err := a.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    a.model,
			Messages: prompt,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create completion: %w", err)
	}

	msg := resp.Choices[0].Message.Content
	if msg == "" {
		return nil, nil
	}

	return strings.Split(msg, ","), nil
}

func buildPrompt(payload, labels, details string) []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: buildSystemPrompt(labels, details),
		},
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: payload,
		},
	}
}

func buildSystemPrompt(labels string, details string) string {
	systemPrompt := "You are the developer.\nYour task is to triage discussions on GitHub by defining labels for discussions.\n"
	if details != "" {
		systemPrompt += fmt.Sprintf("Some details:\n%s\n", details)
	}
	systemPrompt += fmt.Sprintf("Available labels:\n%s\n", labels)
	systemPrompt += `Provide the response as a string. The response should contain label IDs separated by commas. Consider the context of the discussion title and text when assigning labels. For example: "LA_kwDOJrb9oM8AAAABTLsvXA, LA_kwDOJrb9oM8AAAABTLsvXQ"`
	return systemPrompt
}
