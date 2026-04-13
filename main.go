package main

import (
	"log"
	"net/http"

	"gitea.ewpt3ch.dev/ewpt3ch/pkgstash/internal/cache"
)

const repoRoot = "/home/ewpt3ch/dev/pacman-cache-server/tmprepo"

func main() {
	const port = "8090"
	cfg := NewConfig()
	c := cache.NewCache(cfg.RepoPath, cfg.MirrorURL)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{repo}/os/{arch}/{file}", func(w http.ResponseWriter, req *http.Request) {
		handlePackage(w, req, c)
	})

	httpServe := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Fatal(httpServe.ListenAndServe())

}
