package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/lnmonja/internal/agent"
	"github.com/yourusername/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "/etc/lnmonja/config.yaml", "Path to config file")
	debug      = flag.Bool("debug", false, "Enable debug mode")
	version    = flag.Bool("version", false, "Show version")
	Version    = "dev"
	BuildTime  = "unknown"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("lnmonja Agent v%s (built: %s)\n", Version, BuildTime)
		return
	}

	// Load configuration
	config, err := utils.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *debug {
		config.Logging.Level = "debug"
	}

	// Setup logger
	logger, err := utils.NewLogger(config.Logging)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting lnmonja Agent",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	// Create agent instance
	ag, err := agent.NewAgent(config, logger)
	if err != nil {
		logger.Fatal("Failed to create agent", zap.Error(err))
	}

	// Start agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := ag.Start(ctx); err != nil {
		logger.Fatal("Failed to start agent", zap.Error(err))
	}

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down agent...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := ag.Stop(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown gracefully", zap.Error(err))
	}

	logger.Info("Agent stopped")
}