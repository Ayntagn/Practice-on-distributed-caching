module cache-benchmark

go 1.26.3

require github.com/redis/go-redis/v9 v9.22.0

require (
	caches-core v0.0.0
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

replace caches-core => ../caches-core
