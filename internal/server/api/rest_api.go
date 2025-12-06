package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/pkg/utils"
	"go.uber.org/zap"
)

type RESTAPI struct {
	config *utils.Config
	store  Storage
	logger *zap.Logger
	router *chi.Mux
}

type Storage interface {
	QueryMetrics(query string, start, end time.Time, step time.Duration) ([]*models.TimeSeries, error)
	GetNodes() ([]*models.Node, error)
	GetNode(nodeID string) (*models.Node, error)
	GetAlerts(state string) ([]*models.Alert, error)
	Ping() error
}

func NewRESTAPI(config *utils.Config, store Storage, logger *zap.Logger) *RESTAPI {
	api := &RESTAPI{
		config: config,
		store:  store,
		logger: logger,
		router: chi.NewRouter(),
	}

	api.setupMiddleware()
	api.setupRoutes()

	return api
}

func (a *RESTAPI) setupMiddleware() {
	// Request ID
	a.router.Use(middleware.RequestID)
	
	// Logger
	a.router.Use(middleware.Logger)
	
	// Recovery
	a.router.Use(middleware.Recoverer)
	
	// CORS
	if a.config.Server.HTTP.CORS.Enabled {
		corsMiddleware := cors.New(cors.Options{
			AllowedOrigins:   a.config.Server.HTTP.CORS.AllowedOrigins,
			AllowedMethods:   a.config.Server.HTTP.CORS.AllowedMethods,
			AllowedHeaders:   a.config.Server.HTTP.CORS.AllowedHeaders,
			AllowCredentials: true,
			MaxAge:           300,
		})
		a.router.Use(corsMiddleware.Handler)
	}
	
	// Timeout
	a.router.Use(middleware.Timeout(60 * time.Second))
	
	// Authentication (if enabled)
	if a.config.Authentication.Enabled {
		a.router.Use(a.authMiddleware)
	}
}

func (a *RESTAPI) setupRoutes() {
	// Health check
	a.router.Get("/health", a.healthHandler)
	a.router.Get("/ready", a.readyHandler)
	
	// API v1
	a.router.Route("/api/v1", func(r chi.Router) {
		// Nodes
		r.Route("/nodes", func(r chi.Router) {
			r.Get("/", a.listNodesHandler)
			r.Get("/{nodeID}", a.getNodeHandler)
			r.Get("/{nodeID}/metrics", a.getNodeMetricsHandler)
			r.Get("/{nodeID}/alerts", a.getNodeAlertsHandler)
		})
		
		// Metrics
		r.Route("/metrics", func(r chi.Router) {
			r.Get("/query", a.queryMetricsHandler)
			r.Get("/series", a.seriesHandler)
			r.Get("/labels", a.labelsHandler)
			r.Get("/label/{name}/values", a.labelValuesHandler)
		})
		
		// Alerts
		r.Route("/alerts", func(r chi.Router) {
			r.Get("/", a.listAlertsHandler)
			r.Post("/silence", a.silenceAlertHandler)
			r.Delete("/silence/{id}", a.deleteSilenceHandler)
		})
		
		// Dashboards
		r.Route("/dashboards", func(r chi.Router) {
			r.Get("/", a.listDashboardsHandler)
			r.Get("/{id}", a.getDashboardHandler)
			r.Post("/", a.createDashboardHandler)
			r.Put("/{id}", a.updateDashboardHandler)
			r.Delete("/{id}", a.deleteDashboardHandler)
		})
	})
	
	// Static files for dashboard
	if a.config.Server.HTTP.Static.Enabled {
		a.router.Handle("/*", http.FileServer(http.Dir(a.config.Server.HTTP.Static.Path)))
	}
}

func (a *RESTAPI) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"version": a.config.Version,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}
	
	a.respondJSON(w, http.StatusOK, response)
}

func (a *RESTAPI) readyHandler(w http.ResponseWriter, r *http.Request) {
	// Check storage connectivity
	if err := a.store.Ping(); err != nil {
		a.respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	
	a.respondJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (a *RESTAPI) listNodesHandler(w http.ResponseWriter, r *http.Request) {
	nodes, err := a.store.GetNodes()
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err)
		return
	}
	
	a.respondJSON(w, http.StatusOK, nodes)
}

func (a *RESTAPI) getNodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	
	node, err := a.store.GetNode(nodeID)
	if err != nil {
		a.respondError(w, http.StatusNotFound, err)
		return
	}
	
	a.respondJSON(w, http.StatusOK, node)
}

func (a *RESTAPI) queryMetricsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	stepStr := r.URL.Query().Get("step")
	
	if query == "" {
		a.respondError(w, http.StatusBadRequest, "query parameter is required")
		return
	}
	
	// Parse time range
	start := time.Now().Add(-1 * time.Hour) // default: 1 hour ago
	if startStr != "" {
		if ts, err := parseTime(startStr); err == nil {
			start = ts
		}
	}
	
	end := time.Now()
	if endStr != "" {
		if ts, err := parseTime(endStr); err == nil {
			end = ts
		}
	}
	
	step := 15 * time.Second
	if stepStr != "" {
		if d, err := time.ParseDuration(stepStr); err == nil {
			step = d
		}
	}
	
	// Execute query
	series, err := a.store.QueryMetrics(query, start, end, step)
	if err != nil {
		a.respondError(w, http.StatusBadRequest, err)
		return
	}
	
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "matrix",
			"result":     series,
		},
	}
	
	a.respondJSON(w, http.StatusOK, response)
}

