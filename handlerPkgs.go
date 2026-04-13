package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"gitea.ewpt3ch.dev/ewpt3ch/pkgstash/internal/cache"
)

func handlePackage(w http.ResponseWriter, req *http.Request, c *cache.Cache) {
	// build file paths from the request, they follow archlinux repo
	// <reporoot>/[core, extra, etc]/os/[x86_64, arm, etc]/package.pkg.tar.zst[.sig]
	repo := req.PathValue("repo")
	arch := req.PathValue("arch")
	file := req.PathValue("file")
	relPath := filepath.Join(repo, "os", arch, file) //path from repo root to pkg or db file
	pkgPath := filepath.Join(repoRoot, relPath)      //path for local read of the file

	if _, err := os.Stat(pkgPath); err != nil {
		err = c.Fetch(relPath)
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
