package response_cache

import "net/http"
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

func (cache memoryCache) Get(key string, w http.ResponseWriter, miss func() error) error {
	cache.RLock()
	defer cache.RUnlock()
	entry, ok := cache.Entries[key]
	if !ok {
		return miss()
	}

	CopyHeader(w.Header(), entry.header)
	w.WriteHeader(entry.status)
	w.Write(entry.body)
	return nil
}

type memoryCacheWriter struct {
	cache memoryCache
	key   string
	entry *memoryCacheEntry
}

func (writer *memoryCacheWriter) WriteHeader(status int, header http.Header) error {
	writer.entry = &memoryCacheEntry{
		status: status,
		header: make(http.Header),
	}

	CopyHeader(writer.entry.header, header)

	return nil
}

func (writer *memoryCacheWriter) Write(data []byte) (int, error) {
	writer.entry.body = append(writer.entry.body, data...)
	return len(data), nil
}

func (writer *memoryCacheWriter) Finish() error {
	writer.cache.RLock()
	defer writer.cache.RUnlock()
	writer.cache.Entries[writer.key] = *writer.entry
	return nil
}

func (writer *memoryCacheWriter) Abort() error {
	// do nothing
	return nil
}

func (cache memoryCache) BeginWrite(key string) (CacheWriter, error) {
	return &memoryCacheWriter{
		cache: cache,
		key:   key,
		entry: &memoryCacheEntry{},
	}, nil
}
