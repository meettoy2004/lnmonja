<script>
  export let title = '';
  export let value = 0;
  export let icon = '📊';
  export let color = '#3498db';
</script>

<div class="stat-card" style="border-left-color: {color}">
  <div class="icon" style="color: {color}">
    {icon}
  </div>
  <div class="content">
    <div class="title">{title}</div>
    <div class="value">{value}</div>
  </div>
</div>

<style>
  .stat-card {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    border-left: 4px solid;
    display: flex;
    gap: 1rem;
    align-items: center;
    transition: transform 0.2s, box-shadow 0.2s;
  }

  .stat-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
  }

  .icon {
    font-size: 2.5rem;
  }

  .content {
    flex: 1;
  }

  .title {
    font-size: 0.9rem;
    color: #7f8c8d;
    margin-bottom: 0.5rem;
  }

  .value {
    font-size: 2rem;
    font-weight: 600;
    color: #2c3e50;
  }
</style>
