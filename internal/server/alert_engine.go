package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/internal/storage"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

// AlertManager manages alert rules and notifications
type AlertManager struct {
	config       *utils.Config
	store        storage.Storage
	logger       *zap.Logger
	rules        map[string]*AlertRule
	rulesMu      sync.RWMutex
	activeAlerts map[string]*models.Alert
	alertsMu     sync.RWMutex
}

// AlertRule represents an alert rule
type AlertRule struct {
	Name        string
	Expression  string
	For         time.Duration
	Labels      map[string]string
	Annotations map[string]string
	Severity    string
	Enabled     bool
	Threshold   float64
	Operator    string // >, <, >=, <=, ==, !=
	MetricName  string
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config *utils.Config, store storage.Storage, logger *zap.Logger) *AlertManager {
	am := &AlertManager{
		config:       config,
		store:        store,
		logger:       logger,
		rules:        make(map[string]*AlertRule),
		activeAlerts: make(map[string]*models.Alert),
	}

	// Load default alert rules
	am.loadDefaultRules()

	return am
}

// loadDefaultRules loads the default alert rules
func (am *AlertManager) loadDefaultRules() {
	defaultRules := []*AlertRule{
		{
			Name:       "HighCPUUsage",
			Expression: "system_cpu_usage > 80",
			For:        2 * time.Minute,
			Labels: map[string]string{
				"severity": "warning",
				"category": "system",
			},
			Annotations: map[string]string{
				"summary":     "High CPU usage detected",
				"description": "CPU usage is above 80%",
			},
			Enabled:    true,
			Threshold:  80.0,
			Operator:   ">",
			MetricName: "system_cpu_usage",
		},
		{
			Name:       "HighMemoryUsage",
			Expression: "system_memory_usage_percent > 90",
			For:        2 * time.Minute,
			Labels: map[string]string{
				"severity": "warning",
				"category": "system",
			},
			Annotations: map[string]string{
				"summary":     "High memory usage detected",
				"description": "Memory usage is above 90%",
			},
			Enabled:    true,
			Threshold:  90.0,
			Operator:   ">",
			MetricName: "system_memory_usage_percent",
		},
		{
			Name:       "LowDiskSpace",
			Expression: "system_disk_usage_percent > 85",
			For:        5 * time.Minute,
			Labels: map[string]string{
				"severity": "warning",
				"category": "system",
			},
			Annotations: map[string]string{
				"summary":     "Low disk space",
				"description": "Disk usage is above 85%",
			},
			Enabled:    true,
			Threshold:  85.0,
			Operator:   ">",
			MetricName: "system_disk_usage_percent",
		},
	}

	am.rulesMu.Lock()
	defer am.rulesMu.Unlock()

	for _, rule := range defaultRules {
		am.rules[rule.Name] = rule
	}

	am.logger.Info("Loaded default alert rules", zap.Int("count", len(defaultRules)))
}

// CheckMetrics checks metrics against alert rules
func (am *AlertManager) CheckMetrics(nodeID string, metrics []*models.Metric) {
	am.rulesMu.RLock()
	defer am.rulesMu.RUnlock()

	for _, metric := range metrics {
		for ruleName, rule := range am.rules {
			if !rule.Enabled {
				continue
			}

			// Check if metric matches the rule
			if metric.Name != rule.MetricName {
				continue
			}

			// Evaluate the rule
			if am.evaluateRule(rule, metric.Value) {
				am.fireAlert(nodeID, rule, metric)
			} else {
				am.resolveAlert(nodeID, ruleName)
			}
		}
	}
}

// evaluateRule evaluates an alert rule against a metric value
func (am *AlertManager) evaluateRule(rule *AlertRule, value float64) bool {
	switch rule.Operator {
	case ">":
		return value > rule.Threshold
	case "<":
		return value < rule.Threshold
	case ">=":
		return value >= rule.Threshold
	case "<=":
		return value <= rule.Threshold
	case "==":
		return value == rule.Threshold
	case "!=":
		return value != rule.Threshold
	default:
		return false
	}
}

