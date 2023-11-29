package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

func envOrFatal(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("env %q is required\n", key)
	}
	return val
}

type config struct {
	timeout         int
	eventName       string
	eventPath       string
	details         string
	gptToken        string
	gptModel        string
	ghToken         string
	graphQLEndpoint string
	repoOwner       string
	repoName        string
}

const defaultTimeoutS = 60

func main() {

	timeout := flag.Int("timeout", defaultTimeoutS, fmt.Sprintf("timeout in seconds (default %ds)", defaultTimeoutS))
	gptModel := flag.String("gpt-model", openai.GPT3Dot5Turbo, fmt.Sprintf("the chat-gpt model used. (default %s)", openai.GPT3Dot5Turbo))
	details := flag.String("details", "", "additional details for label suggestions.")

	flag.Parse()

	ghRepo := envOrFatal("GITHUB_REPOSITORY")
	parts := strings.Split(ghRepo, "/")

	c := config{
		timeout:         *timeout,
		eventName:       envOrFatal("GITHUB_EVENT_NAME"),
		eventPath:       envOrFatal("GITHUB_EVENT_PATH"),
		details:         *details,
		gptToken:        envOrFatal("OPENAI_API_KEY"),
		gptModel:        *gptModel,
		ghToken:         envOrFatal("GITHUB_TOKEN"),
		graphQLEndpoint: envOrFatal("GITHUB_GRAPHQL_URL"),
		repoOwner:       parts[0],
		repoName:        parts[1],
	}

	if err := run(c); err != nil {
		log.Fatal(err)
	}
}

func run(cfg config) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.timeout)*time.Second)
	defer cancel()

	ghapi := NewGithubClient(cfg.ghToken, cfg.graphQLEndpoint, nil)

	availableLabels, err := ghapi.FetchRepoLabels(ctx, cfg.repoOwner, cfg.repoName)
	if err != nil {
		return err
	}

	ef, err := os.Open(cfg.eventPath)
	if err != nil {
		return fmt.Errorf("failed to open event file: %w", err)
	}

	payload, err := payloadFromEvent(cfg.eventName, ef)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	assistant := newLabelingAssistant(cfg.gptToken, cfg.gptModel, nil)

	labels, err := json.Marshal(availableLabels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	gptsLabels, err := assistant.GetLabels(ctx, getLabelsRequest{
		labels:  string(labels),
		payload: payload.String(),
		details: cfg.details,
	})

	if err != nil {
		return err
	}

	if len(gptsLabels) == 0 {
		log.Println("ChatGPT returned an empty message.")
		return nil
	}

	if err := ghapi.ReplaceLabels(ctx, payload.nodeID, gptsLabels); err != nil {
		return err
	}

	return nil
}

type payload struct {
	nodeID string
	title  string
	body   string
}

func (d payload) String() string {
	return fmt.Sprintf("Title: %s\nBody: %s", d.title, d.body)
}

func payloadFromEvent(eventName string, r io.Reader) (payload, error) {
	var event map[string]any
	if err := json.NewDecoder(r).Decode(&event); err != nil {
		return payload{}, fmt.Errorf("failed to decode %s event: %w", eventName, err)
	}

	objName := eventName
	if eventName == "issues" {
		objName = "issue"
	}

	obj, exist := event[objName]
	if !exist {
		return payload{}, fmt.Errorf("invalid event name %q", eventName)
	}

	if m, ok := obj.(map[string]any); ok {
		title := m["title"].(string)
		body := m["body"].(string)
		id := m["node_id"].(string)
		return payload{title: title, body: body, nodeID: id}, nil
	}

	return payload{}, errors.New("invalid event")
}
