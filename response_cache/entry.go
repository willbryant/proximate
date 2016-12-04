package response_cache

import "io"
import "net/http"

type ReadSizer interface {
	io.Reader
	Size() int64
}

type Entry interface {
	Status() int
	Header() http.Header
	Body() ReadSizer
	WriteTo(w http.ResponseWriter)
}

func WriteEntryTo(entry Entry, w http.ResponseWriter) {
	CopyHeader(w.Header(), entry.Header())
	w.WriteHeader(entry.Status())
	io.Copy(w, entry.Body())
}