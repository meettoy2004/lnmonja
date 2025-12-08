<script>
  import { onMount, onDestroy } from 'svelte';
  import api from '../services/api.js';

  let nodes = [];
  let selectedNode = null;
  let loading = true;
  let error = null;
  let refreshInterval;
  let searchQuery = '';
  let filterStatus = 'all'; // all, online, offline

  onMount(async () => {
    await loadNodes();
    refreshInterval = setInterval(loadNodes, 5000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  async function loadNodes() {
    try {
      const data = await api.getNodes();
      nodes = data.nodes || data || [];
      loading = false;
      error = null;
    } catch (err) {
      console.error('Failed to load nodes:', err);
      error = err.message;
      loading = false;
    }
  }

  async function selectNode(node) {
    selectedNode = node;
  }

  async function deleteNode(nodeId) {
    if (!confirm(`Are you sure you want to delete node ${nodeId}?`)) {
      return;
    }

    try {
      await api.deleteNode(nodeId);
      nodes = nodes.filter(n => (n.node_id || n.id) !== nodeId);
      if (selectedNode && (selectedNode.node_id || selectedNode.id) === nodeId) {
        selectedNode = null;
      }
    } catch (err) {
      alert(`Failed to delete node: ${err.message}`);
    }
  }

  function getNodeStatus(node) {
    if (node.connected || node.status === 'active') return 'online';
    return 'offline';
  }

  function formatBytes(bytes) {
    if (!bytes) return 'N/A';
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    if (bytes === 0) return '0 Bytes';
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
  }

  function formatUptime(seconds) {
    if (!seconds) return 'N/A';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${days}d ${hours}h ${mins}m`;
  }

  $: filteredNodes = nodes.filter(node => {
    const matchesSearch = !searchQuery ||
      (node.node_id || node.id || '').toLowerCase().includes(searchQuery.toLowerCase());

    const matchesStatus = filterStatus === 'all' ||
      getNodeStatus(node) === filterStatus;

    return matchesSearch && matchesStatus;
  });
</script>

<div class="nodes-page">
  <header class="page-header">
    <h1>Nodes & Agents</h1>
    <div class="header-actions">
      <button class="btn btn-secondary" on:click={loadNodes}>
        🔄 Refresh
      </button>
    </div>
  </header>

  {#if error}
    <div class="error-banner">
      <strong>Error:</strong> {error}
      <button on:click={loadNodes}>Retry</button>
    </div>
  {/if}

  <div class="controls">
    <input
      type="text"
      placeholder="Search nodes..."
      bind:value={searchQuery}
      class="search-input"
    />
    <select bind:value={filterStatus} class="filter-select">
      <option value="all">All Status</option>
      <option value="online">Online</option>
      <option value="offline">Offline</option>
    </select>
  </div>

  <div class="content-grid">
    <!-- Nodes List -->
    <div class="nodes-list">
      <div class="list-header">
        <h2>Nodes ({filteredNodes.length})</h2>
      </div>
      {#if loading}
        <div class="loading">Loading nodes...</div>
      {:else if filteredNodes.length === 0}
        <div class="empty-state">
          {#if searchQuery || filterStatus !== 'all'}
            <p>No nodes match your filters.</p>
          {:else}
            <p>No nodes connected yet.</p>
            <p class="hint">Start an agent to see it appear here.</p>
          {/if}
        </div>
      {:else}
        <div class="node-items">
          {#each filteredNodes as node}
            <button
              class="node-item"
              class:selected={selectedNode && (selectedNode.node_id || selectedNode.id) === (node.node_id || node.id)}
              on:click={() => selectNode(node)}
            >
              <div class="node-item-header">
                <span class="node-status" class:online={getNodeStatus(node) === 'online'}>●</span>
                <span class="node-name">{node.node_id || node.id || 'Unknown'}</span>
              </div>
              <div class="node-item-meta">
                <span class="meta-item">{getNodeStatus(node)}</span>
                {#if node.last_heartbeat}
                  <span class="meta-item">
                    {new Date(node.last_heartbeat).toLocaleTimeString()}
                  </span>
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Node Details -->
    <div class="node-details">
      {#if !selectedNode}
        <div class="empty-state">
          <p>Select a node to view details</p>
        </div>
      {:else}
        <div class="details-header">
          <h2>{selectedNode.node_id || selectedNode.id}</h2>
          <button
            class="btn btn-danger"
            on:click={() => deleteNode(selectedNode.node_id || selectedNode.id)}
          >
            🗑️ Delete
          </button>
        </div>

        <div class="details-section">
          <h3>Status</h3>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="label">Status:</span>
              <span class="value status-{getNodeStatus(selectedNode)}">
                {getNodeStatus(selectedNode)}
              </span>
            </div>
            <div class="detail-item">
              <span class="label">Session ID:</span>
              <span class="value">{selectedNode.session_id || 'N/A'}</span>
            </div>
            <div class="detail-item">
              <span class="label">Connected At:</span>
              <span class="value">
                {selectedNode.connected_at ? new Date(selectedNode.connected_at).toLocaleString() : 'N/A'}
              </span>
            </div>
            <div class="detail-item">
              <span class="label">Last Heartbeat:</span>
              <span class="value">
                {selectedNode.last_heartbeat ? new Date(selectedNode.last_heartbeat).toLocaleString() : 'N/A'}
              </span>
            </div>
          </div>
        </div>

        {#if selectedNode.metadata}
          <div class="details-section">
            <h3>System Information</h3>
            <div class="detail-grid">
              {#if selectedNode.metadata.os}
                <div class="detail-item">
                  <span class="label">OS:</span>
                  <span class="value">{selectedNode.metadata.os}</span>
                </div>
              {/if}
              {#if selectedNode.metadata.kernel}
                <div class="detail-item">
                  <span class="label">Kernel:</span>
                  <span class="value">{selectedNode.metadata.kernel}</span>
                </div>
              {/if}
              {#if selectedNode.metadata.cpu_cores}
                <div class="detail-item">
                  <span class="label">CPU Cores:</span>
                  <span class="value">{selectedNode.metadata.cpu_cores}</span>
                </div>
              {/if}
              {#if selectedNode.metadata.total_memory}
                <div class="detail-item">
                  <span class="label">Total Memory:</span>
                  <span class="value">{formatBytes(selectedNode.metadata.total_memory)}</span>
                </div>
              {/if}
              {#if selectedNode.uptime}
                <div class="detail-item">
                  <span class="label">Uptime:</span>
                  <span class="value">{formatUptime(selectedNode.uptime)}</span>
                </div>
              {/if}
            </div>
          </div>
        {/if}

        {#if selectedNode.collectors}
          <div class="details-section">
            <h3>Active Collectors</h3>
            <div class="collectors-list">
              {#each Object.entries(selectedNode.collectors) as [name, enabled]}
                <div class="collector-item" class:enabled>
                  <span class="collector-name">{name}</span>
                  <span class="collector-status">{enabled ? '✅ Enabled' : '❌ Disabled'}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if selectedNode.tags && Object.keys(selectedNode.tags).length > 0}
          <div class="details-section">
            <h3>Tags</h3>
            <div class="tags-list">
              {#each Object.entries(selectedNode.tags) as [key, value]}
                <span class="tag">{key}: {value}</span>
              {/each}
            </div>
          </div>
        {/if}
      {/if}
    </div>
  </div>
</div>

<style>
  .nodes-page {
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

  .btn-danger {
    background: #e74c3c;
    color: white;
  }

  .btn-danger:hover {
    background: #c0392b;
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

  .controls {
    display: flex;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .search-input {
    flex: 1;
    padding: 0.75rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
  }

  .filter-select {
    padding: 0.75rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    background: white;
  }

  .content-grid {
    display: grid;
    grid-template-columns: 350px 1fr;
    gap: 1.5rem;
  }

  .nodes-list {
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    overflow: hidden;
  }

  .list-header {
    padding: 1rem 1.5rem;
    border-bottom: 1px solid #e9ecef;
  }

  .list-header h2 {
    margin: 0;
    font-size: 1.2rem;
    color: #2c3e50;
  }

  .loading,
  .empty-state {
    padding: 2rem;
    text-align: center;
    color: #7f8c8d;
  }

  .empty-state .hint {
    font-size: 0.9rem;
    color: #95a5a6;
    margin-top: 0.5rem;
  }

  .node-items {
    max-height: calc(100vh - 250px);
    overflow-y: auto;
  }

  .node-item {
    width: 100%;
    padding: 1rem 1.5rem;
    background: white;
    border: none;
    border-bottom: 1px solid #e9ecef;
    cursor: pointer;
    text-align: left;
    transition: background 0.2s;
  }

  .node-item:hover {
    background: #f8f9fa;
  }

  .node-item.selected {
    background: #e3f2fd;
    border-left: 3px solid #3498db;
  }

  .node-item-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .node-status {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #95a5a6;
  }

  .node-status.online {
    background: #2ecc71;
  }

  .node-name {
    font-weight: 500;
    color: #2c3e50;
  }

  .node-item-meta {
    display: flex;
    gap: 1rem;
    font-size: 0.85rem;
    color: #7f8c8d;
  }

  .node-details {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  }

  .details-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
    padding-bottom: 1rem;
    border-bottom: 2px solid #e9ecef;
  }

  .details-header h2 {
    margin: 0;
    font-size: 1.5rem;
    color: #2c3e50;
  }

  .details-section {
    margin-bottom: 2rem;
  }

  .details-section h3 {
    margin: 0 0 1rem 0;
    font-size: 1.1rem;
    color: #2c3e50;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 1rem;
  }

  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .detail-item .label {
    font-size: 0.85rem;
    color: #7f8c8d;
  }

  .detail-item .value {
    font-weight: 500;
    color: #2c3e50;
  }

  .status-online {
    color: #2ecc71 !important;
  }

  .status-offline {
    color: #e74c3c !important;
  }

  .collectors-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .collector-item {
    display: flex;
    justify-content: space-between;
    padding: 0.75rem;
    background: #f8f9fa;
    border-radius: 4px;
  }

  .collector-name {
    font-weight: 500;
  }

  .collector-status {
    font-size: 0.9rem;
  }

  .tags-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .tag {
    padding: 0.5rem 1rem;
    background: #e3f2fd;
    border-radius: 20px;
    font-size: 0.9rem;
    color: #2c3e50;
  }

  @media (max-width: 768px) {
    .content-grid {
      grid-template-columns: 1fr;
    }

    .nodes-list {
      max-height: 400px;
    }
  }
</style>
