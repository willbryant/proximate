package response_cache

import "errors"
import "net/http"

var Uncacheable = errors.New("Uncacheable")

type ResponseCache interface {
	Clear() error
	Get(key string, miss func() (*http.Response, error)) (*http.Response, error)
}

func CacheableResponse(status int, header http.Header) bool {
	return (status == http.StatusOK)
}
