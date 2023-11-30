package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayloadFromEvent(t *testing.T) {

	type args struct {
		eventName string
		event     string
	}
	tests := []struct {
		name     string
		args     args
		expected payload
		wantErr  bool
	}{
		{
			name: "discussion created",
			args: args{
				eventName: "discussion",
				event: `{
					"action": "created",
					"discussion": {
						"body": "Some body",
						"title": "Some title",
						"node_id": "D_kwDOKgkPac4AWfor"
					}
				}`,
			},
			expected: payload{title: "Some title", body: "Some body", nodeID: "D_kwDOKgkPac4AWfor"},
		},
		{
			name: "issue opened",
			args: args{
				eventName: "issues",
				event: `{
					"action": "opened",
					"issue": {
						"body": "Some body",
						"title": "Some title",
						"node_id": "D_kwDOKgkPac4AWfor"
					}
				}`,
			},
			expected: payload{title: "Some title", body: "Some body", nodeID: "D_kwDOKgkPac4AWfor"},
		},
		{
			name: "pull_request opened",
			args: args{
				eventName: "pull_request",
				event: `{
					"action": "opened",
					"pull_request": {
						"body": "Some body",
						"title": "Some title",
						"node_id": "D_kwDOKgkPac4AWfor"
					}
				}`,
			},
			expected: payload{title: "Some title", body: "Some body", nodeID: "D_kwDOKgkPac4AWfor"},
		},
		{
			name: "pull_request opened with empty body",
			args: args{
				eventName: "pull_request",
				event: `{
					"action": "opened",
					"pull_request": {
						"title": "Some title",
						"node_id": "D_kwDOKgkPac4AWfor"
					}
				}`,
			},
			expected: payload{title: "Some title", nodeID: "D_kwDOKgkPac4AWfor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := payloadFromEvent(tt.args.eventName, strings.NewReader(tt.args.event))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, p)
		})
	}
}

func TestFilterLabels(t *testing.T) {
	type args struct {
		labels   []Label
		excluded []string
	}
	tests := []struct {
		name     string
		args     args
		expected []Label
	}{
		{
			name: "empty available labels",
			args: args{
				labels:   []Label{},
				excluded: []string{"bug"},
			},
			expected: nil,
		},
		{
			name: "empty excluded",
			args: args{
				labels: []Label{
					{Name: "bug"},
					{Name: "doc"},
				},
				excluded: []string{},
			},
			expected: []Label{
				{Name: "bug"},
				{Name: "doc"},
			},
		},
		{
			name: "happy",
			args: args{
				labels: []Label{
					{Name: "bug"},
					{Name: "doc"},
				},
				excluded: []string{"bug", "todo"},
			},
			expected: []Label{
				{Name: "doc"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, filterLabels(tt.args.labels, tt.args.excluded))
		})
	}
}
