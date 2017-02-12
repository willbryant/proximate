package response_cache

import "net/url"
import "sort"
import "strings"

type Paths []string

type Upstreams struct {
	hosts map[string]Paths
}

func NewUpstreams(cacheServers string) *Upstreams {
	result := Upstreams{
		hosts: make(map[string]Paths),
	}

	sortedServers := sort.StringSlice(strings.Split(cacheServers, ","))
	sort.Sort(sortedServers)

	for _, str := range sortedServers {
		segments := strings.SplitN(str, "/", 2)
		host := strings.ToLower(segments[0])
		path := ""
		if len(segments) > 1 {
			path = "/" + segments[1]
		}
		result.hosts[host] = append(result.hosts[host], path)
	}

	return &result
}

func (upstreams *Upstreams) UpstreamListed(url *url.URL) bool {
	if paths, ok := upstreams.hosts[url.Host]; ok {
		// although the docs say url.Parse will set both Path and RawPath, the requests passed in
		// from ServePath only seem to set RawPath if there were encoded characters.  but if there
		// were, we want to use RawPath so we correctly handle matching / characters etc.
		urlPath := url.Path
		if url.RawPath != "" {
			urlPath = url.RawPath
		}

		for _, path := range paths {
			if strings.HasPrefix(urlPath, path) {
				return true
			}
		}
	}
	return false
}
