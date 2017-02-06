package response_cache

import "net/http"
import "os"
import "sync"

type memoryCacheEntry struct {
	status int
	header http.Header
	body   []byte
}

type memoryCache struct {
	sync.RWMutex
	Entries map[string]memoryCacheEntry
}

func NewMemoryCache() ResponseCache {
	return &memoryCache{
		Entries: make(map[string]memoryCacheEntry),
	}
}

func (cache *memoryCache) Get(key string, realWriter http.ResponseWriter, miss func(writer http.ResponseWriter) error) error {
	cache.RLock()
	entry, ok := cache.Entries[key]
	cache.RUnlock()

	if ok {
		return cache.serveCacheHit(realWriter, entry)
	}

	cacheWriter := memoryCacheWriter{
		entry: memoryCacheEntry{
			header: make(http.Header),
		},
		realWriter: realWriter,
	}
	if err := miss(&cacheWriter); err != nil {
		cacheWriter.Abort()
		return err
	}
	if err := cacheWriter.Finish(cache, key); err != nil {
		return err
	}
	return os.ErrNotExist // indicates a cache miss
}

func (cache *memoryCache) serveCacheHit(w http.ResponseWriter, entry memoryCacheEntry) error {
	CopyHeader(w.Header(), entry.header)
	w.WriteHeader(entry.status)
	_, err := w.Write(entry.body)
	return err
}

type memoryCacheWriter struct {
	entry memoryCacheEntry
	realWriter http.ResponseWriter
}

func (writer *memoryCacheWriter) Header() http.Header {
	return writer.entry.header
}

func (writer *memoryCacheWriter) WriteHeader(status int) {
	CopyHeader(writer.realWriter.Header(), writer.Header())
	writer.realWriter.WriteHeader(status)

	if writer.Aborted() {
		return
	}

	if !CacheableResponse(status, writer.Header()) {
		writer.Uncacheable()
		return
	}

	writer.entry.status = status
}

func (writer *memoryCacheWriter) Write(data []byte) (int, error) {
	n, err := writer.realWriter.Write(data)

	if writer.Aborted() {
		return n, err
	}

	writer.entry.body = append(writer.entry.body, data...)

	return len(data), nil
}

func (writer *memoryCacheWriter) Finish(cache *memoryCache, key string) error {
	if writer.Aborted() {
		return nil
	}

	cache.RLock()
	defer cache.RUnlock()
	cache.Entries[key] = writer.entry
	return nil
}

func (writer *memoryCacheWriter) Abort() error {
	writer.entry.status = -1
	return nil
}

func (writer *memoryCacheWriter) Uncacheable() error {
	return writer.Abort()
}

func (writer *memoryCacheWriter) Aborted() bool {
	return (writer.entry.status == -1)
}
