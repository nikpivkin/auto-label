package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

type chosenLabel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Explanation string `json:"explanation"`
}

type getLabelsResponse struct {
	Labels      []chosenLabel `json:"labels"`
	Explanation string        `json:"explanation"`
}

func (r getLabelsResponse) labelIDs() []string {
	var ids []string
	for _, label := range r.Labels {
		ids = append(ids, label.ID)
	}
	return ids
}

var ErrEmptyMessage = errors.New("empty message")

func (a labelingAssistant) GetLabels(ctx context.Context, request getLabelsRequest) (getLabelsResponse, error) {
	prompt := buildPrompt(request.payload, request.labels, request.details)
	chatResponse, err := a.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    a.model,
			Messages: prompt,
		},
	)
	if err != nil {
		return getLabelsResponse{}, fmt.Errorf("failed to create completion: %w", err)
	}

	msg := chatResponse.Choices[0].Message.Content
	if msg == "" {
		return getLabelsResponse{}, ErrEmptyMessage
	}

	var resp getLabelsResponse
	if err := json.Unmarshal([]byte(msg), &resp); err != nil {
		return getLabelsResponse{}, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return resp, nil
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
		systemPrompt += fmt.Sprintf("Also consider the details when assigning labels:\n%s\n", details)
	}
	systemPrompt += fmt.Sprintf("The following labels are available to you in json format:\n%s\n", labels)
	systemPrompt += `Provide the answer as json. For example:
{
  "labels": [
    {
      "id": 1,
      "name": "bug",
      "explanation": "Found a bug in the code."
    },
    {
      "id": 2,
      "name": "enhancement",
      "explanation": "Proposing an enhancement to the functionality."
    },
    {
      "id": 3,
      "name": "question",
      "explanation": "A question that requires clarification."
    }
  ],
  "explanation": "A general explanation of the choice of labels"
}

The "id" field is the label identifier and "explanation" is an explanation of why you chose that label.
`
	return systemPrompt
}
