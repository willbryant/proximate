package response_cache

import "testing"
import "fmt"
import "net/http"
import "os"
import "reflect"

type responseData struct {
	Status int
	Header http.Header
	Data   [][]byte
}

func (responseData responseData) copyResponseTo(writer http.ResponseWriter) error {
	CopyHeader(writer.Header(), responseData.Header)
	writer.WriteHeader(responseData.Status)
	for _, datum := range responseData.Data {
		if _, err := writer.Write(datum); err != nil {
			return err
		}
	}
	return nil
}

type dummyResponseWriter struct {
	response responseData
	t *testing.T
}

func (responseWriter *dummyResponseWriter) Header() http.Header {
	return responseWriter.response.Header
}

func (responseWriter *dummyResponseWriter) WriteHeader(status int) {
	if responseWriter.response.Status != 0 {
		responseWriter.t.Error("header was already written")
	}
	responseWriter.response.Status = status
}

func (responseWriter *dummyResponseWriter) Write(data []byte) (int, error) {
	if responseWriter.response.Status == 0 {
		responseWriter.t.Error("header has not been written yet")
	}
	responseWriter.response.Data = append(responseWriter.response.Data, data)
	return len(data), nil
}

func newDummyResponseWriter(t *testing.T) *dummyResponseWriter {
	responseWriter := &dummyResponseWriter{
		response: responseData{
			Header: make(http.Header),
		},
		t: t,
	}
	return responseWriter
}

func testResponse(t *testing.T, response responseData, expectedStatus int, expectedHeader http.Header, expectedData []byte) {
	if response.Status != expectedStatus {
		t.Error("cache stored wrong status")
	}
	if !reflect.DeepEqual(response.Header, expectedHeader) {
		t.Error("Header was not restored from the cache")
	}
	responseData := make([]byte, 0)
	for _, datum := range response.Data {
		responseData = append(responseData, datum...)
	}
	if !reflect.DeepEqual(responseData, expectedData) {
		t.Error("Data was not restored from the cache accurately")
	}
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
		responseWriter := newDummyResponseWriter(t)

		// write the scenario to the cache adapter
		err := cache.Get(cacheKey, responseWriter, func(writer http.ResponseWriter) error {
			scenario.copyResponseTo(writer)
			return nil
		})
		if !os.IsNotExist(err) {
			t.Error("request callback wasn't called")
		}

		// check it was all forwarded through to the real HTTP response writer
		expectedData := make([]byte, 0)
		for _, datum := range scenario.Data {
			expectedData = append(expectedData, datum...)
		}
		testResponse(t, responseWriter.response, scenario.responseData.Status, scenario.responseData.Header, expectedData)

		// check it was stored or not stored in the cache as expected
		responseWriter = newDummyResponseWriter(t)
		var missed = false
		err = cache.Get(cacheKey, responseWriter, func(writer http.ResponseWriter) error {
			missed = true
			return nil
		})
		if !scenario.ShouldStore {
			if !os.IsNotExist(err) {
				t.Error("couldn't perform cache miss: " + err.Error())
			}
			if !missed {
				t.Error("response cached when it should not have been")
			}
		} else if err != nil {
			if os.IsNotExist(err) {
				t.Error("response was not written to cache")
			} else {
				t.Error("couldn't read response from cache: " + err.Error())
			}
		} else {
			// check it was stored in the cache correctly
			testResponse(t, responseWriter.response, scenario.responseData.Status, scenario.responseData.Header, expectedData)
		}
	}
}
