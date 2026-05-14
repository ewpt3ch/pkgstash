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
	c, err := NewCache(t.TempDir(), mirrorURLs, mirroredRepos)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	c.client.Timeout = 500 * time.Millisecond
	return c
}

func TestCacheHit(t *testing.T) {
	const expected = "This is fake file contents"

	c := newTestCache(t, []string{"http://example.com/"})
	tmpFileName := "fakeFile"
	err := c.cr.WriteFile(tmpFileName, []byte(expected), 0644)
	if err != nil {
		t.Fatalf("failed to create tempfile: %v", err)
	}

	cachedFile, err := c.Fetch(tmpFileName)
	if err != nil {
		t.Fatalf("failed to fetch file: %v", err)
	}
	//nolint:errcheck //ephemeral no need to check
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
	//nolint:errcheck //ephemeral no need to check
	defer cachedFile.Reader.Close()

	if !bytes.Equal(data, []byte(expected)) {
		t.Errorf("expected file to contain %s got %s", expected, data)
	}
}

func TestCacheMissExists(t *testing.T) {
	const expected = "This is fake file contents"

	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck //ephemeral no need to check
		fmt.Fprint(w, expected)
	}))

	c := newTestCache(t, []string{svr.URL + "/"})

	_, err := c.Fetch("fakefile")
	if err != nil {
		t.Fatalf("Fetch failed %v", err)
	}

	data, err := c.cr.ReadFile("fakefile")
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
			//nolint:errcheck //ephemeral no need to check
			fmt.Fprint(w, "too late")
		}
	}))
	defer svr.Close()

	c := newTestCache(t, []string{svr.URL + "/"})

	_, err := c.Fetch("fakefile")
	if err == nil {
		t.Fatal("expected err got nil")
	}

	if _, ok := errors.AsType[*UpstreamError](err); ok {
		t.Error("expected network error not UpstreamError")
	}
}

func TestFetchRetryExists(t *testing.T) {
	const expected = "This is fake file contents"

	svr1 := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	svr2 := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck //ephemeral no need to check
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

	data, err := c.cr.ReadFile("fakefile")
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

func TestCreateSymlinks(t *testing.T) {
	repos := []string{"core", "extra"}
	tmp := t.TempDir()
	cr, err := os.OpenRoot(tmp)
	if err != nil {
		t.Fatalf("unable to create tmp dir: %v", err)
	}

	if err := checkSymLinks(cr, repos); err != nil {
		t.Fatalf("error creating links: %v", err)
	}

	for _, repo := range repos {
		lnfile := filepath.Join(repo, "os/x86_64", repo+".db")
		expected := repo + ".db.tar.gz"
		lnval, err := cr.Readlink(lnfile)
		if err != nil {
			t.Errorf("%s has no link: %v", repo, err)
		}
		if lnval != expected {
			t.Errorf("expected %s got %s", expected, lnval)
		}
	}
}

func TestGetStreamMultiplClient(t *testing.T) {
	firstBytesSend := make(chan struct{})
	const expectedOne = "This is fake file contents"
	const expectedTwo = "More fake file contents"
	expected := expectedOne + expectedTwo

	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck //ephemeral no need to check
		fmt.Fprint(w, expectedOne)
		w.(http.Flusher).Flush()
		close(firstBytesSend)
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, expectedTwo)
	}))

	c := newTestCache(t, []string{svr.URL + "/"})
	c.client.Timeout = 10 * time.Second

	type fetchResult struct {
		data []byte
		err  error
	}

	results := make(chan fetchResult, 2)

	for range 2 {
		go func() {
			cf, err := c.Fetch("fakefile")
			if err != nil {
				results <- fetchResult{err: err}
				return
			}
			defer cf.Reader.Close()
			data, err := io.ReadAll(cf.Reader)
			results <- fetchResult{data: data, err: err}
		}()
	}

	<-firstBytesSend
	c.inFlightMu.Lock()
	_, ok := c.inFlight["fakefile"]
	c.inFlightMu.Unlock()
	if !ok {
		t.Errorf("no matching key in map: %v", c.inFlight)
	}

	for range 2 {
		result := <-results
		if result.err != nil {
			t.Errorf("a fetch failed: %v", result.err)
		}
		if !bytes.Equal(result.data, []byte(expected)) {
			t.Errorf("expected result to contain %s got %s", expected, result.data)
		}
	}

	data, err := c.cr.ReadFile("fakefile")
	if err != nil {
		t.Fatalf("Error reading file back: %v", err)
	}
	if !bytes.Equal(data, []byte(expected)) {
		t.Errorf("expected file to contain %s got %s", expected, data)
	}
}
