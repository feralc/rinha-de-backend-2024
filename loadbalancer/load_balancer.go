package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	targetBackends []*url.URL
	proxy          *httputil.ReverseProxy
)

func main() {
	backends := strings.Split(os.Getenv("APP_BACKENDS"), ",")
	targetBackends = make([]*url.URL, len(backends))

	for i, address := range backends {
		parsed, err := url.Parse(address)
		if err != nil {
			panic(err)
		}
		targetBackends[i] = parsed
	}

	maxConnsPerHost, _ := strconv.Atoi(os.Getenv("APP_MAX_CONNS_PER_HOST"))
	maxIdleConns, _ := strconv.Atoi(os.Getenv("APP_MAX_IDLE_CONNS"))
	maxIdleConnsPerHost, _ := strconv.Atoi(os.Getenv("APP_MAX_IDLE_CONNS_PER_HOST"))
	idleConnTimeout, _ := strconv.Atoi(os.Getenv("APP_IDLE_CONN_TIMEOUT_SECONDS"))
	port, _ := strconv.Atoi(os.Getenv("APP_PORT"))

	fmt.Printf("APP_MAX_CONNS_PER_HOST=%d\n", maxConnsPerHost)
	fmt.Printf("APP_MAX_IDLE_CONNS=%d\n", maxIdleConns)
	fmt.Printf("APP_MAX_IDLE_CONNS_PER_HOST=%d\n", maxIdleConnsPerHost)
	fmt.Printf("APP_IDLE_CONN_TIMEOUT_SECONDS=%d\n", idleConnTimeout)

	initProxy(maxIdleConns, maxIdleConnsPerHost, idleConnTimeout, maxConnsPerHost)

	http.HandleFunc("/", handleRequest)

	fmt.Printf("Load balancer listening on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func initProxy(maxIdleConns, maxIdleConnsPerHost, idleConnTimeout, maxConnsPerHost int) {
	proxy = &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(idleConnTimeout) * time.Second,
			MaxConnsPerHost:     maxConnsPerHost,
			DisableKeepAlives:   false,
		},
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	proxy.ServeHTTP(w, r)
}

func director(req *http.Request) {
	clientID := getClientID(req.URL.Path)

	targetIndex := clientID % len(targetBackends)
	targetURL := targetBackends[targetIndex]

	req.URL.Scheme = targetURL.Scheme
	req.URL.Host = targetURL.Host
}

func getClientID(path string) int {
	parts := strings.Split(path, "/")
	clientIDStr := parts[2]
	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil {
		return 0
	}
	return clientID
}
