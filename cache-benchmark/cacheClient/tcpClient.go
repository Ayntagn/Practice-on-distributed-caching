package cacheClient

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"caches-core/ports"
)

type tcpClient struct {
	net.Conn
	r *bufio.Reader
}

func (c *tcpClient) sendGet(key string) {
	klen := len(key)
	c.Write([]byte(fmt.Sprintf("G%d %s", klen, key)))
}

func (c *tcpClient) sendSet(key, value string) {
	klen, vlen := len(key), len(value)
	c.Write([]byte(fmt.Sprintf("S%d %d %s%s", klen, vlen, key, value)))
}

func (c *tcpClient) sendDel(key string) {
	klen := len(key)
	c.Write([]byte(fmt.Sprintf("D%d %s", klen, key)))
}

func readLen(r *bufio.Reader) int {
	tmp, e := r.ReadString(' ')
	if e != nil {
		log.Println(e)
		return 0
	}
	l, e := strconv.Atoi(strings.TrimSpace(tmp))
	if e != nil {
		log.Println(tmp, e)
		return 0
	}
	return l
}

func (c *tcpClient) recvResponse() (string, error) {
	vlen := readLen(c.r)
	if vlen == 0 {
		return "", nil
	}
	if vlen < 0 {
		err := make([]byte, -vlen)
		_, e := io.ReadFull(c.r, err)
		if e != nil {
			return "", e
		}
		return "", errors.New(string(err))
	}
	value := make([]byte, vlen)
	_, e := io.ReadFull(c.r, value)
	if e != nil {
		return "", e
	}
	return string(value), nil
}

const redirectPrefix = "redirect "

// followRedirect re-executes a command rejected with a
// "redirect <addr>" error on the owner node (D1). The owner is
// authoritative, so a single retry is enough.
func (c *tcpClient) followRedirect(cmd *Cmd) {
	if cmd.Error == nil || !strings.HasPrefix(cmd.Error.Error(), redirectPrefix) {
		return
	}
	addr := strings.TrimPrefix(cmd.Error.Error(), redirectPrefix)
	owner := newTCPClient(addr)
	defer owner.Close()
	cmd.Error = nil
	owner.Run(cmd)
}

func (c *tcpClient) Run(cmd *Cmd) {
	switch cmd.Name {
	case "get":
		c.sendGet(cmd.Key)
		cmd.Value, cmd.Error = c.recvResponse()
	case "set":
		c.sendSet(cmd.Key, cmd.Value)
		_, cmd.Error = c.recvResponse()
	case "del":
		c.sendDel(cmd.Key)
		_, cmd.Error = c.recvResponse()
	default:
		panic("unknown cmd name " + cmd.Name)
	}
	c.followRedirect(cmd)
}

func (c *tcpClient) PipelinedRun(cmds []*Cmd) {
	if len(cmds) == 0 {
		return
	}
	for _, cmd := range cmds {
		if cmd.Name == "get" {
			c.sendGet(cmd.Key)
		}
		if cmd.Name == "set" {
			c.sendSet(cmd.Key, cmd.Value)
		}
		if cmd.Name == "del" {
			c.sendDel(cmd.Key)
		}
	}
	for _, cmd := range cmds {
		cmd.Value, cmd.Error = c.recvResponse()
		c.followRedirect(cmd)
	}
}

func newTCPClient(server string) *tcpClient {
	c, e := net.Dial("tcp", server+":"+ports.TCP)
	if e != nil {
		panic(e)
	}
	r := bufio.NewReader(c)
	return &tcpClient{c, r}
}
