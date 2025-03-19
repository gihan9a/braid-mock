package server

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"gihan9a/braidmock/internal/utils"
)

// handleBraidRequest handles all Braid protocol requests
func (s *BraidMockServer) handleBraidRequest(w http.ResponseWriter, r *http.Request) {
	resourceID := r.URL.Path

	// Check if we have a local mock file for this resource
	if !s.fileExists(resourceID) {
		// If not and we have a proxy configured, forward the request
		if s.config.ProxyURL != nil {
			log.Printf("Resource %s not found locally, proxying to %s", resourceID, s.config.ProxyURL.String())
			s.proxyRequest(w, r)
			return
		}

		// No proxy configured, return 404
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Add CORS headers for mock server responses if enabled
	if s.config.CORS.Enabled {
		s.addCORSHeaders(w, r)

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	// Get path to the .braid file
	filePath := s.getPathFromResourceID(resourceID)

	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading resource: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate hash for the resource
	hash := utils.CalculateHash(data)

	s.mu.Lock()
	s.versions[resourceID] = hash
	s.hashes[resourceID] = hash
	s.mu.Unlock()

	// Set common headers
	w.Header().Set("Range-Request-Allow-Methods", "PATCH, PUT")
	w.Header().Set("Range-Request-Allow-Units", "json")
	w.Header().Set("Content-Type", "application/json")

	// Check if this is a subscription request
	if r.Header.Get("Subscribe") == "true" || r.Header.Get("subscribe") == "true" {
		// Ensure we can flush the response
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Set headers for streaming
		w.Header().Set("subscribe", "true")
		w.Header().Set("cache-control", "no-cache, no-transform")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(209) // 209 is the status code for a successful subscription

		// Add subscription
		subID := s.AddSubscription(resourceID, w, flusher, data)

		// Send initial state
		fmt.Fprintf(w, "Version: %s\r\n", hash)
		fmt.Fprintf(w, "Parents: \r\n")
		fmt.Fprintf(w, "Content-Length: %d\r\n", len(data))
		fmt.Fprintf(w, "\r\n")
		w.Write(data)
		fmt.Fprintf(w, "\r\n\r\n\r\n\r\n\r\n")
		flusher.Flush()

		// Remove subscription when client disconnects
		notify := r.Context().Done()
		go func() {
			<-notify
			s.RemoveSubscription(resourceID, subID)
		}()

		// Keep the connection open until client disconnects
		<-notify
	} else {
		// Regular GET request
		w.Header().Set("Version", hash)
		w.Header().Set("Parents", "")

		w.Write(data)
	}
}

// addCORSHeaders adds CORS headers to the response
func (s *BraidMockServer) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", s.config.CORS.AllowOrigins)
	w.Header().Set("Access-Control-Allow-Methods", s.config.CORS.AllowMethods)
	w.Header().Set("Access-Control-Allow-Headers", s.config.CORS.AllowHeaders)

	if s.config.CORS.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", s.config.CORS.MaxAge))
}
