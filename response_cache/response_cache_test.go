package response_cache

import "testing"
import "bytes"
import "io"
import "fmt"
import "net/http"
import "os"
import "reflect"

type responseData struct {
	StatusCode int
	Header     http.Header
	Data       [][]byte // more than one will result in multiple short reads, simulating network traffic
}

type multiByteReader struct {
	data [][]byte
}

func (reader *multiByteReader) Read(p []byte) (n int, err error) {
	if len(reader.data) == 0 {
		return 0, io.EOF
	}
	datum := reader.data[0]
	reader.data = reader.data[1:]
	return bytes.NewReader(datum).Read(p)
}

func (reader *multiByteReader) Close() error {
	return nil
}

func (responseData responseData) copyResponse(contentLengthKnown bool) *http.Response {
	contentLength := -1
	if contentLengthKnown {
		contentLength = 0
		for _, b := range responseData.Data {
			contentLength += len(b)
		}
	}
	return &http.Response{
		StatusCode:    responseData.StatusCode,
		Header:        responseData.Header,
		ContentLength: int64(contentLength),
		Body:          &multiByteReader{responseData.Data},
	}
}

func testResponse(t *testing.T, response *http.Response, expectedStatus int, expectedHeader http.Header, expectedData []byte) {
	if response.StatusCode != expectedStatus {
		t.Error(fmt.Sprintf("cache stored wrong status (%d instead of %d)", response.StatusCode, expectedStatus))
	}
	if !reflect.DeepEqual(response.Header, expectedHeader) {
		t.Error("Header was not restored from the cache")
	}
	var responseData bytes.Buffer
	io.Copy(&responseData, response.Body)
	response.Body.Close()
	if !reflect.DeepEqual(responseData.Bytes(), expectedData) {
		t.Error("Data was not restored from the cache accurately")
	}
}

type cacheWriterTestScenario struct {
	responseData
	ShouldStore bool
}

func testScenario(t *testing.T, cache ResponseCache, index int, scenario cacheWriterTestScenario, contentLengthKnown bool) {
	cache.Clear()
	cacheKey := fmt.Sprintf("cache_key_%d", index)

	// write the scenario to the cache adapter
	called := false
	response, err := cache.Get(cacheKey, func() (*http.Response, error) {
		called = true
		return scenario.copyResponse(contentLengthKnown), nil
	})
	if !called {
		t.Error("request callback wasn't called")
	}
	if !os.IsNotExist(err) && err != Uncacheable {
		t.Error(fmt.Sprintf("result wasn't an IsNotExist or Uncacheable, was %s", err))
	}

	// check it was all forwarded through to the HTTP response object
	expectedData := make([]byte, 0)
	for _, datum := range scenario.Data {
		expectedData = append(expectedData, datum...)
	}
	testResponse(t, response, scenario.responseData.StatusCode, scenario.responseData.Header, expectedData)

	// check it was stored or not stored in the cache as expected
	var missed = false
	response, err = cache.Get(cacheKey, func() (*http.Response, error) {
		missed = true
		return scenario.copyResponse(contentLengthKnown), nil
	})
	if !scenario.ShouldStore {
		if err != Uncacheable {
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
	}
	// check it was replayed from the cache correctly - or the miss function results copied through correctly
	testResponse(t, response, scenario.responseData.StatusCode, scenario.responseData.Header, expectedData)
}

func TestCacheWriter(t *testing.T) {
	scenarios := []cacheWriterTestScenario{
		{
			responseData: responseData{
				StatusCode: 200,
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
				StatusCode: 301,
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
				StatusCode: 200,
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

	cache := NewDiskCache("test/cache")

	for index, scenario := range scenarios {
		fmt.Fprintf(os.Stderr, "--- scenario %d\n", index)
		testScenario(t, cache, index, scenario, false)
		testScenario(t, cache, index, scenario, true)
	}
}
