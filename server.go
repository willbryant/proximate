package main

import "fmt"
import "net"
import "net/http"
import "net/http/httputil"
import "os"
import "github.com/willbryant/proximate/response_cache"
import "strings"

type proximateServer struct {
	Listener net.Listener
	Tracker  *ConnectionTracker
	Closed   uint32
	Quiet    bool
	Cache    response_cache.ResponseCache
	Proxy    *httputil.ReverseProxy
}

func ProximateServer(listener net.Listener, cacheDirectory string, quiet bool) proximateServer {
	return proximateServer{
		Listener: listener,
		Tracker:  NewConnectionTracker(),
		Quiet:    quiet,
		Cache:    response_cache.NewDiskCache(cacheDirectory),
		Proxy:    &httputil.ReverseProxy{Director: setProxyUserAgentDirector},
	}
}

func setProxyUserAgentDirector(req *http.Request) {
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

func (server proximateServer) serveGitPackRequest(w http.ResponseWriter, req *http.Request) {
	hash, err := response_cache.HashRequestAndBody(req)
	if err != nil {
		http.Error(w, err.Error(), 401)
	}

	if cacheEntry, ok := server.Cache.Get(hash); ok {
		fmt.Fprintf(os.Stdout, "%s request to %s is cached, request hash %s\n", req.Method, req.URL, hash)
		cacheEntry.WriteTo(w)
	} else {
		fmt.Fprintf(os.Stdout, "%s request to %s is cacheable, request hash %s\n", req.Method, req.URL, hash)
		writer := response_cache.NewResponseCacheWriter(server.Cache, hash, w)
		server.Proxy.ServeHTTP(writer, req)
		writer.Close()
	}
}

func (server proximateServer) extractHostFromPrefix(req *http.Request) {
	req.URL.Scheme = "https"
	parts := strings.SplitN(req.URL.Path, "/", 3)
	req.URL.Host = parts[1]
	req.URL.Path = "/" + parts[2]
	req.Host = req.URL.Host
}

func (server proximateServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := &responseLogger{w: w, req: req}

	server.extractHostFromPrefix(req)

	if cachableUploadGitPackRequest(req) {
		server.serveGitPackRequest(logger, req)
	} else {
		server.Proxy.ServeHTTP(logger, req)
	}

	if !server.Quiet && server.Active() {
		logger.ClfLog()
	}
}
