package anomaly

import (
	"fmt"
	"math"
)

// EWMADetector implements anomaly detection using Exponentially Weighted Moving Average
type EWMADetector struct {
	alpha       float64 // Smoothing factor (0 < alpha < 1)
	mean        float64 // EWMA mean
	variance    float64 // EWMA variance
	threshold   float64 // Number of standard deviations for anomaly
	initialized bool
	count       int
}

// NewEWMADetector creates a new EWMA-based anomaly detector
func NewEWMADetector(alpha, threshold float64) *EWMADetector {
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.2 // Default to 0.2 if invalid
	}

	return &EWMADetector{
		alpha:     alpha,
		threshold: threshold,
	}
}

// Train initializes the detector with historical data
func (ed *EWMADetector) Train(data []float64) error {
	if len(data) == 0 {
		return fmt.Errorf("empty training data")
	}

	// Initialize with first value
	ed.mean = data[0]
	ed.variance = 0
	ed.count = 1

	// Process remaining values
	for i := 1; i < len(data); i++ {
		ed.updateEWMA(data[i])
	}

	ed.initialized = true
	return nil
}

// Detect checks if a value is anomalous
func (ed *EWMADetector) Detect(value float64) (bool, float64, error) {
	if !ed.initialized {
		return false, 0, fmt.Errorf("detector not trained")
	}

	// Calculate bounds
	stddev := math.Sqrt(ed.variance)
	upperBound := ed.mean + ed.threshold*stddev
	lowerBound := ed.mean - ed.threshold*stddev

	// Check if value is outside bounds
	isAnomaly := value > upperBound || value < lowerBound

	// Calculate anomaly score (normalized distance from mean)
	var score float64
	if stddev > 0 {
		score = math.Abs(value-ed.mean) / (ed.threshold * stddev)
	} else {
		if value != ed.mean {
			score = 1.0
		} else {
			score = 0.0
		}
	}

	return isAnomaly, score, nil
}

// Update updates the detector with a new value
func (ed *EWMADetector) Update(value float64) error {
	if !ed.initialized {
		ed.mean = value
		ed.variance = 0
		ed.initialized = true
		ed.count = 1
		return nil
	}

	ed.updateEWMA(value)
	return nil
}

// updateEWMA updates the EWMA statistics
func (ed *EWMADetector) updateEWMA(value float64) {
	// Update EWMA mean
	prevMean := ed.mean
	ed.mean = ed.alpha*value + (1-ed.alpha)*ed.mean

	// Update EWMA variance
	diff := value - prevMean
	ed.variance = ed.alpha*diff*diff + (1-ed.alpha)*ed.variance

	ed.count++
}

// Reset resets the detector
func (ed *EWMADetector) Reset() {
	ed.mean = 0
	ed.variance = 0
	ed.initialized = false
	ed.count = 0
}

// GetStats returns current statistics
func (ed *EWMADetector) GetStats() (mean, stddev float64) {
	return ed.mean, math.Sqrt(ed.variance)
}

// GetBounds returns the current upper and lower bounds
func (ed *EWMADetector) GetBounds() (lower, upper float64) {
	stddev := math.Sqrt(ed.variance)
	return ed.mean - ed.threshold*stddev, ed.mean + ed.threshold*stddev
}

// SetAlpha changes the smoothing factor
func (ed *EWMADetector) SetAlpha(alpha float64) {
	if alpha > 0 && alpha < 1 {
		ed.alpha = alpha
	}
}

// SetThreshold changes the anomaly threshold
func (ed *EWMADetector) SetThreshold(threshold float64) {
	if threshold > 0 {
		ed.threshold = threshold
	}
}
