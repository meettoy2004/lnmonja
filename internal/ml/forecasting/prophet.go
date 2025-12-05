package forecasting

import (
	"fmt"
	"math"
	"time"
)

// Forecast represents a forecasted value
type Forecast struct {
	Timestamp time.Time
	Value     float64
	Lower     float64 // Lower confidence bound
	Upper     float64 // Upper confidence bound
}

// Prophet implements a simplified time-series forecasting model
type Prophet struct {
	trend       *TrendModel
	seasonality *SeasonalityModel
	trained     bool
	data        []DataPoint
}

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// TrendModel represents the trend component
type TrendModel struct {
	slope     float64
	intercept float64
}

// SeasonalityModel represents the seasonality component
type SeasonalityModel struct {
	period     time.Duration
	components map[int]float64 // hour/day -> seasonal value
	enabled    bool
}

// NewProphet creates a new Prophet forecaster
func NewProphet() *Prophet {
	return &Prophet{
		trend: &TrendModel{},
		seasonality: &SeasonalityModel{
			period:     24 * time.Hour, // Daily seasonality
			components: make(map[int]float64),
			enabled:    true,
		},
		data: make([]DataPoint, 0),
	}
}

// Train trains the model with historical data
func (p *Prophet) Train(data []DataPoint) error {
	if len(data) < 10 {
		return fmt.Errorf("insufficient training data: need at least 10 points")
	}

	p.data = make([]DataPoint, len(data))
	copy(p.data, data)

	// Extract trend using linear regression
	if err := p.fitTrend(data); err != nil {
		return fmt.Errorf("failed to fit trend: %w", err)
	}

	// Extract seasonality
	if p.seasonality.enabled {
		if err := p.fitSeasonality(data); err != nil {
			return fmt.Errorf("failed to fit seasonality: %w", err)
		}
	}

	p.trained = true
	return nil
}

// Predict forecasts future values
func (p *Prophet) Predict(periods int, interval time.Duration) ([]Forecast, error) {
	if !p.trained {
		return nil, fmt.Errorf("model not trained")
	}

	if len(p.data) == 0 {
		return nil, fmt.Errorf("no training data")
	}

	forecasts := make([]Forecast, periods)
	lastTime := p.data[len(p.data)-1].Timestamp

	for i := 0; i < periods; i++ {
		timestamp := lastTime.Add(interval * time.Duration(i+1))
		forecast := p.predictSingle(timestamp, i+1)
		forecasts[i] = forecast
	}

	return forecasts, nil
}

// predictSingle predicts a single value
func (p *Prophet) predictSingle(timestamp time.Time, stepsAhead int) Forecast {
	// Calculate trend component
	x := float64(stepsAhead)
	trendValue := p.trend.intercept + p.trend.slope*x

	// Calculate seasonal component
	var seasonalValue float64
	if p.seasonality.enabled {
		seasonalValue = p.getSeasonalValue(timestamp)
	}

	// Combine components
	predictedValue := trendValue + seasonalValue

	// Calculate confidence intervals (simplified)
	// In reality, this would use residual analysis
	variance := p.calculateVariance()
	stdError := math.Sqrt(variance * float64(stepsAhead))
	confidenceInterval := 1.96 * stdError // 95% confidence

	return Forecast{
		Timestamp: timestamp,
		Value:     predictedValue,
		Lower:     predictedValue - confidenceInterval,
		Upper:     predictedValue + confidenceInterval,
	}
}

// fitTrend fits a linear trend to the data
func (p *Prophet) fitTrend(data []DataPoint) error {
	if len(data) == 0 {
		return fmt.Errorf("no data to fit")
	}

	n := float64(len(data))
	var sumX, sumY, sumXY, sumX2 float64

	for i, point := range data {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Linear regression: y = mx + b
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		p.trend.slope = 0
		p.trend.intercept = sumY / n
		return nil
	}

	p.trend.slope = (n*sumXY - sumX*sumY) / denominator
	p.trend.intercept = (sumY - p.trend.slope*sumX) / n

	return nil
}

// fitSeasonality fits seasonal patterns
func (p *Prophet) fitSeasonality(data []DataPoint) error {
	if len(data) == 0 {
		return fmt.Errorf("no data to fit")
	}

	// Remove trend from data
	detrended := make([]DataPoint, len(data))
	for i, point := range data {
		trendValue := p.trend.intercept + p.trend.slope*float64(i)
		detrended[i] = DataPoint{
			Timestamp: point.Timestamp,
			Value:     point.Value - trendValue,
		}
	}

	// Calculate average value for each hour of the day
	hourSums := make(map[int]float64)
	hourCounts := make(map[int]int)

	for _, point := range detrended {
		hour := point.Timestamp.Hour()
		hourSums[hour] += point.Value
		hourCounts[hour]++
	}

	// Calculate averages
	for hour := 0; hour < 24; hour++ {
		if count, exists := hourCounts[hour]; exists && count > 0 {
			p.seasonality.components[hour] = hourSums[hour] / float64(count)
		} else {
			p.seasonality.components[hour] = 0
		}
	}

	// Remove mean to center seasonality around zero
	var mean float64
	for _, value := range p.seasonality.components {
		mean += value
	}
	mean /= float64(len(p.seasonality.components))

	for hour := range p.seasonality.components {
		p.seasonality.components[hour] -= mean
	}

	return nil
}

// getSeasonalValue gets the seasonal component for a timestamp
func (p *Prophet) getSeasonalValue(timestamp time.Time) float64 {
	hour := timestamp.Hour()
	if value, exists := p.seasonality.components[hour]; exists {
		return value
	}
	return 0
}

// calculateVariance calculates the variance of residuals
func (p *Prophet) calculateVariance() float64 {
	if len(p.data) == 0 {
		return 1.0
	}

	var sumSquaredResiduals float64
	for i, point := range p.data {
		// Calculate predicted value
		trendValue := p.trend.intercept + p.trend.slope*float64(i)
		seasonalValue := p.getSeasonalValue(point.Timestamp)
		predicted := trendValue + seasonalValue

		// Calculate residual
		residual := point.Value - predicted
		sumSquaredResiduals += residual * residual
	}

	variance := sumSquaredResiduals / float64(len(p.data))
	return math.Max(variance, 0.01) // Minimum variance to avoid division by zero
}

// Update updates the model with new data
func (p *Prophet) Update(point DataPoint) error {
	if !p.trained {
		return fmt.Errorf("model not trained")
	}

	p.data = append(p.data, point)

	// Keep only recent data to avoid unbounded growth
	maxSize := 1000
	if len(p.data) > maxSize {
		p.data = p.data[len(p.data)-maxSize:]
	}

	// Periodically retrain (e.g., every 100 points)
	if len(p.data)%100 == 0 {
		return p.Train(p.data)
	}

	return nil
}

// Reset resets the model
func (p *Prophet) Reset() {
	p.trend = &TrendModel{}
	p.seasonality = &SeasonalityModel{
		period:     24 * time.Hour,
		components: make(map[int]float64),
		enabled:    true,
	}
	p.data = make([]DataPoint, 0)
	p.trained = false
}
