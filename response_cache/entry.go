package response_cache

import "net/http"

type Entry struct {
	Status int
	Header http.Header
	Body []byte
}

func NewCacheEntry() Entry {
	return Entry {
		Header: make(http.Header),
	}
}

func (entry Entry) WriteTo(w http.ResponseWriter) {
	CopyHeader(w.Header(), entry.Header)
	w.WriteHeader(entry.Status)
	w.Write(entry.Body)
}