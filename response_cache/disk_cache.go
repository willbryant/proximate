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

func (cache diskCache) Get(key string, w http.ResponseWriter, miss func() error) error {
	file, err := os.Open(cache.cacheEntryPath(key))
	if os.IsNotExist(err) {
		return miss()
	} else if err != nil {
		return err
	}

	defer file.Close()
	streamer := msgp.NewReader(file)

	var diskCacheHeader DiskCacheHeader
	diskCacheHeader.DecodeMsg(streamer)

	CopyHeader(w.Header(), diskCacheHeader.Header)
	w.WriteHeader(diskCacheHeader.Status)
	io.Copy(w, streamer)
	return nil
}

type diskCacheWriter struct {
	cache    diskCache
	key      string
	tempfile *os.File
}

func (writer diskCacheWriter) WriteHeader(status int, header http.Header) error {
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

func (writer diskCacheWriter) Write(data []byte) (int, error) {
	return writer.tempfile.Write(data)
}

func (writer diskCacheWriter) Finish() error {
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

func (writer diskCacheWriter) Abort() error {
	if err := writer.tempfile.Close(); err != nil {
		return err
	}

	if err := os.Remove(writer.tempfile.Name()); err != nil {
		return err
	}

	return nil
}

func (cache diskCache) BeginWrite(key string) (CacheWriter, error) {
	tempfile, err := ioutil.TempFile(cache.cacheDirectory, "_temp")
	if err != nil {
		return nil, err
	}

	return diskCacheWriter{
		cache:    cache,
		key:      key,
		tempfile: tempfile,
	}, nil
}
