package response_cache

import "io"
import "net/http"

type CacheWriter interface {
	WriteHeader(status int, header http.Header) error
	io.Writer
	Finish() error
	Abort() error
}

type ResponseCache interface {
	Get(key string) (Entry, error)
	BeginWrite(key string) (CacheWriter, error)
}
