package cluster

import (
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"stathat.com/c/consistent"
)

func newNodeForTest(addr string, members []string) *node {
	circle := consistent.New()
	circle.NumberOfReplicas = 256
	circle.Set(members)
	return &node{Consistent: circle, addr: addr}
}

func TestShouldProcessOwnsLocalKeys(t *testing.T) {
	members := []string{"10.0.0.1", "10.0.0.2"}
	n1 := newNodeForTest("10.0.0.1", members)
	n2 := newNodeForTest("10.0.0.2", members)
	for _, key := range []string{"a", "hello", "user:42", "123456789", ""} {
		owner1, ok1 := n1.ShouldProcess(key)
		owner2, ok2 := n2.ShouldProcess(key)
		if owner1 != owner2 {
			t.Fatalf("key %q resolved to %q on n1 and %q on n2", key, owner1, owner2)
		}
		if ok1 == ok2 {
			t.Fatalf("key %q claimed by both or neither (n1=%v n2=%v)", key, ok1, ok2)
		}
	}
}

func TestShouldProcessEmptyRing(t *testing.T) {
	n := newNodeForTest("10.0.0.1", nil)
	addr, ok := n.ShouldProcess("key")
	if ok || addr != "" {
		t.Fatalf("empty ring: got (%q, %v)", addr, ok)
	}
}

func TestShouldProcessSingleMember(t *testing.T) {
	n := newNodeForTest("10.0.0.1", []string{"10.0.0.1"})
	addr, ok := n.ShouldProcess("anything")
	if !ok || addr != "10.0.0.1" {
		t.Fatalf("got (%q, %v)", addr, ok)
	}
}

func TestRebuildSeedsRing(t *testing.T) {
	n := newNodeForTest("10.0.0.1", nil)
	n.rebuild()
	addr, ok := n.ShouldProcess("key")
	if ok || addr != "" {
		t.Fatalf("got (%q, %v)", addr, ok)
	}
}

// TestEventDelegateCallbacksReturn guards the D3 regression: memberlist
// invokes these callbacks while holding its internal lock, so the
// callbacks must return without touching the memberlist API (the rebuild
// happens on a fresh goroutine). A synchronous rebuild here deadlocks.
func TestEventDelegateCallbacksReturn(t *testing.T) {
	n := newNodeForTest("10.0.0.1", nil)
	d := &eventDelegate{n}
	done := make(chan struct{})
	go func() {
		d.NotifyJoin(&memberlist.Node{Name: "10.0.0.2"})
		d.NotifyLeave(&memberlist.Node{Name: "10.0.0.2"})
		d.NotifyUpdate(&memberlist.Node{Name: "10.0.0.2"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("event callbacks blocked (rebuild must be asynchronous)")
	}
}
