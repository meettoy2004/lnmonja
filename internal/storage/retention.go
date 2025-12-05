package storage

import (
	"fmt"
	"time"

	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

// RetentionManager handles data retention policies
type RetentionManager struct {
	config *utils.StorageConfig
	store  *BadgerStore
	logger *zap.Logger
}

// NewRetentionManager creates a new retention manager
func NewRetentionManager(config *utils.StorageConfig, store *BadgerStore, logger *zap.Logger) *RetentionManager {
	return &RetentionManager{
		config: config,
		store:  store,
		logger: logger,
	}
}

// Cleanup removes old metrics based on retention policy
func (rm *RetentionManager) Cleanup() error {
	rm.logger.Info("Starting retention cleanup")

	// Calculate cutoff times for each tier
	now := time.Now()
	cutoffTime := now.Add(-rm.config.RetentionPeriod)

	rm.logger.Debug("Retention cleanup parameters",
		zap.Time("cutoff_time", cutoffTime),
		zap.Duration("retention_period", rm.config.RetentionPeriod),
	)

	// Delete metrics older than retention period
	deleted, err := rm.store.DeleteMetricsOlderThan(cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}

	rm.logger.Info("Retention cleanup completed",
		zap.Int64("deleted_metrics", deleted),
	)

	// Run garbage collection if enabled
	if err := rm.store.RunGC(); err != nil {
		rm.logger.Warn("Failed to run garbage collection", zap.Error(err))
	}

	return nil
}

// ApplyTieringPolicy applies tiered retention (hot/warm/cold)
func (rm *RetentionManager) ApplyTieringPolicy() error {
	if !rm.config.Tiering.Enabled {
		return nil
	}

	now := time.Now()

	// Hot tier: Most recent data, kept in fast storage
	hotCutoff := now.Add(-rm.config.Tiering.HotRetention)

	// Warm tier: Older data, can be compressed more aggressively
	warmCutoff := now.Add(-rm.config.Tiering.WarmRetention)

	// Cold tier: Oldest data, maximum compression or archival
	coldCutoff := now.Add(-rm.config.Tiering.ColdRetention)

	rm.logger.Debug("Applying tiering policy",
		zap.Time("hot_cutoff", hotCutoff),
		zap.Time("warm_cutoff", warmCutoff),
		zap.Time("cold_cutoff", coldCutoff),
	)

	// Move warm data to compressed storage
	if err := rm.store.CompactMetricsInRange(warmCutoff, hotCutoff); err != nil {
		rm.logger.Warn("Failed to compact warm data", zap.Error(err))
	}

	// Archive cold data
	if rm.config.Tiering.ColdPath != "" {
		if err := rm.archiveColdData(coldCutoff, warmCutoff); err != nil {
			rm.logger.Warn("Failed to archive cold data", zap.Error(err))
		}
	}

	return nil
}

// archiveColdData moves old metrics to archive storage
func (rm *RetentionManager) archiveColdData(start, end time.Time) error {
	rm.logger.Info("Archiving cold data",
		zap.Time("start", start),
		zap.Time("end", end),
	)

	// This is a placeholder for actual archival logic
	// In a real implementation, you would:
	// 1. Export metrics in the time range to a file
	// 2. Compress the file
	// 3. Move to cold storage (S3, etc.)
	// 4. Delete from hot storage

	return nil
}

// GetRetentionStats returns statistics about data retention
func (rm *RetentionManager) GetRetentionStats() (*RetentionStats, error) {
	stats := &RetentionStats{
		RetentionPeriod: rm.config.RetentionPeriod,
		TieringEnabled:  rm.config.Tiering.Enabled,
	}

	if rm.config.Tiering.Enabled {
		stats.HotRetention = rm.config.Tiering.HotRetention
		stats.WarmRetention = rm.config.Tiering.WarmRetention
		stats.ColdRetention = rm.config.Tiering.ColdRetention
	}

	return stats, nil
}

// RetentionStats contains retention statistics
type RetentionStats struct {
	RetentionPeriod time.Duration
	TieringEnabled  bool
	HotRetention    time.Duration
	WarmRetention   time.Duration
	ColdRetention   time.Duration
	TotalMetrics    int64
	HotMetrics      int64
	WarmMetrics     int64
	ColdMetrics     int64
}
