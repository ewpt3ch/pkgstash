package cache

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	svr := httptest.NewServer(handler)
	t.Cleanup(func() { svr.Close() })
	return svr
}

func newTestCache(t *testing.T, mirrorURL []string) *Cache {
	t.Helper()
	mirroredRepos := []string{"core", "extra"}
	c := NewCache(t.TempDir(), mirrorURL, mirroredRepos)
	c.client.Timeout = 500 * time.Millisecond
	return c
}

func TestFetchFileExists(t *testing.T) {
	const expected = "This is fake file contents"

	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expected)
	}))

	c := newTestCache(t, []string{svr.URL + "/"})

	err := c.Fetch("fakefile")
	if err != nil {
		t.Fatalf("Fetch failed %v", err)
	}

	fakefilepath := filepath.Join(c.cfg.cacheRoot, "fakefile")

	data, err := os.ReadFile(fakefilepath)
	if err != nil {
		t.Fatalf("Error reading file back: %v", err)
	}
	if !bytes.Equal(data, []byte(expected)) {
		t.Errorf("expected file to contain %s got %s", expected, data)
	}
}

func TestFetchNotFound(t *testing.T) {
	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	c := newTestCache(t, []string{svr.URL + "/"})

	err := c.Fetch("fakefile")
	var upstreamErr *UpstreamError
	if !errors.As(err, &upstreamErr) {
		t.Fatalf("expected UpstreamError fot %v", err)
	}
	if upstreamErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 got %d", upstreamErr.StatusCode)
	}
}

func TestFetchSrvError(t *testing.T) {
	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	c := newTestCache(t, []string{svr.URL + "/"})

	err := c.Fetch("fakefile")
	var upstreamErr *UpstreamError
	if !errors.As(err, &upstreamErr) {
		t.Fatalf("expected UpstreamError fot %v", err)
	}
	if upstreamErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 got %d", upstreamErr.StatusCode)
	}
}

func TestFetchSrvDead(t *testing.T) {
	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(60 * time.Second):
			fmt.Fprint(w, "too late")
		}
	}))
	defer svr.Close()

	c := newTestCache(t, []string{svr.URL + "/"})

	err := c.Fetch("fakefile")
	if err == nil {
		t.Fatal("expected err got nil")
	}

	var upstreamErr *UpstreamError
	if errors.As(err, &upstreamErr) {
		t.Error("expected network error not UpstreamError")
	}
}
