package response_cache

import "net/http"

type Entry struct {
	Header http.Header
	Body []byte
}

func NewCacheEntry() Entry {
	return Entry {
		Header: make(http.Header),
	}
}
