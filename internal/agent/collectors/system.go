package collectors

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type SystemCollector struct {
	config     SystemConfig
	lastCPU    map[string]cpu.TimesStat
	lastNet    map[string]net.IOCountersStat
	lastDisk   map[string]disk.IOCountersStat
	processors int
}

type SystemConfig struct {
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
	Disk struct {
		IgnoreFSTypes   []string `yaml:"ignore_fs_types"`
		IgnoreMounts    []string `yaml:"ignore_mounts"`
		IncludePartitions bool   `yaml:"include_partitions"`
	} `yaml:"disk"`
	Network struct {
		Interfaces      []string `yaml:"interfaces"`
		IncludeAll      bool     `yaml:"include_all"`
		IncludeLoopback bool     `yaml:"include_loopback"`
	} `yaml:"network"`
}

func NewSystemCollector(config SystemConfig) (*SystemCollector, error) {
	c := &SystemCollector{
		config:     config,
		lastCPU:    make(map[string]cpu.TimesStat),
		lastNet:    make(map[string]net.IOCountersStat),
		lastDisk:   make(map[string]disk.IOCountersStat),
		processors: runtime.NumCPU(),
	}

	return c, nil
}

func (c *SystemCollector) Name() string {
	return "system"
}

func (c *SystemCollector) Enabled() bool {
	return c.config.Enabled
}

func (c *SystemCollector) Interval() time.Duration {
	return c.config.Interval
}

func (c *SystemCollector) Collect(ctx context.Context) ([]*Metric, error) {
	var metrics []*Metric

	// Collect CPU metrics
	if c.config.Metrics.CPU {
		cpuMetrics, err := c.collectCPUMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to collect CPU metrics: %w", err)
		}
		metrics = append(metrics, cpuMetrics...)
	}

	// Collect memory metrics
	if c.config.Metrics.Memory {
		memMetrics, err := c.collectMemoryMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to collect memory metrics: %w", err)
		}
		metrics = append(metrics, memMetrics...)
	}

	// Collect load metrics
	if c.config.Metrics.Load {
		loadMetrics, err := c.collectLoadMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to collect load metrics: %w", err)
		}
		metrics = append(metrics, loadMetrics...)
	}

	// Collect disk metrics
	if c.config.Metrics.Disk {
		diskMetrics, err := c.collectDiskMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to collect disk metrics: %w", err)
		}
		metrics = append(metrics, diskMetrics...)
	}

	// Collect network metrics
	if c.config.Metrics.Network {
		netMetrics, err := c.collectNetworkMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to collect network metrics: %w", err)
		}
		metrics = append(metrics, netMetrics...)
	}

	// Collect uptime
	if c.config.Metrics.Uptime {
		uptimeMetrics, err := c.collectUptimeMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to collect uptime metrics: %w", err)
		}
		metrics = append(metrics, uptimeMetrics...)
	}

	return metrics, nil
}

func (c *SystemCollector) collectCPUMetrics(ctx context.Context) ([]*Metric, error) {
	var metrics []*Metric

	// Get CPU times
	cpuTimes, err := cpu.Times(true)
	if err != nil {
		return nil, err
	}

	// Get CPU percentages
	percentages, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	// CPU count
	metrics = append(metrics, &Metric{
		Name:  "system_cpu_cores",
		Value: float64(c.processors),
		Type:  Gauge,
		Help:  "Number of CPU cores",
	})

	// Per-CPU usage
	for i, cpuTime := range cpuTimes {
		cpuLabel := fmt.Sprintf("cpu%d", i)

		// Calculate usage from previous sample
		if last, exists := c.lastCPU[cpuLabel]; exists {
			totalDelta := totalCPUTime(cpuTime) - totalCPUTime(last)

			if totalDelta > 0 {
				userPercent := 100 * (cpuTime.User - last.User) / totalDelta
				systemPercent := 100 * (cpuTime.System - last.System) / totalDelta
				idlePercent := 100 * (cpuTime.Idle - last.Idle) / totalDelta
				iowaitPercent := 100 * (cpuTime.Iowait - last.Iowait) / totalDelta

				metrics = append(metrics,
					&Metric{
						Name:   "system_cpu_user",
						Value:  userPercent,
						Labels: map[string]string{"cpu": cpuLabel},
						Type:   Gauge,
						Help:   "CPU user time percentage",
						Unit:   "percent",
					},
					&Metric{
						Name:   "system_cpu_system",
						Value:  systemPercent,
						Labels: map[string]string{"cpu": cpuLabel},
						Type:   Gauge,
						Help:   "CPU system time percentage",
						Unit:   "percent",
					},
					&Metric{
						Name:   "system_cpu_idle",
						Value:  idlePercent,
						Labels: map[string]string{"cpu": cpuLabel},
						Type:   Gauge,
						Help:   "CPU idle time percentage",
						Unit:   "percent",
					},
					&Metric{
						Name:   "system_cpu_iowait",
						Value:  iowaitPercent,
						Labels: map[string]string{"cpu": cpuLabel},
						Type:   Gauge,
						Help:   "CPU I/O wait time percentage",
						Unit:   "percent",
					},
				)
			}
		}

		c.lastCPU[cpuLabel] = cpuTime

		// Add percentage from psutil
		if i < len(percentages) {
			metrics = append(metrics, &Metric{
				Name:   "system_cpu_usage",
				Value:  percentages[i],
				Labels: map[string]string{"cpu": cpuLabel},
				Type:   Gauge,
				Help:   "CPU usage percentage",
				Unit:   "percent",
			})
		}
	}

	// Overall CPU usage
	overallPercent, err := cpu.Percent(0, false)
	if err == nil && len(overallPercent) > 0 {
		metrics = append(metrics, &Metric{
			Name:  "system_cpu_usage_total",
			Value: overallPercent[0],
			Type:  Gauge,
			Help:  "Total CPU usage percentage",
			Unit:  "percent",
		})
	}

	return metrics, nil
}

