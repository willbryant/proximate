package response_cache

import "io"
import "net/http"

type CacheBodyWriter interface {
	io.Writer
	Finish()
	Abort()
}

type ResponseCache interface {
	Get(key string) (Entry, error)
	BeginWrite(key string, status int, header http.Header) (CacheBodyWriter, error)
}
