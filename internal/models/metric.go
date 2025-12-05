package models

import (
	"time"
)

type Metric struct {
	ID        string            `json:"id"`
	NodeID    string            `json:"node_id"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
	Type      MetricType        `json:"type"`
	Help      string            `json:"help,omitempty"`
	Unit      string            `json:"unit,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type MetricType int

const (
	MetricTypeGauge MetricType = iota
	MetricTypeCounter
	MetricTypeHistogram
	MetricTypeSummary
)

func (t MetricType) String() string {
	switch t {
	case MetricTypeGauge:
		return "gauge"
	case MetricTypeCounter:
		return "counter"
	case MetricTypeHistogram:
		return "histogram"
	case MetricTypeSummary:
		return "summary"
	default:
		return "unknown"
	}
}

func MetricTypeFromString(s string) MetricType {
	switch s {
	case "gauge":
		return MetricTypeGauge
	case "counter":
		return MetricTypeCounter
	case "histogram":
		return MetricTypeHistogram
	case "summary":
		return MetricTypeSummary
	default:
		return MetricTypeGauge
	}
}

type TimeSeries struct {
	Labels  map[string]string `json:"labels"`
	Samples []Sample          `json:"samples"`
}

type Sample struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type Node struct {
	ID        string            `json:"id"`
	Hostname  string            `json:"hostname"`
	OS        string            `json:"os"`
	Arch      string            `json:"arch"`
	Version   string            `json:"version"`
	Labels    map[string]string `json:"labels"`
	Status    NodeStatus        `json:"status"`
	LastSeen  time.Time         `json:"last_seen"`
	CreatedAt time.Time         `json:"created_at"`
}

type NodeStatus int

const (
	NodeStatusUnknown NodeStatus = iota
	NodeStatusHealthy
	NodeStatusDegraded
	NodeStatusUnhealthy
	NodeStatusOffline
)

type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Expression  string            `json:"expression"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       AlertState        `json:"state"`
	Value       float64           `json:"value"`
	ActiveAt    time.Time         `json:"active_at"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type AlertState int

const (
	AlertStateInactive AlertState = iota
	AlertStatePending
	AlertStateFiring
	AlertStateResolved
)