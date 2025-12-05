package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/internal/storage"
	"go.uber.org/zap"
)

// NodeManager manages node lifecycle and health
type NodeManager struct {
	store   storage.Storage
	logger  *zap.Logger
	nodes   map[string]*NodeInfo
	nodesMu sync.RWMutex
}

// NodeInfo contains runtime information about a node
type NodeInfo struct {
	Node         *models.Node
	LastHeartbeat time.Time
	IsHealthy    bool
	SessionCount int
	MetricsCount int64
	Collectors   []string
}

// NewNodeManager creates a new node manager
func NewNodeManager(store storage.Storage, logger *zap.Logger) *NodeManager {
	return &NodeManager{
		store:  store,
		logger: logger,
		nodes:  make(map[string]*NodeInfo),
	}
}

// RegisterNode registers a new node
func (nm *NodeManager) RegisterNode(node *models.Node) error {
	if node == nil || node.ID == "" {
		return fmt.Errorf("invalid node")
	}

	nm.nodesMu.Lock()
	defer nm.nodesMu.Unlock()

	// Check if node already exists
	if existing, exists := nm.nodes[node.ID]; exists {
		nm.logger.Info("Node re-registering",
			zap.String("node_id", node.ID),
			zap.Int("session_count", existing.SessionCount+1),
		)
		existing.Node = node
		existing.LastHeartbeat = time.Now()
		existing.SessionCount++
		existing.IsHealthy = true
	} else {
		nm.logger.Info("New node registered",
			zap.String("node_id", node.ID),
			zap.String("hostname", node.Hostname),
		)
		nm.nodes[node.ID] = &NodeInfo{
			Node:          node,
			LastHeartbeat: time.Now(),
			IsHealthy:     true,
			SessionCount:  1,
			MetricsCount:  0,
		}
	}

	// Save to storage
	return nm.store.SaveNode(node)
}

// UpdateNodeStatus updates the health status of a node
func (nm *NodeManager) UpdateNodeStatus(nodeID string, status models.NodeStatus) error {
	nm.nodesMu.Lock()
	defer nm.nodesMu.Unlock()

	nodeInfo, exists := nm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	oldStatus := nodeInfo.Node.Status
	nodeInfo.Node.Status = status
	nodeInfo.Node.LastSeen = time.Now()

	// Update health flag
	nodeInfo.IsHealthy = (status == models.NodeStatusHealthy)

	if oldStatus != status {
		nm.logger.Info("Node status changed",
			zap.String("node_id", nodeID),
			zap.String("old_status", oldStatus.String()),
			zap.String("new_status", status.String()),
		)
	}

	// Persist to storage
	return nm.store.SaveNode(nodeInfo.Node)
}

// UpdateHeartbeat updates the last heartbeat time for a node
func (nm *NodeManager) UpdateHeartbeat(nodeID string) error {
	nm.nodesMu.Lock()
	defer nm.nodesMu.Unlock()

	nodeInfo, exists := nm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	nodeInfo.LastHeartbeat = time.Now()
	nodeInfo.Node.LastSeen = time.Now()

	// Mark as healthy if it was down
	if !nodeInfo.IsHealthy {
		nodeInfo.IsHealthy = true
		nodeInfo.Node.Status = models.NodeStatusHealthy
		nm.logger.Info("Node recovered",
			zap.String("node_id", nodeID),
		)
	}

	return nm.store.SaveNode(nodeInfo.Node)
}

// GetNode returns information about a node
func (nm *NodeManager) GetNode(nodeID string) (*NodeInfo, error) {
	nm.nodesMu.RLock()
	defer nm.nodesMu.RUnlock()

	nodeInfo, exists := nm.nodes[nodeID]
	if !exists {
		// Try to load from storage
		node, err := nm.store.GetNode(nodeID)
		if err != nil {
			return nil, fmt.Errorf("node %s not found", nodeID)
		}

		// Add to cache
		nm.nodesMu.RUnlock()
		nm.nodesMu.Lock()
		nodeInfo = &NodeInfo{
			Node:          node,
			LastHeartbeat: node.LastSeen,
			IsHealthy:     node.Status == models.NodeStatusHealthy,
		}
		nm.nodes[nodeID] = nodeInfo
		nm.nodesMu.Unlock()
		nm.nodesMu.RLock()
	}

	return nodeInfo, nil
}

// ListNodes returns all registered nodes
func (nm *NodeManager) ListNodes() []*NodeInfo {
	nm.nodesMu.RLock()
	defer nm.nodesMu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(nm.nodes))
	for _, nodeInfo := range nm.nodes {
		nodes = append(nodes, nodeInfo)
	}

	return nodes
}

// CheckHealth checks the health of all nodes
func (nm *NodeManager) CheckHealth(timeout time.Duration) {
	nm.nodesMu.Lock()
	defer nm.nodesMu.Unlock()

	now := time.Now()

	for nodeID, nodeInfo := range nm.nodes {
		timeSinceHeartbeat := now.Sub(nodeInfo.LastHeartbeat)

		if timeSinceHeartbeat > timeout {
			if nodeInfo.IsHealthy {
				nm.logger.Warn("Node unhealthy - heartbeat timeout",
					zap.String("node_id", nodeID),
					zap.Duration("time_since_heartbeat", timeSinceHeartbeat),
				)
				nodeInfo.IsHealthy = false
				nodeInfo.Node.Status = models.NodeStatusUnhealthy

				// Persist status change
				if err := nm.store.SaveNode(nodeInfo.Node); err != nil {
					nm.logger.Error("Failed to save node status",
						zap.String("node_id", nodeID),
						zap.Error(err),
					)
				}
			}

			// Mark as offline if no heartbeat for extended period
			if timeSinceHeartbeat > timeout*3 {
				if nodeInfo.Node.Status != models.NodeStatusOffline {
					nm.logger.Warn("Node offline",
						zap.String("node_id", nodeID),
						zap.Duration("time_since_heartbeat", timeSinceHeartbeat),
					)
					nodeInfo.Node.Status = models.NodeStatusOffline

					if err := nm.store.SaveNode(nodeInfo.Node); err != nil {
						nm.logger.Error("Failed to save node status",
							zap.String("node_id", nodeID),
							zap.Error(err),
						)
					}
				}
			}
		}
	}
}

// IncrementMetricCount increments the metric count for a node
func (nm *NodeManager) IncrementMetricCount(nodeID string, count int64) {
	nm.nodesMu.Lock()
	defer nm.nodesMu.Unlock()

	if nodeInfo, exists := nm.nodes[nodeID]; exists {
		nodeInfo.MetricsCount += count
	}
}

// GetStats returns statistics about all nodes
func (nm *NodeManager) GetStats() *NodeStats {
	nm.nodesMu.RLock()
	defer nm.nodesMu.RUnlock()

	stats := &NodeStats{
		TotalNodes:    len(nm.nodes),
		HealthyNodes:  0,
		UnhealthyNodes: 0,
		OfflineNodes:  0,
	}

	for _, nodeInfo := range nm.nodes {
		switch nodeInfo.Node.Status {
		case models.NodeStatusHealthy:
			stats.HealthyNodes++
		case models.NodeStatusUnhealthy, models.NodeStatusDegraded:
			stats.UnhealthyNodes++
		case models.NodeStatusOffline:
			stats.OfflineNodes++
		}
	}

	return stats
}

// NodeStats contains node statistics
type NodeStats struct {
	TotalNodes     int
	HealthyNodes   int
	UnhealthyNodes int
	OfflineNodes   int
}
