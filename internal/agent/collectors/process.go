package collectors

import (
	"context"
	"time"
)

// ProcessCollector collects process-level metrics
type ProcessCollector struct {
	*BaseCollector
	maxProcesses int
}

// ProcessCollectorConfig holds configuration for process collector
type ProcessCollectorConfig struct {
	Enabled      bool
	Interval     time.Duration
	MaxProcesses int
}

// NewProcessCollector creates a new process collector
func NewProcessCollector(config ProcessCollectorConfig) (*ProcessCollector, error) {
	return &ProcessCollector{
		BaseCollector: NewBaseCollector("process", config.Enabled, config.Interval),
		maxProcesses:  config.MaxProcesses,
	}, nil
}

// Collect collects process metrics
func (pc *ProcessCollector) Collect(ctx context.Context) ([]*Metric, error) {
	return make([]*Metric, 0), nil
}
