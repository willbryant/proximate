package response_cache

import "io"
import "github.com/tinylib/msgp/msgp"
import "net/http"
import "sync"
import "os"

type diskCache struct {
	cacheDirectory        string
	progressTrackersMutex sync.Mutex
	progressTrackers      map[string]*progressTracker
}

func NewDiskCache(cacheDirectory string) ResponseCache {
	return &diskCache{
		cacheDirectory:   cacheDirectory,
		progressTrackers: make(map[string]*progressTracker),
	}
}

func (cache *diskCache) cacheEntryPath(key string) string {
	return cache.cacheDirectory + "/" + key
}

func (cache *diskCache) Get(key string, realWriter http.ResponseWriter, miss func(writer http.ResponseWriter) error) error {
	path := cache.cacheEntryPath(key)
	file, err := os.Open(path)

	if err == nil {
		return cache.serveFromCache(realWriter, file)
	}

	if !os.IsNotExist(err) {
		return err
	}

	// cache miss
	progress := cache.progressTrackerFor(path, miss)

	err = progress.WaitForResponse()
	if err == Uncacheable {
		// TODO: this results in a second upstream request, which is not ideal - would be better to stream the first request to just one client
		err = miss(realWriter)
		if err != nil {
			return err
		}
		return Uncacheable
	} else if err != nil {
		return err
	}

	// TODO: race here if the miss function completes quickly
	file, err = os.Open(path + ".temp")
	if err != nil {
		return err
	}
	defer file.Close()
	err = cache.streamFromCacheInProgress(realWriter, file, progress)

	return os.ErrNotExist // indicates a cache miss
}

func (cache *diskCache) progressTrackerFor(path string, miss func(writer http.ResponseWriter) error) *progressTracker {
	cache.progressTrackersMutex.Lock()
	defer cache.progressTrackersMutex.Unlock()
	progress := cache.progressTrackers[path]
	if progress == nil {
		progress = newProgressTracker()
		cache.progressTrackers[path] = progress
		go cache.populate(path, progress, miss)
	}
	return progress
}

func (cache *diskCache) clearProgressTrackerFor(path string) {
	cache.progressTrackersMutex.Lock()
	defer cache.progressTrackersMutex.Unlock()
	delete(cache.progressTrackers, path)
}

func (cache *diskCache) populate(path string, progress *progressTracker, miss func(writer http.ResponseWriter) error) {
	file, err := os.OpenFile(path+".temp", os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return
	}
	writer := diskCacheWriter{
		tempfile: file,
		header:   make(http.Header),
		progress: progress,
	}
	err = miss(&writer)
	if err == nil {
		err = writer.Finish(path)
	}
	if err != nil {
		writer.Abort(err)
	}
	cache.clearProgressTrackerFor(path)
}

func (cache *diskCache) serveHeaderFromCache(w http.ResponseWriter, streamer *msgp.Reader) error {
	var diskCacheHeader DiskCacheHeader
	if err := diskCacheHeader.DecodeMsg(streamer); err != nil {
		return err
	}

	CopyHeader(w.Header(), diskCacheHeader.Header)
	w.WriteHeader(diskCacheHeader.Status)
	return nil
}

func (cache *diskCache) serveFromCache(w http.ResponseWriter, file *os.File) error {
	reader := msgp.NewReader(file)
	if err := cache.serveHeaderFromCache(w, reader); err != nil {
		return err
	}
	_, err := io.Copy(w, reader)
	return err
}

func (cache *diskCache) streamFromCacheInProgress(w http.ResponseWriter, file *os.File, progress *progressTracker) error {
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

func (cache *diskCache) Clear() error {
	return clearDirectory(cache.cacheDirectory)
}

func clearDirectory(directory string) error {
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dir.Close()

	for {
		filenames, err := dir.Readdirnames(1000)

		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		for _, filename := range filenames {
			// ignore hidden files in the cache root directory
			if filename[0] != '.' {
				err := os.RemoveAll(directory + string(os.PathSeparator) + filename)

				if err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
	}
}
