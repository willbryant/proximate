package response_cache

import "testing"
import "bytes"
import "io"
import "fmt"
import "math/rand"
import "net/http"
import "os"
import "reflect"
import "strconv"
import "syscall"

var cache ResponseCache = NewDiskCache("test/cache")

type multiByteReader struct {
	data      [][]byte
	bodyError error
	offset int
}

func (reader *multiByteReader) Read(p []byte) (n int, err error) {
	if len(reader.data) == 0 {
		if reader.bodyError != nil {
			return 0, reader.bodyError
		}
		return 0, io.EOF
	}
	datum := reader.data[0]
	n, err = bytes.NewReader(datum[reader.offset:len(datum)]).Read(p)
	reader.offset += n
	if reader.offset == len(datum) {
		reader.data = reader.data[1:]
		reader.offset = 0
	}
	return n, err
}

func (reader *multiByteReader) Close() error {
	return nil
}

type scenarioData struct {
	StatusCode        int
	Header            http.Header
	Data              [][]byte // more than one will result in multiple short reads, simulating network traffic
	BodyError         error
	ShouldCache       bool
	ExpectedError     error
	ExpectedErrorLogs []string
	ExpectMismatchedData bool
}

func (scenarioData scenarioData) copyResponse() *http.Response {
	var contentLength int64
	if cl, ok := scenarioData.Header["Content-Length"]; ok {
		contentLength, _ = strconv.ParseInt(cl[0], 10, 64)
	} else {
		contentLength = -1
	}
	return &http.Response{
		StatusCode:    scenarioData.StatusCode,
		Header:        scenarioData.Header,
		ContentLength: int64(contentLength),
		Body:          &multiByteReader{data: scenarioData.Data, bodyError: scenarioData.BodyError},
	}
}

func testResponse(t *testing.T, response *http.Response, expectedStatus int, expectedHeader http.Header, expectedData []byte, expectMismatchedData bool) {
	if response.StatusCode != expectedStatus {
		t.Fatal(fmt.Sprintf("cache stored wrong status (%d instead of %d)", response.StatusCode, expectedStatus))
	}
	if !reflect.DeepEqual(response.Header, expectedHeader) {
		t.Fatal("Header was not restored from the cache")
	}
	var scenarioData bytes.Buffer
	io.Copy(&scenarioData, response.Body)
	response.Body.Close()
	if !reflect.DeepEqual(scenarioData.Bytes(), expectedData) && !expectMismatchedData {
		t.Fatal("Data was not restored from the cache accurately")
	}
}

func testScenario(t *testing.T, scenario scenarioData) {
	cache.Clear()
	cacheKey := fmt.Sprintf("some_cache_key")

	originalLogCacheError := logCacheError
	var errorLogs []string
	logCacheError = func(format string, a ...interface{}) { errorLogs = append(errorLogs, fmt.Sprintf(format, a...)) }

	// write the scenario to the cache adapter
	forwarded := false
	response, err := cache.Get(cacheKey, func() (*http.Response, error) {
		forwarded = true
		return scenario.copyResponse(), nil
	})
	if !forwarded {
		t.Fatal("request callback wasn't forwarded")
	}
	if err != nil && err != Uncacheable {
		t.Fatal(fmt.Sprintf("result wasn't nil or Uncacheable, was %s", err))
	}

	// check it was all forwarded through to the HTTP response object
	expectedData := make([]byte, 0)
	for _, datum := range scenario.Data {
		expectedData = append(expectedData, datum...)
	}
	testResponse(t, response, scenario.StatusCode, scenario.Header, expectedData, scenario.ExpectMismatchedData)

	// check it was stored or not stored in the cache as expected
	forwarded = false
	response, err = cache.Get(cacheKey, func() (*http.Response, error) {
		forwarded = true
		return scenario.copyResponse(), nil
	})
	// check it was replayed from the cache correctly - or the miss function results copied through correctly
	testResponse(t, response, scenario.StatusCode, scenario.Header, expectedData, scenario.ExpectMismatchedData)
	if scenario.ShouldCache {
		if forwarded {
			t.Fatal("response was forwarded again when it should have been retrieved from cache")
		}
	} else {
		if !forwarded {
			t.Fatal("response retrieved from cache when it should have forwarded")
		}
	}
	if err != scenario.ExpectedError {
		t.Fatal(fmt.Sprintf("expected error %s, got %s", scenario.ExpectedError, err))
	}
	if !reflect.DeepEqual(scenario.ExpectedErrorLogs, errorLogs) {
		t.Fatal(fmt.Sprintf("expected error logs %v, got %v", scenario.ExpectedErrorLogs, errorLogs))
	}

	logCacheError = originalLogCacheError
}

