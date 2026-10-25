package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Result is the outcome of checking one URL.
type Result struct {
	URL     string
	Status  int           // HTTP status code; 0 if the request never completed
	Latency time.Duration // time taken by the final attempt
	Err     error         // non-nil if every attempt failed at the transport level
}

// OK reports whether the check counts as healthy: a completed request
// with a 2xx or 3xx status and no transport error.
func (r Result) OK() bool {
	return r.Err == nil && r.Status >= 200 && r.Status < 400
}

// Check performs a single HTTP GET against url and records the outcome.
func Check(ctx context.Context, client *http.Client, url string) Result {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{URL: url, Latency: time.Since(start), Err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{URL: url, Latency: time.Since(start), Err: err}
	}
	defer resp.Body.Close()
	return Result{URL: url, Status: resp.StatusCode, Latency: time.Since(start)}
}

// checkWithRetry retries transport errors and 5xx responses with a
// simple linear backoff, honouring context cancellation between tries.
func checkWithRetry(ctx context.Context, client *http.Client, url string, retries int) Result {
	var res Result
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 200 * time.Millisecond
			select {
			case <-ctx.Done():
				return Result{URL: url, Err: ctx.Err()}
			case <-time.After(backoff):
			}
		}
		res = Check(ctx, client, url)
		if res.Err == nil && res.Status < 500 {
			return res
		}
	}
	return res
}

// CheckAll checks every URL concurrently with bounded parallelism.
// Each check runs under its own timeout derived from ctx. Results are
// returned in the same order as the input slice.
func CheckAll(ctx context.Context, urls []string, concurrency int, timeout time.Duration, retries int) []Result {
	if concurrency < 1 {
		concurrency = 1
	}
	client := &http.Client{} // per-request timeout comes from the context
	results := make([]Result, len(urls))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			results[i] = checkWithRetry(cctx, client, u, retries)
		}(i, u)
	}
	wg.Wait()
	return results
}

// Summarize counts healthy and failed results.
func Summarize(results []Result) (ok, failed int) {
	for _, r := range results {
		if r.OK() {
			ok++
		} else {
			failed++
		}
	}
	return ok, failed
}

// Format renders one result as a single log-style line.
func Format(r Result) string {
	if r.Err != nil {
		return fmt.Sprintf("FAIL %s error=%v latency=%s", r.URL, r.Err, r.Latency.Round(time.Millisecond))
	}
	state := "OK  "
	if !r.OK() {
		state = "FAIL"
	}
	return fmt.Sprintf("%s %s status=%d latency=%s", state, r.URL, r.Status, r.Latency.Round(time.Millisecond))
}
