module cache-client

go 1.26.3

require cache-benchmark v0.0.0

require (
	caches-core v0.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

// replace directives do not propagate across modules: cache-benchmark's
// own replace of caches-core is invisible here, so re-declare both local
// paths (caches-core is a transitive dependency via cache-benchmark).
replace (
	cache-benchmark => ../cache-benchmark
	caches-core => ../caches-core
)
