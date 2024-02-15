package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var apiAddresses = []string{
	"http://web01:8080",
	"http://web02:8080",
}

func main() {
	proxy := &httputil.ReverseProxy{Director: director, Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	fmt.Println("Load balancer listening on port 9999...")
	log.Fatal(http.ListenAndServe(":9999", nil))
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
