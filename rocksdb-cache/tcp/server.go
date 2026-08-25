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

func (s *Server) get(rq *protocol.ResultQueue, r *bufio.Reader) {
	c := make(chan *protocol.Result)
	rq.Push(c)
	k, e := protocol.ReadKey(r)
	if e != nil {
		c <- &protocol.Result{V: nil, E: e}
		return
	}
	go func() {
		v, e := s.Get(k)
		c <- &protocol.Result{V: v, E: e}
	}()
}

func (s *Server) set(rq *protocol.ResultQueue, r *bufio.Reader) {
	c := make(chan *protocol.Result)
	rq.Push(c)
	k, v, e := protocol.ReadKeyAndValue(r)
	if e != nil {
		c <- &protocol.Result{V: nil, E: e}
		return
	}
	go func() {
		c <- &protocol.Result{V: nil, E: s.Set(k, v)}
	}()
}

func (s *Server) del(rq *protocol.ResultQueue, r *bufio.Reader) {
	c := make(chan *protocol.Result)
	rq.Push(c)
	k, e := protocol.ReadKey(r)
	if e != nil {
		c <- &protocol.Result{V: nil, E: e}
		return
	}
	go func() {
		c <- &protocol.Result{V: nil, E: s.Del(k)}
	}()
}

func (s *Server) process(conn net.Conn) {
	r := bufio.NewReader(conn)
	rq := protocol.NewResultQueue()
	defer rq.Close()
	go reply(conn, rq)
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
			s.set(rq, r)
		case 'G':
			s.get(rq, r)
		case 'D':
			s.del(rq, r)
		default:
			log.Println("close connection due to invalid operation: ", op)
			return
		}
	}
}

func reply(conn net.Conn, rq *protocol.ResultQueue) {
	defer conn.Close()
	for {
		c, open := rq.Pop()
		if !open {
			return
		}
		r := <-c
		e := protocol.SendResponse(r.V, r.E, conn)
		if e != nil {
			log.Println("close connection due to error: ", e)
			return
		}
	}
}
