package models

import "time"

// Dashboard represents a monitoring dashboard
type Dashboard struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Panels      []*Panel          `json:"panels"`
	Variables   map[string]string `json:"variables"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Panel represents a dashboard panel
type Panel struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Type        PanelType              `json:"type"`
	Query       string                 `json:"query"`
	Position    *PanelPosition         `json:"position"`
	Options     map[string]interface{} `json:"options"`
	Datasource  string                 `json:"datasource"`
	RefreshRate time.Duration          `json:"refresh_rate"`
}

// PanelType represents the type of dashboard panel
type PanelType string

const (
	PanelTypeGraph      PanelType = "graph"
	PanelTypeTable      PanelType = "table"
	PanelTypeSingleStat PanelType = "singlestat"
	PanelTypeHeatmap    PanelType = "heatmap"
	PanelTypeText       PanelType = "text"
	PanelTypeAlert      PanelType = "alert"
)

// PanelPosition defines the position and size of a panel
type PanelPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardFilter represents filters for querying dashboards
type DashboardFilter struct {
	Tags      []string
	CreatedBy string
	Since     *time.Time
}
