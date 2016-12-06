package response_cache

import "testing"
import "bytes"
import "net/http"
import "reflect"

func testCacheSetAndGet(t *testing.T, cache ResponseCache) {
	dummyHeader := make(http.Header)
	dummyHeader.Add("Host", "www.example.com")
	dummyHeader.Add("Content-Type", "text/html")
	dummyData := []byte("Test response body\x00test.")

	// test starting but not finishing a write
	bodyWriter, err := cache.BeginWrite("key1", http.StatusOK, dummyHeader)
	if err != nil { panic(err) }

	_, err = cache.Get("key1")
	if err == nil { t.Error("Cache should not contain key not finished") }

	bodyWriter.Write(dummyData)
	_, err = cache.Get("key1")
	if err == nil { t.Error("Cache should not contain key not finished") }

	bodyWriter.Abort()
	_, err = cache.Get("key1")
	if err == nil { t.Error("Cache should not contain key not finished") }

	// test an actual write
	bodyWriter, err = cache.BeginWrite("key2", http.StatusOK, dummyHeader)
	if err != nil { panic(err) }
	bodyWriter.Write(dummyData)
	bodyWriter.Finish()

	entry, err := cache.Get("key2")

	if err != nil { t.Error("Cache did not contain written key") }
	if entry.Status() != http.StatusOK { t.Error("Status was not restored from the cache") }
	if !reflect.DeepEqual(entry.Header(), dummyHeader) { t.Error("Header was not restored from the cache") }

	buffer := bytes.Buffer{}
	_, err = buffer.ReadFrom(entry.Body())
	if err != nil { t.Error("Data could not be read from the cache: " + err.Error()) }
	if !reflect.DeepEqual(buffer.Bytes(), dummyData) { t.Error("Data was not restored from the cache accurately") }

	// test other keys are still not present
	_, err = cache.Get("key3")
	if err == nil { t.Error("Cache should not contain key not finished") }
}

func TestResponseCacheSetAndGet(t *testing.T) {
	testCacheSetAndGet(t, NewMemoryCache())
	testCacheSetAndGet(t, NewDiskCache("test/cache"))
}
