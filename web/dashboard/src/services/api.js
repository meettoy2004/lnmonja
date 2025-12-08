import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

class APIService {
  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add response interceptor for error handling
    this.client.interceptors.response.use(
      response => response,
      error => {
        console.error('API Error:', error);
        return Promise.reject(error);
      }
    );
  }

  // Health check
  async getHealth() {
    const response = await this.client.get('/health');
    return response.data;
  }

  // Nodes
  async getNodes() {
    const response = await this.client.get('/nodes');
    return response.data;
  }

  async getNode(nodeId) {
    const response = await this.client.get(`/nodes/${nodeId}`);
    return response.data;
  }

  async deleteNode(nodeId) {
    const response = await this.client.delete(`/nodes/${nodeId}`);
    return response.data;
  }

  // Metrics
  async getMetrics(params = {}) {
    const response = await this.client.get('/metrics', { params });
    return response.data;
  }

  async getMetricsByNode(nodeId, params = {}) {
    const response = await this.client.get(`/nodes/${nodeId}/metrics`, { params });
    return response.data;
  }

  // Query metrics with time range
  async queryMetrics({ node, metric, start, end, limit = 1000 }) {
    const params = { node, metric, start, end, limit };
    const response = await this.client.get('/metrics', { params });
    return response.data;
  }

  // Alerts
  async getAlerts() {
    const response = await this.client.get('/alerts');
    return response.data;
  }

  async createAlert(alert) {
    const response = await this.client.post('/alerts', alert);
    return response.data;
  }

  async updateAlert(alertId, alert) {
    const response = await this.client.put(`/alerts/${alertId}`, alert);
    return response.data;
  }

  async deleteAlert(alertId) {
    const response = await this.client.delete(`/alerts/${alertId}`);
    return response.data;
  }

  async getAlertHistory(params = {}) {
    const response = await this.client.get('/alerts/history', { params });
    return response.data;
  }

  // Alert rules
  async getAlertRules() {
    const response = await this.client.get('/alert-rules');
    return response.data;
  }

  async createAlertRule(rule) {
    const response = await this.client.post('/alert-rules', rule);
    return response.data;
  }

  async updateAlertRule(ruleId, rule) {
    const response = await this.client.put(`/alert-rules/${ruleId}`, rule);
    return response.data;
  }

  async deleteAlertRule(ruleId) {
    const response = await this.client.delete(`/alert-rules/${ruleId}`);
    return response.data;
  }

  // Stats
  async getStats() {
    const response = await this.client.get('/stats');
    return response.data;
  }
}

export const api = new APIService();
export default api;
