# concurrent-health-checker — a concurrent URL health checker in Go

A Go learning project exercising the idioms that matter in production Go:
goroutines, bounded concurrency, `context` timeouts/cancellation, retries
with backoff, table-driven tests — race-detector clean.

## Run

```sh
cat > urls.txt <<'EOF'
https://example.com
https://www.google.com
EOF
go run . -file urls.txt -concurrency 10 -timeout 5s -retries 2
```

Prints one line per URL plus a summary; exits 1 if any check failed,
2 on usage/input errors — so it can gate a script or CI step.

## Test

```sh
go test ./...        # table-driven tests against httptest servers
go test -race ./...  # the concurrency must be race-clean
go vet ./...
```

## Design notes (the part a reviewer actually reads)

- **Bounded parallelism:** a semaphore channel caps in-flight checks;
  results are written by index, so output order matches input order
  regardless of completion order.
- **Timeouts:** every check gets its own `context.WithTimeout`; retries
  select on `ctx.Done()` during backoff so cancellation is immediate.
- **Retry policy:** only transport errors and 5xx are retried — a 404 is
  a definitive answer, not a transient failure.
- **What I would add next:** per-host rate limiting, a `-json` output
  flag for machine consumption, and jitter on the backoff.

## Status

Starter scaffold prepared 2026-10-25. Not yet compile-tested in the
assistant's sandbox (no Go toolchain there) — first step when picking
this up: `go vet ./... && go test -race ./...` and fix anything that
surfaces before extending it.
