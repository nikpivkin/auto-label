package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestGetLabels(t *testing.T) {

	mockResponse := `{"choices": [{"message": {"content": "suggestion1,suggestion2"}}]}`
	httpClient := http.DefaultClient
	httpClient.Transport = &fakeTransport{statusCode: 200, response: mockResponse}

	assistant := newLabelingAssistant("key", openai.GPT3Dot5TurboInstruct, httpClient)

	ctx := context.Background()
	request := getLabelsRequest{
		labels:  "LA_kwDOJrb9oM8AAAABTLsvXA:bug",
		payload: "Description of the issue...",
		details: "Additional details...",
	}

	labels, err := assistant.GetLabels(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, []string{"suggestion1", "suggestion2"}, labels)
}
