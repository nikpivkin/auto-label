package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLabels(t *testing.T) {
	t.Parallel()

	t.Run("happy", func(t *testing.T) {
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: `{"labels":[{"id":"some_id","name":"bug","explanation":"Found a bug in the code."}],"explanation":"A general explanation of the choice of labels"}`,
					},
				},
			},
		}

		b, err := json.Marshal(resp)
		require.NoError(t, err)

		httpClient := &http.Client{}
		httpClient.Transport = &fakeTransport{statusCode: 200, response: string(b)}

		assistant := newLabelingAssistant("key", openai.GPT3Dot5Turbo, httpClient)

		ctx := context.Background()
		request := getLabelsRequest{
			labels:  "LA_kwDOJrb9oM8AAAABTLsvXA:bug",
			payload: "Description of the issue...",
		}

		gptsResp, err := assistant.GetLabels(ctx, request)

		require.NoError(t, err)
		expected := getLabelsResponse{
			Labels: []chosenLabel{
				{ID: "some_id", Name: "bug", Explanation: "Found a bug in the code."},
			},
			Explanation: "A general explanation of the choice of labels",
		}
		assert.Equal(t, expected, gptsResp)
	})

	t.Run("empty msg", func(t *testing.T) {
		mockResponse := `{"choices": [{"message": {"content": ""}}]}`
		httpClient := &http.Client{}
		httpClient.Transport = &fakeTransport{statusCode: 200, response: mockResponse}

		assistant := newLabelingAssistant("key", openai.GPT3Dot5Turbo, httpClient)

		ctx := context.Background()
		request := getLabelsRequest{
			labels:  "LA_kwDOJrb9oM8AAAABTLsvXA:bug",
			payload: "Description of the issue...",
		}

		_, err := assistant.GetLabels(ctx, request)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyMessage)
	})
}
