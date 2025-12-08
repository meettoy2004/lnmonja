<script>
  import { onMount } from 'svelte';
  import api from '../services/api.js';

  let activeAlerts = [];
  let alertRules = [];
  let alertHistory = [];
  let loading = true;
  let showCreateModal = false;
  let newRule = {
    name: '',
    metric: 'system_cpu_usage_total',
    condition: '>',
    threshold: 80,
    duration: '5m',
    severity: 'warning',
    enabled: true
  };

  onMount(async () => {
    await loadAlerts();
  });

  async function loadAlerts() {
    try {
      const [alerts, rules, history] = await Promise.all([
        api.getAlerts().catch(() => ({ alerts: [] })),
        api.getAlertRules().catch(() => ({ rules: [] })),
        api.getAlertHistory({ limit: 50 }).catch(() => ({ history: [] }))
      ]);

      activeAlerts = alerts.alerts || alerts || [];
      alertRules = rules.rules || rules || [];
      alertHistory = history.history || history || [];
      loading = false;
    } catch (err) {
      console.error('Failed to load alerts:', err);
      loading = false;
    }
  }

  async function createRule() {
    try {
      await api.createAlertRule(newRule);
      showCreateModal = false;
      newRule = {
        name: '',
        metric: 'system_cpu_usage_total',
        condition: '>',
        threshold: 80,
        duration: '5m',
        severity: 'warning',
        enabled: true
      };
      await loadAlerts();
    } catch (err) {
      alert(`Failed to create rule: ${err.message}`);
    }
  }

  async function toggleRule(rule) {
    try {
      await api.updateAlertRule(rule.id, { ...rule, enabled: !rule.enabled });
      await loadAlerts();
    } catch (err) {
      alert(`Failed to update rule: ${err.message}`);
    }
  }

  async function deleteRule(ruleId) {
    if (!confirm('Are you sure you want to delete this rule?')) {
      return;
    }
    try {
      await api.deleteAlertRule(ruleId);
      await loadAlerts();
    } catch (err) {
      alert(`Failed to delete rule: ${err.message}`);
    }
  }

  function getSeverityColor(severity) {
    const colors = {
      critical: '#e74c3c',
      warning: '#f39c12',
      info: '#3498db'
    };
    return colors[severity] || '#95a5a6';
  }

  function formatTimestamp(ts) {
    if (!ts) return 'N/A';
    return new Date(ts).toLocaleString();
  }
</script>