func (c *SystemCollector) collectMemoryMetrics(ctx context.Context) ([]*Metric, error) {
	var metrics []*Metric

	virtMem, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	swapMem, err := mem.SwapMemory()
	if err != nil {
		return nil, err
	}

	// Virtual memory
	metrics = append(metrics,
		&Metric{
			Name:  "system_memory_total_bytes",
			Value: float64(virtMem.Total),
			Type:  Gauge,
			Help:  "Total memory in bytes",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_memory_available_bytes",
			Value: float64(virtMem.Available),
			Type:  Gauge,
			Help:  "Available memory in bytes",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_memory_used_bytes",
			Value: float64(virtMem.Used),
			Type:  Gauge,
			Help:  "Used memory in bytes",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_memory_free_bytes",
			Value: float64(virtMem.Free),
			Type:  Gauge,
			Help:  "Free memory in bytes",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_memory_usage_percent",
			Value: virtMem.UsedPercent,
			Type:  Gauge,
			Help:  "Memory usage percentage",
			Unit:  "percent",
		},
		&Metric{
			Name:  "system_memory_buffers_bytes",
			Value: float64(virtMem.Buffers),
			Type:  Gauge,
			Help:  "Memory used by buffers",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_memory_cached_bytes",
			Value: float64(virtMem.Cached),
			Type:  Gauge,
			Help:  "Memory used by cache",
			Unit:  "bytes",
		},
	)

	// Swap memory
	metrics = append(metrics,
		&Metric{
			Name:  "system_swap_total_bytes",
			Value: float64(swapMem.Total),
			Type:  Gauge,
			Help:  "Total swap space in bytes",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_swap_used_bytes",
			Value: float64(swapMem.Used),
			Type:  Gauge,
			Help:  "Used swap space in bytes",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_swap_free_bytes",
			Value: float64(swapMem.Free),
			Type:  Gauge,
			Help:  "Free swap space in bytes",
			Unit:  "bytes",
		},
		&Metric{
			Name:  "system_swap_usage_percent",
			Value: swapMem.UsedPercent,
			Type:  Gauge,
			Help:  "Swap usage percentage",
			Unit:  "percent",
		},
	)

	return metrics, nil
}

func (c *SystemCollector) collectLoadMetrics(ctx context.Context) ([]*Metric, error) {
	var metrics []*Metric

	avg, err := load.Avg()
	if err != nil {
		return nil, err
	}

	metrics = append(metrics,
		&Metric{
			Name:  "system_load1",
			Value: avg.Load1,
			Type:  Gauge,
			Help:  "1-minute load average",
		},
		&Metric{
			Name:  "system_load5",
			Value: avg.Load5,
			Type:  Gauge,
			Help:  "5-minute load average",
		},
		&Metric{
			Name:  "system_load15",
			Value: avg.Load15,
			Type:  Gauge,
			Help:  "15-minute load average",
		},
	)

	// Load per CPU
	metrics = append(metrics, &Metric{
		Name:  "system_load_per_cpu",
		Value: avg.Load1 / float64(c.processors),
		Type:  Gauge,
		Help:  "Load average per CPU",
	})

	return metrics, nil
}

