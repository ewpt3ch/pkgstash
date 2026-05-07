package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ewpt3ch/pkgstash/internal/cache"
)

func (s *Server) handlerPackage(w http.ResponseWriter, req *http.Request) {

	// db files are not signed so we ignore as to not spam mirrors
	if strings.HasSuffix(req.PathValue("file"), ".db.sig") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// record the useragent from requestor
	slog.Debug("Requestors User Agent", "UA", req.Header.Get("User-Agent"))

	// build file paths from the request, they follow archlinux repo
	// <mirrorroot>/[core, extra, etc]/os/[x86_64, arm, etc]/package.pkg.tar.zst[.sig]
	repo := req.PathValue("repo")
	arch := req.PathValue("arch")
	file := req.PathValue("file")
	repoPath := filepath.Join(repo, "os", arch, file) //path from mirror root to requested file

	cachedFile, err := s.c.Fetch(repoPath)
	if err != nil {
		if upstreamErr, ok := errors.AsType[*cache.UpstreamError](err); ok {
			slog.Warn("upstream error", "err", upstreamErr.Error())
			http.Error(w, "Not found upstream", upstreamErr.StatusCode)
			return
		}
		slog.Warn("fetch error", "err", err)
		http.Error(w, "Failed to fetch from upstream", http.StatusBadGateway)
		return
	}
	defer func() {
		if closeErr := cachedFile.Reader.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+cachedFile.Filename)
	w.Header().Set("Content-Length", strconv.FormatInt(cachedFile.Size, 10))
	_, err = io.Copy(w, cachedFile.Reader)
	if err != nil {
		slog.Warn("streaming error", "err", err)
	}

}
