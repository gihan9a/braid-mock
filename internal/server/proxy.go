package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
)

// proxyRequest forwards the request to the configured proxy server
func (s *BraidMockServer) proxyRequest(w http.ResponseWriter, r *http.Request) {
	if s.reverseProxy != nil {
		// Use the configured reverse proxy
		s.reverseProxy.ServeHTTP(w, r)
		return
	}

	// If we don't have a reverse proxy (should not happen, but just in case),
	// create a new request and handle it manually
	proxyURL := *s.config.ProxyURL
	proxyURL.Path = r.URL.Path
	proxyURL.RawQuery = r.URL.RawQuery

	// Create a new request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating proxy request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Create HTTP client with optional insecure TLS
	client := &http.Client{}
	if s.config.InsecureProxy {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// Send the request
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error proxying request: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}
