package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"

	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/internal/storage"
	"github.com/meettoy2004/lnmonja/pkg/protocol"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	config     *utils.Config
	server     *grpc.Server
	listener   net.Listener
	logger     *zap.Logger
	store      storage.Storage
	nodeMgr    *NodeManager
	alertMgr   *AlertManager
	sessions   map[string]*Session
	sessionsMu sync.RWMutex
}

type Session struct {
	NodeID      string
	SessionID   string
	LastSeen    time.Time
	Stream      protocol.MonitorService_StreamMetricsServer
	Labels      map[string]string
	Collectors  []string
	ConnectedAt time.Time
}

func NewGRPCServer(config *utils.Config, store storage.Storage, logger *zap.Logger) (*GRPCServer, error) {
	s := &GRPCServer{
		config:   config,
		logger:   logger,
		store:    store,
		sessions: make(map[string]*Session),
	}

	s.nodeMgr = NewNodeManager(store, logger)
	s.alertMgr = NewAlertManager(config, store, logger)

	return s, nil
}

func (s *GRPCServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.GRPC.Address, s.config.Server.GRPC.Port)

	// Create listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener

	// Setup gRPC options
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.StreamInterceptor(s.streamInterceptor),
		grpc.UnaryInterceptor(s.unaryInterceptor),
	}

	// Add TLS if enabled
	if s.config.Server.GRPC.TLS.Enabled {
		creds, err := s.loadTLSCredentials()
		if err != nil {
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Create gRPC server
	s.server = grpc.NewServer(opts...)
	protocol.RegisterMonitorServiceServer(s.server, s)

	s.logger.Info("Starting gRPC server",
		zap.String("address", addr),
		zap.Bool("tls", s.config.Server.GRPC.TLS.Enabled),
	)

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.logger.Error("gRPC server failed", zap.Error(err))
		}
	}()

	return nil
}

func (s *GRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// Implement gRPC methods
func (s *GRPCServer) Register(ctx context.Context, req *protocol.RegisterRequest) (*protocol.RegisterResponse, error) {
	s.logger.Info("Node registration",
		zap.String("node_id", req.NodeId),
		zap.String("hostname", req.Hostname),
		zap.String("os", req.Os),
	)

	// Validate node
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	// Generate session ID
	sessionID := utils.GenerateSessionID()

	// Store session
	session := &Session{
		NodeID:      req.NodeId,
		SessionID:   sessionID,
		LastSeen:    time.Now(),
		Labels:      req.Labels,
		ConnectedAt: time.Now(),
	}

	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	// Update node in storage
	node := &models.Node{
		ID:        req.NodeId,
		Hostname:  req.Hostname,
		OS:        req.Os,
		Arch:      req.Arch,
		Version:   req.Version,
		Labels:    req.Labels,
		Status:    models.NodeStatusHealthy,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}

	if err := s.store.SaveNode(node); err != nil {
		s.logger.Error("Failed to save node", zap.Error(err))
	}

	// Determine which collectors to enable
	collectorConfigs := s.getCollectorConfigs(req)

	resp := &protocol.RegisterResponse{
		Success:          true,
		Message:          "Registration successful",
		SessionId:        sessionID,
		HeartbeatInterval: int64(s.config.Server.GRPC.HeartbeatInterval.Seconds()),
		Collectors:       collectorConfigs,
	}

	return resp, nil
}

func (s *GRPCServer) StreamMetrics(stream protocol.MonitorService_StreamMetricsServer) error {
	// First message should contain session ID
	firstMsg, err := stream.Recv()
	if err != nil {
		return status.Error(codes.InvalidArgument, "failed to receive first message")
	}

	sessionID := firstMsg.SessionId
	if sessionID == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Get session
	s.sessionsMu.RLock()
	session, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		return status.Error(codes.Unauthenticated, "invalid session")
	}

	session.Stream = stream
	session.LastSeen = time.Now()

	s.logger.Info("Starting metric stream",
		zap.String("node_id", session.NodeID),
		zap.String("session_id", sessionID),
	)

	// Start heartbeat goroutine
	heartbeatCtx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go s.handleHeartbeat(heartbeatCtx, session)

	// Process incoming metrics
	for {
		batch, err := stream.Recv()
		if err != nil {
			s.logger.Info("Stream closed",
				zap.String("node_id", session.NodeID),
				zap.Error(err),
			)
			break
		}

		session.LastSeen = time.Now()

		// Process metrics in background
		go s.processMetrics(session, batch)
	}

	// Cleanup session
	s.sessionsMu.Lock()
	delete(s.sessions, sessionID)
	s.sessionsMu.Unlock()

	return nil
}

