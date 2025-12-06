package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/meettoy2004/lnmonja/internal/agent/collectors"
	"github.com/meettoy2004/lnmonja/internal/agent/client"
	"github.com/meettoy2004/lnmonja/pkg/protocol"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

type Agent struct {
	config     *utils.Config
	logger     *zap.Logger
	client     *client.GRPCClient
	collectors map[string]collectors.Collector
	wg         sync.WaitGroup
	cancel     context.CancelFunc
	ctx        context.Context
	metricsCh  chan []*collectors.Metric
	nodeID     string
	sessionID  string
}

func NewAgent(config *utils.Config, logger *zap.Logger) (*Agent, error) {
	agent := &Agent{
		config:     config,
		logger:     logger,
		collectors: make(map[string]collectors.Collector),
		metricsCh:  make(chan []*collectors.Metric, 1000),
	}

	// Generate node ID if not provided
	if config.Agent.NodeID == "" {
		hostname, _ := os.Hostname()
		config.Agent.NodeID = hostname
	}
	agent.nodeID = config.Agent.NodeID

	// Initialize client
	grpcClient, err := client.NewGRPCClient(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}
	agent.client = grpcClient

	// Initialize collectors
	if err := agent.initCollectors(); err != nil {
		return nil, fmt.Errorf("failed to initialize collectors: %w", err)
	}

	return agent, nil
}

func (a *Agent) Start(ctx context.Context) error {
	a.ctx, a.cancel = context.WithCancel(ctx)

	// Connect to server
	if err := a.client.Connect(a.ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Register with server
	sessionID, err := a.client.Register(a.nodeID)
	if err != nil {
		return fmt.Errorf("failed to register with server: %w", err)
	}
	a.sessionID = sessionID

	a.logger.Info("Agent registered",
		zap.String("node_id", a.nodeID),
		zap.String("session_id", sessionID),
	)

	// Start collectors
	for name, collector := range a.collectors {
		if collector.Enabled() {
			a.wg.Add(1)
			go a.runCollector(name, collector)
		}
	}

	// Start metric processor
	a.wg.Add(1)
	go a.processMetrics()

	// Start heartbeat
	a.wg.Add(1)
	go a.heartbeat()

	a.logger.Info("Agent started successfully")
	return nil
}

func (a *Agent) Stop(ctx context.Context) error {
	a.logger.Info("Stopping agent...")

	// Cancel context to stop all goroutines
	if a.cancel != nil {
		a.cancel()
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("All goroutines stopped")
	case <-ctx.Done():
		a.logger.Warn("Timeout waiting for goroutines to stop")
	}

	// Close client connection
	if a.client != nil {
		a.client.Close()
	}

	close(a.metricsCh)
	a.logger.Info("Agent stopped")
	return nil
}

func (a *Agent) initCollectors() error {
	// System collector
	if a.config.Collectors.System.Enabled {
		sysConfig := collectors.SystemConfig{
			Enabled:  a.config.Collectors.System.Enabled,
			Interval: a.config.Collectors.System.Interval,
			Metrics:  a.config.Collectors.System.Metrics,
		}
		sysCollector, err := collectors.NewSystemCollector(sysConfig)
		if err != nil {
			return fmt.Errorf("failed to create system collector: %w", err)
		}
		a.collectors["system"] = sysCollector
	}

	// Process collector
	if a.config.Collectors.Process.Enabled {
		procConfig := collectors.ProcessCollectorConfig{
			Enabled:      a.config.Collectors.Process.Enabled,
			Interval:     a.config.Collectors.Process.Interval,
			MaxProcesses: a.config.Collectors.Process.MaxProcesses,
		}
		procCollector, err := collectors.NewProcessCollector(procConfig)
		if err != nil {
			return fmt.Errorf("failed to create process collector: %w", err)
		}
		a.collectors["process"] = procCollector
	}

	// Container collector
	if a.config.Collectors.Container.Enabled {
		containerConfig := collectors.ContainerCollectorConfig{
			Enabled: a.config.Collectors.Container.Enabled,
			Runtime: a.config.Collectors.Container.Runtime,
		}
		containerCollector, err := collectors.NewContainerCollector(containerConfig)
		if err != nil {
			a.logger.Warn("Failed to create container collector", zap.Error(err))
			// Don't fail agent if container collector fails
		} else {
			a.collectors["container"] = containerCollector
		}
	}

	a.logger.Info("Collectors initialized",
		zap.Int("count", len(a.collectors)),
		zap.Strings("collectors", a.getCollectorNames()),
	)

	return nil
}

func (a *Agent) runCollector(name string, collector collectors.Collector) {
	defer a.wg.Done()

	interval := collector.Interval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.logger.Debug("Starting collector",
		zap.String("name", name),
		zap.Duration("interval", interval),
	)

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Debug("Collector stopped", zap.String("name", name))
			return
		case <-ticker.C:
			start := time.Now()
			
			metrics, err := collector.Collect(a.ctx)
			if err != nil {
				a.logger.Error("Collector failed",
					zap.String("name", name),
					zap.Error(err),
				)
				continue
			}
			
			// Add node and collector labels
			for _, metric := range metrics {
				if metric.Labels == nil {
					metric.Labels = make(map[string]string)
				}
				metric.Labels["node"] = a.nodeID
				metric.Labels["collector"] = name
			}
			
			// Send metrics to channel
			select {
			case a.metricsCh <- metrics:
				// Metrics sent successfully
			default:
				a.logger.Warn("Metrics channel full, dropping batch",
					zap.String("collector", name),
					zap.Int("metrics", len(metrics)),
				)
			}
			
			collectorDuration := time.Since(start)
			if collectorDuration > interval {
				a.logger.Warn("Collector taking longer than interval",
					zap.String("name", name),
					zap.Duration("duration", collectorDuration),
					zap.Duration("interval", interval),
				)
			}
		}
	}
}

