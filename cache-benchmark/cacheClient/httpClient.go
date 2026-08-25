package cacheClient

import (
	"io"
	"log"
	"net/http"
	"strings"

	"caches-core/ports"
)

type httpClient struct {
	*http.Client
	server string
}

func (c *httpClient) get(key string) string {
	resp, e := c.Get(c.server + key)
	if e != nil {
		log.Println(key)
		panic(e)
	}
	if resp.StatusCode == http.StatusNotFound {
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}
	b, e := io.ReadAll(resp.Body)
	if e != nil {
		panic(e)
	}
	return string(b)
}

func (c *httpClient) set(key, value string) {
	req, e := http.NewRequest(
		http.MethodPut,
		c.server+key,
		strings.NewReader(value),
	)
	if e != nil {
		log.Println(key)
		panic(e)
	}
	resp, e := c.Do(req)
	if e != nil {
		log.Println(key)
		panic(e)
	}
	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}
}

func (c *httpClient) Run(cmd *Cmd) {
	if cmd.Name == "get" {
		cmd.Value = c.get(cmd.Key)
		return
	} else if cmd.Name == "set" {
		c.set(cmd.Key, cmd.Value)
		return
	} else {
		panic("unknown cmd name " + cmd.Name)
	}
}

func newHTTPClient(server string) *httpClient {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 1,
		},
	}
	return &httpClient{
		client,
		"http://" + server + ":" + ports.HTTP + "/cache/",
	}

}

func (c *httpClient) PipelinedRun([]*Cmd) {
	panic("httpClient pipelined run not implement")
}
