package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// ConnectionManager manages gRPC connection lifecycle
type ConnectionManager struct {
	address    string
	conn       *grpc.ClientConn
	logger     *zap.Logger
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	reconnectC chan struct{}
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(address string, logger *zap.Logger) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionManager{
		address:    address,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		reconnectC: make(chan struct{}, 1),
	}
}

// Connect establishes a connection to the server
func (cm *ConnectionManager) Connect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.conn != nil {
		state := cm.conn.GetState()
		if state == connectivity.Ready || state == connectivity.Connecting {
			return nil
		}
	}

	conn, err := grpc.Dial(
		cm.address,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", cm.address, err)
	}

	cm.conn = conn
	cm.logger.Info("Connected to server", zap.String("address", cm.address))

	// Start connection monitor
	go cm.monitorConnection()

	return nil
}

// GetConnection returns the current connection
func (cm *ConnectionManager) GetConnection() *grpc.ClientConn {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.conn
}

// Close closes the connection
func (cm *ConnectionManager) Close() error {
	cm.cancel()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.conn != nil {
		return cm.conn.Close()
	}

	return nil
}

// Reconnect attempts to reconnect to the server
func (cm *ConnectionManager) Reconnect() error {
	cm.logger.Info("Attempting to reconnect...")

	cm.mu.Lock()
	if cm.conn != nil {
		cm.conn.Close()
		cm.conn = nil
	}
	cm.mu.Unlock()

	return cm.Connect()
}

// monitorConnection monitors the connection state
func (cm *ConnectionManager) monitorConnection() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.mu.RLock()
			conn := cm.conn
			cm.mu.RUnlock()

			if conn != nil {
				state := conn.GetState()
				if state == connectivity.TransientFailure || state == connectivity.Shutdown {
					cm.logger.Warn("Connection in bad state, triggering reconnect",
						zap.String("state", state.String()),
					)
					select {
					case cm.reconnectC <- struct{}{}:
					default:
					}
				}
			}
		case <-cm.reconnectC:
			if err := cm.Reconnect(); err != nil {
				cm.logger.Error("Reconnection failed", zap.Error(err))
			}
		}
	}
}
