package response_cache

import "fmt"
import "os"
import "net/http"

// intercepts an HTTP response and as well as sending it to the original writer (which belongs to the
// real client), stores the response in the cache (if it is a 200 OK response).
type ResponseCacheWriter struct {
	cache    ResponseCache
	key      string
	body     CacheBodyWriter
	original http.ResponseWriter
}

func NewResponseCacheWriter(cache ResponseCache, key string, original http.ResponseWriter) *ResponseCacheWriter {
	return &ResponseCacheWriter{
		cache:    cache,
		key:      key,
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
		body, err := writer.cache.BeginWrite(writer.key, status, writer.original.Header())
		if err != nil {
			fmt.Fprintf(os.Stderr, "couldn't start cache store for request hash %s, error %s\n", writer.key, err)
		} else {
			writer.body = body
		}
	}

	writer.original.WriteHeader(status)
}

func (writer *ResponseCacheWriter) Write(data []byte) (int, error) {
	// if we're actually caching this response, keep a copy of the data
	if writer.body != nil {
		_, err := writer.body.Write(data)
		if err != nil {
			// we can continue sending the response to the client, but we can't store to the cache
			fmt.Fprintf(os.Stderr, "couldn't write cache data for request hash %s, error %s\n", writer.key, err)
			writer.body.Abort()
			writer.body = nil
		}
	}

	len, err := writer.original.Write(data)
	return len, err
}

func (writer *ResponseCacheWriter) Finish() error {
	if writer.body != nil {
		return writer.body.Finish()
	}
	return nil
}

func (writer *ResponseCacheWriter) Abort() error {
	if writer.body != nil {
		return writer.body.Abort()
	}
	return nil
}
