package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/meettoy2004/lnmonja/internal/server/api"
	"github.com/meettoy2004/lnmonja/internal/storage"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

// Server represents the main lnmonja server
type Server struct {
	config    *utils.Config
	logger    *zap.Logger
	store     storage.Storage
	grpc      *GRPCServer
	http      *http.Server
	websocket *api.WebSocketServer
	nodeMgr   *NodeManager
	alertMgr  *AlertManager
}

// NewServer creates a new server instance
func NewServer(config *utils.Config, store storage.Storage, logger *zap.Logger) (*Server, error) {
	s := &Server{
		config: config,
		logger: logger,
		store:  store,
	}

	// Initialize node manager
	s.nodeMgr = NewNodeManager(store, logger)

	// Initialize alert manager
	s.alertMgr = NewAlertManager(config, store, logger)

	// Initialize gRPC server
	grpcServer, err := NewGRPCServer(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC server: %w", err)
	}
	s.grpc = grpcServer

	// Initialize WebSocket server
	s.websocket = api.NewWebSocketServer(store, logger)

	// Initialize HTTP server
	s.http = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.HTTP.Address, config.Server.HTTP.Port),
		Handler:      s.setupHTTPRoutes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s, nil
}

// StartGRPC starts the gRPC server
func (s *Server) StartGRPC() error {
	return s.grpc.Start()
}

// StartHTTP starts the HTTP server
func (s *Server) StartHTTP() error {
	s.logger.Info("Starting HTTP server", zap.String("addr", s.http.Addr))
	return s.http.ListenAndServe()
}

// StartWebSocket starts the WebSocket server
func (s *Server) StartWebSocket() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.WebSocket.Address, s.config.Server.WebSocket.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.websocket.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("Starting WebSocket server", zap.String("addr", addr))
	return server.ListenAndServe()
}

// StartAlertEngine starts the alert engine
func (s *Server) StartAlertEngine() {
	s.logger.Info("Starting alert engine")
	// The alert engine is event-driven and doesn't need a separate goroutine
	// It will be called by the gRPC server when metrics are received
}

// StartRetentionJob starts the data retention job
func (s *Server) StartRetentionJob() {
	s.logger.Info("Starting retention job")
	// The retention job is handled by the TimeSeriesDB internally
}

// StartHealthCheck starts the health check routine
func (s *Server) StartHealthCheck() {
	s.logger.Info("Starting health check")
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Check node health
			timeout := s.config.Server.GRPC.HeartbeatTimeout
			if timeout == 0 {
				timeout = 90 * time.Second
			}
			s.nodeMgr.CheckHealth(timeout)
		}
	}()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	// Stop gRPC server
	if s.grpc != nil {
		s.grpc.Stop()
	}

	// Stop HTTP server
	if s.http != nil {
		if err := s.http.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to shutdown HTTP server", zap.Error(err))
		}
	}

	// Stop WebSocket server
	if s.websocket != nil {
		if err := s.websocket.Close(); err != nil {
			s.logger.Error("Failed to close WebSocket server", zap.Error(err))
		}
	}

	return nil
}

// setupHTTPRoutes sets up HTTP routes
func (s *Server) setupHTTPRoutes() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Metrics endpoint (for Prometheus scraping)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Placeholder for Prometheus metrics
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Prometheus metrics\n"))
	})

	// API endpoints
	mux.HandleFunc("/api/v1/nodes", s.handleNodes)
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)
	mux.HandleFunc("/api/v1/query", s.handleQuery)

	return mux
}

// HTTP handlers
func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.store.ListNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Marshal and write nodes
	fmt.Fprintf(w, `{"nodes":%d}`, len(nodes))
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := s.alertMgr.GetActiveAlerts()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"alerts":%d}`, len(alerts))
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
