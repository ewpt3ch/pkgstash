package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

func (s *Server) handlerRefresh(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Authorization") != "Bearer "+s.cfg.Token {
		ip := req.Header.Get("X-Real-IP")
		if ip == "" {
			ip = req.RemoteAddr
		}
		slog.Warn("unauthorized request", "ip", ip, "path", req.URL.Path, "method", req.Method)
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := s.rs.Sync(); err != nil {
		slog.Error("refresh failed", "err", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlerLogLevel(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Authorization") != "Bearer "+s.cfg.Token {
		ip := req.Header.Get("X-Real-IP")
		if ip == "" {
			ip = req.RemoteAddr
		}
		slog.Warn("unauthorized request", "ip", ip, "path", req.URL.Path, "method", req.Method)
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	type reqParameters struct {
		NewLevel string `json:"loglevel"`
	}

	decoder := json.NewDecoder(req.Body)
	reqParams := reqParameters{}
	err := decoder.Decode(&reqParams)
	if err != nil {
		slog.Debug("json decode erro", "err", err)
		respondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	switch strings.ToLower(reqParams.NewLevel) {
	case "debug":
		s.logLevel.Set(slog.LevelDebug)
	case "info":
		s.logLevel.Set(slog.LevelInfo)
	case "warn":
		s.logLevel.Set(slog.LevelWarn)
	case "error":
		s.logLevel.Set(slog.LevelError)
	default:
		respondWithError(w, http.StatusBadRequest, "invalid log level")
		return
	}

	slog.Info("log level changed", "level", reqParams.NewLevel)
	w.WriteHeader(http.StatusNoContent)
}
