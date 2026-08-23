package http

import (
	"net/http"
	"tcp-cache/cache"
)

type Server struct {
	cache.Cache
}

func New(c cache.Cache) *Server {
	return &Server{
		Cache: c,
	}
}

func (s *Server) Listen() {
	http.Handle("/cache/", s.cacheHandler())
	http.Handle("/status", s.statusHandler())
	http.ListenAndServe(":12345", nil)
}

func (s *Server) cacheHandler() http.Handler {
	return &cacheHandler{
		Server: s,
	}
}

func (s *Server) statusHandler() http.Handler {
	return &statusHandler{
		Server: s,
	}
}
