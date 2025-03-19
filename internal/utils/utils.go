package utils

import (
	"fmt"
	"hash/crc32"
	"time"
)

// CalculateHash generates a CRC32 hash of the data
func CalculateHash(data []byte) string {
	table := crc32.MakeTable(crc32.IEEE)
	return fmt.Sprintf("\"%08x\"", crc32.Checksum(data, table))
}

// GenerateRandomID generates a random ID for subscriptions
func GenerateRandomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
