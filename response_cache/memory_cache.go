package response_cache

import "net/http"
import "bytes"
import "sync"

type memoryCacheEntry struct {
	status int
	header http.Header
	body []byte
}

func (entry memoryCacheEntry) Status() int {
	return entry.status;
}

func (entry memoryCacheEntry) Header() http.Header {
	return entry.header;
}

func (entry memoryCacheEntry) Body() ReadSizer {
	return bytes.NewReader(entry.body)
}

func (entry memoryCacheEntry) WriteTo(w http.ResponseWriter) {
	WriteEntryTo(entry, w)
}

type memoryCache struct {
	sync.RWMutex
	Entries map[string]memoryCacheEntry
}

func NewMemoryCache() memoryCache {
	return memoryCache{
		Entries: make(map[string]memoryCacheEntry),
	}
}

func (cache memoryCache) Get(key string) (Entry, bool) {
	cache.RLock()
	defer cache.RUnlock()
	entry, ok := cache.Entries[key]
	return entry, ok
}

type memoryCacheBodyWriter struct {
	cache memoryCache
	key string
	entry *memoryCacheEntry
}

func (writer memoryCacheBodyWriter) Write(data []byte) (int, error) {
	writer.entry.body = append(writer.entry.body, data...)
	return len(data), nil
}

func (writer memoryCacheBodyWriter) Finish() {
	writer.cache.RLock()
	defer writer.cache.RUnlock()
	writer.cache.Entries[writer.key] = *writer.entry
}

func (writer memoryCacheBodyWriter) Abort() {
	// do nothing
}

func (cache memoryCache) BeginWrite(key string, status int, header http.Header) (CacheBodyWriter, error) {
	entry := memoryCacheEntry{
		status: status,
		header: make(http.Header),
	}
	CopyHeader(entry.Header(), header)

	return memoryCacheBodyWriter{
		cache: cache,
		key: key,
		entry: &entry,
	}, nil
}
