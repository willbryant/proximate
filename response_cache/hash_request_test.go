package response_cache

import "testing"

import "io/ioutil"
import "net/http"
import "net/url"
import "strings"

func parseURL(s string) *url.URL {
	url, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return url
}

func dummyURL() *url.URL {
	return parseURL("http://user:pass@example.com/some/path?some=query")
}

func dummyHeader() http.Header {
	header := make(http.Header)
	header.Add("Host", "www.example.com")
	header.Add("Content-Type", "text/html")
	return header
}

func dummyRequest() *http.Request {
	return &http.Request {
		Method: "GET",
		URL: dummyURL(),
		Proto: "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: dummyHeader(),
		ContentLength: 16,
		Body: ioutil.NopCloser(strings.NewReader("0123456789abcdef")),
	}
}

func hashOf(req *http.Request) string {
	hash, err := HashRequestAndBody(req)
	if err != nil {
		panic(err)
	}
	return hash
}

func TestHashRequestAndBody(t *testing.T) {
	dummyReq := dummyRequest()
	hash := hashOf(dummyReq)

	req := dummyRequest()
	if hashOf(req) != hash { t.Error("hash not repeatable on the same request") }

	req = dummyRequest()
	req.Method = "POST"
	if hashOf(req) == hash { t.Error("hash did not vary on Method") }

	req = dummyRequest()
	req.URL = parseURL("http://user:pass@example.com/some/path?some=query&other=1")
	if hashOf(req) == hash { t.Error("hash did not vary on URL") }

	req = dummyRequest()
	req.URL = parseURL("https://user:pass@example.com/some/path?some=query")
	if hashOf(req) == hash { t.Error("hash did not vary on URL") }

	req = dummyRequest()
	req.Proto = "HTTP/1.0"
	req.ProtoMinor = 0
	if hashOf(req) == hash { t.Error("hash did not vary on Proto or ProtoMajor/ProtoMinor") }

	req = dummyRequest()
	req.Header = dummyHeader()
	req.Header.Add("X-Served-By", "test case")
	if hashOf(req) == hash { t.Error("hash did not vary on Header additions") }

	req = dummyRequest()
	req.Header = dummyHeader()
	req.Header.Set("Content-Type", "text/plain")
	if hashOf(req) == hash { t.Error("hash did not vary on Header changes") }

	req = dummyRequest()
	req.Body = ioutil.NopCloser(strings.NewReader("123456789abcdef0"))
	if hashOf(req) == hash { t.Error("hash did not vary on Body content changes") }

	req = dummyRequest()
	req.ContentLength = 17
	req.Body = ioutil.NopCloser(strings.NewReader("0123456789abcdef_"))
	if hashOf(req) == hash { t.Error("hash did not vary on Body length changes") }
}
