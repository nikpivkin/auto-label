package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
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

const (
	defaultTimeoutS = 60

	discussionsLink = "https://github.com/nikpivkin/auto-label/discussions"
	issuesLink      = "https://github.com/nikpivkin/auto-label/issues"
)

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

	gptResponse, err := assistant.GetLabels(ctx, getLabelsRequest{
		labels:  string(labels),
		payload: payload.String(),
		details: cfg.details,
	})

	if errors.Is(err, ErrEmptyMessage) {
		log.Println("ChatGPT returned an empty message.")
		return nil
	} else if err != nil {
		return err
	}

	if err := ghapi.ReplaceLabels(ctx, payload.nodeID, gptResponse.labelIDs()); err != nil {
		return err
	}

	artifactName := cfg.eventName
	if artifactName == "issues" {
		artifactName = "issue"
	}

	body := strconv.Quote(createComment(artifactName, cfg.repoOwner, cfg.repoName, gptResponse))

	var addCommentFn = ghapi.AddComment
	if cfg.eventName == "discussion" {
		addCommentFn = ghapi.AddDiscussionComment
	}

	if err := addCommentFn(ctx, payload.nodeID, body); err != nil {
		return err
	}

	return nil
}

func createComment(artifactName string, repoOwner string, repoName string, r getLabelsResponse) string {
	body := `**Automated Label Assignment:**

Hello there! 👋 This is an automated message from the ChatGPT Auto Labeler Action.

The ChatGPT Auto Labeler has analyzed the title and content of this %s and assigned the following labels:
`
	body = fmt.Sprintf(body, artifactName)

	for _, l := range r.Labels {
		body += fmt.Sprintf("- **%s**: %s", l.Name, l.Explanation)
	}

	body += "\n\n" + r.Explanation

	body += "\n\nIf you have any concerns or need further clarification, please don't hesitate to reach out. You can discuss this further in the "
	body += fmt.Sprintf("[Issues](%s) or [Discussions](%s) section of our repository.\n\n", issuesLink, discussionsLink)
	body += "You can click on the following links to quickly access each label:\n"

	for _, l := range r.Labels {
		labelURL := fmt.Sprintf("https://github.com/%s/%s/labels/%s", repoOwner, repoName, url.QueryEscape(l.Name))
		body += fmt.Sprintf("- [%s](%s)", l.Name, labelURL)
	}

	footer := "\n\n*Note: This message is generated automatically, and the labels were assigned based on the analysis of the %s's content.*"

	footer = fmt.Sprintf(footer, artifactName)
	return body + footer
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
		p := payload{}
		p.nodeID = m["node_id"].(string)
		p.title = m["title"].(string)
		if body, ok := m["body"].(string); ok {
			p.body = body
		}

		return p, nil
	}

	return payload{}, errors.New("invalid event")
}
