package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantStatus int
		wantOK     bool
	}{
		{"ok", http.StatusOK, http.StatusOK, true},
		{"redirect", http.StatusFound, http.StatusFound, true},
		{"server error", http.StatusInternalServerError, http.StatusInternalServerError, false},
		{"not found", http.StatusNotFound, http.StatusNotFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := testServer(t, tt.status)
			res := Check(context.Background(), srv.Client(), srv.URL)
			if res.Status != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.Status, tt.wantStatus)
			}
			if res.OK() != tt.wantOK {
				t.Errorf("OK() = %v, want %v (result %+v)", res.OK(), tt.wantOK, res)
			}
		})
	}
}

func TestCheckAllPreservesOrderAndCounts(t *testing.T) {
	okSrv := testServer(t, http.StatusOK)
	badSrv := testServer(t, http.StatusInternalServerError)
	urls := []string{okSrv.URL, badSrv.URL, okSrv.URL}
	results := CheckAll(context.Background(), urls, 2, 2*time.Second, 0)
	if len(results) != len(urls) {
		t.Fatalf("got %d results, want %d", len(results), len(urls))
	}
	for i, u := range urls {
		if results[i].URL != u {
			t.Errorf("results[%d].URL = %s, want %s (order must be preserved)", i, results[i].URL, u)
		}
	}
	ok, failed := Summarize(results)
	if ok != 2 || failed != 1 {
		t.Errorf("Summarize = %d ok / %d failed, want 2/1", ok, failed)
	}
}

func TestCheckContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	res := Check(ctx, srv.Client(), srv.URL)
	if res.Err == nil {
		t.Errorf("expected a timeout error, got result %+v", res)
	}
}
