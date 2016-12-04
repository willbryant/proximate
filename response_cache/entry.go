package response_cache

import "io"
import "net/http"

type Entry interface {
	Status() int
	Header() http.Header
	Body() io.Reader
	WriteTo(w http.ResponseWriter)
}

func WriteEntryTo(entry Entry, w http.ResponseWriter) {
	CopyHeader(w.Header(), entry.Header())
	w.WriteHeader(entry.Status())
	io.Copy(w, entry.Body())
}