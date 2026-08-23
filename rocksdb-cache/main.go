package main

import (
	"flag"
	"log"
	"rocksdb-cache/cache"
	"rocksdb-cache/http"
	"rocksdb-cache/tcp"
)

func main() {
	typ := flag.String("type", "inmemory", "cache type")
	flag.Parse()
	log.Println("type is: ", *typ)
	ca := cache.New(*typ)
	go tcp.New(ca).Listen()
	http.New(ca).Listen()
}
