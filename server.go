package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"github.com/wI2L/jsondiff"
)

// Patch represents a patch in the Braid protocol.
type Patch struct {
	Unit    string `json:"unit"`    // Unit represents the operational unit of the patch, e.g. "replace"
	Range   string `json:"range"`   // Range represents the path of the patch, e.g. "/foo/bar/0/id"
	Content string `json:"content"` // Content is the actual content of the patch, can be a JSON object
}

// Subscription represents a client subscription to resource changes
type Subscription struct {
	ID           string
	W            http.ResponseWriter
	F            http.Flusher
	LastResource []byte // Store the last resource state to calculate patches
	LastHash     string // Store the hash of the last resource
}

// BraidMockServer implements a mock server for the Braid protocol
type BraidMockServer struct {
	rootDir       string
	subscriptions map[string]map[string]Subscription
	versions      map[string]string
	hashes        map[string]string
	proxyURL      *url.URL
	proxyClient   *http.Client
	reverseProxy  *httputil.ReverseProxy
	mu            sync.RWMutex
	watcher       *fsnotify.Watcher
}

// NewBraidMockServer creates a new BraidMockServer
func NewBraidMockServer(rootDir string, proxyURLStr string) (*BraidMockServer, error) {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	server := &BraidMockServer{
		rootDir:       rootDir,
		subscriptions: make(map[string]map[string]Subscription),
		versions:      make(map[string]string),
		hashes:        make(map[string]string),
		proxyClient:   &http.Client{Timeout: 30 * time.Second},
		watcher:       watcher,
	}

	// Configure reverse proxy if URL is provided
	if proxyURLStr != "" {
		proxyURL, err := url.Parse(proxyURLStr)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		server.proxyURL = proxyURL
		server.reverseProxy = httputil.NewSingleHostReverseProxy(proxyURL)

		// Customize the director to preserve the original path
		director := server.reverseProxy.Director
		server.reverseProxy.Director = func(req *http.Request) {
			director(req)
			req.Host = proxyURL.Host
		}

		log.Printf("Proxy mode enabled: Requests not found locally will be forwarded to %s", proxyURLStr)
	}

	// Start watching for file changes
	go server.watchFiles()

	return server, nil
}

// setupWatchers recursively adds directories to the watcher
func (s *BraidMockServer) setupWatchers() error {
	return filepath.Walk(s.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return s.watcher.Add(path)
		}
		return nil
	})
}

// watchFiles monitors file changes and sends updates to subscribers
func (s *BraidMockServer) watchFiles() {
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only process .braid file writes
			if !strings.HasSuffix(event.Name, ".braid") || event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			// Get resource ID from file path
			resourceID, err := s.getResourceIDFromPath(event.Name)
			if err != nil {
				log.Printf("Error determining resource ID: %v", err)
				continue
			}

			log.Printf("File changed: %s, resourceID: %s", event.Name, resourceID)

			// Read updated content
			data, err := ioutil.ReadFile(event.Name)
			if err != nil {
				log.Printf("Error reading file: %v", err)
				continue
			}

			// Calculate hash for the resource
			hash := calculateHash(data)

			s.mu.Lock()
			s.versions[resourceID] = hash
			s.hashes[resourceID] = hash
			s.mu.Unlock()

			// Notify subscribers
			s.notifySubscribers(resourceID, data)

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// getResourceIDFromPath converts a file path to a resource ID
func (s *BraidMockServer) getResourceIDFromPath(path string) (string, error) {
	// Make the path relative to the root directory
	relPath, err := filepath.Rel(s.rootDir, path)
	if err != nil {
		return "", err
	}

	// Remove .braid extension
	resourceID := strings.TrimSuffix(relPath, ".braid")

	// Convert Windows path separators to URL path separators
	resourceID = strings.ReplaceAll(resourceID, "\\", "/")

	// Ensure the path starts with /
	if !strings.HasPrefix(resourceID, "/") {
		resourceID = "/" + resourceID
	}

	return resourceID, nil
}

// getPathFromResourceID converts a resource ID to a file path
func (s *BraidMockServer) getPathFromResourceID(resourceID string) string {
	// Remove leading / if present
	if strings.HasPrefix(resourceID, "/") {
		resourceID = resourceID[1:]
	}

	// Create complete path
	return filepath.Join(s.rootDir, resourceID+".braid")
}

// fileExists checks if a mock file exists for the given resource ID
func (s *BraidMockServer) fileExists(resourceID string) bool {
	filePath := s.getPathFromResourceID(resourceID)
	_, err := os.Stat(filePath)
	return err == nil
}

// AddSubscription adds a new subscription for a resource
func (s *BraidMockServer) AddSubscription(resourceID string, w http.ResponseWriter, f http.Flusher, initialResource []byte) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	subID := generateRandomID()
	hash := calculateHash(initialResource)

	if _, exists := s.subscriptions[resourceID]; !exists {
		s.subscriptions[resourceID] = make(map[string]Subscription)
	}

	s.subscriptions[resourceID][subID] = Subscription{
		ID:           subID,
		W:            w,
		F:            f,
		LastResource: initialResource,
		LastHash:     hash,
	}

	log.Printf("Added subscription %s for resource %s", subID, resourceID)
	return subID
}