func (c *SystemCollector) collectDiskMetrics(ctx context.Context) ([]*Metric, error) {
	var metrics []*Metric

	// Disk usage
	partitions, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}

	for _, partition := range partitions {
		// Skip ignored filesystem types
		if c.shouldIgnoreDisk(partition.Fstype, partition.Mountpoint) {
			continue
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue // Skip partitions we can't read
		}

		labels := map[string]string{
			"device":     partition.Device,
			"mount":      partition.Mountpoint,
			"fstype":     partition.Fstype,
		}

		metrics = append(metrics,
			&Metric{
				Name:   "system_disk_total_bytes",
				Value:  float64(usage.Total),
				Labels: labels,
				Type:   Gauge,
				Help:   "Total disk space in bytes",
				Unit:   "bytes",
			},
			&Metric{
				Name:   "system_disk_used_bytes",
				Value:  float64(usage.Used),
				Labels: labels,
				Type:   Gauge,
				Help:   "Used disk space in bytes",
				Unit:   "bytes",
			},
			&Metric{
				Name:   "system_disk_free_bytes",
				Value:  float64(usage.Free),
				Labels: labels,
				Type:   Gauge,
				Help:   "Free disk space in bytes",
				Unit:   "bytes",
			},
			&Metric{
				Name:   "system_disk_usage_percent",
				Value:  usage.UsedPercent,
				Labels: labels,
				Type:   Gauge,
				Help:   "Disk usage percentage",
				Unit:   "percent",
			},
			&Metric{
				Name:   "system_disk_inodes_total",
				Value:  float64(usage.InodesTotal),
				Labels: labels,
				Type:   Gauge,
				Help:   "Total inodes",
			},
			&Metric{
				Name:   "system_disk_inodes_used",
				Value:  float64(usage.InodesUsed),
				Labels: labels,
				Type:   Gauge,
				Help:   "Used inodes",
			},
			&Metric{
				Name:   "system_disk_inodes_free",
				Value:  float64(usage.InodesFree),
				Labels: labels,
				Type:   Gauge,
				Help:   "Free inodes",
			},
			&Metric{
				Name:   "system_disk_inodes_usage_percent",
				Value:  usage.InodesUsedPercent,
				Labels: labels,
				Type:   Gauge,
				Help:   "Inode usage percentage",
				Unit:   "percent",
			},
		)
	}

	// Disk I/O
	ioCounters, err := disk.IOCounters()
	if err == nil {
		for device, io := range ioCounters {
			labels := map[string]string{"device": device}

			// Calculate rates if we have previous values
			if last, exists := c.lastDisk[device]; exists {
				timeDelta := float64(time.Second) // Assuming 1-second interval

				metrics = append(metrics,
					&Metric{
						Name:   "system_disk_read_bytes_per_second",
						Value:  float64(io.ReadBytes-last.ReadBytes) / timeDelta,
						Labels: labels,
						Type:   Gauge,
						Help:   "Disk read bytes per second",
						Unit:   "bytes",
					},
					&Metric{
						Name:   "system_disk_write_bytes_per_second",
						Value:  float64(io.WriteBytes-last.WriteBytes) / timeDelta,
						Labels: labels,
						Type:   Gauge,
						Help:   "Disk write bytes per second",
						Unit:   "bytes",
					},
					&Metric{
						Name:   "system_disk_read_ops_per_second",
						Value:  float64(io.ReadCount-last.ReadCount) / timeDelta,
						Labels: labels,
						Type:   Gauge,
						Help:   "Disk read operations per second",
					},
					&Metric{
						Name:   "system_disk_write_ops_per_second",
						Value:  float64(io.WriteCount-last.WriteCount) / timeDelta,
						Labels: labels,
						Type:   Gauge,
						Help:   "Disk write operations per second",
					},
				)
			}

			c.lastDisk[device] = io
		}
	}

	return metrics, nil
}

