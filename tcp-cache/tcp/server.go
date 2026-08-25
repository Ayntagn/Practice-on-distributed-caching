package tcp

import (
	"bufio"
	"io"
	"log"
	"net"

	ccore "caches-core/cache"
	"caches-core/ports"
	"caches-core/protocol"
)

type Server struct {
	ccore.Cache
}

func New(c ccore.Cache) *Server {
	return &Server{
		Cache: c,
	}
}

func (s *Server) Listen() {
	l, e := net.Listen("tcp", ":"+ports.TCP)
	if e != nil {
		panic(e)
	}
	for {
		c, e := l.Accept()
		if e != nil {
			panic(e)
		}
		go s.process(c)
	}
}

// Serial protocol processing: each command is read, applied and
// answered before the next one is parsed (no pipelining here, unlike
// the rocksdb/distributedCache variants).
func (s *Server) process(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		op, e := r.ReadByte()
		if e != nil {
			if e != io.EOF {
				log.Println("close connection error:", e)
			}
			return
		}
		switch op {
		case 'S':
			e = s.set(conn, r)
		case 'G':
			e = s.get(conn, r)
		case 'D':
			e = s.del(conn, r)
		default:
			log.Println("close connection due to invalid operation: ", op)
			return
		}
		if e != nil {
			log.Println("close connection due to error: ", e)
			return
		}
	}
}

func (s *Server) get(conn net.Conn, r *bufio.Reader) error {
	k, e := protocol.ReadKey(r)
	if e != nil {
		return e
	}
	v, e := s.Get(k)
	return protocol.SendResponse(v, e, conn)
}

func (s *Server) set(conn net.Conn, r *bufio.Reader) error {
	k, v, e := protocol.ReadKeyAndValue(r)
	if e != nil {
		return e
	}
	e = s.Set(k, v)
	return protocol.SendResponse(nil, e, conn)
}

func (s *Server) del(conn net.Conn, r *bufio.Reader) error {
	k, e := protocol.ReadKey(r)
	if e != nil {
		return e
	}
	e = s.Del(k)
	return protocol.SendResponse(nil, e, conn)
}
