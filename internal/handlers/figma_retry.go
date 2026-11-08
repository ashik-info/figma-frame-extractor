package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	maxFigmaRetries   = 3
	maxFigmaRetryWait = 5 * time.Minute
)

func requestWithRetry(client *http.Client, req *http.Request, wait func(context.Context, time.Duration) error) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		delay := retryDelay(resp.Header.Get("Retry-After"), attempt, time.Now())
		// Release the connection before waiting. Limit draining an unexpected error body.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()

		if attempt >= maxFigmaRetries {
			return nil, fmt.Errorf("Figma request failed with HTTP status 429 after %d attempts; retry after %s", attempt+1, delay)
		}
		if delay > maxFigmaRetryWait {
			return nil, fmt.Errorf("Figma request failed with HTTP status 429; retry after %s (exceeds automatic wait limit of %s); check your Figma file's plan and seat rate limits", delay, maxFigmaRetryWait)
		}
		log.Printf("Figma rate limit reached (HTTP 429); retrying in %s (retry %d/%d)", delay, attempt+1, maxFigmaRetries)
		if err := wait(req.Context(), delay); err != nil {
			return nil, fmt.Errorf("wait for Figma rate limit: %w", err)
		}
	}
}

func retryDelay(value string, attempt int, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		// Saturate durations that would overflow rather than retrying too early.
		const maxDuration time.Duration = 1<<63 - 1
		if seconds > int64(maxDuration/time.Second) {
			return maxDuration
		}
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		if delay := date.Sub(now); delay > 0 {
			return delay
		}
	}
	// Missing or invalid headers fall back to exponential backoff: 1s, 2s, 4s.
	return time.Second << attempt
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
