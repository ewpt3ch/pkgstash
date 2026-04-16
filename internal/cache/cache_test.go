package cache

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	svr := httptest.NewServer(handler)
	t.Cleanup(func() { svr.Close() })
	return svr
}

func newTestCache(t *testing.T, mirrorURL string) *Cache {
	t.Helper()
	return NewCache(t.TempDir(), mirrorURL)
}

func TestFetchFileExists(t *testing.T) {
	const expected = "This is fake file contents"

	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expected)
	}))

	c := newTestCache(t, svr.URL+"/")

	err := c.Fetch("fakefile")
	if err != nil {
		t.Fatalf("Fetch failed %v", err)
	}

	fakefilepath := filepath.Join(c.cacheRoot, "fakefile")

	data, err := os.ReadFile(fakefilepath)
	if err != nil {
		t.Fatalf("Error reading file back: %v", err)
	}
	if !bytes.Equal(data, []byte(expected)) {
		t.Errorf("expected file to contain %s got %s", expected, data)
	}
}
