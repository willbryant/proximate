package response_cache

import "net/http"

// intercepts an HTTP response and as well as sending it to the original writer (which belongs to the
// real client), stores the response in the cache (if it is a 200 OK response).
type ResponseCacheWriter struct {
	cache ResponseCache
	key string
	body CacheBodyWriter
	original http.ResponseWriter
}

func NewResponseCacheWriter(cache ResponseCache, key string, original http.ResponseWriter) *ResponseCacheWriter {
	return &ResponseCacheWriter {
		cache: cache,
		key: key,
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
		var err error
		writer.body, err = writer.cache.BeginWrite(writer.key, status, writer.original.Header())
		if err != nil {
			panic(err)
		}
	}

	writer.original.WriteHeader(status)
}

func (writer *ResponseCacheWriter) Write(data []byte) (int, error) {
	// if we're actually caching this response, keep a copy of the data
	if writer.body != nil {
		writer.body.Write(data)
	}

	len, err := writer.original.Write(data)
	return len, err
}

func (writer *ResponseCacheWriter) Finish() {
	if writer.body != nil {
		writer.body.Finish()
	}
}

func (writer *ResponseCacheWriter) Abort() {
	if writer.body != nil {
		writer.body.Abort()
	}
}
