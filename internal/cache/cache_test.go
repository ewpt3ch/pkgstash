package cache

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

func newTestCache(t *testing.T, mirrorURLs []string) *Cache {
	t.Helper()
	mirroredRepos := []string{"core", "extra"}
	c := NewCache(t.TempDir(), mirrorURLs, mirroredRepos)
	c.client.Timeout = 500 * time.Millisecond
	return c
}

func TestCacheHit(t *testing.T) {
	const expected = "This is fake file contents"

	c := newTestCache(t, []string{"http://example.com/"})
	tmpFileName := "fakeFile"
	tmpPath := filepath.Join(c.cfg.cacheRoot, tmpFileName)
	err := os.WriteFile(tmpPath, []byte(expected), 0644)
	if err != nil {
		t.Fatalf("failed to create tempfile: %v", err)
	}

	cachedFile, err := c.Fetch(tmpFileName)
	if err != nil {
		t.Fatalf("failed to fetch file: %v", err)
	}
	defer cachedFile.Reader.Close()

	if cachedFile.Filename != tmpFileName {
		t.Errorf("expected filename %s got %s", tmpFileName, cachedFile.Filename)
	}

	if int64(len(expected)) != cachedFile.Size {
		t.Errorf("expected %d got %d", len(expected), cachedFile.Size)
	}

	data, err := io.ReadAll(cachedFile.Reader)
	if err != nil {
		t.Fatalf("error reading file back: %v", err)
	}
	defer cachedFile.Reader.Close()

	if !bytes.Equal(data, []byte(expected)) {
		t.Errorf("expected file to contain %s got %s", expected, data)
	}
}

func TestCacheMissExists(t *testing.T) {
	const expected = "This is fake file contents"

	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expected)
	}))

	c := newTestCache(t, []string{svr.URL + "/"})

	_, err := c.Fetch("fakefile")
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

	_, err := c.Fetch("fakefile")
	var upstreamErr *UpstreamError
	if !errors.As(err, &upstreamErr) {
		t.Fatalf("expected UpstreamError got %v", err)
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

	_, err := c.Fetch("fakefile")
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

	_, err := c.Fetch("fakefile")
	if err == nil {
		t.Fatal("expected err got nil")
	}

	var upstreamErr *UpstreamError
	if errors.As(err, &upstreamErr) {
		t.Error("expected network error not UpstreamError")
	}
}

func TestFetchRetryExists(t *testing.T) {
	const expected = "This is fake file contents"

	svr1 := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	svr2 := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expected)
	}))
	fakeURLs := []string{
		svr1.URL + "/",
		svr2.URL + "/",
	}

	c := newTestCache(t, fakeURLs)

	_, err := c.Fetch("fakefile")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	fakefilepath := filepath.Join(c.cfg.cacheRoot, "fakefile")
	data, err := os.ReadFile(fakefilepath)
	if err != nil {
		t.Fatalf("error reading file back: %v", err)
	}
	if !bytes.Equal(data, []byte(expected)) {
		t.Errorf("expected file to contain %s got %s", expected, data)
	}
}

func TestFetchRetryNonExist(t *testing.T) {

	svr1 := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	svr2 := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	fakeURLs := []string{
		svr1.URL + "/",
		svr2.URL + "/",
	}

	c := newTestCache(t, fakeURLs)

	_, err := c.Fetch("fakefile")
	var upstreamErr *UpstreamError
	if !errors.As(err, &upstreamErr) {
		t.Errorf("expected UpstreamError got %v", err)
	}
	if upstreamErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 got %d", upstreamErr.StatusCode)
	}
}
