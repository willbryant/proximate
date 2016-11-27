package response_cache

import "sync"

type ResponseCache struct {
	sync.RWMutex
	Entries map[string]Entry
}

func NewResponseCache(cacheDirectory string) *ResponseCache {
	return &ResponseCache{
		Entries: make(map[string]Entry),
	}
}

func (cache ResponseCache) Get(key string) (Entry, bool) {
	cache.RLock()
	defer cache.RUnlock()
	response, ok := cache.Entries[key]
	return response, ok
}

func (cache ResponseCache) Set(key string, entry Entry) {
	cache.RLock()
	defer cache.RUnlock()
	cache.Entries[key] = entry
}
