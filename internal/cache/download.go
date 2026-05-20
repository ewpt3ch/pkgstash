package cache

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
)

func (c *Cache) downloadWrangle(relPath string, flight *inFlight, tmpFile *os.File) {

	slog.Debug("wrangle starting", "relPath", relPath)
	// defer map cleanup and signal client handle done
	defer c.cleanupFlight(relPath, flight)

	// declare vars outside loop
	var err error
	// fetch pkgs from mirror with retry logic
	for range len(c.cfg.mirrorURLs) {
		url := c.nextMirror() + relPath
		err = c.downloadToDisk(url, flight, tmpFile)
		if err == nil {
			break
		}
		var upstreamErr *UpstreamError
		if !errors.As(err, &upstreamErr) {
			// network error, transfer interupted, bail
			break
		}
		slog.Warn("mirror failed", "url", url, "status", upstreamErr.StatusCode)
	}
	if err != nil {
		slog.Warn("download error", "err", err)
		flight.err = err
		removeErr := c.cr.Remove(flight.tmpPath)
		if removeErr != nil {
			slog.Warn("failed to remove temp file", "path", flight.tmpPath, "err", removeErr)
		}
		return
	}
	slog.Debug("wrangle download complete", "err", err)

	// mv file to final location
	err = c.cr.Rename(flight.tmpPath, relPath)
	if err != nil {
		removeErr := c.cr.Remove(flight.tmpPath)
		if removeErr != nil {
			slog.Warn("failed to remove temp file", "path", flight.tmpPath, "err", removeErr)
		}
		return
	}
	slog.Debug("file moved to final location", "err", err)
}

func (c *Cache) downloadToDisk(url string, flight *inFlight, tmpFile *os.File) error {
	slog.Info("fetching", "url", url)

	// set the user agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("failed create request", "err", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		slog.Warn("fetch failed", "url", url, "err", err)
		return err
	}
	if resp.StatusCode != 200 {
		slog.Info("fetch returned", "url", url, "status", resp.StatusCode)
		return &UpstreamError{StatusCode: resp.StatusCode}
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	size := resp.ContentLength
	flight.contentLength = size
	close(flight.headerReady)

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return err
	}
	if err = tmpFile.Sync(); err != nil {
		return err
	}
	return nil
}

func (c *Cache) cleanupFlight(key string, f *inFlight) {
	c.inFlightMu.Lock()
	delete(c.inFlight, key)
	c.inFlightMu.Unlock()
	slog.Debug("closing channels")
	safeClose(f.headerReady)
	safeClose(f.done)
}

func safeClose(ch chan struct{}) {
	select {
	case <-ch:
		// already closed
	default:
		close(ch)
	}
}
