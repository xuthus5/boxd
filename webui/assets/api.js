const API_ROOT = '/api/v1';

export class APIError extends Error {
  constructor(message, { code = 'request_failed', status = 0, op = '', hint = '' } = {}) {
    super(message);
    this.name = 'APIError';
    this.code = code;
    this.status = status;
    this.op = op;
    this.hint = hint;
  }
}

async function request(path, options = {}) {
  const requestOptions = {
    method: options.method || 'GET',
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
  };
  if (options.body !== undefined) {
    requestOptions.headers['Content-Type'] = 'application/json';
    requestOptions.body = JSON.stringify(options.body);
  }
  let response;
  try {
    response = await fetch(`${API_ROOT}${path}`, requestOptions);
  } catch {
    throw new APIError('无法连接控制面', { code: 'network_error', hint: '检查 boxd 服务是否正在运行' });
  }
  const contentType = response.headers.get('content-type') || '';
  let payload;
  if (contentType.includes('application/json')) {
    payload = await response.json().catch(() => null);
  } else {
    payload = await response.text().catch(() => '');
  }
  if (!response.ok) {
    const detail = payload?.error || {};
    const error = new APIError(detail.message || `请求失败 (${response.status})`, {
      code: detail.code || 'request_failed',
      status: response.status,
      op: detail.op || '',
      hint: detail.hint || '',
    });
    if (response.status === 401 && !path.startsWith('/auth/')) {
      window.dispatchEvent(new CustomEvent('boxd:unauthorized'));
    }
    throw error;
  }
  return payload;
}

export const api = {
  authStatus: () => request('/auth/status'),
  initialize: (password) => request('/auth/initialize', { method: 'POST', body: { password } }),
  login: (password) => request('/auth/login', { method: 'POST', body: { password } }),
  rotatePassword: () => request('/auth/rotate-password', { method: 'POST' }),
  logout: () => request('/auth/logout', { method: 'POST' }),
  snapshot: () => request('/snapshot'),
  kernel: {
    start: () => request('/kernel/start', { method: 'POST' }),
    stop: () => request('/kernel/stop', { method: 'POST' }),
    restart: () => request('/kernel/restart', { method: 'POST' }),
  },
  inbounds: {
    create: (profile) => request('/inbounds', { method: 'POST', body: profile }),
    save: (id, patch) => request(`/inbounds/${encodeURIComponent(id)}`, { method: 'PATCH', body: patch }),
    remove: (id) => request(`/inbounds/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  },
  nodes: {
    get: (id) => request(`/nodes/${encodeURIComponent(id)}`),
    update: (id, node) => request(`/nodes/${encodeURIComponent(id)}`, { method: 'PUT', body: node }),
    importNodes: (nodes) => request('/nodes/import', { method: 'POST', body: { nodes } }),
    importLinks: (links, groupId = '') => request('/nodes/import-links', { method: 'POST', body: { links, group_id: groupId } }),
    delay: (id, url = '') => request(`/nodes/${encodeURIComponent(id)}/delay`, { method: 'POST', body: url ? { url } : {} }),
    test: (nodeIds, kind, url = '') => request('/nodes/test', { method: 'POST', body: { node_ids: nodeIds, kind, url } }),
    remove: (id) => request(`/nodes/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  },
  groups: {
    select: (id, nodeId) => request(`/groups/${encodeURIComponent(id)}/select`, { method: 'POST', body: { node_id: nodeId } }),
    save: (group) => request('/groups', { method: 'POST', body: group }),
    remove: (id) => request(`/groups/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  },
  chains: {
    save: (chain) => request('/chains', { method: 'POST', body: chain }),
    remove: (id) => request(`/chains/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  },
  routes: {
    mode: (mode) => request('/routes/mode', { method: 'PATCH', body: { mode } }),
    defaultEgress: (kind, id = '') => request('/routes/default-egress', { method: 'PATCH', body: { kind, id } }),
    presets: (presets) => request('/routes/presets', { method: 'PATCH', body: presets }),
    replaceRules: (rules) => request('/routes/rules', { method: 'PUT', body: { rules } }),
    saveRule: (rule) => request('/routes/rules', { method: 'POST', body: rule }),
    patchRule: (id, patch) => request(`/routes/rules/${encodeURIComponent(id)}`, { method: 'PATCH', body: patch }),
    removeRule: (id) => request(`/routes/rules/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  },
  subscriptions: {
    create: (subscription) => request('/subscriptions', { method: 'POST', body: subscription }),
    patch: (id, patch) => request(`/subscriptions/${encodeURIComponent(id)}`, { method: 'PATCH', body: patch }),
    remove: (id) => request(`/subscriptions/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    refresh: (id) => request(`/subscriptions/${encodeURIComponent(id)}/refresh`, { method: 'POST' }),
    refreshAll: () => request('/subscriptions/refresh', { method: 'POST' }),
  },
  rulesets: {
    create: (definition) => request('/rulesets', { method: 'POST', body: definition }),
    patch: (name, patch) => request(`/rulesets/${encodeURIComponent(name)}`, { method: 'PATCH', body: patch }),
    refresh: (name) => request(`/rulesets/${encodeURIComponent(name)}/refresh`, { method: 'POST' }),
    refreshAll: () => request('/rulesets/refresh', { method: 'POST' }),
    remove: (name) => request(`/rulesets/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  },
  dns: {
    updateProfile: (profile) => request('/dns', { method: 'PUT', body: profile }),
    patch: (patch) => request('/dns', { method: 'PATCH', body: patch }),
    update: (patch) => request('/dns', { method: 'PATCH', body: patch }),
    flush: () => request('/dns/flush', { method: 'POST' }),
    test: () => request('/dns/test', { method: 'POST' }),
  },
  settings: {
    update: (settings) => request('/settings', { method: 'PUT', body: { settings } }),
  },
  connections: {
    close: (id) => request(`/observability/connections/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    closeAll: () => request('/observability/connections', { method: 'DELETE' }),
  },
  history: {
    rulesets: () => request('/config/history?limit=20'),
  },
};

export function formatLogTime(value) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  const pad = (number) => String(number).padStart(2, '0');
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${String(date.getMilliseconds()).padStart(3, '0')}`;
}

export function formatClock(value) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  const pad = (number) => String(number).padStart(2, '0');
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

export function formatBytes(value) {
  const bytes = Number(value) || 0;
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let size = bytes / 1024;
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index += 1;
  }
  return `${size.toFixed(size >= 100 ? 0 : 1)} ${units[index]}`;
}

export function formatRate(value) {
  return `${formatBytes(value)}/s`;
}

export function formatUptime(seconds) {
  const total = Math.max(0, Number(seconds) || 0);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = Math.floor(total % 60);
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
}

export function formatDateTime(value) {
  if (!value) return '—';
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString('zh-CN', { hour12: false });
}
