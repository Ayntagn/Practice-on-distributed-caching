// Package protocol implements the length-prefixed TCP wire protocol
// shared by all cache servers:
//
//	<op><klen> <vlen> <key><value>   (set)
//	<op><klen> <key>                 (get/del)
//	<vlen> <value>                   (response)
//	-<elen> <error>                  (error response)
package protocol

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

// MaxLen bounds key/value lengths advertised by the peer (E1: prevent
// unbounded allocations from a malformed length prefix).
const MaxLen = 1 << 20 // 1 MiB

// ReadLen reads a space-terminated length prefix.
func ReadLen(r *bufio.Reader) (int, error) {
	tmp, e := r.ReadString(' ')
	if e != nil {
		return 0, e
	}
	l, e := strconv.Atoi(strings.TrimSpace(tmp))
	if e != nil {
		return 0, e
	}
	if l < 0 || l > MaxLen {
		return 0, fmt.Errorf("invalid length %d", l)
	}
	return l, nil
}

// ReadKey reads a length-prefixed key.
func ReadKey(r *bufio.Reader) (string, error) {
	klen, e := ReadLen(r)
	if e != nil {
		return "", e
	}
	k := make([]byte, klen)
	_, e = io.ReadFull(r, k)
	if e != nil {
		return "", e
	}
	return string(k), nil
}

// ReadKeyAndValue reads a length-prefixed key followed by a
// length-prefixed value.
func ReadKeyAndValue(r *bufio.Reader) (string, []byte, error) {
	klen, e := ReadLen(r)
	if e != nil {
		return "", nil, e
	}
	vlen, e := ReadLen(r)
	if e != nil {
		return "", nil, e
	}
	k := make([]byte, klen)
	_, e = io.ReadFull(r, k)
	if e != nil {
		return "", nil, e
	}
	v := make([]byte, vlen)
	_, e = io.ReadFull(r, v)
	if e != nil {
		return "", nil, e
	}
	return string(k), v, nil
}

// SendResponse writes a length-prefixed value, or a length-prefixed
// error message when err != nil.
func SendResponse(value []byte, err error, conn net.Conn) error {
	if err != nil {
		errString := err.Error()
		tmp := fmt.Sprintf("-%d ", len(errString)) + errString
		_, e := conn.Write([]byte(tmp))
		return e
	}
	vlen := fmt.Sprintf("%d ", len(value))
	_, e := conn.Write(append([]byte(vlen), value...))
	return e
}
