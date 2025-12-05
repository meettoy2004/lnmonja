package storage

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

// CompressionEngine handles metric compression
type CompressionEngine struct {
	config *utils.StorageConfig
	logger *zap.Logger
}

// NewCompressionEngine creates a new compression engine
func NewCompressionEngine(config *utils.StorageConfig, logger *zap.Logger) *CompressionEngine {
	return &CompressionEngine{
		config: config,
		logger: logger,
	}
}

// CompressedMetrics represents compressed metric data
type CompressedMetrics struct {
	Data          []byte
	OriginalSize  int
	CompressedSize int
	MetricCount   int
}

// CompressMetrics compresses a batch of metrics
func (ce *CompressionEngine) CompressMetrics(metrics []*models.Metric) (*CompressedMetrics, error) {
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics to compress")
	}

	// Serialize metrics to JSON
	data, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics: %w", err)
	}

	originalSize := len(data)

	// Compress using gzip
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if _, err := gzipWriter.Write(data); err != nil {
		gzipWriter.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	compressed := buf.Bytes()
	compressedSize := len(compressed)

	compressionRatio := float64(originalSize) / float64(compressedSize)
	ce.logger.Debug("Metrics compressed",
		zap.Int("metric_count", len(metrics)),
		zap.Int("original_size", originalSize),
		zap.Int("compressed_size", compressedSize),
		zap.Float64("compression_ratio", compressionRatio),
	)

	return &CompressedMetrics{
		Data:           compressed,
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		MetricCount:    len(metrics),
	}, nil
}

// DecompressMetrics decompresses metric data
func (ce *CompressionEngine) DecompressMetrics(compressed *CompressedMetrics) ([]*models.Metric, error) {
	if compressed == nil || len(compressed.Data) == 0 {
		return nil, fmt.Errorf("no data to decompress")
	}

	// Decompress using gzip
	buf := bytes.NewReader(compressed.Data)
	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	// Deserialize metrics
	var metrics []*models.Metric
	if err := json.Unmarshal(decompressed, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	ce.logger.Debug("Metrics decompressed",
		zap.Int("metric_count", len(metrics)),
		zap.Int("decompressed_size", len(decompressed)),
	)

	return metrics, nil
}

// CompressBytes compresses raw bytes
func (ce *CompressionEngine) CompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if _, err := gzipWriter.Write(data); err != nil {
		gzipWriter.Close()
		return nil, err
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecompressBytes decompresses raw bytes
func (ce *CompressionEngine) DecompressBytes(compressed []byte) ([]byte, error) {
	buf := bytes.NewReader(compressed)
	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	return io.ReadAll(gzipReader)
}

// DeltaEncode applies delta encoding to numeric values (for better compression)
func (ce *CompressionEngine) DeltaEncode(values []float64) []float64 {
	if len(values) <= 1 {
		return values
	}

	deltas := make([]float64, len(values))
	deltas[0] = values[0]

	for i := 1; i < len(values); i++ {
		deltas[i] = values[i] - values[i-1]
	}

	return deltas
}

// DeltaDecode reverses delta encoding
func (ce *CompressionEngine) DeltaDecode(deltas []float64) []float64 {
	if len(deltas) <= 1 {
		return deltas
	}

	values := make([]float64, len(deltas))
	values[0] = deltas[0]

	for i := 1; i < len(deltas); i++ {
		values[i] = values[i-1] + deltas[i]
	}

	return values
}
