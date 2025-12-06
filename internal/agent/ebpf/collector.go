package ebpf

import (
	"context"

	"github.com/meettoy2004/lnmonja/internal/agent/collectors"
)

// EBPFCollector collects metrics using eBPF
type EBPFCollector struct {
	*collectors.BaseCollector
}

// Collect collects eBPF metrics
func (ec *EBPFCollector) Collect(ctx context.Context) ([]*collectors.Metric, error) {
	return make([]*collectors.Metric, 0), nil
}
