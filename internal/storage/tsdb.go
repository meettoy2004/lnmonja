package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

// Storage interface defines the methods for metric storage
type Storage interface {
	WriteMetrics(metrics []*models.Metric) error
	QueryMetrics(query *models.Query) ([]*models.TimeSeries, error)
	SaveNode(node *models.Node) error
	GetNode(nodeID string) (*models.Node, error)
	ListNodes() ([]*models.Node, error)
	SaveAlert(alert *models.Alert) error
	GetAlerts(filter *models.AlertFilter) ([]*models.Alert, error)
	Close() error
}

// TimeSeriesDB is the main time-series database implementation
type TimeSeriesDB struct {
	config      *utils.StorageConfig
	logger      *zap.Logger
	badgerStore *BadgerStore
	nodes       map[string]*models.Node
	nodesMu     sync.RWMutex
	retention   *RetentionManager
	compression *CompressionEngine
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewTimeSeriesDB creates a new time-series database instance
func NewTimeSeriesDB(config *utils.StorageConfig, logger *zap.Logger) (*TimeSeriesDB, error) {
	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	// Initialize BadgerDB store
	badgerStore, err := NewBadgerStore(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create badger store: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	tsdb := &TimeSeriesDB{
		config:      config,
		logger:      logger,
		badgerStore: badgerStore,
		nodes:       make(map[string]*models.Node),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize retention manager
	tsdb.retention = NewRetentionManager(config, badgerStore, logger)

	// Initialize compression engine if enabled
	if config.Compression {
		tsdb.compression = NewCompressionEngine(config, logger)
	}

	// Start background jobs
	tsdb.wg.Add(1)
	go tsdb.runRetentionJob()

	logger.Info("Time-series database initialized",
		zap.String("path", config.Path),
		zap.Bool("compression", config.Compression),
	)

	return tsdb, nil
}

// WriteMetrics writes a batch of metrics to the database
func (db *TimeSeriesDB) WriteMetrics(metrics []*models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	// Compress metrics if compression is enabled
	if db.compression != nil {
		compressedMetrics, err := db.compression.CompressMetrics(metrics)
		if err != nil {
			db.logger.Warn("Failed to compress metrics, writing uncompressed",
				zap.Error(err),
			)
		} else {
			return db.badgerStore.WriteCompressedMetrics(compressedMetrics)
		}
	}

	// Write uncompressed metrics
	return db.badgerStore.WriteMetrics(metrics)
}

// QueryMetrics queries metrics based on the given query
func (db *TimeSeriesDB) QueryMetrics(query *models.Query) ([]*models.TimeSeries, error) {
	if query == nil {
		return nil, fmt.Errorf("query is nil")
	}

	// Build query string from Query struct
	queryStr := query.MetricName
	if len(query.Labels) > 0 {
		// Add label filters to query string
		// Format: metric_name{label1="value1",label2="value2"}
		var labelPairs []string
		for k, v := range query.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf("%s=\"%s\"", k, v))
		}
		queryStr = fmt.Sprintf("%s{%s}", query.MetricName, string(labelPairs[0]))
	}

	return db.badgerStore.QueryMetrics(queryStr, query.StartTime, query.EndTime, query.Step)
}

// SaveNode saves a node to the database
func (db *TimeSeriesDB) SaveNode(node *models.Node) error {
	if node == nil || node.ID == "" {
		return fmt.Errorf("invalid node: nil or empty ID")
	}

	// Update in-memory cache
	db.nodesMu.Lock()
	db.nodes[node.ID] = node
	db.nodesMu.Unlock()

	// Persist to storage
	return db.badgerStore.SaveNode(node)
}

// GetNode retrieves a node by ID
func (db *TimeSeriesDB) GetNode(nodeID string) (*models.Node, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("node ID is required")
	}

	// Check in-memory cache first
	db.nodesMu.RLock()
	node, exists := db.nodes[nodeID]
	db.nodesMu.RUnlock()

	if exists {
		return node, nil
	}

	// Fetch from storage
	node, err := db.badgerStore.GetNode(nodeID)
	if err != nil {
		return nil, err
	}

	// Update cache
	db.nodesMu.Lock()
	db.nodes[nodeID] = node
	db.nodesMu.Unlock()

	return node, nil
}

// ListNodes returns all registered nodes
func (db *TimeSeriesDB) ListNodes() ([]*models.Node, error) {
	return db.badgerStore.ListNodes()
}

// SaveAlert saves an alert to the database
func (db *TimeSeriesDB) SaveAlert(alert *models.Alert) error {
	if alert == nil {
		return fmt.Errorf("alert is nil")
	}
	return db.badgerStore.SaveAlert(alert)
}

// GetAlerts retrieves alerts based on the filter
func (db *TimeSeriesDB) GetAlerts(filter *models.AlertFilter) ([]*models.Alert, error) {
	return db.badgerStore.GetAlerts(filter)
}

// Close closes the database and releases resources
func (db *TimeSeriesDB) Close() error {
	db.logger.Info("Shutting down time-series database...")

	// Cancel context to stop background jobs
	db.cancel()

	// Wait for background jobs to finish
	db.wg.Wait()

	// Close BadgerDB
	if db.badgerStore != nil {
		if err := db.badgerStore.Close(); err != nil {
			return fmt.Errorf("failed to close badger store: %w", err)
		}
	}

	db.logger.Info("Time-series database closed")
	return nil
}

// runRetentionJob periodically runs retention cleanup
func (db *TimeSeriesDB) runRetentionJob() {
	defer db.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	db.logger.Info("Retention job started")

	for {
		select {
		case <-db.ctx.Done():
			db.logger.Info("Retention job stopped")
			return
		case <-ticker.C:
			if err := db.retention.Cleanup(); err != nil {
				db.logger.Error("Retention cleanup failed", zap.Error(err))
			} else {
				db.logger.Debug("Retention cleanup completed")
			}
		}
	}
}

// GetStats returns database statistics
func (db *TimeSeriesDB) GetStats() (*DBStats, error) {
	return db.badgerStore.GetStats()
}

// DBStats contains database statistics
type DBStats struct {
	TotalMetrics   int64
	TotalNodes     int64
	TotalAlerts    int64
	DiskUsageBytes int64
	OldestMetric   time.Time
	NewestMetric   time.Time
}
