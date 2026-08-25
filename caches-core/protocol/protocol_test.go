package protocol

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
)

func readKeyFrom(t *testing.T, input string) (string, error) {
	t.Helper()
	return ReadKey(bufio.NewReader(strings.NewReader(input)))
}

func TestReadKey(t *testing.T) {
	k, e := readKeyFrom(t, "5 hello")
	if e != nil {
		t.Fatal(e)
	}
	if k != "hello" {
		t.Fatalf("got %q, want hello", k)
	}
}

func TestReadKeyRejectsNegativeLength(t *testing.T) {
	if _, e := readKeyFrom(t, "-3 abc"); e == nil {
		t.Fatal("expected error for negative length")
	}
}

func TestReadKeyRejectsOversizeLength(t *testing.T) {
	if _, e := readKeyFrom(t, "999999999 abc"); e == nil {
		t.Fatal("expected error for length beyond MaxLen")
	}
}

func TestReadKeyRejectsTruncated(t *testing.T) {
	if _, e := readKeyFrom(t, "5 he"); e == nil {
		t.Fatal("expected error for truncated payload")
	}
}

func TestReadKeyAndValue(t *testing.T) {
	// "2 3 okyes": klen=2 vlen=3, then key "ok" + value "yes".
	r := bufio.NewReader(strings.NewReader("2 3 okyes"))
	k, v, e := ReadKeyAndValue(r)
	if e != nil {
		t.Fatal(e)
	}
	if k != "ok" || string(v) != "yes" {
		t.Fatalf("got %q %q", k, v)
	}
}

// TestSendResponseErrorFormat checks the "-<elen> <err>" wire format
// the benchmark client's recvResponse parses.
func TestSendResponseErrorFormat(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	go SendResponse(nil, io.ErrUnexpectedEOF, server)
	b := make([]byte, 64)
	n, e := client.Read(b)
	if e != nil {
		t.Fatal(e)
	}
	got := string(b[:n])
	want := "-14 unexpected EOF"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSendResponseValue(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	go SendResponse([]byte("data"), nil, server)
	b := make([]byte, 64)
	n, e := client.Read(b)
	if e != nil {
		t.Fatal(e)
	}
	if got := string(b[:n]); got != "4 data" {
		t.Fatalf("got %q, want %q", got, "4 data")
	}
}

func TestResultQueueFIFO(t *testing.T) {
	rq := NewResultQueue()
	c1 := make(chan *Result)
	c2 := make(chan *Result)
	rq.Push(c1)
	rq.Push(c2)
	got1, _ := rq.Pop()
	got2, _ := rq.Pop()
	if got1 != c1 || got2 != c2 {
		t.Fatal("queue order violated")
	}
	rq.Close()
}

// TestResultQueueCloseDrainsThenEnds: after Close, pending items are
// still served, then Pop returns closed.
func TestResultQueueCloseDrainsThenEnds(t *testing.T) {
	rq := NewResultQueue()
	c1 := make(chan *Result)
	rq.Push(c1)
	rq.Close()
	got, open := rq.Pop()
	if !open || got != c1 {
		t.Fatalf("drain failed: open=%v", open)
	}
	if _, open := rq.Pop(); open {
		t.Fatal("expected closed after drain")
	}
}

// TestResultQueueConcurrent hammers Push/Pop from many goroutines;
// -race catches corruption, and a final drain must return exactly the
// number of pushed items.
func TestResultQueueConcurrent(t *testing.T) {
	const producers, perProducer = 8, 2000
	rq := NewResultQueue()
	var wg sync.WaitGroup
	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perProducer; i++ {
				rq.Push(make(chan *Result))
			}
		}()
	}
	// Wait until every push is visible, then count them out.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	count := 0
	for count < producers*perProducer {
		select {
		case <-done:
		default:
		}
		_, open := rq.Pop()
		if !open {
			t.Fatal("queue closed unexpectedly")
		}
		count++
	}
	rq.Close()
}

func TestReadLenTruncated(t *testing.T) {
	if _, e := ReadLen(bufio.NewReader(bytes.NewReader(nil))); e == nil {
		t.Fatal("expected EOF on empty reader")
	}
}
