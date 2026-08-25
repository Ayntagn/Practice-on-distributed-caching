package http

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	ccore "caches-core/cache"
)

// fakeNode is a minimal cluster.Node whose ownership map decides which
// keys belong to this node and which to a fake owner.
type fakeNode struct {
	addr   string
	owners map[string]string
}

func (n *fakeNode) Addr() string      { return n.addr }
func (n *fakeNode) Members() []string { return nil }
func (n *fakeNode) ShouldProcess(key string) (string, bool) {
	owner, exists := n.owners[key]
	if !exists {
		return "", false
	}
	return owner, owner == n.addr
}

func TestRebalanceForwardsNonLocalKeys(t *testing.T) {
	// The fake owner records every PUT it receives.
	received := map[string]string{}
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		received[r.URL.Path] = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer owner.Close()
	_, ownerPort, _ := net.SplitHostPort(strings.TrimPrefix(owner.URL, "http://"))

	c := ccore.NewMemory()
	c.Set("local", []byte("stay"))  // owned here: must not move
	c.Set("remote", []byte("move")) // owned by the fake owner: forwarded+deleted
	fake := &fakeNode{
		addr: "self",
		owners: map[string]string{
			"local":  "self",
			"remote": "127.0.0.1",
		},
	}
	h := &rebalanceHandler{
		Server: &Server{Cache: c, Node: fake},
		client: owner.Client(),
		port:   ownerPort,
	}
	h.rebalance()

	if got := received["/cache/remote"]; got != "move" {
		t.Fatalf("owner received %q for /cache/remote, want %q", got, "move")
	}
	if _, hit := received["/cache/local"]; hit {
		t.Fatalf("owner should not receive local key, got %q", received["/cache/local"])
	}
	if v, _ := c.Get("remote"); v != nil {
		t.Fatalf("remote key still cached locally: %q", v)
	}
	if v, _ := c.Get("local"); string(v) != "stay" {
		t.Fatalf("local key was touched: %q", v)
	}
}

func TestRebalanceKeepsLocalCopyOnForwardFailure(t *testing.T) {
	// The owner rejects every write: the local copy must survive so a
	// later pass can retry it.
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer owner.Close()
	_, ownerPort, _ := net.SplitHostPort(strings.TrimPrefix(owner.URL, "http://"))

	c := ccore.NewMemory()
	c.Set("remote", []byte("move"))
	fake := &fakeNode{
		addr: "self",
		owners: map[string]string{
			"remote": "127.0.0.1",
		},
	}
	h := &rebalanceHandler{
		Server: &Server{Cache: c, Node: fake},
		client: owner.Client(),
		port:   ownerPort,
	}
	h.rebalance()

	if v, _ := c.Get("remote"); string(v) != "move" {
		t.Fatalf("local copy was deleted after a failed forward: %q", v)
	}
}

func TestRebalanceHandlerRejectsGet(t *testing.T) {
	h := &rebalanceHandler{}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/rebalance", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestRebalanceHandlerRejectsConcurrentTrigger(t *testing.T) {
	h := &rebalanceHandler{mu: sync.Mutex{}}
	h.mu.Lock() // simulate an in-flight pass
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/rebalance", nil))
	if w.Code != http.StatusConflict {
		t.Fatalf("got %d, want %d", w.Code, http.StatusConflict)
	}
}
