package main

import (
	"cache/cache"
	"cache/http"
)

func main() {
	c := cache.New("inmemory")
	http.New(c).Listen()
}
