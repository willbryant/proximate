package response_cache

import "io"
import "net/http"

type Entry interface {
	Status() int
	Header() http.Header
	Body() io.ReadCloser
	WriteTo(w http.ResponseWriter)
}

func WriteEntryTo(entry Entry, w http.ResponseWriter) {
	body := entry.Body()
	defer body.Close()

	CopyHeader(w.Header(), entry.Header())
	w.WriteHeader(entry.Status())
	io.Copy(w, body)
}