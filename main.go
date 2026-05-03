package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/ewpt3ch/pkgstash/internal/cache"
)

type Server struct {
	cfg *Config
	c   *cache.Cache
}

func main() {

	// set config from flag if available
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()
	if len(configPath) == 0 {
		configPath = "/etc/pkgstash/pkgstash.toml"
	}

	cfg, err := ReadConfig(configPath)
	if err != nil {
		// #log
		log.Fatal(err)
	}

	c := cache.NewCache(cfg.CacheRoot, cfg.MirrorURLs, cfg.MirroredRepos)
	srv := &Server{cfg: cfg, c: c}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{repo}/os/{arch}/{file}", srv.handlePackage)
	mux.HandleFunc("POST /api/refresh", srv.handlerRefresh)

	if err := srv.c.Refresh(); err != nil {
		// #log
		log.Fatal(err)
	}

	httpServe := &http.Server{
		Addr:    ":" + srv.cfg.Port,
		Handler: mux,
	}

	// #log
	log.Printf("serving pkgstash root: %v on port: %v", cfg.CacheRoot, cfg.Port)
	log.Fatal(httpServe.ListenAndServe())

}