// Heartbeat handles heartbeat requests from agents
func (s *GRPCServer) Heartbeat(ctx context.Context, req *protocol.HeartbeatRequest) (*protocol.HeartbeatResponse, error) {
	// Get session
	s.sessionsMu.RLock()
	session, exists := s.sessions[req.SessionId]
	s.sessionsMu.RUnlock()

	if !exists {
		return nil, status.Error(codes.Unauthenticated, "invalid session")
	}

	// Update last seen time
	session.LastSeen = time.Now()

	// Update node status
	s.nodeMgr.UpdateNodeStatus(session.NodeID, models.NodeStatusHealthy)

	return &protocol.HeartbeatResponse{
		Alive:         true,
		NextHeartbeat: time.Now().Add(s.config.Server.GRPC.HeartbeatInterval).Unix(),
	}, nil
}

// UpdateConfig handles configuration update requests
func (s *GRPCServer) UpdateConfig(ctx context.Context, req *protocol.ConfigUpdate) (*protocol.ConfigAck, error) {
	s.logger.Info("Config update received",
		zap.String("node_id", req.NodeId),
		zap.Bool("restart_required", req.RestartRequired),
	)

	// In a real implementation, this would apply the config update
	// For now, we just acknowledge it
	return &protocol.ConfigAck{
		Success: true,
		Message: "Configuration update acknowledged",
	}, nil
}

func (s *GRPCServer) processMetrics(session *Session, batch *protocol.MetricBatch) {
	// Convert protobuf metrics to internal models
	metrics := make([]*models.Metric, 0, len(batch.Metrics))

	for _, pbMetric := range batch.Metrics {
		metric := &models.Metric{
			NodeID:    session.NodeID,
			Name:      pbMetric.Name,
			Value:     pbMetric.Value,
			Timestamp: time.Unix(0, pbMetric.Timestamp),
			Labels:    pbMetric.Labels,
			Type:      models.MetricType(pbMetric.Type),
			Help:      pbMetric.Help,
			Unit:      pbMetric.Unit,
		}
		metrics = append(metrics, metric)
	}

	// Store metrics
	if err := s.store.WriteMetrics(metrics); err != nil {
		s.logger.Error("Failed to store metrics",
			zap.String("node_id", session.NodeID),
			zap.Error(err),
		)
	}

	// Check alerts
	s.alertMgr.CheckMetrics(session.NodeID, metrics)

	// Update node status
	s.nodeMgr.UpdateNodeStatus(session.NodeID, models.NodeStatusHealthy)
}

func (s *GRPCServer) handleHeartbeat(ctx context.Context, session *Session) {
	ticker := time.NewTicker(s.config.Server.GRPC.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if session is still active
			if time.Since(session.LastSeen) > s.config.Server.GRPC.HeartbeatTimeout {
				s.logger.Warn("Node heartbeat timeout",
					zap.String("node_id", session.NodeID),
					zap.Duration("timeout", s.config.Server.GRPC.HeartbeatTimeout),
				)
				s.nodeMgr.UpdateNodeStatus(session.NodeID, models.NodeStatusUnhealthy)
				return
			}
		}
	}
}

func (s *GRPCServer) getCollectorConfigs(req *protocol.RegisterRequest) []*protocol.CollectorConfig {
	configs := []*protocol.CollectorConfig{}

	// Default collectors
	defaultCollectors := map[string]*protocol.CollectorConfig{
		"system": {
			Name:     "system",
			Enabled:  true,
			Interval: 1000, // 1 second in milliseconds
			Params: map[string]string{
				"include_cpu":    "true",
				"include_memory": "true",
				"include_disk":   "true",
			},
		},
		"process": {
			Name:     "process",
			Enabled:  true,
			Interval: 5000, // 5 seconds
		},
	}

	// Check if node has docker
	if s.hasDocker() {
		defaultCollectors["container"] = &protocol.CollectorConfig{
			Name:     "container",
			Enabled:  true,
			Interval: 2000,
			Params: map[string]string{
				"runtime": "docker",
			},
		}
	}

	// Add all enabled collectors
	for _, collector := range defaultCollectors {
		configs = append(configs, collector)
	}

	return configs
}

func (s *GRPCServer) hasDocker() bool {
	// Check if Docker socket exists
	// This is a simplified check
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		return true
	}
	return false
}

func (s *GRPCServer) loadTLSCredentials() (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(
		s.config.Server.GRPC.TLS.CertFile,
		s.config.Server.GRPC.TLS.KeyFile,
	)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    x509.NewCertPool(),
	}

	// Load CA certificate
	if caCert, err := ioutil.ReadFile(s.config.Server.GRPC.TLS.ClientCAFile); err == nil {
		config.ClientCAs.AppendCertsFromPEM(caCert)
	}

	return credentials.NewTLS(config), nil
}

func (s *GRPCServer) streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Authentication and logging for streaming
	start := time.Now()
	err := handler(srv, ss)
	duration := time.Since(start)

	s.logger.Debug("Stream request",
		zap.String("method", info.FullMethod),
		zap.Duration("duration", duration),
		zap.Error(err),
	)

	return err
}

func (s *GRPCServer) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Authentication and logging for unary RPCs
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	s.logger.Debug("Unary request",
		zap.String("method", info.FullMethod),
		zap.Duration("duration", duration),
		zap.Error(err),
	)

	return resp, err
}