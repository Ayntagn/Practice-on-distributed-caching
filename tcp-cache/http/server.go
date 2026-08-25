package http

import (
	"net/http"

	ccore "caches-core/cache"
	"caches-core/ports"
)

type Server struct {
	ccore.Cache
}

func New(c ccore.Cache) *Server {
	return &Server{
		Cache: c,
	}
}

func (s *Server) Listen() {
	http.Handle("/cache/", s.cacheHandler())
	http.Handle("/status", s.statusHandler())
	http.ListenAndServe(":"+ports.HTTP, nil)
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
