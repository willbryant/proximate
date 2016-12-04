package response_cache

import "testing"
import "fmt"
import "net/http"
import "reflect"

type responseData struct {
	Status int
	Header http.Header
	Data   [][]byte
}

type dummyResponseWriter struct {
	response responseData
}

func (responseWriter *dummyResponseWriter) Header() http.Header {
	return responseWriter.response.Header
}

func (responseWriter *dummyResponseWriter) WriteHeader(status int) {
	responseWriter.response.Status = status
}

func (responseWriter *dummyResponseWriter) Write(data []byte) (int, error) {
	responseWriter.response.Data = append(responseWriter.response.Data, data)
	return len(data), nil
}

func newDummyResponseWriter() *dummyResponseWriter {
	responseWriter := &dummyResponseWriter{
		response: responseData{
			Header: make(http.Header),
		},
	}
	return responseWriter
}

type cacheWriterTestScenario struct {
	responseData
	ShouldStore bool
}

func TestCacheWriter(t *testing.T) {
	scenarios := []cacheWriterTestScenario{
		{
			responseData: responseData{
				Status: 200,
				Header: http.Header{
					"Content-Type": []string{"text/html"},
					"X-Served-By":  []string{"test case"},
				},
				Data: [][]byte{
					[]byte("Test response body."),
				},
			},
			ShouldStore: true,
		},
		{
			responseData: responseData{
				Status: 301,
				Header: http.Header{
					"Content-Type": []string{"text/html"},
					"Location":     []string{"http://www.example.com/"},
					"X-Served-By":  []string{"test case"},
				},
				Data: [][]byte{
					[]byte("You are being redirected."),
				},
			},
			ShouldStore: false,
		},
		{
			responseData: responseData{
				Status: 200,
				Header: http.Header{
					"Content-Type": []string{"text/html"},
					"X-Served-By":  []string{"test\ncase", "values"},
				},
				Data: [][]byte{
					[]byte("Test response body\x00"),
					[]byte("test."),
				},
			},
			ShouldStore: true,
		},
	}

	for index, scenario := range scenarios {
		cache := NewMemoryCache()
		cacheKey := fmt.Sprintf("cache_key_%d", index)
		responseWriter := newDummyResponseWriter()
		cacheWriter := NewResponseCacheWriter(cache, cacheKey, responseWriter)

		// write the scenario to the cache adapter
		CopyHeader(cacheWriter.Header(), scenario.Header)
		cacheWriter.WriteHeader(scenario.Status)
		expectedData := make([]byte, 0)
		for _, datum := range scenario.Data {
			cacheWriter.Write(datum)
			expectedData = append(expectedData, datum...)
		}
		cacheWriter.Close()

		// check it was all forwarded through to the real HTTP response writer
		if !reflect.DeepEqual(responseWriter.response, scenario.responseData) {
			t.Error("response was not writer through correctly")
		}

		// check it was stored or not stored in the cache as expected
		entry, present := cache.Get(cacheKey)
		if !scenario.ShouldStore {
			if present {
				t.Error("response was written to cache when it should not have been")
			}
		} else if !present {
			t.Error("response was not written to cache")
		} else {
			// check it was stored in the cache correctly
			if entry.Status != scenario.Status {
				t.Error("cache stored wrong status")
			}
			if !reflect.DeepEqual(entry.Header, scenario.Header) {
				t.Error("Header was not restored from the cache")
			}
			if !reflect.DeepEqual(entry.Body, expectedData) {
				t.Error("Data was not restored from the cache")
			}
		}
	}
}
