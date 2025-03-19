package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gihan9a/braidmock/internal/utils"

	"github.com/wI2L/jsondiff"
)

// AddSubscription adds a new subscription for a resource
func (s *BraidMockServer) AddSubscription(resourceID string, w http.ResponseWriter, f http.Flusher, initialResource []byte) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	subID := utils.GenerateRandomID()
	hash := utils.CalculateHash(initialResource)

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

	newHash := utils.CalculateHash(newData)
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
