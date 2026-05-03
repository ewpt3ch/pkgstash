package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ewpt3ch/pkgstash/internal/cache"
)

func (s *Server) handlePackage(w http.ResponseWriter, req *http.Request) {

	// most mirrors don't have a *db.sig so we 404 it here instead of spamming the mirror
	if strings.HasSuffix(req.PathValue("file"), ".db.sig") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// record the useragent from requestor
	// #log
	log.Printf("Requestors UA: %s", req.Header.Get("User-Agent"))

	// build file paths from the request, they follow archlinux repo
	// <mirrorroot>/[core, extra, etc]/os/[x86_64, arm, etc]/package.pkg.tar.zst[.sig]
	repo := req.PathValue("repo")
	arch := req.PathValue("arch")
	file := req.PathValue("file")
	repoPath := filepath.Join(repo, "os", arch, file) //path from mirror root to pkg or db file

	cachedFile, err := s.c.Fetch(repoPath)
	if err != nil {
		var upstreamErr *cache.UpstreamError
		if errors.As(err, &upstreamErr) {
			// #log
			log.Printf("upstream error: %v", err)
			http.Error(w, "Not found upstream", upstreamErr.StatusCode)
			return
		}
		// #log
		log.Printf("fetch error: %v", err)
		http.Error(w, "Failed to fetch from upstream", http.StatusBadGateway)
		return
	}
	defer cachedFile.Reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+cachedFile.Filename)
	w.Header().Set("Content-Length", strconv.FormatInt(cachedFile.Size, 10))
	_, err = io.Copy(w, cachedFile.Reader)
	if err != nil {
		// #log
		log.Printf("error streaming file to client: %v", err)
	}

}
