package cache

// #cgo CFLAGS: -I${SRCDIR}/../third_party/rocksdb/include
// #cgo LDFLAGS: -L${SRCDIR}/../third_party/rocksdb/build -lrocksdb -lc++ -lpthread -ldl
// #include <rocksdb/c.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"regexp"
	"runtime"
	"strconv"
	"unsafe"

	ccore "caches-core/cache"
)

var propRegexp = regexp.MustCompile(`([^;]+)=([^;]+);`)

// rocksdbCache wraps the rocksdb C API. The db handle is safe for
// concurrent use; each call keeps its error output in a local
// variable so concurrent requests never race on a shared pointer.
type rocksdbCache struct {
	db *C.rocksdb_t
	ro *C.rocksdb_readoptions_t
	wo *C.rocksdb_writeoptions_t
}

type rocksdbScanner struct {
	i           *C.rocksdb_iterator_t
	initialized bool
}

func (s *rocksdbScanner) Close() {
	C.rocksdb_iter_destroy(s.i)
}

func (s *rocksdbScanner) Scan() bool {
	if !s.initialized {
		C.rocksdb_iter_seek_to_first(s.i)
		s.initialized = true
	} else {
		C.rocksdb_iter_next(s.i)
	}
	return C.rocksdb_iter_valid(s.i) != 0
}

func (s *rocksdbScanner) Key() string {
	var length C.size_t
	k := C.rocksdb_iter_key(s.i, &length)
	return C.GoStringN(k, C.int(length))
}

func (s *rocksdbScanner) Value() []byte {
	var length C.size_t
	v := C.rocksdb_iter_value(s.i, &length)
	return C.GoBytes(unsafe.Pointer(v), C.int(length))
}

func (c *rocksdbCache) NewScanner() ccore.Scanner {
	return &rocksdbScanner{C.rocksdb_create_iterator(c.db,
		c.ro), false}
}

func newRocksDBCache() *rocksdbCache {
	options := C.rocksdb_options_create()
	C.rocksdb_options_increase_parallelism(options, C.int(runtime.NumCPU()))
	C.rocksdb_options_set_create_if_missing(options, 1)
	var e *C.char
	db := C.rocksdb_open(options, C.CString("rocksdb"), &e)
	if e != nil {
		panic(C.GoString(e))
	}
	C.rocksdb_options_destroy(options)
	return &rocksdbCache{
		db: db,
		ro: C.rocksdb_readoptions_create(),
		wo: C.rocksdb_writeoptions_create(),
	}
}

func (c *rocksdbCache) Get(key string) ([]byte, error) {
	k := C.CString(key)
	defer C.free(unsafe.Pointer(k))
	var length C.size_t
	var e *C.char
	v := C.rocksdb_get(c.db, c.ro, k, C.size_t(len(key)), &length, &e)
	if e != nil {
		defer C.free(unsafe.Pointer(e))
		return nil, errors.New(C.GoString(e))
	}
	if v == nil {
		return nil, nil
	}
	defer C.free(unsafe.Pointer(v))
	return C.GoBytes(unsafe.Pointer(v), C.int(length)), nil
}

// Set writes synchronously (B1: the previous async write-batch made
// entries invisible until flushed, so Set followed by Get missed).
// rocksdb_put copies the value into the memtable/WAL before returning.
func (c *rocksdbCache) Set(key string, value []byte) error {
	k := C.CString(key)
	defer C.free(unsafe.Pointer(k))
	var v *C.char
	if len(value) > 0 {
		v = (*C.char)(unsafe.Pointer(&value[0]))
	}
	var e *C.char
	C.rocksdb_put(c.db, c.wo, k, C.size_t(len(key)), v, C.size_t(len(value)), &e)
	if e != nil {
		defer C.free(unsafe.Pointer(e))
		return errors.New(C.GoString(e))
	}
	return nil
}

func (c *rocksdbCache) Del(key string) error {
	k := C.CString(key)
	defer C.free(unsafe.Pointer(k))
	var e *C.char
	C.rocksdb_delete(c.db, c.wo, k, C.size_t(len(key)), &e)
	if e != nil {
		defer C.free(unsafe.Pointer(e))
		return errors.New(C.GoString(e))
	}
	return nil
}

func (c *rocksdbCache) GetStat() ccore.Stat {
	k := C.CString("rocksdb.aggregated-table-properties")
	defer C.free(unsafe.Pointer(k))
	v := C.rocksdb_property_value(c.db, k)
	defer C.free(unsafe.Pointer(v))
	p := C.GoString(v)
	s := ccore.Stat{}
	for _, submatches := range propRegexp.FindAllStringSubmatch(p, -1) {
		if submatches[1] == " # entries" {
			s.Count, _ = strconv.ParseInt(submatches[2], 10, 64)
		} else if submatches[1] == " raw key size" {
			s.KeySize, _ = strconv.ParseInt(submatches[2], 10, 64)
		} else if submatches[1] == " raw value size" {
			s.ValueSize, _ = strconv.ParseInt(submatches[2], 10, 64)
		}
	}
	return s
}
