<script>
  let apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
  let wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:3000/ws';
  let refreshInterval = 5000;
  let theme = 'light';
  let notifications = true;

  function saveSettings() {
    localStorage.setItem('lnmonja_settings', JSON.stringify({
      apiUrl,
      wsUrl,
      refreshInterval,
      theme,
      notifications
    }));
    alert('Settings saved successfully!');
  }

  function loadSettings() {
    const saved = localStorage.getItem('lnmonja_settings');
    if (saved) {
      const settings = JSON.parse(saved);
      apiUrl = settings.apiUrl || apiUrl;
      wsUrl = settings.wsUrl || wsUrl;
      refreshInterval = settings.refreshInterval || refreshInterval;
      theme = settings.theme || theme;
      notifications = settings.notifications !== undefined ? settings.notifications : true;
    }
  }

  function resetSettings() {
    if (confirm('Reset all settings to defaults?')) {
      localStorage.removeItem('lnmonja_settings');
      location.reload();
    }
  }

  // Load settings on mount
  loadSettings();
</script>

<div class="settings-page">
  <header class="page-header">
    <h1>Settings</h1>
  </header>

  <div class="settings-content">
    <div class="section">
      <h2>Connection</h2>
      <div class="form-group">
        <label>API Base URL</label>
        <input type="text" bind:value={apiUrl} placeholder="http://localhost:8080/api/v1" />
        <span class="hint">Base URL for the lnmonja server API</span>
      </div>

      <div class="form-group">
        <label>WebSocket URL</label>
        <input type="text" bind:value={wsUrl} placeholder="ws://localhost:3000/ws" />
        <span class="hint">WebSocket URL for real-time updates</span>
      </div>
    </div>

    <div class="section">
      <h2>Display</h2>
      <div class="form-group">
        <label>Refresh Interval (ms)</label>
        <input type="number" bind:value={refreshInterval} min="1000" step="1000" />
        <span class="hint">How often to refresh data (minimum 1000ms)</span>
      </div>

      <div class="form-group">
        <label>Theme</label>
        <select bind:value={theme}>
          <option value="light">Light</option>
          <option value="dark">Dark (Coming Soon)</option>
          <option value="auto">Auto (Coming Soon)</option>
        </select>
      </div>

      <div class="form-group">
        <label class="checkbox-label">
          <input type="checkbox" bind:checked={notifications} />
          <span>Enable Browser Notifications</span>
        </label>
        <span class="hint">Receive notifications for critical alerts</span>
      </div>
    </div>

    <div class="section">
      <h2>About</h2>
      <div class="about-info">
        <div class="info-row">
          <strong>Version:</strong>
          <span>1.0.0</span>
        </div>
        <div class="info-row">
          <strong>Dashboard:</strong>
          <span>Svelte + Vite</span>
        </div>
        <div class="info-row">
          <strong>Documentation:</strong>
          <a href="https://github.com/meettoy2004/lnmonja" target="_blank">
            GitHub Repository
          </a>
        </div>
      </div>
    </div>

    <div class="actions">
      <button class="btn btn-secondary" on:click={resetSettings}>
        Reset to Defaults
      </button>
      <button class="btn btn-primary" on:click={saveSettings}>
        Save Settings
      </button>
    </div>
  </div>
</div>

<style>
  .settings-page {
    max-width: 800px;
  }

  .page-header {
    margin-bottom: 2rem;
  }

  .page-header h1 {
    margin: 0;
    font-size: 2rem;
    color: #2c3e50;
  }

  .settings-content {
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }

  .section {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  }

  .section h2 {
    margin: 0 0 1.5rem 0;
    font-size: 1.3rem;
    color: #2c3e50;
  }

  .form-group {
    margin-bottom: 1.5rem;
  }

  .form-group:last-child {
    margin-bottom: 0;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    color: #2c3e50;
  }

  .form-group input[type="text"],
  .form-group input[type="number"],
  .form-group select {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
  }

  .hint {
    display: block;
    margin-top: 0.5rem;
    font-size: 0.85rem;
    color: #7f8c8d;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .checkbox-label input[type="checkbox"] {
    width: 18px;
    height: 18px;
    cursor: pointer;
  }

  .about-info {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    padding: 0.75rem;
    background: #f8f9fa;
    border-radius: 4px;
  }

  .info-row a {
    color: #3498db;
    text-decoration: none;
  }

  .info-row a:hover {
    text-decoration: underline;
  }

  .actions {
    display: flex;
    gap: 1rem;
    justify-content: flex-end;
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
</style>
