package cache

// Cache is the storage interface shared by all cache servers.
// Implementations: inMemoryCache (this package) and
// module-local rocksdb caches (cgo, kept per-module).
type Cache interface {
	Set(string, []byte) error
	Get(string) ([]byte, error)
	Del(string) error
	GetStat() Stat
}

// NewMemory returns an in-memory cache.
func NewMemory() Cache {
	return newInMemoryCache()
}
