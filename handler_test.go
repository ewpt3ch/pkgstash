package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ewpt3ch/pkgstash/internal/cache"
)

func newTestServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()

	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "fake pkg data")
	}))
	t.Cleanup(func() { mirror.Close() })

	c := cache.NewCache(t.TempDir(), []string{mirror.URL + "/"}, []string{"core"})
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

	ts, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/core/os/x86_64/somepkg.tar.zst")

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
