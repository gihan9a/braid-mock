package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gihan9a/braidmock/internal/config"
	"gihan9a/braidmock/internal/utils"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
)

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
	config        *config.Config
	subscriptions map[string]map[string]Subscription
	versions      map[string]string
	hashes        map[string]string
	reverseProxy  *httputil.ReverseProxy
	mu            sync.RWMutex
	watcher       *fsnotify.Watcher
}

// NewBraidMockServer creates a new BraidMockServer
func NewBraidMockServer(config *config.Config) (*BraidMockServer, error) {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	server := &BraidMockServer{
		config:        config,
		subscriptions: make(map[string]map[string]Subscription),
		versions:      make(map[string]string),
		hashes:        make(map[string]string),
		watcher:       watcher,
	}

	// Configure reverse proxy if URL is provided
	if config.ProxyURL != nil {
		server.setupProxy()
	}

	// Start watching for file changes
	go server.watchFiles()

	return server, nil
}

// setupProxy configures the reverse proxy
func (s *BraidMockServer) setupProxy() {
	// Create a transport with optional insecure TLS setting
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if s.config.InsecureProxy {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Create the reverse proxy with custom transport
	s.reverseProxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = s.config.ProxyURL.Scheme
			req.URL.Host = s.config.ProxyURL.Host
			req.Host = s.config.ProxyURL.Host

			if s.config.ProxyURL.RawQuery != "" {
				if req.URL.RawQuery == "" {
					req.URL.RawQuery = s.config.ProxyURL.RawQuery
				} else {
					req.URL.RawQuery = s.config.ProxyURL.RawQuery + "&" + req.URL.RawQuery
				}
			}
		},
		Transport: transport,
	}

	log.Printf("Proxy mode enabled: Requests not found locally will be forwarded to %s", s.config.ProxyURL.String())
	if s.config.InsecureProxy {
		log.Printf("Warning: SSL certificate verification disabled for proxy requests")
	}
}

// Close cleans up resources used by the server
func (s *BraidMockServer) Close() {
	if s.watcher != nil {
		s.watcher.Close()
	}
}

// SetupWatchers recursively adds directories to the watcher
func (s *BraidMockServer) SetupWatchers() error {
	return filepath.Walk(s.config.RootDir, func(path string, info os.FileInfo, err error) error {
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
			data, err := os.ReadFile(event.Name)
			if err != nil {
				log.Printf("Error reading file: %v", err)
				continue
			}

			// Calculate hash for the resource
			hash := utils.CalculateHash(data)

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
	relPath, err := filepath.Rel(s.config.RootDir, path)
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
	return filepath.Join(s.config.RootDir, resourceID+".braid")
}

// fileExists checks if a mock file exists for the given resource ID
func (s *BraidMockServer) fileExists(resourceID string) bool {
	filePath := s.getPathFromResourceID(resourceID)
	_, err := os.Stat(filePath)
	return err == nil
}

// SetupRoutes configures the HTTP routes for the server
func (s *BraidMockServer) SetupRoutes() http.Handler {
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(s.handleBraidRequest)
	return router
}
