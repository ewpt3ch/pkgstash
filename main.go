package main

import (
	"log"
	"net/http"

	"gitea.ewpt3ch.dev/ewpt3ch/pkgstash/internal/cache"
)

type Server struct {
	cfg *Config
	c   *cache.Cache
}

func main() {
	cfg := NewConfig()
	c := cache.NewCache(cfg.MirrorRoot, cfg.MirrorURL)
	srv := &Server{cfg: cfg, c: c}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{repo}/os/{arch}/{file}", srv.handlePackage)

	if err := srv.c.Refresh(); err != nil {
		log.Fatal(err)
	}

	httpServe := &http.Server{
		Addr:    ":" + srv.cfg.Port,
		Handler: mux,
	}

	log.Fatal(httpServe.ListenAndServe())

}