// RemoveSubscription removes a subscription
func (s *BraidMockServer) RemoveSubscription(resourceID, subID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if subs, exists := s.subscriptions[resourceID]; exists {
		delete(subs, subID)
		log.Printf("Removed subscription %s for resource %s", subID, resourceID)

		// Clean up empty subscription maps
		if len(subs) == 0 {
			delete(s.subscriptions, resourceID)
		}
	}
}

// notifySubscribers sends an update to all subscribers of a resource
func (s *BraidMockServer) notifySubscribers(resourceID string, newData []byte) {
	s.mu.RLock()
	subs := s.subscriptions[resourceID]
	s.mu.RUnlock()

	if len(subs) == 0 {
		return
	}

	newHash := calculateHash(newData)
	log.Printf("Notifying %d subscribers for resource %s", len(subs), resourceID)

	// Process each subscription
	for subID, sub := range subs {
		if sub.LastHash == newHash {
			log.Printf("Resource %s unchanged for subscription %s, skipping update", resourceID, subID)
			continue
		}

		// Create and send update
		if len(sub.LastResource) == 0 {
			// First update - send full resource
			s.sendFullUpdate(sub, newData, newHash)
		} else {
			// Subsequent update - send patch if possible
			err := s.sendPatchUpdate(sub, newData, newHash)
			if err != nil {
				log.Printf("Error sending patch update: %v, falling back to full update", err)
				s.sendFullUpdate(sub, newData, newHash)
			}
		}

		// Update the last resource and hash for this subscription
		s.mu.Lock()
		if subscriptions, exists := s.subscriptions[resourceID]; exists {
			if subscription, exists := subscriptions[subID]; exists {
				subscription.LastResource = make([]byte, len(newData))
				copy(subscription.LastResource, newData)
				subscription.LastHash = newHash
				subscriptions[subID] = subscription
			}
		}
		s.mu.Unlock()
	}
}

// sendFullUpdate sends a full resource update to a subscriber
func (s *BraidMockServer) sendFullUpdate(sub Subscription, data []byte, hash string) error {
	// Write headers
	fmt.Fprintf(sub.W, "Version: %s\r\n", hash)
	fmt.Fprintf(sub.W, "Parents: \r\n")
	fmt.Fprintf(sub.W, "Content-Length: %d\r\n", len(data))
	fmt.Fprintf(sub.W, "\r\n")

	// Write body
	if _, err := sub.W.Write(data); err != nil {
		return err
	}

	// Add separator for subscription stream
	fmt.Fprintf(sub.W, "\r\n\r\n\r\n\r\n\r\n")
	sub.F.Flush()
	return nil
}

