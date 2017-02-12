package response_cache

import "testing"

import "net/url"

func assertListed(t *testing.T, upstreams *Upstreams, uri string) {
	url, err := url.Parse(uri)

	if err != nil {
		t.Error("Couldn't parse url " + uri)
		return
	}

	if !upstreams.UpstreamListed(url) {
		t.Error("Expected " + uri + " to be listed in the upstream set, but it wasn't")
	}
}

func assertNotListed(t *testing.T, upstreams *Upstreams, uri string) {
	url, err := url.Parse(uri)

	if err != nil {
		t.Error("Couldn't parse url " + uri)
		return
	}

	if upstreams.UpstreamListed(url) {
		t.Error("Expected " + uri + " not to be listed in the upstream set, but it was")
	}
}

func TestUpstreams(t *testing.T) {
	upstreams := NewUpstreams("github.com/willbryant/,gitlab.com/irrelevant,github.com/rails,gitlab.com")

	assertListed(t, upstreams, "https://github.com/willbryant/proximate.git")
	assertListed(t, upstreams, "https://github.com/willbryant/")
	assertNotListed(t, upstreams, "https://github.com/willbryant")
	assertNotListed(t, upstreams, "https://github.com/willbryant%2f")
	assertNotListed(t, upstreams, "https://github.com/other")
	assertNotListed(t, upstreams, "https://github.com/")
	assertNotListed(t, upstreams, "https://github.com")

	assertListed(t, upstreams, "https://gitlab.com/whatever/")
	assertListed(t, upstreams, "https://gitlab.com/")
	assertListed(t, upstreams, "https://gitlab.com")

	assertNotListed(t, upstreams, "https://gitxyz.com/")
	assertNotListed(t, upstreams, "https://gitxyz.com/willbryant/proximate.git")

	empty := NewUpstreams("")
	assertNotListed(t, empty, "https://gitxyz.com/")

	empty = NewUpstreams(",")
	assertNotListed(t, empty, "https://gitxyz.com/")
}
