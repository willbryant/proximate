package response_cache

import "fmt"
import "github.com/tinylib/msgp/msgp"
import "net/http"
import "os"

type diskCacheWriter struct {
	tempfile *os.File
	header http.Header
	progress *progressTracker
}

func (writer *diskCacheWriter) Header() http.Header {
	return writer.header
}

func (writer *diskCacheWriter) WriteHeader(status int) {
	if writer.Aborted() {
		return
	}

	if !CacheableResponse(status, writer.Header()) {
		writer.Uncacheable()
		return
	}

	diskCacheHeader := DiskCacheHeader{
		Version: 1,
		Status:  status,
		Header:  writer.header,
	}

	streamer := msgp.NewWriter(writer.tempfile)

	if err := diskCacheHeader.EncodeMsg(streamer); err != nil {
		writer.Abort(err)
		return
	}

	if err := streamer.Flush(); err != nil {
		writer.Abort(err)
		return
	}

	writer.progress.Reading()
}

func (writer *diskCacheWriter) Write(data []byte) (int, error) {
	n, err := writer.tempfile.Write(data)
	if err != nil {
		writer.Abort(err)
	}

	writer.progress.Wrote(int64(n))

	return n, err
}

func (writer *diskCacheWriter) Finish(path string) error {
	if writer.Aborted() {
		return nil
	}

	if err := writer.tempfile.Close(); err != nil {
		return err
	}

	if err := os.Rename(writer.tempfile.Name(), path); err != nil {
		return err
	}

	writer.progress.Success()

	return nil
}

func (writer *diskCacheWriter) Abort(reason error) error {
	if reason != nil {
		fmt.Fprintf(os.Stderr, "error writing to cache: %s\n", reason)
	}

	if err := writer.tempfile.Close(); err != nil {
		return err
	}

	if err := os.Remove(writer.tempfile.Name()); err != nil {
		return err
	}

	writer.tempfile = nil

	writer.progress.Failure(reason)

	return nil
}

func (writer *diskCacheWriter) Uncacheable() error {
	return writer.Abort(Uncacheable)
}

func (writer *diskCacheWriter) Aborted() bool {
	return (writer.tempfile == nil)
}
