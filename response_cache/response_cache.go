package response_cache

import "net/http"

type ResponseCache interface {
	Get(key string, w http.ResponseWriter, miss func(writer http.ResponseWriter) error) error
}
