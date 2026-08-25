package cache

import (
	"sync"
	"testing"
)

func TestMemorySetGet(t *testing.T) {
	c := NewMemory()
	if e := c.Set("hello", []byte("world")); e != nil {
		t.Fatal(e)
	}
	v, e := c.Get("hello")
	if e != nil {
		t.Fatal(e)
	}
	if string(v) != "world" {
		t.Fatalf("got %q, want %q", v, "world")
	}
}

func TestMemoryGetMiss(t *testing.T) {
	c := NewMemory()
	v, e := c.Get("missing")
	if e != nil {
		t.Fatal(e)
	}
	if v != nil {
		t.Fatalf("got %q, want nil", v)
	}
}

func TestMemoryOverwriteAndDel(t *testing.T) {
	c := NewMemory()
	c.Set("k", []byte("v1"))
	c.Set("k", []byte("v2"))
	v, _ := c.Get("k")
	if string(v) != "v2" {
		t.Fatalf("overwrite failed: %q", v)
	}
	if e := c.Del("k"); e != nil {
		t.Fatal(e)
	}
	v, _ = c.Get("k")
	if v != nil {
		t.Fatalf("del failed: %q", v)
	}
}

func TestMemoryGetStat(t *testing.T) {
	c := NewMemory()
	c.Set("abc", []byte("12345"))
	c.Set("xy", []byte("12"))
	s := c.GetStat()
	if s.Count != 2 || s.KeySize != 5 || s.ValueSize != 7 {
		t.Fatalf("stat = %+v, want {2 5 7}", s)
	}
	c.Del("abc")
	s = c.GetStat()
	if s.Count != 1 || s.KeySize != 2 || s.ValueSize != 2 {
		t.Fatalf("stat after del = %+v, want {1 2 2}", s)
	}
}

// TestMemoryConcurrent hammers the cache from many goroutines; run with
// -race to catch the shared-state races the original had.
func TestMemoryConcurrent(t *testing.T) {
	c := NewMemory()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				key := string(rune('a' + id)) + string(rune('a' + j%26))
				value := []byte{byte(j)}
				if j%3 == 0 {
					c.Set(key, value)
				} else if j%3 == 1 {
					c.Get(key)
				} else {
					c.Del(key)
				}
				c.GetStat()
			}
		}(i)
	}
	wg.Wait()
}
