package response_cache

import "io"

type readerAndCloser struct {
	io.Reader
	io.Closer
}

func ReaderAndCloser(r io.Reader, cl io.Closer) io.ReadCloser {
	return readerAndCloser{r, cl}
}
