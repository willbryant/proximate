package response_cache

import "io"
import "io/ioutil"
import "github.com/tinylib/msgp/msgp"
import "net/http"
import "os"

type DiskCacheEntry struct {
	header DiskCacheHeader
	body io.Reader
	file io.Closer
}

func (entry DiskCacheEntry) Status() int {
	return entry.header.Status
}

func (entry DiskCacheEntry) Header() http.Header {
	return entry.header.Header
}

func (entry DiskCacheEntry) Body() io.Reader {
	return entry.body
}

func (entry DiskCacheEntry) Close() {
	if entry.file != nil {
		entry.file.Close()
	}
}

func (entry DiskCacheEntry) WriteTo(w http.ResponseWriter) {
	WriteEntryTo(entry, w)
}

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

func (cache diskCache) Get(key string) (Entry, error) {
	file, err := os.Open(cache.cacheEntryPath(key))
	if err != nil {
		return nil, err
	}

	streamer := msgp.NewReader(file)

	var diskCacheHeader DiskCacheHeader
	diskCacheHeader.DecodeMsg(streamer)

	return DiskCacheEntry{
		header: diskCacheHeader,
		body: streamer,
		file: file,
	}, nil
}

type diskCacheBodyWriter struct {
	cache    diskCache
	key      string
	tempfile *os.File
}

func (writer diskCacheBodyWriter) Write(data []byte) (int, error) {
	return writer.tempfile.Write(data)
}

func (writer diskCacheBodyWriter) Finish() error {
	err := writer.tempfile.Close()
	if err != nil {
		return err
	}

	err = os.Link(writer.tempfile.Name(), writer.cache.cacheEntryPath(writer.key))
	if err != nil {
		return err
	}

	os.Remove(writer.tempfile.Name())
	if err != nil {
		return err
	}

	return nil
}

func (writer diskCacheBodyWriter) Abort() error {
	err := writer.tempfile.Close()
	if err != nil {
		return err
	}

	os.Remove(writer.tempfile.Name())
	if err != nil {
		return err
	}

	return nil
}

func (cache diskCache) BeginWrite(key string, status int, header http.Header) (CacheBodyWriter, error) {
	diskCacheHeader := DiskCacheHeader{
		Version: 1,
		Status: status,
		Header: header,
	}

	tempfile, err := ioutil.TempFile(cache.cacheDirectory, "_temp")
	if err != nil {
		return nil, err
	}

	streamer := msgp.NewWriter(tempfile)

	err = diskCacheHeader.EncodeMsg(streamer)
	if err != nil {
		return nil, err
	}

	err = streamer.Flush()
	if err != nil {
		return nil, err
	}

	return diskCacheBodyWriter{
		cache:    cache,
		key:      key,
		tempfile: tempfile,
	}, nil
}
