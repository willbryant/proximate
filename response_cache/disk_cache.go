package response_cache

import "errors"
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
	path := cache.cacheEntryPath(key)
	file, err := os.Open(path)

	if err == nil {
		return cache.serveFromCache(realWriter, file)
	}

	if !os.IsNotExist(err) {
		return err
	}

	// cache miss; open a new tempfile to write to
	tempfile, err := ioutil.TempFile(cache.cacheDirectory, "_temp")
	if err != nil {
		miss(realWriter)
		return err
	}

	// reopen file for the benefit of this reader, so the read position is independent of the write position
	// we have to do this up here to avoid a race condition with populate() calling Finish(), which removes the tempfile
	file, err = os.Open(tempfile.Name())
	if err != nil {
		tempfile.Close()
		return errors.New(fmt.Sprintf("Couldn't reopen tempfile %s: %s", tempfile.Name(), err))
	}
	defer file.Close()

	progress := newProgressTracker()
	go cache.populate(path, tempfile, progress, miss)

	err = progress.WaitForResponse()
	if err == Uncacheable {
		return miss(realWriter)
	} else if err != nil {
		return err;
	}

	err = cache.streamFromCacheInProgress(realWriter, file, progress)

	return os.ErrNotExist // indicates a cache miss
}

func (cache diskCache) populate(path string, tempfile *os.File, progress *progressTracker, miss func(writer http.ResponseWriter) error) {
	writer := diskCacheWriter{
		tempfile: tempfile,
		header: make(http.Header),
		progress: progress,
	}
	if err := miss(&writer); err != nil {
		writer.Abort(err)
		return
	}
	if err := writer.Finish(path); err != nil {
		writer.Abort(err)
		return
	}
}

func (cache diskCache) serveHeaderFromCache(w http.ResponseWriter, streamer *msgp.Reader) error {
	var diskCacheHeader DiskCacheHeader
	if err := diskCacheHeader.DecodeMsg(streamer); err != nil {
		return err;
	}

	CopyHeader(w.Header(), diskCacheHeader.Header)
	w.WriteHeader(diskCacheHeader.Status)
	return nil
}

func (cache diskCache) serveFromCache(w http.ResponseWriter, file *os.File) error {
	reader := msgp.NewReader(file)
	if err := cache.serveHeaderFromCache(w, reader); err != nil {
		return err
	}
	_, err := io.Copy(w, reader)
	return err
}

func (cache diskCache) streamFromCacheInProgress(w http.ResponseWriter, file *os.File, progress *progressTracker) error {
	reader := msgp.NewReader(file)
	if err := cache.serveHeaderFromCache(w, reader); err != nil {
		return err
	}

	var position int64 = 0
	for {
		n, err := io.Copy(w, reader)
		position += n
		if err != nil {
			return err
		}

		err = progress.WaitForMore(position)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}
