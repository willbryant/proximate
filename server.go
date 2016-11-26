package main

import "net"
import "net/http"

type approximateServer struct {
	Listener    net.Listener
	Tracker     *ConnectionTracker
	Closed      uint32
	RootDataDir string
	RootHttpDir http.Dir
	Quiet       bool
}

func ApproximateServer(listener net.Listener, cacheDirectory string, quiet bool) approximateServer {
	return approximateServer{
		Listener: listener,
		Tracker: NewConnectionTracker(),
		RootDataDir: cacheDirectory,
		RootHttpDir: http.Dir(cacheDirectory),
		Quiet: quiet,
	}
}

func (server approximateServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// we need to keep track of the response code and count the bytes so we can log them below
	logger := &responseLogger{w: w, req: req}

	http.Error(logger, "Method not supported", 405)

	if !server.Quiet && server.Active() {
		logger.ClfLog()
	}
}
