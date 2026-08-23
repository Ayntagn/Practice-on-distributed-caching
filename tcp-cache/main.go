package main

import (
	"tcp-cache/cache"
	"tcp-cache/http"
	"tcp-cache/tcp"
)

func main() {
	ca := cache.New("inmemory")
	go tcp.New(ca).Listen()
	http.New(ca).Listen()
}
