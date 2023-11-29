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
		httpClient = &http.Client{}
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
	systemPrompt := `You are the developer.
Your task is to triage discussions on GitHub by defining labels for discussions. You must analyse the title and content of the discussion and assign it one or more of the available labels.
You will receive discussions in the following format:

Title: Some Title.
Body: Some body

Consider the context of the discussion title and text when assigning labels.
`
	if details != "" {
		systemPrompt += fmt.Sprintf("Some details:\n%s\n", details)
	}
	systemPrompt += fmt.Sprintf("The following labels are available to you in json format:\n%s\n", labels)
	systemPrompt += `Provide the response as a string. The response should contain only label identifiers ("id" key from the json above) separated by commas and nothing else. For example: "LA_kwDOJrb9oM8AAAABTLsvXA, LA_kwDOJrb9oM8AAAABTLsvXQ"`
	return systemPrompt
}
