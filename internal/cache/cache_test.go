package cache

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	svr := httptest.NewServer(handler)
	t.Cleanup(func() { svr.Close() })
	return svr
}

func newTestCache(t *testing.T, mirrorURLs []string) *Cache {
	t.Helper()
	// set slog to debug
	slog.SetLogLoggerLevel(slog.LevelDebug)
	mirroredRepos := []string{"core", "extra"}
	c, err := NewCache(t.TempDir(), mirrorURLs, mirroredRepos)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	c.client.Timeout = 500 * time.Millisecond
	return c
}

func TestFetch(t *testing.T) {
	// test happy paths on fetch, the error paths all return through
	// the handler so need to be tested from the handler
	//Test: cache hit
	const expected = "This is fake file contents"
	c := newTestCache(t, []string{"http://example.com/"})
	tmpFileName := "fakeFile"
	err := c.cr.WriteFile(tmpFileName, []byte(expected), 0644)
	if err != nil {
		t.Fatalf("failed to create tempfile: %v", err)
	}

	cachedFile, err := c.Fetch(tmpFileName)
	require.NoError(t, err, "expected no error got %v", err)
	require.NotNil(t, cachedFile, "expected CacheFile got nil")
	assert.Equal(t, tmpFileName, cachedFile.Filename, "expected tmp %s to equal cached %s", tmpFileName, cachedFile.Filename)
	assert.Equal(t, int64(len(expected)), cachedFile.Size)
	data, err := io.ReadAll(cachedFile.Reader)
	require.NoError(t, err, "failed to read back file %v", err)
	assert.Equal(t, []byte(expected), data, "expected: %s; got: %s", expected, string(data))

	// Test: cache miss file exists
	svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck //ephemeral no need to check
		fmt.Fprint(w, expected)
	}))

	c = newTestCache(t, []string{svr.URL + "/"})

	cf, err := c.Fetch("fakefile")
	require.NoError(t, err, "expected no error got: %v", err)
	require.NotNil(t, cf, "expected CacheFile got nil")

	io.Copy(io.Discard, cf.Reader)
	cf.Reader.Close()

	data, err = c.cr.ReadFile("fakefile")
	require.NoError(t, err, "expected no error got: %v", err)
	assert.Equal(t, []byte(expected), data, "expected: %s; got: %s", expected, string(data))
}

func TestCreateSymlinks(t *testing.T) {
	// reafactor to use testify
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
	// refactor tests to use testify
	// Test: test mutiple clients
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

func TestDownloadWrangle(t *testing.T) {
	// Test: Upload error propagates to flight.err
	// Test: Network error propagates to flight.err
	// Test: retry works across mirror
	// Test: cleanup runs on failure
}

func TestTailer(t *testing.T) {
	// Test: reads available bytes
	// Test: blocks until done
	// Test: propagates flight.err
	// Test: return true EOF
}
