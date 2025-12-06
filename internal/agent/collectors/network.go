package collectors

import "context"

// NetworkCollector collects network metrics
type NetworkCollector struct {
	*BaseCollector
}

// Collect collects network metrics
func (nc *NetworkCollector) Collect(ctx context.Context) ([]*Metric, error) {
	return make([]*Metric, 0), nil
}