func (c *SystemCollector) collectNetworkMetrics(ctx context.Context) ([]*Metric, error) {
	var metrics []*Metric

	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	for _, io := range ioCounters {
		// Skip loopback if not requested
		if !c.config.Network.IncludeLoopback && strings.HasPrefix(io.Name, "lo") {
			continue
		}

		// Filter interfaces
		if !c.shouldIncludeInterface(io.Name) {
			continue
		}

		labels := map[string]string{"interface": io.Name}

		// Calculate rates if we have previous values
		if last, exists := c.lastNet[io.Name]; exists {
			timeDelta := float64(time.Second) // Assuming 1-second interval

			metrics = append(metrics,
				&Metric{
					Name:   "system_network_receive_bytes_per_second",
					Value:  float64(io.BytesRecv-last.BytesRecv) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network receive bytes per second",
					Unit:   "bytes",
				},
				&Metric{
					Name:   "system_network_transmit_bytes_per_second",
					Value:  float64(io.BytesSent-last.BytesSent) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network transmit bytes per second",
					Unit:   "bytes",
				},
				&Metric{
					Name:   "system_network_receive_packets_per_second",
					Value:  float64(io.PacketsRecv-last.PacketsRecv) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network receive packets per second",
				},
				&Metric{
					Name:   "system_network_transmit_packets_per_second",
					Value:  float64(io.PacketsSent-last.PacketsSent) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network transmit packets per second",
				},
				&Metric{
					Name:   "system_network_receive_errors_per_second",
					Value:  float64(io.Errin-last.Errin) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network receive errors per second",
				},
				&Metric{
					Name:   "system_network_transmit_errors_per_second",
					Value:  float64(io.Errout-last.Errout) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network transmit errors per second",
				},
				&Metric{
					Name:   "system_network_receive_drops_per_second",
					Value:  float64(io.Dropin-last.Dropin) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network receive drops per second",
				},
				&Metric{
					Name:   "system_network_transmit_drops_per_second",
					Value:  float64(io.Dropout-last.Dropout) / timeDelta,
					Labels: labels,
					Type:   Gauge,
					Help:   "Network transmit drops per second",
				},
			)
		}

		// Absolute counters
		metrics = append(metrics,
			&Metric{
				Name:   "system_network_receive_bytes_total",
				Value:  float64(io.BytesRecv),
				Labels: labels,
				Type:   Counter,
				Help:   "Total network receive bytes",
				Unit:   "bytes",
			},
			&Metric{
				Name:   "system_network_transmit_bytes_total",
				Value:  float64(io.BytesSent),
				Labels: labels,
				Type:   Counter,
				Help:   "Total network transmit bytes",
				Unit:   "bytes",
			},
			&Metric{
				Name:   "system_network_receive_errors_total",
				Value:  float64(io.Errin),
				Labels: labels,
				Type:   Counter,
				Help:   "Total network receive errors",
			},
			&Metric{
				Name:   "system_network_transmit_errors_total",
				Value:  float64(io.Errout),
				Labels: labels,
				Type:   Counter,
				Help:   "Total network transmit errors",
			},
		)

		c.lastNet[io.Name] = io
	}

	return metrics, nil
}

func (c *SystemCollector) collectUptimeMetrics(ctx context.Context) ([]*Metric, error) {
	var metrics []*Metric

	uptime, err := host.Uptime()
	if err != nil {
		// Fallback to reading /proc/uptime
		uptime = c.readUptimeFromProc()
	}

	metrics = append(metrics, &Metric{
		Name:  "system_uptime_seconds",
		Value: float64(uptime),
		Type:  Gauge,
		Help:  "System uptime in seconds",
		Unit:  "seconds",
	})

	return metrics, nil
}

func (c *SystemCollector) readUptimeFromProc() uint64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) > 0 {
		if uptime, err := strconv.ParseFloat(fields[0], 64); err == nil {
			return uint64(uptime)
		}
	}

	return 0
}

func (c *SystemCollector) shouldIgnoreDisk(fsType, mountpoint string) bool {
	// Check filesystem type
	for _, ignoreFS := range c.config.Disk.IgnoreFSTypes {
		if fsType == ignoreFS {
			return true
		}
	}

	// Check mount point
	for _, ignoreMount := range c.config.Disk.IgnoreMounts {
		if strings.HasPrefix(mountpoint, ignoreMount) {
			return true
		}
	}

	return false
}

func (c *SystemCollector) shouldIncludeInterface(name string) bool {
	if c.config.Network.IncludeAll {
		return true
	}

	for _, pattern := range c.config.Network.Interfaces {
		if strings.HasPrefix(name, pattern) {
			return true
		}
	}

	return false
}

func totalCPUTime(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Nice + t.Idle + t.Iowait + t.Irq +
		t.Softirq + t.Steal + t.Guest + t.GuestNice
}