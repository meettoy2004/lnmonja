package collectors

import "context"

// DiskCollector collects disk metrics
type DiskCollector struct {
	*BaseCollector
}

// Collect collects disk metrics
func (dc *DiskCollector) Collect(ctx context.Context) ([]*Metric, error) {
	return make([]*Metric, 0), nil
}
