package main

import (
	"log"
	"net/http"
)

func (s *Server) handlerRefresh(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Authorization") != "Bearer "+s.cfg.Auth.Token {
		http.Error(w, "unauthorized", http.StatusInternalServerError)
		return
	}
	if err := s.c.Refresh(); err != nil {
		// #log
		log.Printf("refresh failed: %v", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
