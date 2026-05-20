package cache

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func newTestFlight(tmpPath string) *inFlight {
	return &inFlight{
		tmpPath:     tmpPath,
		headerReady: make(chan struct{}),
		done:        make(chan struct{}),
	}
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

	//nolint:errcheck //ephemeral no need to check
	io.Copy(io.Discard, cf.Reader)
	//nolint:errcheck //ephemeral no need to check
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
		//nolint:errcheck //ephemeral no need to check
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
			//nolint:errcheck //ephemeral no need to check
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
	const expected = "This is fake file contents"
	t.Run("Download error propagates to flight.err", func(t *testing.T) {
		svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:errcheck //ephemeral no need to check
			w.WriteHeader(http.StatusNotFound)
		}))

		c := newTestCache(t, []string{svr.URL + "/"})
		relPath := "fakefile"
		tmpPath := "fakefile.tmp"
		flight := newTestFlight(tmpPath)
		tmpFile, err := c.cr.Create(tmpPath)
		require.NoError(t, err, "failed open test file")

		c.downloadWrangle(relPath, flight, tmpFile)
		select {
		case <-flight.done:
			//closed, pass
		case <-time.After(time.Second):
			t.Fatal("done channel never closed")
		}
		assert.Error(t, flight.err, "expected err got nil, err: %v", err)
	})

	t.Run("Network error propagates to flight.err", func(t *testing.T) {
		c := newTestCache(t, []string{"http://127.0.0.1/"})
		relPath := "fakefile"
		tmpPath := "fakefile.tmp"
		flight := newTestFlight(tmpPath)
		tmpFile, err := c.cr.Create(tmpPath)
		require.NoError(t, err, "failed open test file")

		c.downloadWrangle(relPath, flight, tmpFile)
		select {
		case <-flight.done:
			//closed, pass
		case <-time.After(time.Second):
			t.Fatal("done channel never closed")
		}
		assert.Error(t, flight.err, "expected err got none, err: %v", err)
	})

	t.Run("Retry works across mirror", func(t *testing.T) {
		svrMiss := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:errcheck //ephemeral no need to check
			w.WriteHeader(http.StatusNotFound)
		}))

		svrFound := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:errcheck //ephemeral no need to check
			fmt.Fprintf(w, "%s", expected)
		}))

		c := newTestCache(t, []string{
			svrMiss.URL + "/",
			svrFound.URL + "/",
		})
		relPath := "fakefile"
		tmpPath := "fakefile.tmp"
		flight := newTestFlight(tmpPath)
		tmpFile, err := c.cr.Create(tmpPath)
		require.NoError(t, err, "failed open test file")

		c.downloadWrangle(relPath, flight, tmpFile)
		select {
		case <-flight.done:
			//closed, pass
		case <-time.After(time.Second):
			t.Fatal("done channel never closed")
		}
		assert.NoError(t, flight.err, "expected no err got: %v", err)
	})

	t.Run("Cleanup runs on failure", func(t *testing.T) {
		svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:errcheck //ephemeral no need to check
			w.WriteHeader(http.StatusNotFound)
		}))

		c := newTestCache(t, []string{svr.URL + "/"})
		relPath := "fakefile"
		tmpPath := "fakefile.tmp"
		flight := newTestFlight(tmpPath)
		tmpFile, err := c.cr.Create(tmpPath)
		require.NoError(t, err, "failed open test file")

		c.downloadWrangle(relPath, flight, tmpFile)
		_, err = os.Stat(tmpPath)
		assert.ErrorIs(t, err, os.ErrNotExist)
		select {
		case <-flight.headerReady:
			//closed
		default:
			t.Error("headerReady not closes")
		}
		select {
		case <-flight.done:
			//closed
		default:
			t.Error("done not closed")
		}
		c.inFlightMu.Lock()
		_, ok := c.inFlight[relPath]
		c.inFlightMu.Unlock()
		assert.False(t, ok, "expected inFlight entry to be removed")
	})

	t.Run("Size propagates to flight", func(t *testing.T) {
		svr := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:errcheck //ephemeral no need to check
			fmt.Fprintf(w, "%s", expected)
		}))
		c := newTestCache(t, []string{svr.URL + "/"})
		relPath := "fakefile"
		tmpPath := "fakefile.tmp"
		flight := newTestFlight(tmpPath)
		tmpFile, err := c.cr.Create(tmpPath)
		require.NoError(t, err, "failed open test file")

		c.downloadWrangle(relPath, flight, tmpFile)
		var size int64
		select {
		case <-flight.headerReady:
			size = flight.contentLength
		case <-time.After(time.Second):
			t.Fatal("content-length never got set")
		}
		assert.Equal(t, int64(len(expected)), size)

	})
}

func TestTailer(t *testing.T) {
	const expected = "This is fake file contents"
	const filename = "fakefile"
	t.Run("Read from completed file", func(t *testing.T) {
		tmpPath := filepath.Join(t.TempDir(), filename)
		flight := &inFlight{
			done: make(chan struct{}),
		}

		err := os.WriteFile(tmpPath, []byte(expected), 0660)
		require.NoError(t, err)
		f, err := os.Open(tmpPath)
		require.NoError(t, err)
		close(flight.done)

		tr := &tailer{f: f, flight: flight}
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		assert.Equal(t, []byte(expected), data)
	})

	t.Run("Read chunks until done", func(t *testing.T) {
		tmpPath := filepath.Join(t.TempDir(), filename)
		flight := &inFlight{
			done: make(chan struct{}),
		}

		wf, err := os.Create(tmpPath)
		require.NoError(t, err)

		go func() {
			for range 3 {
				//nolint:errcheck //ephemeral no need to check
				fmt.Fprintf(wf, "%s", expected)
				time.Sleep(100 * time.Millisecond)
			}
			//nolint:errcheck //ephemeral no need to check
			wf.Sync()
			//nolint:errcheck //ephemeral no need to check
			wf.Close()
			close(flight.done)
		}()

		f, err := os.Open(tmpPath)
		require.NoError(t, err)

		tr := &tailer{f: f, flight: flight}
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		assert.Equal(t, []byte(strings.Repeat(expected, 3)), data)
	})

	t.Run("propagate flight.err", func(t *testing.T) {
		expectedErr := errors.New("upstream failed")
		tmpPath := filepath.Join(t.TempDir(), filename)

		err := os.WriteFile(tmpPath, []byte{}, 0660)
		require.NoError(t, err)

		f, err := os.Open(tmpPath)
		require.NoError(t, err)

		flight := &inFlight{
			done: make(chan struct{}),
			err:  expectedErr,
		}
		close(flight.done)

		tr := &tailer{f: f, flight: flight}
		_, err = io.ReadAll(tr)
		assert.ErrorIs(t, err, expectedErr)
	})
}
