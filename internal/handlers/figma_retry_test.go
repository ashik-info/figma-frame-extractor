package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestExtractFrameRecoversFromRateLimit(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if req.Header.Get("X-Figma-Token") != "test-token" {
			t.Fatal("retry lost authentication header")
		}
		if attempts == 1 {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Retry-After": {"0"}},
				Body:       io.NopCloser(strings.NewReader(`{"err":"rate limited"}`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"nodes":{"1:2":{"document":{"id":"1:2","name":"Example","type":"FRAME"}}}}`)),
		}, nil
	})}
	var output bytes.Buffer
	if err := ExtractFrame(client, "https://figma.com/design/example/Example?node-id=1-2", "test-token", &output); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || !strings.Contains(output.String(), `"id": "1:2"`) {
		t.Fatalf("attempts = %d, output = %q", attempts, output.String())
	}
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

func TestRequestWithRetry(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		status     int
		wantCalls  int
		wantWaits  []time.Duration
		wantErr    string
	}{
		{name: "exponential fallback", status: 429, wantCalls: 4, wantWaits: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}, wantErr: "HTTP status 429 after 4 attempts"},
		{name: "respect server delay", status: 429, retryAfter: "12", wantCalls: 4, wantWaits: []time.Duration{12 * time.Second, 12 * time.Second, 12 * time.Second}, wantErr: "retry after 12s"},
		{name: "long limit", status: 429, retryAfter: "3600", wantCalls: 1, wantErr: "retry after 1h0m0s (exceeds automatic wait limit"},
		{name: "do not retry forbidden", status: 403, wantCalls: 1},
		{name: "do not retry server error", status: 500, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			var bodies []*trackedBody
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				body := &trackedBody{Reader: strings.NewReader(`{}`)}
				bodies = append(bodies, body)
				return &http.Response{StatusCode: tt.status, Header: http.Header{"Retry-After": {tt.retryAfter}}, Body: body}, nil
			})}
			req, err := http.NewRequest(http.MethodGet, "https://api.figma.com/test", nil)
			if err != nil {
				t.Fatal(err)
			}
			var waits []time.Duration
			resp, err := requestWithRetry(client, req, func(_ context.Context, delay time.Duration) error {
				if !bodies[len(bodies)-1].closed {
					t.Fatal("rate-limit body must be closed before waiting")
				}
				waits = append(waits, delay)
				return nil
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				for _, body := range bodies {
					if !body.closed {
						t.Fatal("rate-limit body was not closed")
					}
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
			}
			if calls != tt.wantCalls || !reflect.DeepEqual(waits, tt.wantWaits) {
				t.Fatalf("calls = %d, waits = %v; want %d, %v", calls, waits, tt.wantCalls, tt.wantWaits)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	now := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		value   string
		attempt int
		want    time.Duration
	}{
		{value: "15", want: 15 * time.Second},
		{value: " 2 ", want: 2 * time.Second},
		{value: "0", want: 0},
		{value: now.Add(time.Minute).Format(http.TimeFormat), want: time.Minute},
		{value: now.Add(-time.Minute).Format(http.TimeFormat), want: time.Second},
		{value: "", attempt: 2, want: 4 * time.Second},
		{value: "invalid", attempt: 1, want: 2 * time.Second},
		{value: "-1", want: time.Second},
		{value: "9223372036854775807", want: time.Duration(1<<63 - 1)},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := retryDelay(tt.value, tt.attempt, now); got != tt.want {
				t.Fatalf("delay = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimitWaitCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		cancel()
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Retry-After": {"60"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})}
	var output bytes.Buffer
	err := ExtractFrameContext(ctx, client, "https://figma.com/design/example/Example?node-id=1-2", "test-token", &output)
	if !errors.Is(err, context.Canceled) || calls != 1 || output.Len() != 0 {
		t.Fatalf("error = %v, calls = %d, output = %q", err, calls, output.String())
	}
}
