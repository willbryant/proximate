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
	return memoryCache{
		Entries: make(map[string]memoryCacheEntry),
	}
}

func (cache memoryCache) Get(key string, realWriter http.ResponseWriter, miss func(writer http.ResponseWriter) error) error {
	cache.RLock()
	entry, ok := cache.Entries[key]
	cache.RUnlock()

	if ok {
		return cache.ServeCacheHit(realWriter, entry)
	}

	cacheWriter := memoryCacheWriter{
		cache: cache,
		key:   key,
		entry: &memoryCacheEntry{},
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

func (cache memoryCache) ServeCacheHit(w http.ResponseWriter, entry memoryCacheEntry) error {
	CopyHeader(w.Header(), entry.header)
	w.WriteHeader(entry.status)
	_, err := w.Write(entry.body)
	return err
}

type memoryCacheWriter struct {
	cache memoryCache
	key   string
	entry *memoryCacheEntry
}

func (writer *memoryCacheWriter) WriteHeader(status int, header http.Header) error {
	if writer.Aborted() {
		return nil
	}

	writer.entry.status = status
	writer.entry.header = make(http.Header)
	CopyHeader(writer.entry.header, header)

	return nil
}

func (writer *memoryCacheWriter) Write(data []byte) (int, error) {
	if writer.Aborted() {
		return 0, nil
	}

	writer.entry.body = append(writer.entry.body, data...)

	return len(data), nil
}

func (writer *memoryCacheWriter) Finish() error {
	if writer.Aborted() {
		return nil
	}

	writer.cache.RLock()
	defer writer.cache.RUnlock()
	writer.cache.Entries[writer.key] = *writer.entry
	return nil
}

func (writer *memoryCacheWriter) Abort() error {
	writer.entry = nil
	return nil
}

func (writer *memoryCacheWriter) Aborted() bool {
	return (writer.entry == nil)
}
