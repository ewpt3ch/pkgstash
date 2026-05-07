package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ewpt3ch/pkgstash/internal/cache"
)

type Server struct {
	cfg      *Config
	c        *cache.Cache
	logLevel *slog.LevelVar
}

func main() {

	// get options from cli flags
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	logFlag := flag.String("loglevel", "INFO", "loglevel: DEBUG, INFO, WARN, ERROR")
	flag.Parse()

	// set config from flag if available
	if len(configPath) == 0 {
		configPath = "/etc/pkgstash/pkgstash.toml"
	}

	//set log level from flag if available
	logLevel := new(slog.LevelVar)
	if err := logLevel.UnmarshalText([]byte(*logFlag)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level %q, defaulting to INFO\n", *logFlag)
		logLevel.Set(slog.LevelInfo)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	cfg, err := ReadConfig(configPath)
	if err != nil {
		slog.Error("failed to read config", "err", err)
		os.Exit(1)
	}

	c, err := cache.NewCache(cfg.CacheRoot, cfg.MirrorURLs, cfg.MirroredRepos)
	if err != nil {
		slog.Error("failed to create cache", "err", err)
		os.Exit(1)
	}
	defer c.Close()

	srv := &Server{
		cfg:      cfg,
		c:        c,
		logLevel: logLevel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{repo}/os/{arch}/{file}", srv.handlerPackage)
	mux.HandleFunc("POST /api/refresh", srv.handlerRefresh)
	mux.HandleFunc("POST /api/loglevel", srv.handlerLogLevel)

	if err := srv.c.Refresh(); err != nil {
		slog.Error("failed to refesh db files", "err", err)
		c.Close()
		os.Exit(1)
	}

	httpServe := &http.Server{
		Addr:              ":" + srv.cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// gracefully quit the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("serving pkgstash", "root", cfg.CacheRoot, "port", cfg.Port)
		if err = httpServe.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("server failed", "err", err)
			c.Close()
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServe.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "err", err)
	}
}
