package utils

import (
	"os"
	"runtime"
)

// GetSystemInfo returns basic system information
func GetSystemInfo() *SystemInfo {
	hostname, _ := os.Hostname()

	return &SystemInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		NumCPU:   runtime.NumCPU(),
	}
}

// SystemInfo contains system information
type SystemInfo struct {
	Hostname string
	OS       string
	Arch     string
	NumCPU   int
}