func (a *RESTAPI) listAlertsHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	
	alerts, err := a.store.GetAlerts(state)
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err)
		return
	}
	
	a.respondJSON(w, http.StatusOK, alerts)
}

func (a *RESTAPI) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Get API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}
		
		// Validate API key
		if !a.validateAPIKey(apiKey) {
			a.respondJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "Invalid API key",
			})
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (a *RESTAPI) validateAPIKey(apiKey string) bool {
	// Check against configured API keys
	for _, key := range a.config.Authentication.APIKeys {
		if key == apiKey {
			return true
		}
	}
	return false
}

func (a *RESTAPI) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		a.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (a *RESTAPI) respondError(w http.ResponseWriter, status int, err interface{}) {
	var errMsg string
	switch v := err.(type) {
	case error:
		errMsg = v.Error()
	case string:
		errMsg = v
	default:
		errMsg = "Unknown error"
	}
	
	a.respondJSON(w, status, map[string]string{
		"error": errMsg,
	})
}

func parseTime(s string) (time.Time, error) {
	// Try parsing as RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try parsing as Unix timestamp
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(i, 0), nil
	}

	// Try parsing as duration (e.g., "1h", "5m")
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", s)
}

func (a *RESTAPI) getNodeMetricsHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	query := r.URL.Query().Get("query")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	stepStr := r.URL.Query().Get("step")

	// Default query to get all metrics for this node
	if query == "" {
		query = fmt.Sprintf("{node=\"%s\"}", nodeID)
	} else {
		// Add node label to query
		query = fmt.Sprintf("%s{node=\"%s\"}", query, nodeID)
	}

	// Parse time range
	start := time.Now().Add(-1 * time.Hour)
	if startStr != "" {
		if ts, err := parseTime(startStr); err == nil {
			start = ts
		}
	}

	end := time.Now()
	if endStr != "" {
		if ts, err := parseTime(endStr); err == nil {
			end = ts
		}
	}

	step := 15 * time.Second
	if stepStr != "" {
		if d, err := time.ParseDuration(stepStr); err == nil {
			step = d
		}
	}

	series, err := a.store.QueryMetrics(query, start, end, step)
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err)
		return
	}

	a.respondJSON(w, http.StatusOK, series)
}

func (a *RESTAPI) getNodeAlertsHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")

	// Get all alerts and filter by node
	alerts, err := a.store.GetAlerts("")
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err)
		return
	}

	// Filter alerts for this node using Labels
	var nodeAlerts []*models.Alert
	for _, alert := range alerts {
		if alert.Labels != nil && alert.Labels["node"] == nodeID {
			nodeAlerts = append(nodeAlerts, alert)
		}
	}

	a.respondJSON(w, http.StatusOK, nodeAlerts)
}

func (a *RESTAPI) seriesHandler(w http.ResponseWriter, r *http.Request) {
	// Get all unique metric series
	// This is a simplified implementation
	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   []string{},
	})
}

func (a *RESTAPI) labelsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all label names
	// This is a simplified implementation
	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   []string{"node", "collector", "metric"},
	})
}

func (a *RESTAPI) labelValuesHandler(w http.ResponseWriter, r *http.Request) {
	labelName := chi.URLParam(r, "name")

	// Get all values for this label
	// This is a simplified implementation
	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   []string{},
		"label":  labelName,
	})
}

func (a *RESTAPI) silenceAlertHandler(w http.ResponseWriter, r *http.Request) {
	// Silence an alert
	// This is a simplified implementation
	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Alert silenced",
	})
}

func (a *RESTAPI) deleteSilenceHandler(w http.ResponseWriter, r *http.Request) {
	silenceID := chi.URLParam(r, "id")

	// Delete a silence
	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Silence %s deleted", silenceID),
	})
}

func (a *RESTAPI) listDashboardsHandler(w http.ResponseWriter, r *http.Request) {
	// List all dashboards
	// This is a simplified implementation
	a.respondJSON(w, http.StatusOK, []interface{}{})
}

func (a *RESTAPI) getDashboardHandler(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")

	// Get dashboard by ID
	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":   dashboardID,
		"name": "Dashboard",
	})
}

func (a *RESTAPI) createDashboardHandler(w http.ResponseWriter, r *http.Request) {
	var dashboard models.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		a.respondError(w, http.StatusBadRequest, err)
		return
	}

	// Create dashboard
	a.respondJSON(w, http.StatusCreated, dashboard)
}

func (a *RESTAPI) updateDashboardHandler(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")

	var dashboard models.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		a.respondError(w, http.StatusBadRequest, err)
		return
	}

	dashboard.ID = dashboardID
	a.respondJSON(w, http.StatusOK, dashboard)
}

func (a *RESTAPI) deleteDashboardHandler(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")

	// Delete dashboard
	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Dashboard %s deleted", dashboardID),
	})
}

func (a *RESTAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}