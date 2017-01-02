package response_cache

import "bytes"
import "io"
import "net/http"
import "os"
import "sync"

type memoryCacheEntry struct {
	status int
	header http.Header
	body   []byte
}

func (entry memoryCacheEntry) Status() int {
	return entry.status
}

func (entry memoryCacheEntry) Header() http.Header {
	return entry.header
}

func (entry memoryCacheEntry) Body() io.Reader {
	return bytes.NewReader(entry.body)
}

func (entry memoryCacheEntry) Close() {
	// do nothing
}

func (entry memoryCacheEntry) WriteTo(w http.ResponseWriter) {
	WriteEntryTo(entry, w)
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

func (cache memoryCache) Get(key string) (Entry, error) {
	cache.RLock()
	defer cache.RUnlock()
	entry, ok := cache.Entries[key]
	if ok {
		return entry, nil
	}
	return nil, os.ErrNotExist
}

type memoryCacheWriter struct {
	cache memoryCache
	key   string
	entry *memoryCacheEntry
}

func (writer memoryCacheWriter) WriteHeader(status int, header http.Header) error {
	writer.entry.status = status

	writer.entry.header = make(http.Header)
	CopyHeader(writer.entry.Header(), header)

	return nil
}

func (writer memoryCacheWriter) Write(data []byte) (int, error) {
	writer.entry.body = append(writer.entry.body, data...)
	return len(data), nil
}

func (writer memoryCacheWriter) Finish() error {
	writer.cache.RLock()
	defer writer.cache.RUnlock()
	writer.cache.Entries[writer.key] = *writer.entry
	return nil
}

func (writer memoryCacheWriter) Abort() error {
	// do nothing
	return nil
}

func (cache memoryCache) BeginWrite(key string) (CacheWriter, error) {
	return memoryCacheWriter{
		cache: cache,
		key:   key,
		entry: &memoryCacheEntry{},
	}, nil
}
