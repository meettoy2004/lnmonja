package collectors

import "context"

// KubernetesCollector collects Kubernetes metrics
type KubernetesCollector struct {
	*BaseCollector
}

// Collect collects Kubernetes metrics
func (kc *KubernetesCollector) Collect(ctx context.Context) ([]*Metric, error) {
	return make([]*Metric, 0), nil
}
