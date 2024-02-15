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

var apiAddresses = []string{
	"http://127.0.0.1:8080",
	"http://127.0.0.1:8081",
}

func main() {
	maxConnsPerHost, _ := strconv.Atoi(os.Getenv("APP_MAX_CONNS_PER_HOST"))
	maxIdleConns, _ := strconv.Atoi(os.Getenv("APP_MAX_IDLE_CONNS"))
	maxIdleConnsPerHost, _ := strconv.Atoi(os.Getenv("APP_MAX_IDLE_CONNS_PER_HOST"))
	idleConnTimeout, _ := strconv.Atoi(os.Getenv("APP_IDLE_CONN_TIMEOUT_SECONDS"))
	port, _ := strconv.Atoi(os.Getenv("APP_PORT"))

	fmt.Printf("APP_MAX_CONNS_PER_HOST=%d\n", maxConnsPerHost)
	fmt.Printf("APP_MAX_IDLE_CONNS=%d\n", maxIdleConns)
	fmt.Printf("APP_MAX_IDLE_CONNS_PER_HOST=%d\n", maxIdleConnsPerHost)
	fmt.Printf("APP_IDLE_CONN_TIMEOUT_SECONDS=%d\n", idleConnTimeout)

	proxy := &httputil.ReverseProxy{Director: director, Transport: &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(idleConnTimeout) * time.Second,
		MaxConnsPerHost:     maxConnsPerHost,
	}}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	fmt.Printf("Load balancer listening on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func director(req *http.Request) {
	clientID := getClientID(req.URL.Path)

	targetIndex := clientID % len(apiAddresses)
	targetURL, _ := url.Parse(apiAddresses[targetIndex])
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
