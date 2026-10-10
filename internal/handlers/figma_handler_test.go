package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestExtractFrame(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Host != "api.figma.com" || req.URL.Path != "/v1/files/example/nodes" || req.URL.Query().Get("ids") != "1:2" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
		}
		if req.Header.Get("X-Figma-Token") != "test-token" {
			t.Fatal("missing authentication header")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"nodes":{"1:2":{"document":{"id":"1:2","name":"Example","type":"FRAME","ignored":"value"}}}}`)),
		}, nil
	})}
	var output bytes.Buffer
	if err := ExtractFrame(client, "https://www.figma.com/design/example/Example?node-id=1-2", "test-token", &output); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"id\": \"1:2\",\n  \"name\": \"Example\",\n  \"type\": \"FRAME\"\n}\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestExtractFrameErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "HTTP error", status: http.StatusForbidden, body: `{}`, want: "HTTP status 403"},
		{name: "invalid JSON", status: http.StatusOK, body: `{`, want: "parse Figma response JSON"},
		{name: "missing node", status: http.StatusOK, body: `{"nodes":{}}`, want: "node was not found"},
		{name: "null node", status: http.StatusOK, body: `{"nodes":{"1:2":null}}`, want: "node was not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body))}, nil
			})}
			var output bytes.Buffer
			err := ExtractFrame(client, "https://figma.com/file/example/Example?node-id=1-2", "test-token", &output)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if output.Len() != 0 {
				t.Fatal("unexpected output on failure")
			}
		})
	}
}

func TestExtractFrameInvalidURL(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid URL should not trigger a request")
		return nil, nil
	})}
	if err := ExtractFrame(client, "https://example.com", "test-token", io.Discard); err == nil {
		t.Fatal("expected URL parsing error")
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestExtractFrameOutputError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"nodes":{"1:2":{"document":{"id":"1:2"}}}}`)),
		}, nil
	})}
	want := errors.New("output unavailable")
	err := ExtractFrame(client, "https://figma.com/file/example/Example?node-id=1-2", "test-token", failingWriter{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
