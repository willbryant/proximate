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
		for _, path := range paths {
			if strings.HasPrefix(url.Path, path) {
				return true
			}
		}
	}
	return false
}
