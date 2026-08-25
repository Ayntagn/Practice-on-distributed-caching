package http

import (
	"log"
	"net/http"

	ccore "caches-core/cache"
	"caches-core/ports"
	"distributedCache/cluster"
)

type Server struct {
	ccore.Cache
	cluster.Node
}

func New(c ccore.Cache, n cluster.Node) *Server {
	return &Server{
		Cache: c,
		Node:  n,
	}
}

func (s *Server) Listen() {
	http.Handle("/cache/", s.cacheHandler())
	http.Handle("/status", s.statusHandler())
	http.Handle("/cluster", s.clusterHandler())
	// Fatal instead of swallowing the error: a failed bind would
	// otherwise exit the node silently with no log line at all.
	if e := http.ListenAndServe(s.Addr()+":"+ports.HTTP, nil); e != nil {
		log.Fatal(e)
	}
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

func (s *Server) clusterHandler() http.Handler {
	return &clusterHandler{s}
}
