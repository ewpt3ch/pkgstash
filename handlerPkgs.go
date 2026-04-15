package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"gitea.ewpt3ch.dev/ewpt3ch/pkgstash/internal/cache"
)

func (s *Server) handlePackage(w http.ResponseWriter, req *http.Request) {
	// build file paths from the request, they follow archlinux repo
	// <mirrorroot>/[core, extra, etc]/os/[x86_64, arm, etc]/package.pkg.tar.zst[.sig]
	repo := req.PathValue("repo")
	arch := req.PathValue("arch")
	file := req.PathValue("file")
	repoPath := filepath.Join(repo, "os", arch, file)    //path from mirror root to pkg or db file
	pkgPath := filepath.Join(s.cfg.MirrorRoot, repoPath) //absolute path for local read of the file

	if _, err := os.Stat(pkgPath); err != nil {
		err = s.c.Fetch(repoPath)
		if err != nil {
			var upstreamErr *cache.UpstreamError
			if errors.As(err, &upstreamErr) {
				http.Error(w, "Not found upstream", upstreamErr.StatusCode)
				return
			}
			http.Error(w, "Failed to fetch from upstream", http.StatusBadGateway)

		}
	}

	http.ServeFile(w, req, pkgPath)
}
