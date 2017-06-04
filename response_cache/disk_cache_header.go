package response_cache

//go:generate msgp
type DiskCacheHeader struct {
	Version int `msg:"version"`

	StatusCode int `msg:"status_code"`

	// equivalent to: Header http.Header `msg:"header"`
	Header map[string][]string `msg:"header"`

	ContentLength int64 `msg:"content_length"`
}