func TestCacheable200(t *testing.T) {
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"X-Served-By":    []string{"test case"},
			"Content-Length": []string{"19"},
		},
		Data: [][]byte{
			[]byte("Test response body."),
		},
		ShouldCache: true,
	})

	// same but chunked
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"text/html"},
			"X-Served-By":  []string{"test case"},
		},
		Data: [][]byte{
			[]byte("Test response body."),
		},
		ShouldCache: true,
	})
}

func TestCacheable200WithMultipleReads(t *testing.T) {
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"X-Served-By":    []string{"test\ncase", "values"},
			"Content-Length": []string{"32"},
		},
		Data: [][]byte{
			[]byte("Test response body"),
			[]byte("more\x00data"),
			[]byte("test."),
		},
		ShouldCache: true,
	})

	// same but chunked
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"text/html"},
			"X-Served-By":  []string{"test\ncase", "values"},
		},
		Data: [][]byte{
			[]byte("Test response body"),
			[]byte("more\x00data"),
			[]byte("test."),
		},
		ShouldCache: true,
	})
}

func TestUncacheable301(t *testing.T) {
	testScenario(t, scenarioData{
		StatusCode: 301,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"Location":       []string{"http://www.example.com/"},
			"X-Served-By":    []string{"test case"},
			"Content-Length": []string{"25"},
		},
		Data: [][]byte{
			[]byte("You are being redirected."),
		},
		ShouldCache:   false,
		ExpectedError: Uncacheable,
	})

	// same but chunked
	testScenario(t, scenarioData{
		StatusCode: 301,
		Header: http.Header{
			"Content-Type": []string{"text/html"},
			"Location":     []string{"http://www.example.com/"},
			"X-Served-By":  []string{"test case"},
		},
		Data: [][]byte{
			[]byte("You are being redirected."),
		},
		ShouldCache:   false,
		ExpectedError: Uncacheable,
	})
}

func TestMediumFile(t *testing.T) {
	var buffer bytes.Buffer
	io.CopyN(&buffer, rand.New(rand.NewSource(421337)), 800000)
	data := buffer.Bytes()
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"application/octet-stream"},
			"Content-Length": []string{"800000"},
		},
		Data: [][]byte{
			data,
		},
		ShouldCache: true,
	})

	// same but chunked
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"application/octet-stream"},
		},
		Data: [][]byte{
			data,
		},
		ShouldCache: true,
	})
}

func TestFileCreateFailure(t *testing.T) {
	originalOsCreate := osCreate
	osCreate = func(path string) (File, error) { return nil, os.ErrPermission }
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"X-Served-By":    []string{"test case"},
			"Content-Length": []string{"19"},
		},
		Data: [][]byte{
			[]byte("Test response body."),
		},
		ShouldCache: false,

		// the file open error handler should handle and log the error, and then proceed the same as a cache miss - without returning an error
		ExpectedError: nil,

		// we should see this log twice because testScenario does the request twice
		ExpectedErrorLogs: []string{
			"Error opening cache path test/cache/some_cache_key for writing: permission denied\n",
			"Error opening cache path test/cache/some_cache_key for writing: permission denied\n",
		},
	})
	osCreate = originalOsCreate
}

func TestFileOpenFailure(t *testing.T) {
	originalOsOpen := osOpen
	osOpen = func(path string) (File, error) { return nil, os.ErrPermission }
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"X-Served-By":    []string{"test case"},
			"Content-Length": []string{"19"},
		},
		Data: [][]byte{
			[]byte("Test response body."),
		},
		ShouldCache: false,

		// the file open error handler should handle and log the error, and then proceed the same as a cache miss - without returning an error
		ExpectedError: nil,

		// we should see this log twice because testScenario does the request twice
		ExpectedErrorLogs: []string{
			"Error opening cache path test/cache/some_cache_key for reading: permission denied\n",
			"Error opening cache path test/cache/some_cache_key for reading: permission denied\n",
		},
	})
	osOpen = originalOsOpen
}

