package cache

import (
	"log"

	ccore "caches-core/cache"
)

// New returns the cache implementation for typ. The in-memory
// implementation lives in caches-core; this local package keeps the
// factory signature stable for callers across modules.
func New(typ string) ccore.Cache {
	var c ccore.Cache
	if typ == "inmemory" {
		c = ccore.NewMemory()
	}
	if c == nil {
		panic("unknown cache type: " + typ)
	}
	log.Println(typ, "ready to serve")
	return c
}
