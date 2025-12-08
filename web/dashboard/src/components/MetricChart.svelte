<script>
  import { onMount, onDestroy } from 'svelte';
  import api from '../services/api.js';

  export let title = 'Metric';
  export let metric = '';
  export let nodes = [];

  let chartData = [];
  let loading = true;
  let error = null;
  let canvas;
  let updateInterval;

  onMount(async () => {
    await loadData();
    updateInterval = setInterval(loadData, 5000);
  });

  onDestroy(() => {
    if (updateInterval) {
      clearInterval(updateInterval);
    }
  });

  async function loadData() {
    try {
      const now = Math.floor(Date.now() / 1000);
      const start = now - 300;

      const promises = nodes.map(node =>
        api.queryMetrics({
          node,
          metric,
          start,
          end: now,
          limit: 60
        }).catch(() => [])
      );

      const results = await Promise.all(promises);
      chartData = results.flat();
      loading = false;
      error = null;

      drawChart();
    } catch (err) {
      console.error('Failed to load metric data:', err);
      error = err.message;
      loading = false;
    }
  }

  function drawChart() {
    if (!canvas || chartData.length === 0) return;

    const ctx = canvas.getContext('2d');
    const width = canvas.width;
    const height = canvas.height;

    ctx.clearRect(0, 0, width, height);

    const values = chartData.map(d => d.value);
    const max = Math.max(...values, 0);
    const min = Math.min(...values, 0);
    const range = max - min || 1;

    ctx.strokeStyle = '#e9ecef';
    ctx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
      const y = (height / 4) * i;
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(width, y);
      ctx.stroke();
    }

    if (chartData.length > 0) {
      ctx.strokeStyle = '#3498db';
      ctx.lineWidth = 2;
      ctx.beginPath();

      chartData.forEach((point, i) => {
        const x = (width / (chartData.length - 1 || 1)) * i;
        const y = height - ((point.value - min) / range) * height;

        if (i === 0) {
          ctx.moveTo(x, y);
        } else {
          ctx.lineTo(x, y);
        }
      });

      ctx.stroke();

      ctx.lineTo(width, height);
      ctx.lineTo(0, height);
      ctx.closePath();
      ctx.fillStyle = 'rgba(52, 152, 219, 0.1)';
      ctx.fill();
    }

    if (chartData.length > 0) {
      const lastValue = chartData[chartData.length - 1].value;
      ctx.fillStyle = '#2c3e50';
      ctx.font = '14px sans-serif';
      ctx.fillText(lastValue.toFixed(2), width - 60, 20);
    }
  }
</script>

<div class="metric-chart">
  <div class="chart-header">
    <h3>{title}</h3>
    {#if loading && chartData.length === 0}
      <span class="status">Loading...</span>
    {:else if error}
      <span class="status error">Error</span>
    {:else}
      <span class="status">Live</span>
    {/if}
  </div>
  <div class="chart-container">
    {#if chartData.length === 0}
      <div class="empty-state">No data available</div>
    {:else}
      <canvas bind:this={canvas} width="600" height="200"></canvas>
    {/if}
  </div>
</div>

<style>
  .metric-chart {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  }

  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .chart-header h3 {
    margin: 0;
    font-size: 1.1rem;
    color: #2c3e50;
  }

  .status {
    font-size: 0.85rem;
    color: #2ecc71;
    font-weight: 500;
  }

  .status.error {
    color: #e74c3c;
  }

  .chart-container {
    position: relative;
    width: 100%;
    height: 200px;
  }

  canvas {
    width: 100%;
    height: 100%;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #7f8c8d;
  }
</style>
