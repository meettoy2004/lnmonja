package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/badgerodon/badger/v3"
	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

type BadgerStore struct {
	db     *badger.DB
	config *utils.StorageConfig
	logger *zap.Logger
}

func NewBadgerStore(config *utils.StorageConfig, logger *zap.Logger) (*BadgerStore, error) {
	opts := badger.DefaultOptions(config.Path)
	opts.Logger = &badgerLogger{logger: logger}
	opts.SyncWrites = config.SyncWrites
	opts.ValueLogFileSize = config.ValueLogFileSize
	opts.MemTableSize = config.MemTableSize

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &BadgerStore{
		db:     db,
		config: config,
		logger: logger,
	}

	// Start compaction goroutine
	go store.runCompaction()

	logger.Info("Badger storage initialized",
		zap.String("path", config.Path),
	)

	return store, nil
}

func (s *BadgerStore) WriteMetrics(metrics []*models.Metric) error {
	return s.db.Update(func(txn *badger.Txn) error {
		for _, metric := range metrics {
			key := s.encodeMetricKey(metric)
			value, err := s.encodeMetricValue(metric)
			if err != nil {
				s.logger.Error("Failed to encode metric", zap.Error(err))
				continue
			}

			if err := txn.Set(key, value); err != nil {
				return fmt.Errorf("failed to write metric: %w", err)
			}
		}
		return nil
	})
}

func (s *BadgerStore) QueryMetrics(query string, start, end time.Time, step time.Duration) ([]*models.TimeSeries, error) {
	// Parse query (simplified for now)
	// In production, you'd want to implement a proper query parser
	metricName, filters := parseSimpleQuery(query)

	var series []*models.TimeSeries
	seriesMap := make(map[string]*models.TimeSeries)

	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(fmt.Sprintf("metric:%s:", metricName))
		
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()
			
			// Decode metric from key/value
			metric, err := s.decodeMetric(item)
			if err != nil {
				s.logger.Warn("Failed to decode metric", zap.Error(err))
				continue
			}
			
			// Filter by time range
			if metric.Timestamp.Before(start) || metric.Timestamp.After(end) {
				continue
			}
			
			// Apply filters
			if !s.matchesFilters(metric, filters) {
				continue
			}
			
			// Group by labels
			seriesKey := s.seriesKey(metric.Labels)
			if _, exists := seriesMap[seriesKey]; !exists {
				seriesMap[seriesKey] = &models.TimeSeries{
					Labels: metric.Labels,
					Samples: make([]models.Sample, 0),
				}
			}
			
			// Apply downsampling based on step
			roundedTime := metric.Timestamp.Truncate(step)
			series := seriesMap[seriesKey]
			
			// Find or create sample for this time bucket
			found := false
			for i := range series.Samples {
				if series.Samples[i].Timestamp.Equal(roundedTime) {
					// Aggregate (average for now)
					series.Samples[i].Value = (series.Samples[i].Value + metric.Value) / 2
					found = true
					break
				}
			}
			
			if !found {
				series.Samples = append(series.Samples, models.Sample{
					Timestamp: roundedTime,
					Value:     metric.Value,
				})
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Convert map to slice
	for _, ts := range seriesMap {
		series = append(series, ts)
	}
	
	return series, nil
}

func (s *BadgerStore) encodeMetricKey(metric *models.Metric) []byte {
	// Key format: metric:name:timestamp:labels_hash
	timestamp := metric.Timestamp.UnixNano()
	labelsHash := utils.HashLabels(metric.Labels)
	
	key := fmt.Sprintf("metric:%s:%d:%s", 
		metric.Name, 
		timestamp,
		labelsHash,
	)
	
	return []byte(key)
}

func (s *BadgerStore) encodeMetricValue(metric *models.Metric) ([]byte, error) {
	data := struct {
		Value     float64           `json:"v"`
		Labels    map[string]string `json:"l,omitempty"`
		NodeID    string            `json:"n"`
		Type      string            `json:"t"`
		Help      string            `json:"h,omitempty"`
		Unit      string            `json:"u,omitempty"`
	}{
		Value:  metric.Value,
		Labels: metric.Labels,
		NodeID: metric.NodeID,
		Type:   metric.Type.String(),
		Help:   metric.Help,
		Unit:   metric.Unit,
	}
	
	return json.Marshal(data)
}

func (s *BadgerStore) decodeMetric(item *badger.Item) (*models.Metric, error) {
	var data struct {
		Value  float64           `json:"v"`
		Labels map[string]string `json:"l"`
		NodeID string            `json:"n"`
		Type   string            `json:"t"`
		Help   string            `json:"h"`
		Unit   string            `json:"u"`
	}
	
	err := item.Value(func(val []byte) error {
		return json.Unmarshal(val, &data)
	})
	if err != nil {
		return nil, err
	}
	
	// Parse key to get timestamp
	key := item.Key()
	parts := bytes.Split(key, []byte(":"))
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid key format")
	}
	
	timestamp, err := strconv.ParseInt(string(parts[2]), 10, 64)
	if err != nil {
		return nil, err
	}
	
	metric := &models.Metric{
		Name:      string(parts[1]),
		Value:     data.Value,
		Timestamp: time.Unix(0, timestamp),
		Labels:    data.Labels,
		NodeID:    data.NodeID,
		Type:      models.MetricTypeFromString(data.Type),
		Help:      data.Help,
		Unit:      data.Unit,
	}
	
	return metric, nil
}

func (s *BadgerStore) seriesKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(labels[k])
		b.WriteString(",")
	}
	
	return b.String()
}

func (s *BadgerStore) matchesFilters(metric *models.Metric, filters map[string]string) bool {
	for key, value := range filters {
		if metric.Labels[key] != value {
			return false
		}
	}
	return true
}

func (s *BadgerStore) runCompaction() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.logger.Debug("Running database compaction")
		
		for {
			err := s.db.RunValueLogGC(0.5)
			if err != nil {
				if err == badger.ErrNoRewrite {
					break
				}
				s.logger.Error("Failed to run GC", zap.Error(err))
				break
			}
		}
	}
}

func (s *BadgerStore) Close() error {
	return s.db.Close()
}

// Helper functions
func parseSimpleQuery(query string) (string, map[string]string) {
	// Simple parser for queries like "metric_name{label1="value1",label2="value2"}"
	parts := strings.SplitN(query, "{", 2)
	metricName := strings.TrimSpace(parts[0])
	
	filters := make(map[string]string)
	
	if len(parts) > 1 {
		filterStr := strings.TrimSuffix(parts[1], "}")
		pairs := strings.Split(filterStr, ",")
		
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.Trim(strings.TrimSpace(kv[1]), "\"")
				filters[key] = value
			}
		}
	}
	
	return metricName, filters
}

type badgerLogger struct {
	logger *zap.Logger
}

func (l *badgerLogger) Errorf(f string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(f, v...))
}

func (l *badgerLogger) Warningf(f string, v ...interface{}) {
	l.logger.Warn(fmt.Sprintf(f, v...))
}

func (l *badgerLogger) Infof(f string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(f, v...))
}

func (l *badgerLogger) Debugf(f string, v ...interface{}) {
	l.logger.Debug(fmt.Sprintf(f, v...))
}