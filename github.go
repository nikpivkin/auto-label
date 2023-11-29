package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GitHubGraphQLClient struct {
	token    string
	endpoint string
	client   *http.Client
}

func NewGithubClient(token string, endpoint string, client *http.Client) *GitHubGraphQLClient {
	if client == nil {
		client = &http.Client{}
	}

	return &GitHubGraphQLClient{
		token:    token,
		endpoint: endpoint,
		client:   client,
	}
}

func buildAddLabelsToLabelableRequest(labelableID string, labelIDs []string) string {
	tpl := `{"query":"mutation{addLabelsToLabelable(input:{labelableId: \"%s\", labelIds: [\"%s\"]}){clientMutationId}}"}`
	return fmt.Sprintf(tpl, labelableID, strings.Join(labelIDs, `\", \"`))
}

func (c *GitHubGraphQLClient) ReplaceLabels(ctx context.Context, labelableID string, labelIDs []string) error {
	payload := buildAddLabelsToLabelableRequest(labelableID, labelIDs)
	if _, err := c.request(ctx, payload); err != nil {
		return fmt.Errorf("failed to replace labels: %w", err)
	}
	return nil
}

type gqlRepo struct {
	Labels struct {
		Nodes []Label `json:"nodes"`
	} `json:"labels"`
}

type Label struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ID          string `json:"id"`
}

func (r gqlRepo) labels() []Label {
	return r.Labels.Nodes
}

func buildGetLabelsRequest(owner, name string) string {
	tpl := `{"query":"query{repository(owner:\"%s\",name:\"%s\"){labels(first:100){nodes{name description id}}}}"}`
	return fmt.Sprintf(tpl, owner, name)
}

func (c *GitHubGraphQLClient) FetchRepoLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	data, err := c.request(ctx, buildGetLabelsRequest(owner, repo))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch labels: %w", err)
	}

	var r struct {
		gqlRepo `json:"repository"`
	}

	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return r.gqlRepo.labels(), nil
}

func (c *GitHubGraphQLClient) request(ctx context.Context, payload string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", c.token))
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}
		return nil, fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(b))
	}

	var r ghResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if r.HasErrors() {
		return nil, fmt.Errorf("bad request: %w", r)
	}

	return r.Data, err
}

type ghResponse struct {
	Data   json.RawMessage   `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

func (r ghResponse) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r ghResponse) Error() string {
	if len(r.Errors) == 0 {
		return ""
	}
	var errs []string
	for _, e := range r.Errors {
		errs = append(errs, string(e))
	}

	return strings.Join(errs, ",")
}