// sendPatchUpdate sends a patch update to a subscriber
func (s *BraidMockServer) sendPatchUpdate(sub Subscription, newData []byte, newHash string) error {
	// Calculate patch
	patchOperations, err := jsondiff.CompareJSON(sub.LastResource, newData)
	if err != nil {
		return err
	}

	if len(patchOperations) == 0 {
		// No changes detected
		return nil
	}

	// Write headers
	fmt.Fprintf(sub.W, "Version: %s\r\n", newHash)
	fmt.Fprintf(sub.W, "Parents: %s\r\n", sub.LastHash)

	// Write patches header if more than one patch
	if len(patchOperations) > 1 {
		fmt.Fprintf(sub.W, "Patches: %d\r\n\r\n", len(patchOperations))
	}

	// Write each patch
	for i, op := range patchOperations {
		if i > 0 {
			fmt.Fprintf(sub.W, "\r\n\r\n")
		}

		valueJSON, _ := json.Marshal(op.Value)
		fmt.Fprintf(sub.W, "Content-Length: %d\r\n", len(valueJSON))
		fmt.Fprintf(sub.W, "Content-Range: %s %s\r\n", op.Type, op.Path)
		fmt.Fprintf(sub.W, "\r\n")
		fmt.Fprintf(sub.W, "%s", string(valueJSON))
	}

	// Add separator for subscription stream
	fmt.Fprintf(sub.W, "\r\n\r\n\r\n\r\n\r\n")
	sub.F.Flush()
	return nil
}

// handleBraidRequest handles all Braid protocol requests
func (s *BraidMockServer) handleBraidRequest(w http.ResponseWriter, r *http.Request) {
	resourceID := r.URL.Path

	// Check if we have a local mock file for this resource
	if !s.fileExists(resourceID) {
		// If not and we have a proxy configured, forward the request
		if s.proxyURL != nil {
			log.Printf("Resource %s not found locally, proxying to %s", resourceID, s.proxyURL.String())
			s.proxyRequest(w, r)
			return
		}

		// No proxy configured, return 404
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Get path to the .braid file
	filePath := s.getPathFromResourceID(resourceID)

	// Read file content
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading resource: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate hash for the resource
	hash := calculateHash(data)

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

// proxyRequest forwards the request to the configured proxy server
func (s *BraidMockServer) proxyRequest(w http.ResponseWriter, r *http.Request) {
	if s.reverseProxy != nil {
		// Use the configured reverse proxy
		s.reverseProxy.ServeHTTP(w, r)
		return
	}

	// If we don't have a reverse proxy (should not happen, but just in case),
	// create a new request and handle it manually
	proxyURL := *s.proxyURL
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

	// Send the request
	resp, err := s.proxyClient.Do(proxyReq)
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

// calculateHash generates a CRC32 hash of the data
func calculateHash(data []byte) string {
	table := crc32.MakeTable(crc32.IEEE)
	return fmt.Sprintf("\"%08x\"", crc32.Checksum(data, table))
}

// generateRandomID generates a random ID for subscriptions
func generateRandomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func main() {
	// Parse command line flags
	dirFlag := flag.String("d", ".", "Directory containing .braid mock files")
	portFlag := flag.Int("p", 3000, "Port to listen on")
	proxyFlag := flag.String("proxy", "", "URL to proxy requests to when mock files aren't found")
	flag.Parse()

	// Create server
	server, err := NewBraidMockServer(*dirFlag, *proxyFlag)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.watcher.Close()

	// Set up watchers for the directory
	if err := server.setupWatchers(); err != nil {
		log.Fatalf("Failed to set up file watchers: %v", err)
	}

	// Set up HTTP router
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(server.handleBraidRequest)

	// Start server
	addr := fmt.Sprintf(":%d", *portFlag)
	log.Printf("Braid mock server running at http://localhost%s", addr)
	log.Printf("Serving .braid files from directory: %s", *dirFlag)
	log.Fatal(http.ListenAndServe(addr, router))
}
