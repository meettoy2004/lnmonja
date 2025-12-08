<script>
  import { onMount } from 'svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import Nodes from './pages/Nodes.svelte';
  import Alerts from './pages/Alerts.svelte';
  import Settings from './pages/Settings.svelte';

  let currentPage = 'dashboard';
  let sidebarOpen = true;

  function navigate(page) {
    currentPage = page;
  }

  onMount(() => {
    // Check if we have a hash route
    const hash = window.location.hash.slice(1);
    if (hash) {
      currentPage = hash;
    }
  });

  function toggleSidebar() {
    sidebarOpen = !sidebarOpen;
  }
</script>

<div class="app">
  <!-- Sidebar -->
  <aside class="sidebar" class:closed={!sidebarOpen}>
    <div class="sidebar-header">
      <h1>LnMonja</h1>
      <button class="toggle-btn" on:click={toggleSidebar}>
        {sidebarOpen ? '◀' : '▶'}
      </button>
    </div>

    <nav class="nav">
      <button
        class="nav-item"
        class:active={currentPage === 'dashboard'}
        on:click={() => navigate('dashboard')}
      >
        <span class="icon">📊</span>
        {#if sidebarOpen}<span>Dashboard</span>{/if}
      </button>

      <button
        class="nav-item"
        class:active={currentPage === 'nodes'}
        on:click={() => navigate('nodes')}
      >
        <span class="icon">🖥️</span>
        {#if sidebarOpen}<span>Nodes & Agents</span>{/if}
      </button>

      <button
        class="nav-item"
        class:active={currentPage === 'alerts'}
        on:click={() => navigate('alerts')}
      >
        <span class="icon">🔔</span>
        {#if sidebarOpen}<span>Alerts</span>{/if}
      </button>

      <button
        class="nav-item"
        class:active={currentPage === 'settings'}
        on:click={() => navigate('settings')}
      >
        <span class="icon">⚙️</span>
        {#if sidebarOpen}<span>Settings</span>{/if}
      </button>
    </nav>
  </aside>

  <!-- Main content -->
  <main class="main-content">
    {#if currentPage === 'dashboard'}
      <Dashboard />
    {:else if currentPage === 'nodes'}
      <Nodes />
    {:else if currentPage === 'alerts'}
      <Alerts />
    {:else if currentPage === 'settings'}
      <Settings />
    {/if}
  </main>
</div>

<style>
  .app {
    display: flex;
    height: 100vh;
    background: #f5f7fa;
  }

  .sidebar {
    width: 250px;
    background: #2c3e50;
    color: white;
    transition: width 0.3s ease;
    display: flex;
    flex-direction: column;
  }

  .sidebar.closed {
    width: 70px;
  }

  .sidebar-header {
    padding: 1.5rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .sidebar-header h1 {
    font-size: 1.5rem;
    margin: 0;
    font-weight: 600;
  }

  .sidebar.closed .sidebar-header h1 {
    display: none;
  }

  .toggle-btn {
    background: none;
    border: none;
    color: white;
    cursor: pointer;
    font-size: 1.2rem;
    padding: 0.5rem;
  }

  .nav {
    flex: 1;
    padding: 1rem 0;
  }

  .nav-item {
    width: 100%;
    padding: 1rem 1.5rem;
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 1rem;
    font-size: 1rem;
    transition: all 0.2s;
    text-align: left;
  }

  .nav-item:hover {
    background: rgba(255, 255, 255, 0.05);
    color: white;
  }

  .nav-item.active {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    border-left: 3px solid #3498db;
  }

  .nav-item .icon {
    font-size: 1.5rem;
  }

  .sidebar.closed .nav-item {
    justify-content: center;
    padding: 1rem;
  }

  .main-content {
    flex: 1;
    overflow-y: auto;
    padding: 2rem;
  }

  @media (max-width: 768px) {
    .sidebar {
      position: fixed;
      z-index: 100;
      height: 100vh;
    }

    .sidebar.closed {
      width: 0;
    }

    .main-content {
      margin-left: 0;
    }
  }
</style>
