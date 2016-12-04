package response_cache

import "testing"
import "bytes"
import "reflect"

func dummyCacheEntry() Entry {
	entry := NewCacheEntry()
	entry.Status = 200
	entry.Header.Add("Host", "www.example.com")
	entry.Header.Add("Content-Type", "text/html")
	entry.Body = bytes.NewReader([]byte("Test response body\x00test."))
	return entry
}

func testCacheSetAndGet(t *testing.T, cache ResponseCache) {
	entry := dummyCacheEntry()
	cache.Set("key1", entry)
	retrieved, ok := cache.Get("key1")

	if !ok { t.Error("Cache did not contain written key") }
	if entry.Status != retrieved.Status { t.Error("Status was not restored from the cache") }
	if !reflect.DeepEqual(entry.Header, retrieved.Header) { t.Error("Header was not restored from the cache") }
	if !reflect.DeepEqual(entry.Body, retrieved.Body) { t.Error("Body was not restored from the cache") }

	_, ok = cache.Get("key2")
	if ok { t.Error("Cache should not contain key not set") }
}

func TestResponseCacheSetAndGet(t *testing.T) {
	testCacheSetAndGet(t, NewMemoryCache())
	testCacheSetAndGet(t, NewDiskCache("test"))
}
