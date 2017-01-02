package response_cache

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
		cache:    cache,
		key:      key,
		tempfile: tempfile,
	}
	responseWriter := NewResponseCacheWriter(&cacheWriter, realWriter)
	if err := miss(responseWriter); err != nil {
		cacheWriter.Abort()
		return err
	}
	if err := cacheWriter.Finish(); err != nil {
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
	cache    diskCache
	key      string
	tempfile *os.File
}

func (writer *diskCacheWriter) WriteHeader(status int, header http.Header) error {
	if writer.Aborted() {
		return nil
	}

	diskCacheHeader := DiskCacheHeader{
		Version: 1,
		Status:  status,
		Header:  header,
	}

	streamer := msgp.NewWriter(writer.tempfile)

	if err := diskCacheHeader.EncodeMsg(streamer); err != nil {
		return err
	}

	if err := streamer.Flush(); err != nil {
		return err
	}

	return nil
}

func (writer *diskCacheWriter) Write(data []byte) (int, error) {
	if writer.Aborted() {
		return 0, nil
	}

	return writer.tempfile.Write(data)
}

func (writer *diskCacheWriter) Finish() error {
	if writer.Aborted() {
		return nil
	}

	if err := writer.tempfile.Close(); err != nil {
		return err
	}

	if err := os.Link(writer.tempfile.Name(), writer.cache.cacheEntryPath(writer.key)); err != nil {
		return err
	}

	if err := os.Remove(writer.tempfile.Name()); err != nil {
		return err
	}

	return nil
}

func (writer *diskCacheWriter) Abort() error {
	if err := writer.tempfile.Close(); err != nil {
		return err
	}

	if err := os.Remove(writer.tempfile.Name()); err != nil {
		return err
	}

	writer.tempfile = nil

	return nil
}

func (writer *diskCacheWriter) Aborted() bool {
	return (writer.tempfile == nil)
}
