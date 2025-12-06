package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// HashLabels creates a hash from a labels map
func HashLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	// Sort keys for consistent hashing
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build label string
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}
	labelStr := strings.Join(parts, ",")

	// Hash it
	hash := sha256.Sum256([]byte(labelStr))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter hash
}
