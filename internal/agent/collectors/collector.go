package collectors

import (
	"context"
	"time"
)

// Collector interface defines the contract for metric collectors
type Collector interface {
	// Collect collects metrics and returns them
	Collect(ctx context.Context) ([]*Metric, error)

	// Enabled returns whether this collector is enabled
	Enabled() bool

	// Interval returns the collection interval
	Interval() time.Duration

	// Name returns the collector name
	Name() string
}

// Metric represents a collected metric
type Metric struct {
	Name      string
	Value     float64
	Timestamp int64
	Labels    map[string]string
	Type      MetricType
	Help      string
	Unit      string
}

// MetricType represents the type of metric
type MetricType int

const (
	MetricTypeGauge MetricType = iota
	MetricTypeCounter
	MetricTypeHistogram
	MetricTypeSummary
)

// BaseCollector provides common functionality for collectors
type BaseCollector struct {
	name     string
	enabled  bool
	interval time.Duration
}

// NewBaseCollector creates a new base collector
func NewBaseCollector(name string, enabled bool, interval time.Duration) *BaseCollector {
	return &BaseCollector{
		name:     name,
		enabled:  enabled,
		interval: interval,
	}
}

// Enabled returns whether the collector is enabled
func (bc *BaseCollector) Enabled() bool {
	return bc.enabled
}

// Interval returns the collection interval
func (bc *BaseCollector) Interval() time.Duration {
	return bc.interval
}

// Name returns the collector name
func (bc *BaseCollector) Name() string {
	return bc.name
}
