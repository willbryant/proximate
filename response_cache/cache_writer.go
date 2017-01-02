package response_cache

import "io"
import "fmt"
import "os"
import "net/http"

// interface provided by a particular cache store to the ResponseCacheWriter
type cacheStoreWriter interface {
	WriteHeader(status int, header http.Header) error
	io.Writer
	Finish() error
	Abort() error
}

// intercepts an HTTP response and as well as sending it to the original responseWriter (which
// belongs to the real client), stores the response in the cache (if it is a 200 OK response).
type ResponseCacheWriter struct {
	cacheEntry cacheStoreWriter
	original http.ResponseWriter
}

func NewResponseCacheWriter(cacheEntry cacheStoreWriter, original http.ResponseWriter) *ResponseCacheWriter {
	return &ResponseCacheWriter{
		cacheEntry: cacheEntry,
		original: original,
	}
}

func (w *ResponseCacheWriter) Header() http.Header {
	return w.original.Header()
}

func (w *ResponseCacheWriter) WriteHeader(status int) {
	if status == http.StatusOK {
		// now that we know we're keeping the response, we want to copy the header to the Entry object
		// we could of course have returned that from Header() above instead, but then we'd have to do
		// a header copy even in the !StatusOK case.
		err := w.cacheEntry.WriteHeader(status, w.original.Header())
		if err != nil {
			fmt.Fprintf(os.Stderr, "couldn't write cache headers, error %s\n", err)
			w.cacheEntry.Abort()
		}
	} else {
		// this is fine, but we don't cache non-OK responses, so abort the cache write
		w.cacheEntry.Abort()
	}

	w.original.WriteHeader(status)
}

func (w *ResponseCacheWriter) Write(data []byte) (int, error) {
	// if we're actually caching this response, keep a copy of the data
	_, err := w.cacheEntry.Write(data)
	if err != nil {
		// we can continue sending the response to the client, but we can't store to the cache
		fmt.Fprintf(os.Stderr, "couldn't write cache data, error %s\n", err)
		w.cacheEntry.Abort()
	}

	len, err := w.original.Write(data)
	return len, err
}
