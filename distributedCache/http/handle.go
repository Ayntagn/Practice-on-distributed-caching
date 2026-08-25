package http

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"caches-core/ports"
)

const cachePathPrefix = "/cache/"

// parseKey extracts the cache key from the request path (B3: the old
// Split("/")[2] panicked on paths without enough segments). The path
// must be exactly "/cache/<key>" with a single-segment key.
func parseKey(r *http.Request) (string, bool) {
	p := r.URL.Path
	if !strings.HasPrefix(p, cachePathPrefix) {
		return "", false
	}
	key := strings.TrimPrefix(p, cachePathPrefix)
	if key == "" || strings.Contains(key, "/") {
		return "", false
	}
	return key, true
}

type cacheHandler struct {
	*Server
}

func (h *cacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, ok := parseKey(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// D1: a key owned by another member is answered with a redirect
	// to that member's cache endpoint on this service's port.
	if addr, ok := h.ShouldProcess(key); !ok {
		http.Redirect(w, r, "http://"+addr+":"+ports.HTTP+cachePathPrefix+key, http.StatusTemporaryRedirect)
		return
	}
	m := r.Method
	if m == http.MethodPut {
		b, _ := io.ReadAll(r.Body)
		if len(b) != 0 {
			e := h.Set(key, b)
			if e != nil {
				log.Println(e)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	}
	if m == http.MethodGet {
		b, e := h.Get(key)
		if e != nil {
			log.Println(e)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(b) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if _, e := w.Write(b); e != nil {
			log.Println(e)
		}
		return
	}
	if m == http.MethodDelete {
		e := h.Del(key)
		if e != nil {
			log.Println(e)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

type statusHandler struct {
	*Server
}

func (h *statusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	b, e := json.Marshal(h.GetStat())
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, e := w.Write(b); e != nil {
		log.Println(e)
	}
}

// clusterHandler reports the current member list as JSON.
type clusterHandler struct {
	*Server
}

func (h *clusterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	if _, e := w.Write(b); e != nil {
		log.Println(e)
	}
}
