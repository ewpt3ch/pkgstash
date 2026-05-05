package main

import (
	"log/slog"
	"net/http"
)

func (s *Server) handlerRefresh(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Authorization") != "Bearer "+s.cfg.Auth.Token {
		http.Error(w, "unauthorized", http.StatusInternalServerError)
		return
	}
	if err := s.c.Refresh(); err != nil {
		slog.Error("refresh failed", "err", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlerLogLevel(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Authorization") != "Bearer "+s.cfg.Auth.Token {
		http.Error(w, "unauthorized", http.StatusInternalServerError)
		return
	}

}
