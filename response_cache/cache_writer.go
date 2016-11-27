package response_cache

import "fmt"
import "net/http"
import "os"

type ResponseCacheWriter struct {
	cache *ResponseCache
	key string
	entry Entry
	cacheable bool
	original http.ResponseWriter
}

func NewResponseCacheWriter(cache *ResponseCache, key string, original http.ResponseWriter) *ResponseCacheWriter {
	return &ResponseCacheWriter {
		cache: cache,
		key: key,
		entry: NewCacheEntry(),
		original: original,
	}
}

func (writer *ResponseCacheWriter) Header() http.Header {
	return writer.original.Header()
}

func (writer *ResponseCacheWriter) WriteHeader(status int) {
	if status == http.StatusOK {
		// now that we know we're keeping the response, we want to copy the header to the Entry object
		// we could of course have returned that from Header() above instead, but then we'd have to do
		// a header copy even in the !StatusOK case.
		writer.cacheable = true
		CopyHeader(writer.entry.Header, writer.original.Header())
		fmt.Fprintf(os.Stdout, "could enter %s into cache, status was %d\n", writer.key, status)
	} else {
		fmt.Fprintf(os.Stdout, "not entering %s into cache, status was %d\n", writer.key, status)
	}

	writer.original.WriteHeader(status)
}

func (writer *ResponseCacheWriter) Write(data []byte) (int, error) {
	// if we're actually caching this response, keep a copy of the data
	if writer.cacheable {
		writer.entry.Body = append(writer.entry.Body, data...)
	}

	len, err := writer.original.Write(data)
	return len, err
}

func (writer *ResponseCacheWriter) Close() {
	if writer.cacheable {
		writer.cache.Set(writer.key, writer.entry)
		fmt.Fprintf(os.Stdout, "entered %s into cache, response body was %d bytes\n", writer.key, len(writer.entry.Body))
	}
}
