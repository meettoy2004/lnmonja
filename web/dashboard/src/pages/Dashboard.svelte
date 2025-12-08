<script>
  import { onMount, onDestroy } from 'svelte';
  import api from '../services/api.js';
  import MetricChart from '../components/MetricChart.svelte';
  import StatCard from '../components/StatCard.svelte';

  let nodes = [];
  let stats = {
    totalNodes: 0,
    activeNodes: 0,
    totalMetrics: 0,
    alertsCount: 0
  };
  let loading = true;
  let error = null;
  let refreshInterval;

  onMount(async () => {
    await loadData();
    // Refresh every 5 seconds
    refreshInterval = setInterval(loadData, 5000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  async function loadData() {
    try {
      const [nodesData, statsData] = await Promise.all([
        api.getNodes().catch(() => ({ nodes: [] })),
        api.getStats().catch(() => ({}))
      ]);

      nodes = nodesData.nodes || nodesData || [];

      // Calculate stats
      stats.totalNodes = nodes.length;
      stats.activeNodes = nodes.filter(n => n.status === 'active' || n.connected).length;
      stats.totalMetrics = statsData.total_metrics || 0;
      stats.alertsCount = statsData.active_alerts || 0;

      loading = false;
      error = null;
    } catch (err) {
      console.error('Failed to load dashboard data:', err);
      error = err.message;
      loading = false;
    }
  }

  function getNodeStatus(node) {
    if (node.connected || node.status === 'active') return 'online';
    return 'offline';
  }

  function formatUptime(seconds) {
    if (!seconds) return 'N/A';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${days}d ${hours}h ${mins}m`;
  }
</script>

<div class="dashboard">
  <header class="page-header">
    <h1>Dashboard</h1>
    <div class="header-actions">
      <button class="btn btn-secondary" on:click={loadData}>
        🔄 Refresh
      </button>
    </div>
  </header>

  {#if loading}
    <div class="loading">Loading...</div>
  {:else if error}
    <div class="error-banner">
      <strong>Error:</strong> {error}
      <button on:click={loadData}>Retry</button>
    </div>
  {:else}
    <!-- Stats Cards -->
    <div class="stats-grid">
      <StatCard
        title="Total Nodes"
        value={stats.totalNodes}
        icon="🖥️"
        color="#3498db"
      />
      <StatCard
        title="Active Nodes"
        value={stats.activeNodes}
        icon="✅"
        color="#2ecc71"
      />
      <StatCard
        title="Total Metrics"
        value={stats.totalMetrics.toLocaleString()}
        icon="📊"
        color="#9b59b6"
      />
      <StatCard
        title="Active Alerts"
        value={stats.alertsCount}
        icon="🔔"
        color={stats.alertsCount > 0 ? '#e74c3c' : '#95a5a6'}
      />
    </div>

    <!-- Nodes Overview -->
    <div class="section">
      <h2>Nodes Overview</h2>
      {#if nodes.length === 0}
        <div class="empty-state">
          <p>No nodes connected yet.</p>
          <p class="hint">Start an agent to see it appear here.</p>
        </div>
      {:else}
        <div class="nodes-grid">
          {#each nodes as node}
            <div class="node-card">
              <div class="node-header">
                <span class="node-status" class:online={getNodeStatus(node) === 'online'}>
                  ●
                </span>
                <h3>{node.node_id || node.id || 'Unknown'}</h3>
              </div>
              <div class="node-info">
                <div class="info-item">
                  <span class="label">Status:</span>
                  <span class="value status-{getNodeStatus(node)}">
                    {getNodeStatus(node)}
                  </span>
                </div>
                <div class="info-item">
                  <span class="label">Session:</span>
                  <span class="value">{node.session_id || 'N/A'}</span>
                </div>
                <div class="info-item">
                  <span class="label">Last Seen:</span>
                  <span class="value">
                    {node.last_heartbeat ? new Date(node.last_heartbeat).toLocaleString() : 'N/A'}
                  </span>
                </div>
                {#if node.uptime}
                  <div class="info-item">
                    <span class="label">Uptime:</span>
                    <span class="value">{formatUptime(node.uptime)}</span>
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Recent Metrics Charts -->
    <div class="section">
      <h2>System Metrics</h2>
      <div class="charts-grid">
        {#if nodes.length > 0}
          <MetricChart
            title="CPU Usage"
            metric="system_cpu_usage_total"
            nodes={nodes.map(n => n.node_id || n.id)}
          />
          <MetricChart
            title="Memory Usage"
            metric="system_memory_usage_percent"
            nodes={nodes.map(n => n.node_id || n.id)}
          />
        {:else}
          <div class="empty-state">
            <p>No metrics available yet.</p>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .dashboard {
    max-width: 1400px;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  .page-header h1 {
    margin: 0;
    font-size: 2rem;
    color: #2c3e50;
  }

  .header-actions {
    display: flex;
    gap: 1rem;
  }

  .btn {
    padding: 0.75rem 1.5rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 1rem;
    transition: all 0.2s;
  }

  .btn-secondary {
    background: white;
    color: #2c3e50;
    border: 1px solid #ddd;
  }

  .btn-secondary:hover {
    background: #f8f9fa;
  }

  .loading {
    text-align: center;
    padding: 4rem;
    font-size: 1.2rem;
    color: #7f8c8d;
  }

  .error-banner {
    background: #fee;
    border: 1px solid #fcc;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 2rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 1.5rem;
    margin-bottom: 2rem;
  }

  .section {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    margin-bottom: 2rem;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  }

  .section h2 {
    margin: 0 0 1.5rem 0;
    font-size: 1.5rem;
    color: #2c3e50;
  }

  .empty-state {
    text-align: center;
    padding: 3rem;
    color: #7f8c8d;
  }

  .empty-state .hint {
    font-size: 0.9rem;
    color: #95a5a6;
  }

  .nodes-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1rem;
  }

  .node-card {
    background: #f8f9fa;
    border: 1px solid #e9ecef;
    border-radius: 6px;
    padding: 1rem;
  }

  .node-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .node-status {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: #95a5a6;
  }

  .node-status.online {
    background: #2ecc71;
  }

  .node-header h3 {
    margin: 0;
    font-size: 1.1rem;
    color: #2c3e50;
  }

  .node-info {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .info-item {
    display: flex;
    justify-content: space-between;
    font-size: 0.9rem;
  }

  .info-item .label {
    color: #7f8c8d;
  }

  .info-item .value {
    font-weight: 500;
    color: #2c3e50;
  }

  .status-online {
    color: #2ecc71 !important;
  }

  .status-offline {
    color: #e74c3c !important;
  }

  .charts-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
    gap: 1.5rem;
  }

  @media (max-width: 768px) {
    .stats-grid {
      grid-template-columns: 1fr;
    }

    .nodes-grid {
      grid-template-columns: 1fr;
    }

    .charts-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
