package main

import (
	"io"
	"net/http"
	"strings"
)

type fakeTransport struct {
	response   string
	statusCode int
}

func (t *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Body:       io.NopCloser(strings.NewReader(t.response)),
	}, nil
}
