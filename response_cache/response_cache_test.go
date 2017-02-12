package response_cache

import "testing"
import "errors"
import "net/http"
import "os"

func returnDummyError(writer http.ResponseWriter) error {
	return errors.New("dummy error")
}

func testCacheSetAndGet(t *testing.T, cache ResponseCache) {
	cache.Clear()

	dummyHeader := make(http.Header)
	dummyHeader.Add("Host", "www.example.com")
	dummyHeader.Add("Content-Type", "text/html")
	dummyData := []byte("Test response body\x00test.")
	dummyResponse := responseData{
		Status: http.StatusOK,
		Header: dummyHeader,
		Data:   [][]byte{dummyData},
	}

	// test opening but not starting a write
	err := cache.Get("key1", newDummyResponseWriter(t), returnDummyError)
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	err = cache.Get("key1", newDummyResponseWriter(t), returnDummyError)
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	// test setting up but not performing a write
	err = cache.Get("key1", newDummyResponseWriter(t), func(writer http.ResponseWriter) error {
		writer.Header().Set("dummy", "value")
		return os.ErrNotExist
	})
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	err = cache.Get("key1", newDummyResponseWriter(t), returnDummyError)
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	// test aborting a write before starting the body
	err = cache.Get("key1", newDummyResponseWriter(t), func(writer http.ResponseWriter) error {
		writer.Header().Set("dummy", "value")
		writer.WriteHeader(http.StatusOK)
		return errors.New("aborted")
	})
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	err = cache.Get("key1", newDummyResponseWriter(t), returnDummyError)
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	// test aborting a write after starting the body
	err = cache.Get("key1", newDummyResponseWriter(t), func(writer http.ResponseWriter) error {
		writer.Header().Set("dummy", "value")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("test"))
		return errors.New("aborted")
	})
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	err = cache.Get("key1", newDummyResponseWriter(t), returnDummyError)
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}

	// test an actual write
	responseWriter := newDummyResponseWriter(t)
	err = cache.Get("key2", responseWriter, dummyResponse.copyResponseTo)
	if !os.IsNotExist(err) {
		t.Error("Cache did not contain written key")
	}
	testResponse(t, responseWriter.response, http.StatusOK, dummyHeader, dummyData)

	// and read it again
	responseWriter = newDummyResponseWriter(t)
	err = cache.Get("key2", responseWriter, dummyResponse.copyResponseTo)
	if err != nil {
		t.Error("Cache did not contain written key")
	}
	testResponse(t, responseWriter.response, http.StatusOK, dummyHeader, dummyData)

	// test other keys are still not present
	responseWriter = newDummyResponseWriter(t)
	err = cache.Get("key3", responseWriter, returnDummyError)
	if err == nil {
		t.Error("Cache should not contain key not finished")
	}
}

func TestDiskCacheSetAndGet(t *testing.T) {
	testCacheSetAndGet(t, NewDiskCache("test/cache"))
}
