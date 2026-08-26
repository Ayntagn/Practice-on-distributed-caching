package cache

import (
	"sync"
	"time"
)

type inMemoryCache struct {
	c     map[string]value
	mutex sync.RWMutex
	Stat
	ttl time.Duration
}

// pair is one key/value snapshot handed to the scanner goroutine.
type pair struct {
	k string
	v []byte
}

// value is one cache entry: the payload plus the moment it was written.
// A zero ttl means entries never expire, so expired is always false.
type value struct {
	v       []byte
	created time.Time
}

// expired reports whether an entry written at created has outlived the
// cache's ttl. Callers must hold c.mutex (read or write).
func (c *inMemoryCache) expired(entry value) bool {
	return c.ttl > 0 && entry.created.Add(c.ttl).Before(time.Now())
}

// inMemoryScanner streams a best-effort snapshot of the map: the scan
// goroutine reads entries one at a time under RLock and hands them to the
// consumer through pairCh, so writers are only blocked between entries.
// Close cancels an in-flight scan early.
type inMemoryScanner struct {
	pair
	pairCh  chan *pair
	closeCh chan struct{}
}

func (c *inMemoryCache) Set(k string, v []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	tmp, exist := c.c[k]
	if exist {
		c.del(k, tmp.v)
	}
	c.c[k] = value{v, time.Now()}
	c.add(k, v)
	return nil
}

func (c *inMemoryCache) Get(k string) ([]byte, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	v, ok := c.c[k]
	if !ok || c.expired(v) {
		return nil, nil
	}
	return v.v, nil
}

func (c *inMemoryCache) Del(k string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	v, exist := c.c[k]
	if exist {
		delete(c.c, k)
		c.del(k, v.v)
	}
	return nil
}

func (c *inMemoryCache) GetStat() Stat {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Stat
}

func newInMemoryCache(ttl int) *inMemoryCache {
	c := &inMemoryCache{
		make(map[string]value),
		sync.RWMutex{},
		Stat{},
		time.Duration(ttl) * time.Second,
	}
	if ttl > 0 {
		go c.expirer()
	}
	return c
}

// expirer sweeps expired entries so the map does not leak memory for keys
// that are written but never read again. Get already hides expired entries,
// so this is a reclamation pass, not a correctness mechanism. The created
// timestamp is re-checked under the write lock so a concurrent Set that
// refreshed the key is not deleted.
func (c *inMemoryCache) expirer() {
	for {
		time.Sleep(c.ttl)
		c.mutex.RLock()
		for k, v := range c.c {
			c.mutex.RUnlock()
			if c.expired(v) {
				c.mutex.Lock()
				if cur, ok := c.c[k]; ok && cur.created.Equal(v.created) {
					delete(c.c, k)
					c.del(k, cur.v)
				}
				c.mutex.Unlock()
			}
			c.mutex.RLock()
		}
		c.mutex.RUnlock()
	}
}

func (s *inMemoryScanner) Close() {
	close(s.closeCh)
}

func (s *inMemoryScanner) Scan() bool {
	p, ok := <-s.pairCh
	if ok {
		s.k, s.v = p.k, p.v
	}
	return ok
}

func (s *inMemoryScanner) Key() string {
	return s.k
}

func (s *inMemoryScanner) Value() []byte {
	return s.v
}

func (c *inMemoryCache) NewScanner() Scanner {
	pairCh := make(chan *pair)
	closeCh := make(chan struct{})
	go func() {
		defer close(pairCh)
		c.mutex.RLock()
		for k, v := range c.c {
			c.mutex.RUnlock()
			select {
			case <-closeCh:
				return
			case pairCh <- &pair{k, v.v}:
			}
			c.mutex.RLock()
		}
		c.mutex.RUnlock()
	}()
	return &inMemoryScanner{
		pair:    pair{},
		pairCh:  pairCh,
		closeCh: closeCh,
	}
}
