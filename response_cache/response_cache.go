package response_cache

type ResponseCache interface {
	Get(key string) (Entry, bool)
	Set(key string, entry Entry)
}
