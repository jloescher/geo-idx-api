package sync

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/mlsupstream"
)

const maxRetryAfterSleep = 60 * time.Second

type rateWaiter interface {
	Wait(ctx context.Context) error
}

type oDataGETResult struct {
	Status int
	Body   []byte
	Header http.Header
}

// doODataGET performs a GET with cluster rate limiting and retries on 429/503.
func doODataGET(ctx context.Context, client *http.Client, limiter rateWaiter, req *http.Request, maxRetries int, provider string) (oDataGETResult, error) {
	if maxRetries < 1 {
		maxRetries = 1
	}

	var lastStatus int
	var lastBody []byte
	var lastHeader http.Header

	for attempt := 0; attempt < maxRetries; attempt++ {
		if limiter != nil {
			if err := limiter.Wait(ctx); err != nil {
				return oDataGETResult{}, err
			}
		}

		reqClone := req.Clone(ctx)
		resp, err := client.Do(reqClone)
		if err != nil {
			if ctx.Err() != nil {
				return oDataGETResult{}, ctx.Err()
			}
			return oDataGETResult{}, mlsupstream.ErrTimeout{Provider: provider, Cause: err}
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return oDataGETResult{}, readErr
		}

		lastStatus = resp.StatusCode
		lastBody = body
		lastHeader = resp.Header

		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != http.StatusServiceUnavailable {
			return oDataGETResult{Status: lastStatus, Body: lastBody, Header: lastHeader}, nil
		}

		if attempt+1 >= maxRetries {
			break
		}

		wait := retryBackoff(attempt, resp)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return oDataGETResult{}, ctx.Err()
		case <-timer.C:
		}
	}

	if lastStatus == http.StatusTooManyRequests {
		return oDataGETResult{Status: lastStatus, Body: lastBody, Header: lastHeader},
			mlsupstream.ErrRateLimited{Provider: provider, Status: lastStatus}
	}
	if lastStatus == http.StatusServiceUnavailable {
		return oDataGETResult{Status: lastStatus, Body: lastBody, Header: lastHeader},
			mlsupstream.ErrTimeout{Provider: provider, Status: lastStatus}
	}
	return oDataGETResult{Status: lastStatus, Body: lastBody, Header: lastHeader}, nil
}

func retryBackoff(attempt int, resp *http.Response) time.Duration {
	wait := time.Duration(attempt+1) * 500 * time.Millisecond
	if resp != nil {
		if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
			wait = ra
		}
	}
	if wait > maxRetryAfterSleep {
		wait = maxRetryAfterSleep
	}
	return wait
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

func httpStatusError(provider string, status int) error {
	switch status {
	case http.StatusTooManyRequests:
		return ErrUpstreamRateLimited{Provider: provider, Status: status}
	case http.StatusServiceUnavailable:
		return ErrUpstreamTimeout{Provider: provider, Status: status}
	default:
		return fmt.Errorf("%s fetch http %d", provider, status)
	}
}
