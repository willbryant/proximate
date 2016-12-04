package response_cache

import "io"
import "net/http"

type ReadSizeSeeker interface {
	io.ReadSeeker
	Size() int64
}

type Entry struct {
	Status int
	Header http.Header
	Body ReadSizeSeeker
}

func NewCacheEntry() Entry {
	return Entry {
		Header: make(http.Header),
	}
}

func (entry Entry) WriteTo(w http.ResponseWriter) {
	CopyHeader(w.Header(), entry.Header)
	w.WriteHeader(entry.Status)
	io.Copy(w, entry.Body)
}