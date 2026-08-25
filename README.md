# caches — Distributed Cache in Go

A Go multi-module workspace implementing a distributed cache: consistent-hash
sharding across a memberlist cluster, with HTTP and TCP access, in-memory and
RocksDB storage backends, plus a benchmark client.

```
                     ┌─────────────────────────────┐
 client ──HTTP 307──▶│  node A  ◀──gossip (7946)──▶│  node B
 (curl / ./client)   │  12345/12346                 │  12345/12346
                     └──────────────┬──────────────┘
                                    │ consistent-hash ring (256 replicas)
                     every key is owned by exactly one node
```

## Modules

| Module | Description |
|--------|-------------|
| `caches-core` | Shared core: `Cache` interface + in-memory impl, TCP wire protocol helpers, `ResultQueue` (ordered pipelined replies), port constants |
| `cache` | Single-node in-memory cache server (HTTP) |
| `tcp-cache` | Single-node cache server (HTTP + TCP) |
| `rocksdb-cache` | Single-node cache server backed by RocksDB (HTTP + TCP) |
| `distributedCache` | Sharded cluster: memberlist membership, consistent-hash routing, HTTP 307 / TCP `redirect` forwarding |
| `cache-benchmark` | Benchmark client (http / tcp / redis protocols, pipelining) |
| `cache-client` | Small CLI for manual testing over the TCP protocol |

Modules are independent Go modules connected with `replace` directives
(no `go.work`).

## Ports

| Service | Port |
|---------|------|
| HTTP | 12345 |
| TCP  | 12346 |
| Redis (external server only) | 6379 |
| memberlist gossip | 7946 |

## Quick Start

### 1. Single node

```bash
cd cache
go run main.go            # in-memory
# or rocksdb backend:
cd ../rocksdb-cache && go run main.go -type rocksdb
```

### 2. Sharded cluster

`-node` must be a **real IP of a local interface** (`ifconfig`). The first
node joins nothing; later nodes point `-cluster` at any running node:

```bash
cd distributedCache
go run main.go -node 127.0.0.1 &
go run main.go -node 10.250.180.183 -cluster 127.0.0.1 &
```

Membership converges via gossip (a few seconds). Each key is owned by exactly
one node; requests for non-owned keys are transparently forwarded:

```bash
curl http://127.0.0.1:12345/cluster          # member list
curl -sL -X PUT --data 'v1' http://127.0.0.1:12345/cache/key1   # auto-follows redirect
curl -sL http://127.0.0.1:12345/cache/key1   # -> v1
```

### 3. Manual client

```bash
cd cache-client
go build -o client .
./client -h 127.0.0.1 -c set -k keya -v a    # -> a (echoes the value)
./client -h 127.0.0.1 -c get -k keya         # -> a
./client -h 127.0.0.1 -c del -k keya
```

Redirects are followed transparently, so the client works against any node.

### 4. Benchmark

```bash
cd cache-benchmark
go run . -type tcp -h 127.0.0.1 -t mixed -n 2000 -c 8 -r 100 -d 100 -P 16
```

| Flag | Meaning |
|------|---------|
| `-type` | `tcp`, `http` or `redis` (redis needs an external server on 6379) |
| `-t` | `set` / `get` / `mixed` |
| `-n` | total requests |
| `-c` | parallel connections |
| `-r` | keyspace length (0 = sequential keys) |
| `-d` | value size in bytes |
| `-P` | pipeline length (**http must use `-P 1`** — pipelined http is not implemented) |

## TCP wire protocol

Length-prefixed, space-separated ASCII:

```
request:  <op><klen> <vlen> <key><value>      op: S (set), G (get), D (del)
response: <vlen> <value>                      on success
          -<elen> <error>                     on failure
```

## RocksDB backend

`third_party/rocksdb` vendors the RocksDB source tree. Build it with `make`
(lib is linked as `third_party/rocksdb/build/librocksdb.a` via cgo). On
clang 21+ add `-Wno-nontrivial-memcall` to `CXXFLAGS`, and note that piping
the build (e.g. `make 2>&1 | tee`) can mask the make exit code.

## Testing

```bash
go test -race ./...    # per module
```

Tests cover the cache core (concurrency-safe, `-race` clean), protocol
encoding/decoding (length validation, truncation), `ResultQueue` (FIFO,
close-drain, concurrent push/pop), and cluster routing (ownership, empty
ring, async event callbacks).

## Known limitations

- **No replication or failover** — data is sharded but not copied; when a
  node leaves, its keys are lost (the ring is rebuilt).
- **No TTL or eviction** — the in-memory cache grows without bound.
- **No Redis server** — the benchmark's redis mode targets an external
  `redis-server`.
- **One node per IP** — memberlist binds port 7946, so two nodes need
  different local IPs; `-node` must be a real interface address
  (loopback aliases work after `sudo ifconfig lo0 alias 127.0.0.2 up`).
- **VPN tunnel interfaces (e.g. `198.18.0.1`)** can bind but may not
  converge with other interfaces — routing is hijacked by the VPN client.
