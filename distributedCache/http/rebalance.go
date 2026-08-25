package http

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// rebalanceHandler triggers an async pass that moves every local key whose
// owner (per the consistent-hash ring) is now another member: the value is
// PUT to the new owner and deleted locally. Call it after members join or
// leave so data follows the rebuilt ring.
type rebalanceHandler struct {
	*Server
	client *http.Client // forwarded to the new owner; injectable for tests
	port   string       // forward target port; ports.HTTP in production
	mu     sync.Mutex   // one pass at a time
}

func (h *rebalanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// A second trigger while a pass is running would scan and move
	// concurrently, duplicating or dropping entries.
	if !h.mu.TryLock() {
		w.WriteHeader(http.StatusConflict)
		return
	}
	go func() {
		defer h.mu.Unlock()
		h.rebalance()
	}()
}

// rebalance walks this node's local keys and forwards each one that the
// current ring no longer assigns to this node, then deletes it locally.
func (h *rebalanceHandler) rebalance() {
	s := h.NewScanner()
	defer s.Close()
	for s.Scan() {
		k := s.Key()
		n, ok := h.ShouldProcess(k)
		if !ok {
			// Keep the local copy when the forward fails (network error,
			// owner down, or owner rejecting the write): deleting it here
			// would lose the entry, and a later pass can retry it.
			if e := h.forward(n, k, s.Value()); e != nil {
				log.Println("rebalance: forward", k, "failed:", e)
				continue
			}
			h.Del(k)
		}
	}
}

// forward PUTs one key to its new owner. Any failure (request build,
// transport, or a non-2xx response) is returned so the caller can keep
// the local copy.
func (h *rebalanceHandler) forward(addr, key string, value []byte) error {
	r, e := http.NewRequest(http.MethodPut,
		"http://"+addr+":"+h.port+cachePathPrefix+key,
		bytes.NewReader(value))
	if e != nil {
		return e
	}
	resp, e := h.client.Do(r)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("owner replied %d", resp.StatusCode)
	}
	return nil
}
