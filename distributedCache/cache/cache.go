package cache

import (
	"log"

	ccore "caches-core/cache"
)

// New returns the cache implementation for typ ("inmemory" or "rocksdb").
// The in-memory implementation lives in caches-core; the rocksdb one is
// module-local because it links against this module's third_party build.
func New(typ string) ccore.Cache {
	var c ccore.Cache
	switch typ {
	case "inmemory":
		c = ccore.NewMemory()
	case "rocksdb":
		c = newRocksDBCache()
	default:
		panic("unknown cache type: " + typ)
	}
	log.Println(typ, "ready to serve")
	return c
}
