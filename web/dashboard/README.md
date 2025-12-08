# LnMonja Web Dashboard

A modern, real-time monitoring dashboard for LnMonja built with Svelte and Vite.

## Features

- 📊 **Real-time Dashboard**: Live metrics visualization
- 🖥️ **Node Management**: View and manage all connected agents
- 🔔 **Alert System**: Create and manage alert rules
- ⚙️ **Settings**: Configure dashboard behavior
- 📱 **Responsive**: Works on desktop and mobile devices

## Prerequisites

- Node.js 18+ and npm
- Running lnmonja-server (see main README.md)

## Quick Start

### 1. Install Dependencies

```bash
cd web/dashboard
npm install
```

### 2. Configure API Endpoint

Create a `.env` file:

```bash
# web/dashboard/.env
VITE_API_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:3000/ws
```

### 3. Development Mode

```bash
npm run dev
```

The dashboard will be available at `http://localhost:5173`

### 4. Build for Production

```bash
npm run build
```

Built files will be in `dist/` directory.

### 5. Preview Production Build

```bash
npm run preview
```

## Project Structure

```
web/dashboard/
├── src/
│   ├── App.svelte           # Main app with routing
│   ├── main.js              # Entry point
│   ├── pages/               # Page components
│   │   ├── Dashboard.svelte # Overview dashboard
│   │   ├── Nodes.svelte     # Node management
│   │   ├── Alerts.svelte    # Alert configuration
│   │   └── Settings.svelte  # Settings page
│   ├── components/          # Reusable components
│   │   ├── StatCard.svelte  # Stat display card
│   │   └── MetricChart.svelte # Chart component
│   ├── services/            # API services
│   │   └── api.js           # Backend API client
│   └── styles/
│       └── global.css       # Global styles
├── index.html
├── package.json
└── vite.config.js

```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run check` - Run Svelte check (type checking)

## Configuration

### Environment Variables

Create `.env` file in `web/dashboard/`:

```env
# API Endpoint
VITE_API_URL=http://localhost:8080/api/v1

# WebSocket Endpoint
VITE_WS_URL=ws://localhost:3000/ws
```

### Runtime Configuration

You can also configure endpoints through the Settings page in the UI.

## Deployment

### Static Hosting

After building, deploy the `dist/` directory to any static hosting service:

```bash
npm run build
# Upload dist/ folder to your hosting provider
```

### Nginx Example

```nginx
server {
    listen 80;
    server_name dashboard.example.com;
    root /var/www/lnmonja-dashboard;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # API Proxy (optional)
    location /api/ {
        proxy_pass http://localhost:8080/api/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # WebSocket Proxy (optional)
    location /ws {
        proxy_pass http://localhost:3000/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }
}
```

### Docker

Build dashboard and copy to server:

```bash
# In web/dashboard/
npm run build

# Copy to server container
docker cp dist/. lnmonja-server:/var/www/dashboard/
```

Or use the provided docker-compose.yml which includes an nginx container for the dashboard.

## API Integration

The dashboard communicates with the lnmonja-server via REST API and WebSocket.

### REST API Endpoints

- `GET /api/v1/health` - Health check
- `GET /api/v1/nodes` - List all nodes
- `GET /api/v1/nodes/:id` - Get node details
- `GET /api/v1/metrics` - Query metrics
- `GET /api/v1/alerts` - Get active alerts
- `POST /api/v1/alert-rules` - Create alert rule
- `PUT /api/v1/alert-rules/:id` - Update alert rule
- `DELETE /api/v1/alert-rules/:id` - Delete alert rule

### WebSocket

Connect to `ws://server:3000/ws` for real-time metric updates.

## Customization

### Themes

Edit `src/styles/global.css` to customize colors and typography.

### Adding Pages

1. Create new component in `src/pages/`
2. Add route in `src/App.svelte`
3. Add navigation item in sidebar

### Adding Charts

Use the `MetricChart` component:

```svelte
<MetricChart
  title="CPU Usage"
  metric="system_cpu_usage_total"
  nodes={nodeIds}
/>
```

## Troubleshooting

### Cannot connect to API

- Verify lnmonja-server is running
- Check API URL in Settings page
- Check browser console for CORS errors
- Ensure CORS is enabled in server config

### No metrics showing

- Wait 30-60 seconds after starting agent
- Check that agents are connected (Nodes page)
- Verify metrics are being collected in server logs
- Check time range in query parameters

### Build errors

```bash
# Clear cache and reinstall
rm -rf node_modules package-lock.json
npm install
npm run build
```

## Development

### Hot Reload

Changes to `.svelte` files will automatically reload in dev mode.

### Adding Dependencies

```bash
npm install <package-name>
```

### Code Style

- Use 2 spaces for indentation
- Use single quotes for strings
- Add comments for complex logic

## Contributing

See main project README.md for contribution guidelines.

## License

Same as main project.
