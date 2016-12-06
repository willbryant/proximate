package response_cache

//go:generate msgp
type DiskCacheHeader struct {
	Version int `msg:"version"`

	Status int `msg:"status"`

	// equivalent to: Header http.Header `msg:"header"`
	Header map[string][]string `msg:"header"`
}
