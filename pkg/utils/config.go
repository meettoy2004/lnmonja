package utils

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		GRPC struct {
			Address string `yaml:"address"`
			Port    int    `yaml:"port"`
			TLS     struct {
				Enabled     bool   `yaml:"enabled"`
				CertFile    string `yaml:"cert_file"`
				KeyFile     string `yaml:"key_file"`
				ClientCAFile string `yaml:"client_ca_file"`
			} `yaml:"tls"`
			HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
			HeartbeatTimeout  time.Duration `yaml:"heartbeat_timeout"`
		} `yaml:"grpc"`

		HTTP struct {
			Address string `yaml:"address"`
			Port    int    `yaml:"port"`
			CORS    struct {
				Enabled        bool     `yaml:"enabled"`
				AllowedOrigins []string `yaml:"allowed_origins"`
				AllowedMethods []string `yaml:"allowed_methods"`
				AllowedHeaders []string `yaml:"allowed_headers"`
			} `yaml:"cors"`
			Static struct {
				Enabled bool   `yaml:"enabled"`
				Path    string `yaml:"path"`
			} `yaml:"static"`
		} `yaml:"http"`

		WebSocket struct {
			Address          string        `yaml:"address"`
			Port             int           `yaml:"port"`
			ReadBufferSize   int           `yaml:"read_buffer_size"`
			WriteBufferSize  int           `yaml:"write_buffer_size"`
			MaxMessageSize   int64         `yaml:"max_message_size"`
			PingInterval     time.Duration `yaml:"ping_interval"`
		} `yaml:"websocket"`
	} `yaml:"server"`

	Storage StorageConfig `yaml:"storage"`

	Alerting struct {
		Enabled            bool          `yaml:"enabled"`
		RulesPath          string        `yaml:"rules_path"`
		EvaluationInterval time.Duration `yaml:"evaluation_interval"`
		DefaultCooldown    time.Duration `yaml:"default_cooldown"`
		Notification       struct {
			Slack struct {
				Enabled    bool   `yaml:"enabled"`
				WebhookURL string `yaml:"webhook_url"`
				Channel    string `yaml:"channel"`
			} `yaml:"slack"`
			Email struct {
				Enabled  bool     `yaml:"enabled"`
				SMTPHost string   `yaml:"smtp_host"`
				SMTPPort int      `yaml:"smtp_port"`
				Username string   `yaml:"username"`
				Password string   `yaml:"password"`
				From     string   `yaml:"from"`
				To       []string `yaml:"to"`
			} `yaml:"email"`
		} `yaml:"notification"`
	} `yaml:"alerting"`

	Authentication struct {
		Enabled    bool     `yaml:"enabled"`
		JWTSecret  string   `yaml:"jwt_secret"`
		TokenExpiry time.Duration `yaml:"token_expiry"`
		APIKeys    []string `yaml:"api_keys"`
		Users      []User   `yaml:"users"`
	} `yaml:"authentication"`

	Logging LogConfig `yaml:"logging"`

	// Agent-specific config
	Agent struct {
		NodeID         string        `yaml:"node_id"`
		ServerAddress  string        `yaml:"server_address"`
		BatchSize      int           `yaml:"batch_size"`
		MaxBatchWait   time.Duration `yaml:"max_batch_wait"`
		HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	} `yaml:"agent"`

	// Collectors config
	Collectors struct {
		System struct {
			Enabled  bool          `yaml:"enabled"`
			Interval time.Duration `yaml:"interval"`
			Metrics  struct {
				CPU     bool `yaml:"cpu"`
				Memory  bool `yaml:"memory"`
				Load    bool `yaml:"load"`
				Disk    bool `yaml:"disk"`
				Network bool `yaml:"network"`
				Uptime  bool `yaml:"uptime"`
			} `yaml:"metrics"`
		} `yaml:"system"`

		Process struct {
			Enabled      bool          `yaml:"enabled"`
			Interval     time.Duration `yaml:"interval"`
			MaxProcesses int           `yaml:"max_processes"`
		} `yaml:"process"`

		Container struct {
			Enabled      bool   `yaml:"enabled"`
			Runtime      string `yaml:"runtime"`
			DockerSocket string `yaml:"docker_socket"`
		} `yaml:"container"`

		Custom struct {
			Enabled bool   `yaml:"enabled"`
			Path    string `yaml:"path"`
		} `yaml:"custom"`
	} `yaml:"collectors"`

	Version string `yaml:"-"`
}

