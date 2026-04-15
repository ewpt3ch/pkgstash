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
	cfg, err := ReadConfig("/home/ewpt3ch/dev/pacman-cache-server/tmprepo/app.config.toml")
	if err != nil {
		log.Fatal(err)
	}

	c := cache.NewCache(cfg.CacheRoot, cfg.MirrorURL)
	srv := &Server{cfg: cfg, c: c}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{repo}/os/{arch}/{file}", srv.handlePackage)
	mux.HandleFunc("POST /api/refresh", srv.handlerRefresh)

	if err := srv.c.Refresh(); err != nil {
		log.Fatal(err)
	}

	httpServe := &http.Server{
		Addr:    ":" + srv.cfg.Port,
		Handler: mux,
	}

	log.Fatal(httpServe.ListenAndServe())

}