<div class="alerts-page">
  <header class="page-header">
    <h1>Alerts & Rules</h1>
    <div class="header-actions">
      <button class="btn btn-primary" on:click={() => showCreateModal = true}>
        ➕ Create Rule
      </button>
      <button class="btn btn-secondary" on:click={loadAlerts}>
        🔄 Refresh
      </button>
    </div>
  </header>

  {#if loading}
    <div class="loading">Loading alerts...</div>
  {:else}
    <!-- Active Alerts -->
    {#if activeAlerts.length > 0}
      <div class="section">
        <h2>Active Alerts ({activeAlerts.length})</h2>
        <div class="alerts-list">
          {#each activeAlerts as alert}
            <div class="alert-card" style="border-left-color: {getSeverityColor(alert.severity)}">
              <div class="alert-header">
                <span class="alert-title">{alert.name || alert.rule_name}</span>
                <span class="alert-severity" style="background: {getSeverityColor(alert.severity)}">
                  {alert.severity}
                </span>
              </div>
              <div class="alert-body">
                <p>{alert.description || alert.message}</p>
                <div class="alert-meta">
                  <span>Node: {alert.node_id || 'All'}</span>
                  <span>Started: {formatTimestamp(alert.started_at)}</span>
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Alert Rules -->
    <div class="section">
      <h2>Alert Rules ({alertRules.length})</h2>
      {#if alertRules.length === 0}
        <div class="empty-state">
          <p>No alert rules configured yet.</p>
          <button class="btn btn-primary" on:click={() => showCreateModal = true}>
            Create Your First Rule
          </button>
        </div>
      {:else}
        <div class="rules-table">
          <table>
            <thead>
              <tr>
                <th>Status</th>
                <th>Name</th>
                <th>Condition</th>
                <th>Severity</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each alertRules as rule}
                <tr>
                  <td>
                    <label class="toggle">
                      <input
                        type="checkbox"
                        checked={rule.enabled}
                        on:change={() => toggleRule(rule)}
                      />
                      <span class="slider"></span>
                    </label>
                  </td>
                  <td><strong>{rule.name}</strong></td>
                  <td>
                    <code>{rule.metric} {rule.condition} {rule.threshold}</code>
                    for {rule.duration || '1m'}
                  </td>
                  <td>
                    <span class="severity-badge" style="background: {getSeverityColor(rule.severity)}">
                      {rule.severity}
                    </span>
                  </td>
                  <td>
                    <button class="btn-icon" on:click={() => deleteRule(rule.id)}>
                      🗑️
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>

    <!-- Alert History -->
    {#if alertHistory.length > 0}
      <div class="section">
        <h2>Alert History</h2>
        <div class="history-list">
          {#each alertHistory as alert}
            <div class="history-item">
              <span class="history-severity" style="background: {getSeverityColor(alert.severity)}"></span>
              <div class="history-content">
                <strong>{alert.name || alert.rule_name}</strong>
                <span class="history-meta">
                  {alert.node_id} • {formatTimestamp(alert.triggered_at)}
                </span>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Create Rule Modal -->
{#if showCreateModal}
  <div class="modal-overlay" on:click={() => showCreateModal = false}>
    <div class="modal" on:click|stopPropagation>
      <div class="modal-header">
        <h2>Create Alert Rule</h2>
        <button class="close-btn" on:click={() => showCreateModal = false}>×</button>
      </div>
      <form on:submit|preventDefault={createRule}>
        <div class="form-group">
          <label>Rule Name</label>
          <input type="text" bind:value={newRule.name} required />
        </div>

        <div class="form-group">
          <label>Metric</label>
          <select bind:value={newRule.metric}>
            <option value="system_cpu_usage_total">CPU Usage</option>
            <option value="system_memory_usage_percent">Memory Usage</option>
            <option value="system_disk_usage_percent">Disk Usage</option>
            <option value="system_load1">Load Average</option>
          </select>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label>Condition</label>
            <select bind:value={newRule.condition}>
              <option value=">">Greater than</option>
              <option value="<">Less than</option>
              <option value=">=">Greater or equal</option>
              <option value="<=">Less or equal</option>
              <option value="==">Equal</option>
            </select>
          </div>

          <div class="form-group">
            <label>Threshold</label>
            <input type="number" bind:value={newRule.threshold} required />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label>Duration</label>
            <input type="text" bind:value={newRule.duration} placeholder="5m" />
          </div>

          <div class="form-group">
            <label>Severity</label>
            <select bind:value={newRule.severity}>
              <option value="info">Info</option>
              <option value="warning">Warning</option>
              <option value="critical">Critical</option>
            </select>
          </div>
        </div>

        <div class="form-actions">
          <button type="button" class="btn btn-secondary" on:click={() => showCreateModal = false}>
            Cancel
          </button>
          <button type="submit" class="btn btn-primary">
            Create Rule
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .alerts-page {
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

  .btn-primary {
    background: #3498db;
    color: white;
  }

  .btn-primary:hover {
    background: #2980b9;
  }

  .btn-secondary {
    background: white;
    color: #2c3e50;
    border: 1px solid #ddd;
  }

  .btn-secondary:hover {
    background: #f8f9fa;
  }

  .btn-icon {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 1.2rem;
    padding: 0.5rem;
  }

  .loading {
    text-align: center;
    padding: 4rem;
    color: #7f8c8d;
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

  .alerts-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .alert-card {
    border: 1px solid #e9ecef;
    border-left-width: 4px;
    border-radius: 4px;
    padding: 1rem;
  }

  .alert-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .alert-title {
    font-weight: 600;
    color: #2c3e50;
  }

  .alert-severity {
    padding: 0.25rem 0.75rem;
    border-radius: 20px;
    color: white;
    font-size: 0.85rem;
    text-transform: uppercase;
  }

  .alert-body p {
    margin: 0 0 0.5rem 0;
    color: #7f8c8d;
  }

  .alert-meta {
    display: flex;
    gap: 1rem;
    font-size: 0.85rem;
    color: #95a5a6;
  }

  .rules-table {
    overflow-x: auto;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th, td {
    text-align: left;
    padding: 1rem;
    border-bottom: 1px solid #e9ecef;
  }

  th {
    background: #f8f9fa;
    font-weight: 600;
    color: #2c3e50;
  }

  code {
    background: #f8f9fa;
    padding: 0.2rem 0.4rem;
    border-radius: 3px;
    font-family: monospace;
  }

  .severity-badge {
    padding: 0.25rem 0.75rem;
    border-radius: 20px;
    color: white;
    font-size: 0.85rem;
    text-transform: uppercase;
    display: inline-block;
  }

  .toggle {
    position: relative;
    display: inline-block;
    width: 50px;
    height: 24px;
  }

  .toggle input {
    opacity: 0;
    width: 0;
    height: 0;
  }

  .slider {
    position: absolute;
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: #ccc;
    transition: .4s;
    border-radius: 24px;
  }

  .slider:before {
    position: absolute;
    content: "";
    height: 18px;
    width: 18px;
    left: 3px;
    bottom: 3px;
    background-color: white;
    transition: .4s;
    border-radius: 50%;
  }

  input:checked + .slider {
    background-color: #2ecc71;
  }

  input:checked + .slider:before {
    transform: translateX(26px);
  }

  .history-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .history-item {
    display: flex;
    gap: 1rem;
    padding: 0.75rem;
    background: #f8f9fa;
    border-radius: 4px;
  }

  .history-severity {
    width: 4px;
    border-radius: 2px;
  }

  .history-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .history-meta {
    font-size: 0.85rem;
    color: #7f8c8d;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: white;
    border-radius: 8px;
    width: 90%;
    max-width: 600px;
    max-height: 90vh;
    overflow-y: auto;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.5rem;
    border-bottom: 1px solid #e9ecef;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 1.5rem;
    color: #2c3e50;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 2rem;
    cursor: pointer;
    color: #7f8c8d;
  }

  form {
    padding: 1.5rem;
  }

  .form-group {
    margin-bottom: 1.5rem;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    color: #2c3e50;
  }

  .form-group input,
  .form-group select {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }

  .form-actions {
    display: flex;
    gap: 1rem;
    justify-content: flex-end;
    margin-top: 2rem;
  }
</style>
