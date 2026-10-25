package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// readURLs loads one URL per line, skipping blanks and # comments.
func readURLs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var urls []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls, sc.Err()
}

func run(args []string) int {
	fs := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	file := fs.String("file", "urls.txt", "file with one URL per line")
	concurrency := fs.Int("concurrency", 10, "max parallel checks")
	timeout := fs.Duration("timeout", 5*time.Second, "per-check timeout")
	retries := fs.Int("retries", 2, "retries on transport error or 5xx")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	urls, err := readURLs(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read urls:", err)
		return 2
	}
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "no URLs to check")
		return 2
	}
	results := CheckAll(context.Background(), urls, *concurrency, *timeout, *retries)
	for _, r := range results {
		fmt.Println(Format(r))
	}
	ok, failed := Summarize(results)
	fmt.Printf("summary: %d ok, %d failed, %d total\n", ok, failed, len(results))
	if failed > 0 {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}
