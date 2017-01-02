package response_cache

import "testing"
import "bytes"
import "net/http"
import "os"
import "reflect"

func returnNotExist() error {
	return os.ErrNotExist
}

func testCacheSetAndGet(t *testing.T, cache ResponseCache) {
	dummyHeader := make(http.Header)
	dummyHeader.Add("Host", "www.example.com")
	dummyHeader.Add("Content-Type", "text/html")
	dummyData := []byte("Test response body\x00test.")

	// test opening but not starting a write
	writer, err := cache.BeginWrite("key1")
	if err != nil { panic(err) }

	_, err = cache.Get("key1", returnNotExist)
	if err == nil { t.Error("Cache should not contain key not finished") }

	writer.Write(dummyData)
	_, err = cache.Get("key1", returnNotExist)
	if err == nil { t.Error("Cache should not contain key not finished") }

	writer.Abort()
	_, err = cache.Get("key1", returnNotExist)
	if err == nil { t.Error("Cache should not contain key not finished") }

	// test starting but not finishing a write
	writer, err = cache.BeginWrite("key1")
	writer.WriteHeader(http.StatusOK, dummyHeader)
	if err != nil { panic(err) }

	_, err = cache.Get("key1", returnNotExist)
	if err == nil { t.Error("Cache should not contain key not finished") }

	writer.Write(dummyData)
	_, err = cache.Get("key1", returnNotExist)
	if err == nil { t.Error("Cache should not contain key not finished") }

	writer.Abort()
	_, err = cache.Get("key1", returnNotExist)
	if err == nil { t.Error("Cache should not contain key not finished") }

	// test an actual write
	writer, err = cache.BeginWrite("key2")
	writer.WriteHeader(http.StatusOK, dummyHeader)
	if err != nil { panic(err) }
	writer.Write(dummyData)
	writer.Finish()

	entry, err := cache.Get("key2", returnNotExist)

	if err != nil { t.Error("Cache did not contain written key") }
	if entry.Status() != http.StatusOK { t.Error("Status was not restored from the cache") }
	if !reflect.DeepEqual(entry.Header(), dummyHeader) { t.Error("Header was not restored from the cache") }

	buffer := bytes.Buffer{}
	_, err = buffer.ReadFrom(entry.Body())
	if err != nil { t.Error("Data could not be read from the cache: " + err.Error()) }
	if !reflect.DeepEqual(buffer.Bytes(), dummyData) { t.Error("Data was not restored from the cache accurately") }

	// test other keys are still not present
	_, err = cache.Get("key3", returnNotExist)
	if err == nil { t.Error("Cache should not contain key not finished") }
}

func TestResponseCacheSetAndGet(t *testing.T) {
	testCacheSetAndGet(t, NewMemoryCache())
	testCacheSetAndGet(t, NewDiskCache("test/cache"))
}
