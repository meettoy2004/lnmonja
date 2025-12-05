package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/meettoy2004/lnmonja/internal/server"
	"github.com/meettoy2004/lnmonja/internal/storage"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "/etc/lnmonja/config.yaml", "Path to config file")
	version    = flag.Bool("version", false, "Show version")
	Version    = "dev"
	BuildTime  = "unknown"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("lnmonja Server v%s (built: %s)\n", Version, BuildTime)
		return
	}

	// Load configuration
	config, err := utils.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup logger
	logger, err := utils.NewLogger(config.Logging)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting lnmonja Server",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	// Initialize storage
	store, err := storage.NewTimeSeriesDB(config.Storage)
	if err != nil {
		logger.Fatal("Failed to initialize storage", zap.Error(err))
	}
	defer store.Close()

	// Create server instance
	srv, err := server.NewServer(config, store, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Start servers
	go func() {
		if err := srv.StartGRPC(); err != nil {
			logger.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	go func() {
		if err := srv.StartHTTP(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	go func() {
		if err := srv.StartWebSocket(); err != nil {
			logger.Fatal("Failed to start WebSocket server", zap.Error(err))
		}
	}()

	// Start background jobs
	go srv.StartAlertEngine()
	go srv.StartRetentionJob()
	go srv.StartHealthCheck()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown gracefully", zap.Error(err))
	}

	logger.Info("Server stopped")
}