type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Role     string `yaml:"role"`
	Email    string `yaml:"email"`
}

type StorageConfig struct {
	Engine           string        `yaml:"engine"`
	Path             string        `yaml:"path"`
	RetentionPeriod  time.Duration `yaml:"retention_period"`
	Compression      bool          `yaml:"compression"`
	ShardSize        string        `yaml:"shard_size"`
	SyncInterval     time.Duration `yaml:"sync_interval"`
	SyncWrites       bool          `yaml:"sync_writes"`
	ValueLogFileSize int64         `yaml:"value_log_file_size"`
	MemTableSize     int64         `yaml:"mem_table_size"`
	Tiering          struct {
		Enabled       bool          `yaml:"enabled"`
		HotRetention  time.Duration `yaml:"hot_retention"`
		WarmRetention time.Duration `yaml:"warm_retention"`
		ColdRetention time.Duration `yaml:"cold_retention"`
		ColdPath      string        `yaml:"cold_path"`
	} `yaml:"tiering"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
	Path   string `yaml:"path"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.setDefaults()

	// Validate config
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func (c *Config) setDefaults() {
	if c.Server.GRPC.Address == "" {
		c.Server.GRPC.Address = "0.0.0.0"
	}
	if c.Server.GRPC.Port == 0 {
		c.Server.GRPC.Port = 9090
	}
	if c.Server.GRPC.HeartbeatInterval == 0 {
		c.Server.GRPC.HeartbeatInterval = 30 * time.Second
	}
	if c.Server.GRPC.HeartbeatTimeout == 0 {
		c.Server.GRPC.HeartbeatTimeout = 90 * time.Second
	}

	if c.Server.HTTP.Address == "" {
		c.Server.HTTP.Address = "0.0.0.0"
	}
	if c.Server.HTTP.Port == 0 {
		c.Server.HTTP.Port = 8080
	}

	if c.Storage.Path == "" {
		c.Storage.Path = "./data"
	}
	if c.Storage.RetentionPeriod == 0 {
		c.Storage.RetentionPeriod = 720 * time.Hour // 30 days
	}
	if c.Storage.ShardSize == "" {
		c.Storage.ShardSize = "1GB"
	}
	if c.Storage.SyncInterval == 0 {
		c.Storage.SyncInterval = 30 * time.Second
	}
	if c.Storage.ValueLogFileSize == 0 {
		c.Storage.ValueLogFileSize = 1 << 30 // 1GB
	}
	if c.Storage.MemTableSize == 0 {
		c.Storage.MemTableSize = 64 << 20 // 64MB
	}

	if c.Agent.BatchSize == 0 {
		c.Agent.BatchSize = 1000
	}
	if c.Agent.MaxBatchWait == 0 {
		c.Agent.MaxBatchWait = 1 * time.Second
	}
	if c.Agent.HeartbeatInterval == 0 {
		c.Agent.HeartbeatInterval = 30 * time.Second
	}

	if c.Collectors.System.Interval == 0 {
		c.Collectors.System.Interval = 1 * time.Second
	}
	if c.Collectors.Process.Interval == 0 {
		c.Collectors.Process.Interval = 5 * time.Second
	}
	if c.Collectors.Process.MaxProcesses == 0 {
		c.Collectors.Process.MaxProcesses = 500
	}
	if c.Collectors.Container.DockerSocket == "" {
		c.Collectors.Container.DockerSocket = "/var/run/docker.sock"
	}
}

func (c *Config) validate() error {
	if c.Server.GRPC.Port <= 0 || c.Server.GRPC.Port > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPC.Port)
	}

	if c.Server.HTTP.Port <= 0 || c.Server.HTTP.Port > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTP.Port)
	}

	if c.Server.GRPC.TLS.Enabled {
		if c.Server.GRPC.TLS.CertFile == "" {
			return fmt.Errorf("TLS cert file is required when TLS is enabled")
		}
		if c.Server.GRPC.TLS.KeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled")
		}
	}

	if c.Authentication.Enabled && c.Authentication.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required when authentication is enabled")
	}

	return nil
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}