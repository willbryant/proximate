package response_cache

import "sync"
import "io"

type ResponseCache interface {
	Get(key string) (Entry, bool)
	Set(key string, entry Entry)
}

func NewDiskCache(cacheDirectory string) memoryCache {
	// TODO: implement real disk cache
	return NewMemoryCache()
}

type memoryCache struct {
	sync.RWMutex
	Entries map[string]Entry
}

func NewMemoryCache() memoryCache {
	return memoryCache{
		Entries: make(map[string]Entry),
	}
}

func (cache memoryCache) Get(key string) (Entry, bool) {
	cache.RLock()
	defer cache.RUnlock()
	entry, ok := cache.Entries[key]
	if ok { entry.Body.Seek(0, io.SeekStart) }
	return entry, ok
}

func (cache memoryCache) Set(key string, entry Entry) {
	cache.RLock()
	defer cache.RUnlock()
	cache.Entries[key] = entry
}