// fireAlert fires an alert
func (am *AlertManager) fireAlert(nodeID string, rule *AlertRule, metric *models.Metric) {
	alertKey := fmt.Sprintf("%s:%s", nodeID, rule.Name)

	am.alertsMu.Lock()
	defer am.alertsMu.Unlock()

	// Check if alert is already active
	if existingAlert, exists := am.activeAlerts[alertKey]; exists {
		// Alert is already firing, check if it should stay active
		if time.Since(existingAlert.ActiveAt) < rule.For {
			// Still within the "for" duration, keep as pending
			return
		}

		// Update the alert value
		existingAlert.Value = metric.Value
		am.store.SaveAlert(existingAlert)
		return
	}

	// Create new alert
	alert := &models.Alert{
		ID:          utils.GenerateAlertID(),
		Name:        rule.Name,
		Expression:  rule.Expression,
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
		State:       models.AlertStatePending,
		Value:       metric.Value,
		ActiveAt:    time.Now(),
		CreatedAt:   time.Now(),
	}

	// Add node label
	if alert.Labels == nil {
		alert.Labels = make(map[string]string)
	}
	alert.Labels["node"] = nodeID
	alert.Labels["metric"] = metric.Name

	// Check if alert should fire immediately
	if rule.For == 0 {
		alert.State = models.AlertStateFiring
		am.logger.Warn("Alert firing",
			zap.String("alert", rule.Name),
			zap.String("node", nodeID),
			zap.Float64("value", metric.Value),
		)

		// Send notification
		go am.sendNotification(alert)
	} else {
		am.logger.Debug("Alert pending",
			zap.String("alert", rule.Name),
			zap.String("node", nodeID),
			zap.Duration("for", rule.For),
		)
	}

	am.activeAlerts[alertKey] = alert
	am.store.SaveAlert(alert)
}

// resolveAlert resolves an active alert
func (am *AlertManager) resolveAlert(nodeID string, ruleName string) {
	alertKey := fmt.Sprintf("%s:%s", nodeID, ruleName)

	am.alertsMu.Lock()
	defer am.alertsMu.Unlock()

	alert, exists := am.activeAlerts[alertKey]
	if !exists {
		return
	}

	// Mark alert as resolved
	alert.State = models.AlertStateResolved
	now := time.Now()
	alert.ResolvedAt = &now

	am.logger.Info("Alert resolved",
		zap.String("alert", ruleName),
		zap.String("node", nodeID),
	)

	// Save to storage
	am.store.SaveAlert(alert)

	// Send resolution notification
	go am.sendNotification(alert)

	// Remove from active alerts
	delete(am.activeAlerts, alertKey)
}

// sendNotification sends an alert notification
func (am *AlertManager) sendNotification(alert *models.Alert) {
	// This is a placeholder for notification logic
	// In a real implementation, you would:
	// 1. Check notification configuration
	// 2. Format the alert message
	// 3. Send to configured channels (Slack, Email, etc.)

	am.logger.Info("Sending alert notification",
		zap.String("alert", alert.Name),
		zap.String("state", alert.State.String()),
		zap.Any("labels", alert.Labels),
	)

	// Example: Send to Slack
	if am.config.Alerting.Notification.Slack.Enabled {
		am.sendSlackNotification(alert)
	}

	// Example: Send to Email
	if am.config.Alerting.Notification.Email.Enabled {
		am.sendEmailNotification(alert)
	}
}

// sendSlackNotification sends a notification to Slack
func (am *AlertManager) sendSlackNotification(alert *models.Alert) {
	// Placeholder for Slack notification
	am.logger.Debug("Would send Slack notification", zap.String("alert", alert.Name))
}

// sendEmailNotification sends a notification via email
func (am *AlertManager) sendEmailNotification(alert *models.Alert) {
	// Placeholder for email notification
	am.logger.Debug("Would send email notification", zap.String("alert", alert.Name))
}

// AddRule adds a new alert rule
func (am *AlertManager) AddRule(rule *AlertRule) error {
	if rule == nil || rule.Name == "" {
		return fmt.Errorf("invalid rule")
	}

	am.rulesMu.Lock()
	defer am.rulesMu.Unlock()

	am.rules[rule.Name] = rule
	am.logger.Info("Alert rule added", zap.String("rule", rule.Name))

	return nil
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(ruleName string) error {
	am.rulesMu.Lock()
	defer am.rulesMu.Unlock()

	if _, exists := am.rules[ruleName]; !exists {
		return fmt.Errorf("rule %s not found", ruleName)
	}

	delete(am.rules, ruleName)
	am.logger.Info("Alert rule removed", zap.String("rule", ruleName))

	return nil
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*models.Alert {
	am.alertsMu.RLock()
	defer am.alertsMu.RUnlock()

	alerts := make([]*models.Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// GetRules returns all alert rules
func (am *AlertManager) GetRules() []*AlertRule {
	am.rulesMu.RLock()
	defer am.rulesMu.RUnlock()

	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}

	return rules
}
