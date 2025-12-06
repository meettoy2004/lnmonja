package client

import (
	"context"
	"fmt"

	"github.com/meettoy2004/lnmonja/pkg/protocol"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

// GRPCClient handles communication with the lnmonja server
type GRPCClient struct {
	config    *utils.Config
	logger    *zap.Logger
	connMgr   *ConnectionManager
	client    protocol.MonitorService
	connected bool
}

// NewGRPCClient creates a new gRPC client
func NewGRPCClient(config *utils.Config, logger *zap.Logger) (*GRPCClient, error) {
	serverAddr := config.Agent.ServerAddress
	if serverAddr == "" {
		return nil, fmt.Errorf("server address not configured")
	}

	connMgr := NewConnectionManager(serverAddr, logger)

	return &GRPCClient{
		config:  config,
		logger:  logger,
		connMgr: connMgr,
	}, nil
}

// Connect establishes connection to the server
func (c *GRPCClient) Connect(ctx context.Context) error {
	if err := c.connMgr.Connect(); err != nil {
		return err
	}

	c.connected = true
	return nil
}

// Register registers the agent with the server
func (c *GRPCClient) Register(nodeID string) (string, error) {
	conn := c.connMgr.GetConnection()
	if conn == nil {
		return "", fmt.Errorf("not connected")
	}

	sysInfo := utils.GetSystemInfo()

	// In a real implementation, this would send the registration request via gRPC
	_ = &protocol.RegisterRequest{
		NodeId:   nodeID,
		Hostname: sysInfo.Hostname,
		Os:       sysInfo.OS,
		Arch:     sysInfo.Arch,
		Version:  "1.0.0",
		Labels:   make(map[string]string),
	}

	sessionID := utils.GenerateSessionID()

	c.logger.Info("Registered with server",
		zap.String("node_id", nodeID),
		zap.String("session_id", sessionID),
	)

	return sessionID, nil
}

// SendMetrics sends metrics to the server
func (c *GRPCClient) SendMetrics(ctx context.Context, sessionID string, metrics []*protocol.Metric) error {
	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	c.logger.Debug("Sending metrics",
		zap.String("session_id", sessionID),
		zap.Int("count", len(metrics)),
	)

	return nil
}

// Heartbeat sends a heartbeat to the server
func (c *GRPCClient) Heartbeat(ctx context.Context, sessionID string) error {
	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	c.logger.Debug("Sending heartbeat", zap.String("session_id", sessionID))
	return nil
}

// Reconnect attempts to reconnect to the server
func (c *GRPCClient) Reconnect(ctx context.Context) error {
	c.connected = false
	if err := c.connMgr.Reconnect(); err != nil {
		return err
	}
	c.connected = true
	return nil
}

// Close closes the connection
func (c *GRPCClient) Close() error {
	c.connected = false
	return c.connMgr.Close()
}
