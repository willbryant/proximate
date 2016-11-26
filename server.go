package main

import "fmt"
import "net"
import "net/http"
import "net/http/httputil"
import "os"
import "strings"

type approximateServer struct {
	Listener    net.Listener
	Tracker     *ConnectionTracker
	Closed      uint32
	RootDataDir string
	RootHttpDir http.Dir
	Quiet       bool
	PrefixedHostProxy *httputil.ReverseProxy
}

func ApproximateServer(listener net.Listener, cacheDirectory string, quiet bool) approximateServer {
	return approximateServer{
		Listener: listener,
		Tracker: NewConnectionTracker(),
		RootDataDir: cacheDirectory,
		RootHttpDir: http.Dir(cacheDirectory),
		Quiet: quiet,
		PrefixedHostProxy: &httputil.ReverseProxy{Director: prefixedHostDirector},
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

func (server approximateServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// we need to keep track of the response code and count the bytes so we can log them below
	logger := &responseLogger{w: w, req: req}

	server.PrefixedHostProxy.ServeHTTP(logger, req)

	if !server.Quiet && server.Active() {
		logger.ClfLog()
	}
}
