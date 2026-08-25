package cache

// Cache is the storage interface shared by all cache servers.
// Implementations: inMemoryCache (this package) and
// module-local rocksdb caches (cgo, kept per-module).
type Cache interface {
	Set(string, []byte) error
	Get(string) ([]byte, error)
	Del(string) error
	GetStat() Stat
	NewScanner() Scanner
}

// Scanner iterates over the cache contents. It exists for the distributed
// rebalance path: walk every local key, and for keys whose owner moved to
// another member, forward the value there and delete locally. Implementations
// must not require the cache lock to be held while Scan/Key/Value are called.
type Scanner interface {
	Scan() bool
	Key() string
	Value() []byte
	Close()
}

// NewMemory returns an in-memory cache.
func NewMemory() Cache {
	return newInMemoryCache()
}
