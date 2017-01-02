package response_cache

import "fmt"
import "io"
import "io/ioutil"
import "github.com/tinylib/msgp/msgp"
import "net/http"
import "os"

type diskCache struct {
	cacheDirectory string
}

func NewDiskCache(cacheDirectory string) ResponseCache {
	return diskCache{
		cacheDirectory: cacheDirectory,
	}
}

func (cache diskCache) cacheEntryPath(key string) string {
	return cache.cacheDirectory + "/" + key
}

func (cache diskCache) Get(key string, realWriter http.ResponseWriter, miss func(writer http.ResponseWriter) error) error {
	file, err := os.Open(cache.cacheEntryPath(key))

	if err == nil {
		return cache.ServeCacheHit(realWriter, file)
	}

	if !os.IsNotExist(err) {
		return err
	}

	// cache miss
	tempfile, err := ioutil.TempFile(cache.cacheDirectory, "_temp")
	if err != nil {
		miss(realWriter)
		return err
	}

	cacheWriter := diskCacheWriter{
		tempfile: tempfile,
		header: make(http.Header),
		realWriter: realWriter,
	}
	if err := miss(&cacheWriter); err != nil {
		cacheWriter.Abort(nil)
		return err
	}
	if err := cacheWriter.Finish(cache.cacheEntryPath(key)); err != nil {
		return err
	}
	return os.ErrNotExist // indicates a cache miss
}

func (cache diskCache) ServeCacheHit(w http.ResponseWriter, file *os.File) error {
	defer file.Close()
	streamer := msgp.NewReader(file)

	var diskCacheHeader DiskCacheHeader
	if err := diskCacheHeader.DecodeMsg(streamer); err != nil {
		return err;
	}

	CopyHeader(w.Header(), diskCacheHeader.Header)
	w.WriteHeader(diskCacheHeader.Status)
	_, err := io.Copy(w, streamer)
	return err
}

type diskCacheWriter struct {
	tempfile *os.File
	header http.Header
	realWriter http.ResponseWriter
}

func (writer *diskCacheWriter) Header() http.Header {
	return writer.header
}

func (writer *diskCacheWriter) WriteHeader(status int) {
	CopyHeader(writer.realWriter.Header(), writer.Header())
	writer.realWriter.WriteHeader(status)

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
}

func (writer *diskCacheWriter) Write(data []byte) (int, error) {
	n, err := writer.realWriter.Write(data)

	if writer.Aborted() {
		return n, err
	}

	n, err = writer.tempfile.Write(data)
	if err != nil {
		writer.Abort(err)
	}
	return n, err
}

func (writer *diskCacheWriter) Finish(path string) error {
	if writer.Aborted() {
		return nil
	}

	if err := writer.tempfile.Close(); err != nil {
		return err
	}

	if err := os.Link(writer.tempfile.Name(), path); err != nil {
		return err
	}

	if err := os.Remove(writer.tempfile.Name()); err != nil {
		return err
	}

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

	return nil
}

func (writer *diskCacheWriter) Uncacheable() error {
	return writer.Abort(nil)
}

func (writer *diskCacheWriter) Aborted() bool {
	return (writer.tempfile == nil)
}
