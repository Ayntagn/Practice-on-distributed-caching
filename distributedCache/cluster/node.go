package cluster

import (
	"io"

	"github.com/hashicorp/memberlist"
	"stathat.com/c/consistent"
)

type Node interface {
	ShouldProcess(key string) (string, bool)
	Members() []string
	Addr() string
}

type node struct {
	*consistent.Consistent
	addr string
	l    *memberlist.Memberlist
}

func (n *node) Addr() string {
	return n.addr
}

func (n *node) ShouldProcess(key string) (string, bool) {
	addr, _ := n.Get(key)
	return addr, addr == n.addr
}

// eventDelegate rebuilds the consistent-hash ring on member
// joins/leaves (D3: replaces the 1-second polling loop).
type eventDelegate struct {
	*node
}

// The memberlist callbacks run synchronously while the internal node lock
// is held; calling back into Members() from here deadlocks. Rebuild on a
// fresh goroutine so the lock is released first. Set is idempotent, so
// racing rebuilds (multiple rapid joins/leaves) are harmless.
func (d *eventDelegate) NotifyJoin(n *memberlist.Node) {
	go d.rebuild()
}

func (d *eventDelegate) NotifyLeave(n *memberlist.Node) {
	go d.rebuild()
}

func (d *eventDelegate) NotifyUpdate(n *memberlist.Node) {
	go d.rebuild()
}

// rebuild replaces the ring with the current member set. Consistent
// guards Set/Get with an internal mutex, so it is safe to call from
// the memberlist event callbacks while requests are in flight.
func (n *node) rebuild() {
	if n.l == nil {
		return
	}
	m := n.l.Members()
	nodes := make([]string, 0, len(m))
	for _, x := range m {
		nodes = append(nodes, x.Name)
	}
	n.Set(nodes)
}

func New(addr string, cluster string) (Node, error) {
	circle := consistent.New()
	circle.NumberOfReplicas = 256
	n := &node{
		Consistent: circle,
		addr:       addr,
	}
	conf := memberlist.DefaultLANConfig()
	conf.Name = addr
	conf.BindAddr = addr
	conf.LogOutput = io.Discard
	conf.Events = &eventDelegate{n}
	l, e := memberlist.Create(conf)
	if e != nil {
		return nil, e
	}
	n.l = l
	// Skip Join when no cluster is given: joining ourselves would start
	// a push/pull full-state sync against our own TCP listener, and with
	// other nodes' joins arriving concurrently that deadlocks on the
	// memberlist node lock (all listeners hang before serving). The ring
	// is seeded right below with ourselves as the only member.
	if cluster != "" {
		_, e = l.Join([]string{cluster})
		if e != nil {
			return nil, e
		}
	}
	// D2: seed the ring immediately after join; the event delegate
	// keeps it fresh afterwards, so there is no empty-ring window
	// during which every key would be misrouted.
	n.rebuild()
	return n, nil
}
