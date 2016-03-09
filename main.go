package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

// ref: net/http/httputil/reverseproxy.go
// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; http://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// ref: net/http/httputil/reverseproxy.go
// Remove hop-by-hop headers to the backend. Especially
// important is "Connection" because we want a persistent
// connection, regardless of what the client sent to us.
func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

// ref: etcd/proxy/httpproxy/reverse.go
func setForwardedFor(req *http.Request) {
	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return
	}

	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := req.Header["X-Forwarded-For"]; ok {
		clientIP = strings.Join(prior, ", ") + ", " + clientIP
	}
	req.Header.Set("X-Forwarded-For", clientIP)
}

func handleRequest(w http.ResponseWriter, req *http.Request) {
	log.Println(req)

	client := &http.Client{}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""
	delHopHeaders(req.Header)
	setForwardedFor(req)

	// Do sends an HTTP request and returns an HTTP response
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Proxy server error: ", err)
		http.Error(w, "Proxy server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Println("Remote backend ", req.RemoteAddr, " response status: ", resp.Status)

	delHopHeaders(resp.Header)
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Copy error:%s", err)
	}
}

func main() {
	var addr = flag.String("addr", "127.0.0.1:7070", "The address of the proxy server.")
	flag.Parse()
	http.HandleFunc("/", handleRequest)
	log.Println("Starting proxy server on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
