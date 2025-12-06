package collectors

import (
	"context"
	"time"
)

// ContainerCollector collects container metrics
type ContainerCollector struct {
	*BaseCollector
	runtime string
}

// ContainerCollectorConfig holds configuration
type ContainerCollectorConfig struct {
	Enabled bool
	Runtime string
}

// NewContainerCollector creates a new container collector
func NewContainerCollector(config ContainerCollectorConfig) (*ContainerCollector, error) {
	return &ContainerCollector{
		BaseCollector: NewBaseCollector("container", config.Enabled, 2*time.Second),
		runtime:       config.Runtime,
	}, nil
}

// Collect collects container metrics
func (cc *ContainerCollector) Collect(ctx context.Context) ([]*Metric, error) {
	return make([]*Metric, 0), nil
}
