package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GenerateSessionID generates a unique session ID
func GenerateSessionID() string {
	return uuid.New().String()
}

// GenerateAlertID generates a unique alert ID
func GenerateAlertID() string {
	return fmt.Sprintf("alert-%s", uuid.New().String())
}

// GenerateMetricID generates a unique metric ID
func GenerateMetricID() string {
	return fmt.Sprintf("metric-%d-%s", time.Now().UnixNano(), randomString(8))
}

// GenerateNodeID generates a unique node ID
func GenerateNodeID() string {
	return uuid.New().String()
}

// randomString generates a random hex string of the specified length
func randomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// GenerateAPIKey generates a new API key
func GenerateAPIKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}
