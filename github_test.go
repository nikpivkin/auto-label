package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRepoLabels(t *testing.T) {
	t.Parallel()

	t.Run("happy", func(t *testing.T) {
		fakeResponse := `{
  "data": {
    "repository": {
      "labels": {
        "nodes": [
          {
            "id": "MDU6TGFiZWw1NTU0NDg4MA==",
            "name": "bug",
            "description": "Something isn't working"
          },
          {
            "id": "MDU6TGFiZWw1NTU0NDg4MQ==",
            "name": "enhancement",
            "description": "New feature or request"
          },
          {
            "id": "MDU6TGFiZWw1NTU0NDg4Mg==",
            "name": "question",
            "description": "Further information is requested"
          }
        ]
      }
    }
  }
}`
		client := newFakeGhClient(200, fakeResponse)
		repoLabels, err := client.FetchRepoLabels(context.TODO(), "owner", "repo")
		require.NoError(t, err)

		expected := []Label{
			{ID: "MDU6TGFiZWw1NTU0NDg4MA==", Name: "bug", Description: "Something isn't working"},
			{ID: "MDU6TGFiZWw1NTU0NDg4MQ==", Name: "enhancement", Description: "New feature or request"},
			{ID: "MDU6TGFiZWw1NTU0NDg4Mg==", Name: "question", Description: "Further information is requested"},
		}
		assert.Equal(t, expected, repoLabels)
	})

	t.Run("with errors", func(t *testing.T) {
		fakeResponse := `{
  "data": {
    "repository": null
  },
  "errors": [
    {
      "message": "Could not resolve to a Repository with the name 'nonexistent-repo'.",
      "type": "NOT_FOUND",
      "path": [
        "repository"
      ],
      "locations": [
        {
          "line": 2,
          "column": 3
        }
      ]
    }
  ]
}`
		client := newFakeGhClient(200, fakeResponse)
		_, err := client.FetchRepoLabels(context.TODO(), "owner", "repo")
		require.Error(t, err)
		assert.ErrorContains(t, err, "Could not resolve to a Repository with the name 'nonexistent-repo'.")
	})

	t.Run("not 200 status", func(t *testing.T) {
		client := newFakeGhClient(500, "")
		_, err := client.FetchRepoLabels(context.TODO(), "owner", "repo")
		require.Error(t, err)
		assert.ErrorContains(t, err, "status code: 500")
	})
}

func TestReplaceLabels(t *testing.T) {
	t.Parallel()

	t.Run("happy", func(t *testing.T) {
		fakeResponse := `{
  "data": {
    "addLabelsToLabelable": {
      "clientMutationId": null
    }
  }
}`

		client := newFakeGhClient(200, fakeResponse)
		err := client.ReplaceLabels(context.TODO(), "MDU6SXNzdWUzOTk5MDE2MTg=", []string{"MDU6TGFiZWw1NTU0NDg4MA=="})
		require.NoError(t, err)
	})

	t.Run("with errors", func(t *testing.T) {
		fakeResponse := `{
  "data": {
    "addLabelsToLabelable": null
  },
  "errors": [
    {
      "message": "Could not resolve to a node with the global id of 'MDU6SXNzdWUzOTk5MDE2MTg='.",
      "type": "NOT_FOUND",
      "path": [
        "addLabelsToLabelable"
      ],
      "locations": [
        {
          "line": 1,
          "column": 2
        }
      ]
    }
  ]
}`

		client := newFakeGhClient(200, fakeResponse)
		err := client.ReplaceLabels(context.TODO(), "MDU6SXNzdWUzOTk5MDE2MTg=", []string{"MDU6TGFiZWw1NTU0NDg4MA=="})
		require.Error(t, err)
		assert.ErrorContains(t, err, "Could not resolve to a node with the global id of 'MDU6SXNzdWUzOTk5MDE2MTg='")
	})

	t.Run("not 200 status", func(t *testing.T) {
		client := newFakeGhClient(500, "")
		err := client.ReplaceLabels(context.TODO(), "MDU6SXNzdWUzOTk5MDE2MTg=", []string{"MDU6TGFiZWw1NTU0NDg4MA=="})
		require.Error(t, err)
		assert.ErrorContains(t, err, "status code: 500")
	})
}

func TestAddComment(t *testing.T) {
	t.Parallel()

	t.Run("happy", func(t *testing.T) {
		fakeResponse := `{
  "data": {
    "addComment": {
      "clientMutationId": "your_client_mutation_id_here"
    }
  }
}`
		client := newFakeGhClient(200, fakeResponse)
		err := client.AddComment(context.TODO(), "some_id", "some_body")
		require.NoError(t, err)
	})

	t.Run("with errors", func(t *testing.T) {
		fakeResponse := `{
  "errors": [
    {
      "message": "Node with subjectId not found. Please check the provided identifier.",
      "locations": [
        {
          "line": 1,
          "column": 1
        }
      ],
      "path": ["addComment"]
    }
  ],
  "data": null
}`

		client := newFakeGhClient(200, fakeResponse)
		err := client.AddComment(context.TODO(), "some_id", "some_body")
		require.Error(t, err)
		assert.ErrorContains(t, err, "Node with subjectId not found. Please check the provided identifier.")
	})

	t.Run("not 200 status", func(t *testing.T) {
		client := newFakeGhClient(500, "")
		err := client.AddDiscussionComment(context.TODO(), "some_id", "some_body")
		require.Error(t, err)
		assert.ErrorContains(t, err, "status code: 500")
	})
}

func TestAddDiscussionComment(t *testing.T) {
	t.Parallel()

	t.Run("happy", func(t *testing.T) {
		fakeResponse := `{
  "data": {
    "addDiscussionComment": {
      "clientMutationId": "your_client_mutation_id_here"
    }
  }
}`

		client := newFakeGhClient(200, fakeResponse)
		err := client.AddDiscussionComment(context.TODO(), "some_id", "some_body")
		require.NoError(t, err)
	})

	t.Run("with errors", func(t *testing.T) {
		fakeResponse := `{
  "errors": [
    {
      "message": "Error in executing the query. Please check the parameters.",
      "locations": [
        {
          "line": 1,
          "column": 1
        }
      ],
      "path": ["addDiscussionComment"]
    }
  ],
  "data": {
    "addDiscussionComment": null
  }
}`

		client := newFakeGhClient(200, fakeResponse)
		err := client.AddDiscussionComment(context.TODO(), "some_id", "some_body")
		require.Error(t, err)
		assert.ErrorContains(t, err, "Error in executing the query. Please check the parameters.")
	})

	t.Run("not 200 status", func(t *testing.T) {
		client := newFakeGhClient(500, "")
		err := client.AddDiscussionComment(context.TODO(), "some_id", "some_body")
		require.Error(t, err)
		assert.ErrorContains(t, err, "status code: 500")
	})
}

func newFakeGhClient(statusCode int, response string) *GitHubGraphQLClient {
	httpClient := &http.Client{}
	httpClient.Transport = &fakeTransport{statusCode: statusCode, response: response}
	client := NewGithubClient("token", "url", httpClient)
	return client
}
