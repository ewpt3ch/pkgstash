package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ewpt3ch/pkgstash/internal/cache"
)

var (
	mirrorOK  = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "fake pkg data") })
	mirror404 = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
)

func mirrorOKWithCounter() (http.HandlerFunc, *atomic.Int32) {
	var calls atomic.Int32
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, "fake pkg data")
	}), &calls
}

const testUrlBase = "/core/os/x86_64"

func newTestServer(t *testing.T, mirrorHandler http.HandlerFunc) (*httptest.Server, *Server) {
	t.Helper()

	mirror := httptest.NewServer(mirrorHandler)
	t.Cleanup(func() { mirror.Close() })

	c, err := cache.NewCache(t.TempDir(), []string{mirror.URL + "/"}, []string{"core"})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	cfg := &Config{
		Port: "0",
		Auth: AuthConfig{Token: "testtoken"},
	}
	logLevel := new(slog.LevelVar)
	srv := &Server{
		cfg:      cfg,
		c:        c,
		logLevel: logLevel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{repo}/os/{arch}/{file}", srv.handlerPackage)
	mux.HandleFunc("POST /api/refresh", srv.handlerRefresh)
	mux.HandleFunc("POST /api/loglevel", srv.handlerLogLevel)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, srv
}

func TestHandlerPkgsExist(t *testing.T) {
	const expected = "fake pkg data"
	const expectedFile = "attachment; filename=somepkg.tar.zst"

	ts, _ := newTestServer(t, mirrorOK)

	resp, err := http.Get(ts.URL + testUrlBase + "/somepkg.tar.zst")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	respFile := resp.Header.Get("Content-Disposition")

	if resp.ContentLength != int64(len(expected)) {
		t.Errorf("expected %d got %d", len(expected), resp.ContentLength)
	}
	if respFile != expectedFile {
		t.Errorf("expected %s got %s", expectedFile, respFile)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err reading body: %v", err)
	}
	if string(data) != expected {
		t.Errorf("expected %s got %s", expected, string(data))
	}
}

func TestHandlerPkgsMiss(t *testing.T) {

	ts, _ := newTestServer(t, mirror404)

	resp, err := http.Get(ts.URL + testUrlBase + "/somepkg.tar.zst")
	if err != nil {
		t.Fatalf("GET failed %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 got %d", resp.StatusCode)
	}
}

func TestHandlerPkgsDBSig(t *testing.T) {

	handler, calls := mirrorOKWithCounter()
	ts, _ := newTestServer(t, handler)

	resp, err := http.Get(ts.URL + testUrlBase + "/core.db.sig")
	if err != nil {
		t.Fatalf("GET failed %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 got %d", resp.StatusCode)
	}
	if calls.Load() != 0 {
		t.Error("expected no upstream calls for .db.sig")
	}
}

func TestHandlerRefreshUnauthorized(t *testing.T) {

	ts, _ := newTestServer(t, mirrorOK)

	req, err := http.NewRequest("POST", ts.URL+"/api/refresh", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer badtoken")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestHandlerRefreshOK(t *testing.T) {

	ts, _ := newTestServer(t, mirrorOK)

	req, err := http.NewRequest("POST", ts.URL+"/api/refresh", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer testtoken")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected %d got %d", http.StatusNoContent, resp.StatusCode)
	}
}

func TestHandlerLogLevelValid(t *testing.T) {

	ts, srv := newTestServer(t, mirrorOK)

	body := strings.NewReader(`{"loglevel": "debug"}`)
	req, err := http.NewRequest("POST", ts.URL+"/api/loglevel", body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testtoken")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected %d got %d", http.StatusNoContent, resp.StatusCode)
	}
	got := srv.logLevel.Level()
	if got != slog.LevelDebug {
		t.Errorf("expected DEBUG got %s", got)
	}
}

func TestHandlerLogLevelInvalid(t *testing.T) {

	ts, srv := newTestServer(t, mirrorOK)

	body := strings.NewReader(`{"loglevel": "what"}`)
	req, err := http.NewRequest("POST", ts.URL+"/api/loglevel", body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testtoken")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected %d got %d", http.StatusBadRequest, resp.StatusCode)
	}
	got := srv.logLevel.Level()
	if got != slog.LevelInfo {
		t.Errorf("expected INFO got %s", got)
	}

}
