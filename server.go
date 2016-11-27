package main

import "bytes"
import "fmt"
import "io"
import "io/ioutil"
import "net"
import "net/http"
import "net/http/httputil"
import "os"
import "github.com/willbryant/approximate/response_cache"
import "strings"

type approximateServer struct {
	Listener            net.Listener
	Tracker             *ConnectionTracker
	Closed              uint32
	Quiet               bool
	Cache               *response_cache.ResponseCache
	HostFromPrefixProxy *httputil.ReverseProxy
}

func ApproximateServer(listener net.Listener, cacheDirectory string, quiet bool) approximateServer {
	return approximateServer{
		Listener: listener,
		Tracker: NewConnectionTracker(),
		Quiet: quiet,
		Cache: response_cache.NewResponseCache(cacheDirectory),
		HostFromPrefixProxy: &httputil.ReverseProxy{Director: prefixedHostDirector},
	}
}

func prefixedHostDirector(req *http.Request) {
	req.URL.Scheme = "https"
	parts := strings.SplitN(req.URL.Path, "/", 3)
	req.URL.Host = parts[1]
	req.URL.Path = "/" + parts[2]
	req.Host = req.URL.Host
	if ua, ok := req.Header["User-Agent"]; ok {
		req.Header.Set("X-Proxy-Client-Agent", ua[0])
	}
	req.Header.Set("User-Agent", banner())
	fmt.Fprintf(os.Stdout, "proxying %s request to %s\n", req.Method, req.URL)
}

func cachableUploadGitPackRequest(req *http.Request) bool {
	return req.ContentLength > 0 && req.ContentLength < 65536 && // arbitrary
		req.Method == "POST" &&
		req.Header.Get("Content-Type") == "application/x-git-upload-pack-request" &&
		req.Header.Get("Accept") == "application/x-git-upload-pack-result" &&
		req.Header.Get("Cache-Control") == "" &&
		req.Header.Get("Authorization") == ""
}

func (server approximateServer) ServeGitPackRequest(w http.ResponseWriter, req *http.Request) {
	body := make([]byte, req.ContentLength)
	if _, err := io.ReadFull(req.Body, body); err != nil {
		http.Error(w, err.Error(), 401)
	}

	hash, err := response_cache.HashRequestAndBody(req, body)
	if err != nil {
		http.Error(w, err.Error(), 401)
	}

	req.Body.Close()
	if cachedResponse, ok := server.Cache.Get(hash); ok {
		fmt.Fprintf(os.Stdout, "%s request to %s is cached, request hash %s\n", req.Method, req.URL, hash)
		response_cache.CopyHeader(w.Header(), cachedResponse.Header)
		w.WriteHeader(http.StatusOK)
		w.Write(cachedResponse.Body)
	} else {
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		fmt.Fprintf(os.Stdout, "%s request to %s is cacheable, request hash %s\n", req.Method, req.URL, hash)
		writer := response_cache.NewResponseCacheWriter(server.Cache, hash, w)
		server.HostFromPrefixProxy.ServeHTTP(writer, req)
		writer.Close()
	}
}

func (server approximateServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := &responseLogger{w: w, req: req}

	if cachableUploadGitPackRequest(req) {
		server.ServeGitPackRequest(logger, req)
	} else {
		server.HostFromPrefixProxy.ServeHTTP(logger, req)
	}

	if !server.Quiet && server.Active() {
		logger.ClfLog()
	}
}
