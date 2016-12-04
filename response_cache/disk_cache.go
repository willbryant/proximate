package response_cache

func NewDiskCache(cacheDirectory string) memoryCache {
	// TODO: implement real disk cache
	return NewMemoryCache()
}
