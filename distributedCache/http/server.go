package http

import (
	"encoding/json"
	"log"
	"net/http"

	"distributedCache/cache"
	"distributedCache/cluster"
)

type Server struct {
	cache.Cache
	cluster.Node
}

func New(c cache.Cache, n cluster.Node) *Server {
	return &Server{
		c,
		n,
	}
}

func (s *Server) Listen() {
	http.Handle("/cache/", s.cacheHandler())
	http.Handle("/status", s.statusHandler())
	http.Handle("/cluster", s.clusterHandler())
	http.ListenAndServe(s.Addr()+":12345", nil)
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

type clusterHandler struct {
	*Server
}

func (h *clusterHandler) ServeHTTP(w http.ResponseWriter, r *http.
	Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m := h.Members()
	b, e := json.Marshal(m)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func (s *Server) clusterHandler() http.Handler {
	return &clusterHandler{s}
}
