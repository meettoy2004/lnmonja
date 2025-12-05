package anomaly

import (
	"fmt"
	"math"
	"time"
)

// Detector interface for anomaly detection
type Detector interface {
	Train(data []float64) error
	Detect(value float64) (bool, float64, error)
	Update(value float64) error
	Reset()
}

// AnomalyResult represents the result of anomaly detection
type AnomalyResult struct {
	IsAnomaly    bool
	Score        float64
	Timestamp    time.Time
	Value        float64
	Expected     float64
	UpperBound   float64
	LowerBound   float64
	Confidence   float64
	DetectorType string
}

// MultiDetector combines multiple detection algorithms
type MultiDetector struct {
	detectors []Detector
	weights   []float64
	threshold float64
}

// NewMultiDetector creates a new multi-detector
func NewMultiDetector(threshold float64) *MultiDetector {
	md := &MultiDetector{
		detectors: make([]Detector, 0),
		weights:   make([]float64, 0),
		threshold: threshold,
	}

	// Add EWMA detector with weight 0.6
	ewmaDetector := NewEWMADetector(0.2, 3.0)
	md.AddDetector(ewmaDetector, 0.6)

	// Add Isolation Forest detector with weight 0.4
	iforestDetector := NewIsolationForest(100, 256)
	md.AddDetector(iforestDetector, 0.4)

	return md
}

// AddDetector adds a detector with a weight
func (md *MultiDetector) AddDetector(detector Detector, weight float64) {
	md.detectors = append(md.detectors, detector)
	md.weights = append(md.weights, weight)
}

// Train trains all detectors
func (md *MultiDetector) Train(data []float64) error {
	for _, detector := range md.detectors {
		if err := detector.Train(data); err != nil {
			return err
		}
	}
	return nil
}

// Detect checks if a value is anomalous using all detectors
func (md *MultiDetector) Detect(value float64) (bool, float64, error) {
	if len(md.detectors) == 0 {
		return false, 0, fmt.Errorf("no detectors configured")
	}

	var weightedScore float64
	var totalWeight float64

	for i, detector := range md.detectors {
		isAnomaly, score, err := detector.Detect(value)
		if err != nil {
			continue
		}

		// Convert boolean to score if needed
		anomalyScore := score
		if isAnomaly && score == 0 {
			anomalyScore = 1.0
		}

		weightedScore += anomalyScore * md.weights[i]
		totalWeight += md.weights[i]
	}

	if totalWeight == 0 {
		return false, 0, fmt.Errorf("no valid detections")
	}

	finalScore := weightedScore / totalWeight
	isAnomaly := finalScore > md.threshold

	return isAnomaly, finalScore, nil
}

// Update updates all detectors with a new value
func (md *MultiDetector) Update(value float64) error {
	for _, detector := range md.detectors {
		if err := detector.Update(value); err != nil {
			return err
		}
	}
	return nil
}

// Reset resets all detectors
func (md *MultiDetector) Reset() {
	for _, detector := range md.detectors {
		detector.Reset()
	}
}

// StatisticalDetector is a simple statistical anomaly detector
type StatisticalDetector struct {
	mean        float64
	stddev      float64
	count       int
	sumX        float64
	sumX2       float64
	threshold   float64 // Number of standard deviations
	minSamples  int
	initialized bool
}

// NewStatisticalDetector creates a new statistical detector
func NewStatisticalDetector(threshold float64) *StatisticalDetector {
	return &StatisticalDetector{
		threshold:  threshold,
		minSamples: 10,
	}
}

// Train trains the detector with historical data
func (sd *StatisticalDetector) Train(data []float64) error {
	if len(data) < sd.minSamples {
		return fmt.Errorf("insufficient data for training: need at least %d samples", sd.minSamples)
	}

	sd.count = len(data)
	sd.sumX = 0
	sd.sumX2 = 0

	for _, value := range data {
		sd.sumX += value
		sd.sumX2 += value * value
	}

	sd.mean = sd.sumX / float64(sd.count)
	variance := (sd.sumX2 / float64(sd.count)) - (sd.mean * sd.mean)
	sd.stddev = math.Sqrt(math.Max(0, variance))
	sd.initialized = true

	return nil
}

// Detect checks if a value is anomalous
func (sd *StatisticalDetector) Detect(value float64) (bool, float64, error) {
	if !sd.initialized {
		return false, 0, fmt.Errorf("detector not trained")
	}

	if sd.stddev == 0 {
		return value != sd.mean, 0, nil
	}

	// Calculate z-score
	zScore := math.Abs((value - sd.mean) / sd.stddev)

	isAnomaly := zScore > sd.threshold
	score := zScore / sd.threshold // Normalize score

	return isAnomaly, score, nil
}

// Update updates the detector with a new value
func (sd *StatisticalDetector) Update(value float64) error {
	if !sd.initialized {
		return fmt.Errorf("detector not trained")
	}

	sd.count++
	sd.sumX += value
	sd.sumX2 += value * value

	sd.mean = sd.sumX / float64(sd.count)
	variance := (sd.sumX2 / float64(sd.count)) - (sd.mean * sd.mean)
	sd.stddev = math.Sqrt(math.Max(0, variance))

	return nil
}

// Reset resets the detector
func (sd *StatisticalDetector) Reset() {
	sd.mean = 0
	sd.stddev = 0
	sd.count = 0
	sd.sumX = 0
	sd.sumX2 = 0
	sd.initialized = false
}
