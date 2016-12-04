package response_cache

func NewDiskCache(cacheDirectory string) ResponseCache {
	// TODO: implement real disk cache
	return NewMemoryCache()
}