func (a *Agent) processMetrics() {
	defer a.wg.Done()

	batchSize := a.config.Agent.BatchSize
	maxWait := a.config.Agent.MaxBatchWait
	batch := make([]*collectors.Metric, 0, batchSize)
	batchTimer := time.NewTimer(maxWait)

	for {
		select {
		case <-a.ctx.Done():
			// Send remaining metrics before exiting
			if len(batch) > 0 {
				a.sendMetrics(batch)
			}
			return
			
		case metrics := <-a.metricsCh:
			batch = append(batch, metrics...)
			
			// Send batch if size limit reached
			if len(batch) >= batchSize {
				a.sendMetrics(batch)
				batch = make([]*collectors.Metric, 0, batchSize)
				batchTimer.Reset(maxWait)
			}
			
		case <-batchTimer.C:
			// Send batch if timeout reached and we have metrics
			if len(batch) > 0 {
				a.sendMetrics(batch)
				batch = make([]*collectors.Metric, 0, batchSize)
			}
			batchTimer.Reset(maxWait)
		}
	}
}

func (a *Agent) sendMetrics(metrics []*collectors.Metric) {
	// Convert to protobuf format
	pbMetrics := make([]*protocol.Metric, 0, len(metrics))
	now := time.Now().UnixNano()
	
	for _, metric := range metrics {
		pbMetric := &protocol.Metric{
			Name:      metric.Name,
			Value:     metric.Value,
			Timestamp: metric.Timestamp,
			Labels:    metric.Labels,
			Type:      protocol.MetricType(metric.Type),
			Help:      metric.Help,
			Unit:      metric.Unit,
		}
		
		// Use current time if timestamp is zero
		if pbMetric.Timestamp == 0 {
			pbMetric.Timestamp = now
		}
		
		pbMetrics = append(pbMetrics, pbMetric)
	}
	
	// Send to server
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	
	if err := a.client.SendMetrics(ctx, a.sessionID, pbMetrics); err != nil {
		a.logger.Error("Failed to send metrics",
			zap.Error(err),
			zap.Int("metrics", len(pbMetrics)),
		)
		
		// Buffer metrics for retry
		a.bufferMetrics(metrics)
	} else {
		a.logger.Debug("Metrics sent successfully",
			zap.Int("count", len(pbMetrics)),
		)
	}
}

func (a *Agent) bufferMetrics(metrics []*collectors.Metric) {
	// TODO: Implement disk-backed buffer for retry
	// For now, just log and drop
	a.logger.Warn("Metrics buffer not implemented, dropping metrics",
		zap.Int("count", len(metrics)),
	)
}

func (a *Agent) heartbeat() {
	defer a.wg.Done()

	interval := a.config.Agent.HeartbeatInterval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
			err := a.client.Heartbeat(ctx, a.sessionID)
			cancel()
			
			if err != nil {
				a.logger.Error("Heartbeat failed", zap.Error(err))
				// Attempt to reconnect
				go a.reconnect()
			}
		}
	}
}

func (a *Agent) reconnect() {
	a.logger.Info("Attempting to reconnect...")
	
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if err := a.client.Reconnect(a.ctx); err != nil {
				a.logger.Error("Reconnect failed", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			
			// Re-register
			sessionID, err := a.client.Register(a.nodeID)
			if err != nil {
				a.logger.Error("Re-register failed", zap.Error(err))
				continue
			}
			
			a.sessionID = sessionID
			a.logger.Info("Reconnected successfully")
			return
		}
	}
}

func (a *Agent) getCollectorNames() []string {
	names := make([]string, 0, len(a.collectors))
	for name := range a.collectors {
		names = append(names, name)
	}
	return names
}