package main

import "flag"
import "fmt"
import "net"
import "net/http"
import "os"
import "os/signal"
import "runtime"
import "runtime/pprof"
import "strings"
import "time"
import "syscall"

var compiled_version string
var compiled_cache_directory string

func banner() string {
	if compiled_version != "" {
		return "Proximate " + compiled_version
	} else {
		return "Proximate"
	}
}

func default_cache_directory() string {
	if compiled_cache_directory != "" {
		return compiled_cache_directory
	} else {
		return DefaultCacheDirectory
	}
}

func main() {
	var cacheDirectory, cacheGitPackServers, cacheDebPoolServers, listenAddress, port string
	var healthCheckPath, healthyIfFile, healthyUnlessFile string
	var quiet bool

	flag.StringVar(&cacheDirectory, "data", default_cache_directory(), "Sets the root data directory to /foo.  Must be fully-qualified (ie. it must start with a /).")
	flag.StringVar(&cacheGitPackServers, "cache-git-packs", "", "Cache git pack requests from this comma-separated list of servers.  May include paths (eg. \"github.com/willbryant,github.com/rails,gitlab.com\").")
	flag.StringVar(&cacheDebPoolServers, "cache-deb-pools", "", "Cache deb pool requests from this comma-separated list of servers.  May include paths (eg. \"security.ubuntu.com,somemirrors.org/ubuntu\").")
	flag.StringVar(&listenAddress, "listen", DefaultListenAddress, "Listen on the given IP address.  Default: listen on all network interfaces.")
	flag.StringVar(&port, "port", DefaultPort, "Listen on the given port.")
	flag.BoolVar(&quiet, "quiet", false, "Quiet mode.  Don't print startup/shutdown/request log messages to stdout.")
	flag.StringVar(&healthCheckPath, "health-check-path", "", "Treat requests to this path as health checks from your load balancer, and give a 200 response without trying to serve a file.")
	flag.StringVar(&healthyIfFile, "healthy-if-file", "", "Respond to requests to the health-check-path with a 503 response code if this file doesn't exist.")
	flag.StringVar(&healthyUnlessFile, "healthy-unless-file", "", "Respond to requests to the health-check-path with a 503 response code if this file exists.")
	flag.VisitAll(setFlagFromEnvironment)
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	listener, err := net.Listen("tcp", listenAddress+":"+port)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't listen on %s:%s: %s\n", listenAddress, port, err.Error())
		os.Exit(1)
	} else if !quiet {
		fmt.Fprintf(os.Stdout, "%s listening on http://%s:%s, cache in %s\n", banner(), listenAddress, port, cacheDirectory)
	}

	server := ProximateServer(listener, cacheDirectory, cacheGitPackServers, cacheDebPoolServers, quiet)
	go waitForSignals(&server)

	if healthCheckPath != "" {
		http.Handle(AddRoot(healthCheckPath), HealthCheckServer(healthyIfFile, healthyUnlessFile))
	}
	http.Handle("/", server)

	server.Serve()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unexpected error: %s\n", err.Error())
		os.Exit(1)
	}

	server.Tracker.Shutdown(ShutdownResponseTimeout * time.Second)

	os.Exit(0)
}

func waitForSignals(server *proximateServer) {
	signals := make(chan os.Signal, 3)
	signal.Notify(signals, syscall.SIGINT)
	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGUSR2)
	for {
		switch <-signals {
		case syscall.SIGUSR2:
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			pprof.Lookup("heap").WriteTo(os.Stdout, 1)

		case syscall.SIGINT, syscall.SIGTERM:
			if !server.Quiet {
				fmt.Fprintf(os.Stdout, "Proximate shutting down by request\n")
			}
			server.Shutdown()
		}
	}
}

func setFlagFromEnvironment(f *flag.Flag) {
	env := "PROXIMATE_" + strings.Replace(strings.ToUpper(f.Name), "-", "_", -1)
	if os.Getenv(env) != "" {
		flag.Set(f.Name, os.Getenv(env))
	}
}