type SizeLimitedFile struct {
	path string
	file File
	max int
	size int
}

func (f *SizeLimitedFile) Write(p []byte) (n int, err error) {
	if f.size + len(p) > f.max {
		n = f.max - f.size
		if n > 0 {
			f.file.Write(p[:n])
		}
		f.size = f.max
		return n, &os.PathError{Path: f.path, Err: syscall.ENOSPC}
	} else {
		f.size += len(p)
		return f.file.Write(p)
	}
}

func (f *SizeLimitedFile) Read(p []byte) (int, error) {
	return f.file.Read(p)
}

func (f *SizeLimitedFile) ReadAt(p []byte, off int64) (int, error) {
	return f.file.ReadAt(p, off)
}

func (f *SizeLimitedFile) Sync() error {
	return f.file.Sync()
}

func (f *SizeLimitedFile) Close() error {
	return f.file.Close()
}

func TestFileWriteFirstFailure(t *testing.T) {
	originalOsCreate := osCreate
	maxSize := 0
	osCreate = func(path string) (File, error) {
		osCreate = originalOsCreate
		file, err := osCreate(path)
		return &SizeLimitedFile{path, file, maxSize, 0}, err
	}
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"X-Served-By":    []string{"test case"},
			"Content-Length": []string{"19"},
		},
		Data: [][]byte{
			[]byte("Test response body."),
		},
		ShouldCache: false,

		// the file open error handler should handle and log the error, and then proceed the same as a cache miss - without returning an error
		ExpectedError: nil,

		// we should see this log twice because testScenario does the request twice; the second time, the error should be cleared
		ExpectedErrorLogs: []string{
			"Error writing to cache path test/cache/some_cache_key:  test/cache/some_cache_key.temp: no space left on device\n",
		},
	})
	osCreate = originalOsCreate
}

func TestFileWriteHeaderFailure(t *testing.T) {
	originalOsCreate := osCreate
	maxSize := 1024
	osCreate = func(path string) (File, error) {
		osCreate = originalOsCreate
		file, err := osCreate(path)
		return &SizeLimitedFile{path, file, maxSize, 0}, err
	}
	var buffer bytes.Buffer
	io.CopyN(&buffer, rand.New(rand.NewSource(421337)), 2048)
	value := buffer.Bytes()
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"X-Long-Header":  []string{string(value)},
			"Content-Length": []string{"19"},
		},
		Data: [][]byte{
			[]byte("Test response body."),
		},
		ShouldCache: false,

		// the file open error handler should handle and log the error, and then proceed the same as a cache miss - without returning an error
		ExpectedError: nil,

		// we should see this log twice because testScenario does the request twice; the second time, the error should be cleared
		ExpectedErrorLogs: []string{
			"Error writing to cache path test/cache/some_cache_key:  test/cache/some_cache_key.temp: no space left on device\n",
		},
	})
	osCreate = originalOsCreate
}

func TestFileWriteDuringBodyFailure(t *testing.T) {
	originalOsCreate := osCreate
	maxSize := 4096
	osCreate = func(path string) (File, error) {
		osCreate = originalOsCreate
		file, err := osCreate(path)
		return &SizeLimitedFile{path, file, maxSize, 0}, err
	}
	var buffer bytes.Buffer
	io.CopyN(&buffer, rand.New(rand.NewSource(421337)), 2048)
	value := buffer.Bytes()
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"Content-Length": []string{"8192"}, // TODO: remove
		},
		Data: [][]byte{
			value,
			value,
			value,
			value,
		},
		ExpectMismatchedData: true,
		ShouldCache: false,

		// the file open error handler should handle and log the error, and then proceed the same as a cache miss - without returning an error
		ExpectedError: nil,

		// we should see this log twice because testScenario does the request twice; the second time, the error should be cleared
		ExpectedErrorLogs: []string{
			"Error copying response to cache path test/cache/some_cache_key:  test/cache/some_cache_key.temp: no space left on device\n",
		},
	})
	osCreate = originalOsCreate
}

func TestTruncatedBody(t *testing.T) {
	testScenario(t, scenarioData{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"text/html"},
			"X-Served-By":    []string{"test case"},
			"Content-Length": []string{"19"},
		},
		Data: [][]byte{
			[]byte("Test res"),
		},
		ShouldCache: false,
	})
}
