import { icon, iconSprite } from './assets/icons.js?v=overview-refresh-9';
import {
  api,
  formatBytes,
  formatClock,
  formatDateTime,
  formatLogTime,
  formatRate,
  formatUptime,
} from './assets/api.js?v=overview-refresh-9';

const VIEW_META = {
  overview: { label: '总览', eyebrow: 'OVERVIEW', icon: 'dashboard' },
  inbounds: { label: '入口', eyebrow: 'INBOUNDS', icon: 'inbound' },
  egress: { label: '出口编排', eyebrow: 'EGRESS ORCHESTRATION', icon: 'nodes' },
  routes: { label: '路由', eyebrow: 'ROUTING', icon: 'route' },
  subscriptions: { label: '订阅', eyebrow: 'SUBSCRIPTIONS', icon: 'rss' },
  rulesets: { label: '规则集', eyebrow: 'RULE SETS', icon: 'rules' },
  observability: { label: '观测', eyebrow: 'OBSERVABILITY', icon: 'activity' },
  logs: { label: '日志', eyebrow: 'STREAMING LOGS', icon: 'terminal' },
  settings: { label: '设置', eyebrow: 'SETTINGS', icon: 'settings' },
};

const app = document.getElementById('app');
const modalRoot = document.getElementById('modal-root');
const toastRegion = document.getElementById('toast-region');

const state = {
  view: VIEW_META[location.hash.slice(1)] ? location.hash.slice(1) : 'overview',
  themeChoice: localStorage.getItem('boxd-theme-v2') || 'light',
  egressTab: 'nodes',
  routeMode: 'rules',
  logLevel: 'all',
  logPaused: false,
  logSource: 'all',
  logQuery: '',
  logAutoScroll: true,
  nodeQuery: '',
  selectedNodeIDs: new Set(),
  nodeTestKind: 'url',
  nodeTestURL: '',
  nodeTesting: false,
  rulesetQuery: '',
  connQuery: '',
  snapshot: null,
  lastSnapshotAt: null,
  loading: true,
  sse: null,
  sseConnected: false,
  traffic: { upload_total: 0, download_total: 0, upload_rate: 0, download_rate: 0, at: null },
  trafficHistory: { down: [], up: [] },
  lastTrafficSampleAt: '',
  scheduler: null,
  auth: { initialized: false, authenticated: false, expiresAt: null, credentialUpdatedAt: null, sessionCount: 0, mode: 'local' },
  nodes: [],
  groups: [],
  chains: [],
  inbounds: [],
  subscriptions: [],
  routeRules: [],
  ruleSets: [],
  connections: [],
  logs: [],
  overview: { stats: [], health: [], quickActions: [] },
  dnsProfile: { strategy: 'prefer_ipv4', cache: true, slots: [] },
  kernel: { version: '', revision: '—', uptime: '—', startedAt: '—', mode: 'rules', state: 'stopped', kind: 'sing-box' },
  flags: {
    autoStart: false,
    authRequired: true,
    accessControl: true,
    dnsCache: true,
    requestLog: true,
    autoRefresh: true,
    subscriptionMaxAttempts: 3,
    subscriptionConflictPolicy: 'keep_local',
    compactTable: false,
  },
};

const escapeHtml = (value) => String(value ?? '')
  .replaceAll('&', '&amp;')
  .replaceAll('<', '&lt;')
  .replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;')
  .replaceAll("'", '&#39;');

async function copyTextToClipboard(text) {
  if (window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall through for browsers that expose the API but deny the permission.
    }
  }

  const activeElement = document.activeElement;
  const selection = document.getSelection();
  const ranges = selection ? Array.from({ length: selection.rangeCount }, (_, index) => selection.getRangeAt(index)) : [];
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.setAttribute('readonly', '');
  textarea.setAttribute('aria-hidden', 'true');
  textarea.style.position = 'fixed';
  textarea.style.top = '0';
  textarea.style.left = '-9999px';
  textarea.style.opacity = '0';
  document.body.append(textarea);
  textarea.select();
  let copied;
  try {
    copied = document.execCommand('copy');
  } finally {
    textarea.remove();
    selection?.removeAllRanges();
    ranges.forEach((range) => selection?.addRange(range));
    activeElement?.focus?.({ preventScroll: true });
  }
  return copied;
}

const byId = (id) => state.nodes.find((item) => item.id === id);
const groupById = (id) => state.groups.find((item) => item.id === id);
const kernelVersionLabel = () => {
  const version = String(state.kernel.version || '').trim();
  if (version) return version;
  return `${String(state.kernel.kind || 'sing-box')} · 版本未读取`;
};
const pad = (value) => String(value).padStart(2, '0');
const shortRevision = (value) => {
  const text = String(value || '—');
  if (text.length <= 22) return text;
  return `${text.slice(0, 12)}…${text.slice(-8)}`;
};

let refreshTimer = null;
let refreshInFlight = null;
let authProbeTimer = null;

const KERNEL_STATE_LABELS = {
  stopped: '已停止',
  starting: '启动中',
  running: '运行中',
  stopping: '停止中',
  failed: '异常',
};

function settingEnabled(settings, key, fallback = false) {
  const value = settings?.[key];
  if (value === undefined) return fallback;
  return ['on', 'true', '1', 'yes', 'enabled'].includes(String(value).toLowerCase());
}

function normalizeLog(item, channel) {
  const timestamp = item?.time ? new Date(item.time).getTime() : Date.now();
  return {
    id: item.id || `${channel}-${timestamp}-${item.message}`,
    channel,
    level: String(item.level || 'info').toLowerCase(),
    source: item.source || (channel === 'kernel' ? 'sing-box' : 'app'),
    message: item.message || '',
    fields: item.fields || null,
    timestamp: Number.isNaN(timestamp) ? Date.now() : timestamp,
    time: formatLogTime(item.time || new Date()),
  };
}

function mergeLogs(items) {
  const merged = new Map();
  for (const item of state.logs) {
    merged.set(`${item.timestamp}|${item.channel}|${item.source}|${item.message}`, item);
  }
  for (const item of items) {
    merged.set(`${item.timestamp}|${item.channel}|${item.source}|${item.message}`, item);
  }
  return [...merged.values()].sort((left, right) => left.timestamp - right.timestamp).slice(-500);
}

function appendTrafficSample(traffic) {
  if (!traffic?.available) {
    state.trafficHistory = { down: [], up: [] };
    state.lastTrafficSampleAt = '';
    return;
  }
  const sampledAt = String(traffic.at || '');
  if (sampledAt && sampledAt === state.lastTrafficSampleAt) return;
  state.lastTrafficSampleAt = sampledAt;
  const down = Math.max(0, Number(traffic?.download_rate) || 0) / 1024 / 1024;
  const up = Math.max(0, Number(traffic?.upload_rate) || 0) / 1024 / 1024;
  state.trafficHistory.down.push(down);
  state.trafficHistory.up.push(up);
  if (state.trafficHistory.down.length > 60) state.trafficHistory.down = state.trafficHistory.down.slice(-60);
  if (state.trafficHistory.up.length > 60) state.trafficHistory.up = state.trafficHistory.up.slice(-60);
}

function applySnapshot(snapshot) {
  const previousTests = new Map(state.nodes.map((node) => [node.id, {
    latency: Number(node.latency) || 0,
    lastTestError: node.lastTestError || '',
    lastTestKind: node.lastTestKind || '',
  }]));
  state.snapshot = snapshot;
  state.lastSnapshotAt = new Date(snapshot.server_time || Date.now());
  state.inbounds = (snapshot.inbounds || []).map((item) => ({ ...item }));
  state.nodes = (snapshot.nodes || []).map((item) => ({
    ...item,
    ...(previousTests.get(item.id) || { latency: 0, lastTestError: '', lastTestKind: '' }),
  }));
  const nodeIDs = new Set(state.nodes.map((node) => node.id));
  for (const id of state.selectedNodeIDs) {
    if (!nodeIDs.has(id)) state.selectedNodeIDs.delete(id);
  }
  state.groups = (snapshot.groups || []).map((item) => ({ ...item, members: [...(item.members || [])] }));
  state.chains = (snapshot.chains || []).map((item) => ({ ...item, hops: [...(item.hops || [])] }));
  state.subscriptions = (snapshot.subscriptions || []).map((item) => ({ ...item }));
  state.routeRules = (snapshot.routes?.rules || []).map((item) => ({ ...item }));
  state.routeMode = snapshot.routes?.mode || 'rules';
  state.ruleSets = (snapshot.rulesets || []).map((item) => ({ ...item }));
  state.connections = (snapshot.connections || []).map((item) => ({ ...item }));
  state.overview = snapshot.overview || state.overview;
  state.dnsProfile = snapshot.dns || state.dnsProfile;
  state.scheduler = snapshot.scheduler || state.scheduler;
  state.traffic = snapshot.traffic || state.traffic;
  state.kernel = {
    ...(snapshot.kernel || {}),
    kind: snapshot.kernel?.kind || snapshot.capabilities?.kind || 'sing-box',
    version: snapshot.kernel?.version || snapshot.capabilities?.version || '',
    revision: snapshot.kernel?.revision || snapshot.revision || '—',
    uptime: formatUptime(snapshot.kernel?.uptime_seconds || 0),
    startedAt: snapshot.kernel?.started_at ? formatDateTime(snapshot.kernel.started_at) : '—',
    mode: snapshot.routes?.mode || 'rules',
    state: snapshot.kernel?.state || 'stopped',
  };
  state.flags = {
    ...state.flags,
    authRequired: Boolean(state.auth.initialized),
    autoRefresh: settingEnabled(snapshot.settings, 'autoRefresh', true),
    subscriptionMaxAttempts: Math.max(1, Number(snapshot.settings?.subscriptionMaxAttempts) || 3),
    subscriptionConflictPolicy: snapshot.settings?.subscriptionConflictPolicy === 'follow_remote' ? 'follow_remote' : 'keep_local',
    accessControl: settingEnabled(snapshot.settings, 'accessControl', true),
    dnsCache: snapshot.dns?.cache ?? true,
    requestLog: settingEnabled(snapshot.settings, 'requestLog', true),
    compactTable: settingEnabled(snapshot.settings, 'compactTable', false),
  };
  document.body.classList.toggle('is-compact', state.flags.compactTable);
  const snapshotLogs = [
    ...(snapshot.logs?.kernel || []).map((item) => normalizeLog(item, 'kernel')),
    ...(snapshot.logs?.app || []).map((item) => normalizeLog(item, 'app')),
  ];
  state.logs = mergeLogs(snapshotLogs);
  appendTrafficSample(state.traffic);
}

async function refreshSnapshot({ rerender = true, quiet = true } = {}) {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    try {
      const snapshot = await api.snapshot();
      applySnapshot(snapshot);
      state.loading = false;
      if (rerender) render(true);
      syncChrome();
      return snapshot;
    } catch (error) {
      if (!quiet && error.status !== 401) showToast('数据刷新失败', error.hint || error.message, 'alert');
      throw error;
    } finally {
      refreshInFlight = null;
    }
  })();
  return refreshInFlight;
}

function scheduleSnapshotRefresh() {
  window.clearTimeout(refreshTimer);
  refreshTimer = window.setTimeout(() => {
    refreshSnapshot().catch((error) => {
      if (error.status !== 401) showToast('数据刷新失败', error.hint || error.message, 'alert');
    });
  }, 180);
}

function syncChrome() {
  const running = state.kernel.state === 'running';
  const stateLabel = KERNEL_STATE_LABELS[state.kernel.state] || state.kernel.state;
  const runtimeChip = document.querySelector('.runtime-chip');
  if (runtimeChip) {
    runtimeChip.dataset.state = state.kernel.state;
    const dot = runtimeChip.querySelector('.status-dot');
    if (dot) {
      dot.classList.remove('is-online', 'is-idle', 'is-warn', 'is-error');
      dot.classList.add(state.kernel.state === 'failed' ? 'is-error' : ['starting', 'stopping'].includes(state.kernel.state) ? 'is-warn' : running ? 'is-online' : 'is-idle');
    }
    const copy = runtimeChip.querySelector('div');
    if (copy) copy.innerHTML = `<strong>内核运行时</strong><span>${escapeHtml(kernelVersionLabel())}${running ? ` · Uptime ${escapeHtml(state.kernel.uptime)}` : ''}</span>`;
    const pill = runtimeChip.querySelector('.runtime-pill');
    if (pill) pill.textContent = stateLabel;
  }
  const statusPair = document.querySelector('.status-pair');
  if (statusPair) {
    statusPair.innerHTML = `<span class="status-pill ${running ? 'is-success' : 'is-muted'}"><span class="status-dot ${running ? 'is-online' : 'is-idle'}"></span>Kernel · ${escapeHtml(stateLabel)}</span><span class="status-pill is-muted"><span data-icon="route"></span>${escapeHtml(String(state.routeMode).toUpperCase())}</span>`;
    hydrateStaticIcons(statusPair);
  }
  syncSidebarStatus();
  const counts = {
    inbounds: state.inbounds.length,
    egress: state.nodes.length,
    routes: state.routeRules.length,
    subscriptions: state.subscriptions.length,
    rulesets: state.ruleSets.length,
  };
  Object.entries(counts).forEach(([view, count]) => {
    const badge = document.querySelector(`[data-view="${view}"] .nav-count`);
    if (badge) badge.textContent = String(count);
  });
  const badge = document.querySelector('.topbar-left .runtime-badge');
  if (badge) {
    badge.textContent = state.sseConnected ? 'LIVE' : 'RECONNECTING';
    badge.classList.toggle('is-live', state.sseConnected);
  }
  document.querySelectorAll('.live-indicator').forEach((indicator) => {
    indicator.classList.toggle('is-offline', !state.sseConnected);
    indicator.title = state.sseConnected ? '实时事件已连接' : '实时事件重连中';
  });
  const alertDot = document.querySelector('.has-badge .badge-dot');
  if (alertDot) alertDot.hidden = !state.logs.some((item) => item.level === 'warn' || item.level === 'error');
}

function syncSidebarStatus() {
  const sidebarStatus = document.querySelector('[data-sidebar-status]');
  if (!sidebarStatus) return;
  sidebarStatus.querySelector('.status-dot')?.classList.toggle('is-online', state.sseConnected);
  sidebarStatus.querySelector('.status-dot')?.classList.toggle('is-idle', !state.sseConnected);
  const label = sidebarStatus.querySelector('[data-sidebar-status-label]');
  const clock = formatClock(new Date());
  if (label) label.textContent = `${state.sseConnected ? 'LIVE' : '重连中'} · ${clock}`;
  sidebarStatus.title = state.sseConnected ? '控制面实时事件已连接，本地时间每秒刷新' : '控制面实时事件重连中';
}

function stopEvents() {
  if (state.sse) state.sse.close();
  state.sse = null;
  state.sseConnected = false;
}

function connectEvents() {
  stopEvents();
  const source = new EventSource('/api/v1/events', { withCredentials: true });
  state.sse = source;
  source.addEventListener('open', () => {
    state.sseConnected = true;
    syncChrome();
    if (state.view === 'overview' || state.view === 'observability' || state.view === 'logs') render(true);
  });
  source.addEventListener('ready', () => {
    state.sseConnected = true;
  });
  source.addEventListener('app.log', (event) => appendStreamLog(JSON.parse(event.data), 'app'));
  source.addEventListener('kernel.log', (event) => appendStreamLog(JSON.parse(event.data), 'kernel'));
  source.addEventListener('kernel.state', (event) => {
    const kernel = JSON.parse(event.data);
    state.kernel = {
      ...state.kernel,
      ...kernel,
      uptime: formatUptime(kernel.uptime_seconds || 0),
      startedAt: kernel.started_at ? formatDateTime(kernel.started_at) : '—',
    };
    syncChrome();
    if (state.view === 'overview' || state.view === 'settings') render(true);
    scheduleSnapshotRefresh();
  });
  source.addEventListener('traffic', (event) => {
    const payload = JSON.parse(event.data);
    state.traffic = payload.traffic || state.traffic;
    appendTrafficSample(state.traffic);
    updateLiveMetrics(payload.connections);
  });
  source.addEventListener('snapshot.invalidated', scheduleSnapshotRefresh);
  source.addEventListener('error', () => {
    state.sseConnected = false;
    syncChrome();
    if (state.view === 'overview' || state.view === 'observability' || state.view === 'logs') render(true);
    window.clearTimeout(authProbeTimer);
    authProbeTimer = window.setTimeout(async () => {
      try {
        const status = await api.authStatus();
        applyAuthStatus(status);
        if (!status.authenticated) showAuthScreen(status.initialized ? 'login' : 'initialize');
      } catch {
        // EventSource keeps retrying while the control plane is temporarily unavailable.
      }
    }, 1200);
  });
}

function appendStreamLog(item, channel) {
  const entry = normalizeLog(item, channel);
  const key = `${entry.timestamp}|${entry.channel}|${entry.source}|${entry.message}`;
  if (state.logs.some((existing) => `${existing.timestamp}|${existing.channel}|${existing.source}|${existing.message}` === key)) return;
  state.logs.push(entry);
  if (state.logs.length > 500) state.logs = state.logs.slice(-500);
  if (state.view === 'logs' && !state.logPaused) updateLogConsole();
}

function applyAuthStatus(status = {}) {
  state.auth = {
    initialized: Boolean(status.initialized),
    authenticated: Boolean(status.authenticated),
    expiresAt: status.expires_at || null,
    credentialUpdatedAt: status.credential_updated_at || null,
    sessionCount: Number(status.session_count) || 0,
    mode: status.mode === 'password' ? 'password' : 'local',
  };
  state.flags.authRequired = state.auth.initialized;
}

function updateLiveMetrics(connectionCount) {
  const down = document.getElementById('live-down-rate');
  const up = document.getElementById('live-up-rate');
  const downHint = document.getElementById('live-down-hint');
  const upHint = document.getElementById('live-up-hint');
  const connections = document.getElementById('live-connection-count');
  const total = document.getElementById('live-total-traffic');
  const connectionDelta = document.getElementById('live-connection-delta');
  const trafficDelta = document.getElementById('live-traffic-delta');
  const connectionDetail = document.getElementById('live-connection-detail');
  const trafficDetail = document.getElementById('live-traffic-detail');
  const available = Boolean(state.traffic.available);
  if (down) down.textContent = available ? formatRate(state.traffic.download_rate) : '—';
  if (up) up.textContent = available ? formatRate(state.traffic.upload_rate) : '—';
  if (downHint) downHint.hidden = available;
  if (upHint) upHint.hidden = available;
  if (connections && connectionCount !== undefined) connections.textContent = available ? String(connectionCount) : '—';
  if (total) total.textContent = available ? formatBytes((state.traffic.upload_total || 0) + (state.traffic.download_total || 0)) : '—';
  if (connectionDelta) {
    connectionDelta.textContent = available ? '实时采样' : '内核未运行';
    connectionDelta.classList.toggle('is-muted', !available);
  }
  if (trafficDelta) {
    trafficDelta.textContent = available ? '实时采样' : '内核未启动，未采集数据';
    trafficDelta.classList.toggle('is-muted', !available);
    trafficDelta.classList.toggle('is-hint', !available);
  }
  if (connectionDetail) connectionDetail.textContent = available ? '来自内核连接快照' : '启动内核后统计活动连接';
  if (trafficDetail) trafficDetail.textContent = available ? `本次运行上行 ${formatBytes(state.traffic.upload_total)}` : '启动内核后开始采样';
  const sampling = document.getElementById('overview-sampling-status');
  if (sampling) {
    sampling.innerHTML = `<span class="status-dot ${available ? 'is-online' : 'is-idle'}"></span>${available ? '实时采样' : '等待启动'}`;
  }
  const trafficState = document.getElementById('overview-traffic-state');
  if (trafficState) {
    trafficState.className = `tag ${available ? 'is-mint' : 'is-solid'}`;
    trafficState.textContent = available ? '采样中' : '等待采样';
  }
  if (state.view === 'overview') {
    const chart = document.getElementById('traffic-chart');
    if (chart) chart.outerHTML = trafficChart();
  }
  syncChrome();
}

function showAuthScreen(mode = 'login', message = '') {
  stopEvents();
  if (modalRoot && !modalRoot.hidden) closeModal();
  state.auth = { ...state.auth, authenticated: false, expiresAt: null };
  const shell = document.querySelector('.app-shell');
  if (shell) shell.hidden = true;
  let screen = document.getElementById('auth-screen');
  if (!screen) {
    screen = document.createElement('div');
    screen.id = 'auth-screen';
    screen.className = 'auth-screen';
    document.body.appendChild(screen);
  }
  screen.hidden = false;
  const initializing = mode === 'initialize';
  const brand = `
    <div class="brand-mark" aria-hidden="true"><span></span><span></span><span></span></div>
    <div class="auth-brand-copy"><strong>BOXD</strong><span>CONTROL PLANE</span></div>
  `;
  screen.innerHTML = `
    <div class="auth-ambient auth-ambient-a"></div>
    <div class="auth-ambient auth-ambient-b"></div>
    <div class="auth-panel">
      <aside class="auth-story" aria-label="控制面能力">
        <div class="auth-story-brand">${brand}</div>
        <div class="auth-story-copy">
          <span class="auth-story-kicker">NETWORK CONTROL PLANE</span>
          <h2>一个控制面，<br>管理完整网络链路。</h2>
          <p>统一编排入口、订阅节点与路由策略，让每一次配置变更清晰、可控、可追溯。</p>
        </div>
        <div class="auth-story-capabilities">
          <div class="auth-capability"><span class="auth-capability-icon">${icon('layers')}</span><span><strong>统一编排</strong><small>入口、订阅与出口集中管理</small></span></div>
          <div class="auth-capability"><span class="auth-capability-icon">${icon('route')}</span><span><strong>策略路由</strong><small>规则驱动的流量分发</small></span></div>
          <div class="auth-capability"><span class="auth-capability-icon">${icon('activity')}</span><span><strong>实时观测</strong><small>运行指标与流式日志</small></span></div>
        </div>
        <div class="auth-story-foot"><span>BOXD CONTROL PLANE</span><span>v2</span></div>
      </aside>
      <main class="auth-form-panel">
        <div class="auth-form-top">
          <div class="auth-mobile-brand">${brand}</div>
          <span class="auth-access-chip">${icon('shield')}<span>管理员访问</span></span>
          <button class="auth-theme-toggle" id="auth-theme-toggle" type="button" data-action="toggle-theme" aria-label="切换到深色模式" aria-pressed="false" title="切换到深色模式">${icon('moon')}</button>
        </div>
        <div class="auth-heading">
          <span class="auth-kicker">${initializing ? '首次设置' : '安全登录'}</span>
          <h1>${initializing ? '创建管理员口令' : '欢迎回来'}</h1>
          <p>${initializing ? '为当前控制面创建管理员口令。口令仅在本机校验，并以 PBKDF2 摘要形式保存。' : '输入管理员口令，继续管理当前控制面。登录会话将在 24 小时后自动失效。'}</p>
        </div>
        <form class="auth-form" id="auth-form">
          <label class="auth-field" for="auth-password">
            <span class="auth-field-label"><span>管理员口令</span><small>${initializing ? '至少 8 个字符' : '本地认证'}</small></span>
            <span class="auth-input-shell">
              <input class="input" id="auth-password" type="password" minlength="8" autocomplete="${initializing ? 'new-password' : 'current-password'}" required autofocus>
              <button class="auth-reveal" type="button" data-auth-reveal="auth-password" aria-label="显示口令" aria-pressed="false" title="显示口令">${icon('eye')}</button>
            </span>
          </label>
          ${initializing ? `<label class="auth-field" for="auth-confirm">
            <span class="auth-field-label"><span>确认口令</span><small>再次输入</small></span>
            <span class="auth-input-shell">
              <input class="input" id="auth-confirm" type="password" minlength="8" autocomplete="new-password" required>
              <button class="auth-reveal" type="button" data-auth-reveal="auth-confirm" aria-label="显示口令" aria-pressed="false" title="显示口令">${icon('eye')}</button>
            </span>
          </label>` : ''}
          <div class="auth-error" id="auth-error" role="alert" ${message ? '' : 'hidden'}>${message ? `${icon('alert')}<span>${escapeHtml(message)}</span>` : ''}</div>
          <button class="btn btn-primary auth-submit" id="auth-submit" type="submit"><span data-auth-submit-label>${initializing ? '初始化并进入' : '登录控制面'}</span>${icon('chevron-right')}</button>
        </form>
        <div class="auth-security">
          <span class="auth-security-icon">${icon('shield')}</span>
          <span class="auth-security-copy"><strong>${initializing ? '凭据安全存储' : '受保护的管理员会话'}</strong><span>${initializing ? '口令以 PBKDF2 摘要保存在本机，不会上传至外部服务。' : '仅限授权管理员访问，认证会话将在 24 小时后自动失效。'}</span></span>
        </div>
      </main>
    </div>
  `;
  screen.querySelector('#auth-form').addEventListener('submit', submitAuth);
  screen.querySelectorAll('[data-auth-reveal]').forEach((button) => {
    button.addEventListener('click', () => {
      const input = document.getElementById(button.dataset.authReveal);
      if (!input) return;
      const reveal = input.type === 'password';
      input.type = reveal ? 'text' : 'password';
      button.innerHTML = icon(reveal ? 'eye-off' : 'eye');
      button.setAttribute('aria-pressed', String(reveal));
      const label = reveal ? '隐藏口令' : '显示口令';
      button.setAttribute('aria-label', label);
      button.title = label;
      input.focus({ preventScroll: true });
    });
  });
  applyTheme(state.themeChoice, true);
  window.setTimeout(() => screen.querySelector('#auth-password')?.focus(), 30);
}

async function submitAuth(event) {
  event.preventDefault();
  const password = document.getElementById('auth-password')?.value || '';
  const confirm = document.getElementById('auth-confirm')?.value;
  const errorBox = document.getElementById('auth-error');
  const submit = document.getElementById('auth-submit');
  const submitLabel = submit?.querySelector('[data-auth-submit-label]');
  const initializing = document.getElementById('auth-confirm') !== null;
  const setSubmitLabel = (text) => {
    if (submitLabel) submitLabel.textContent = text;
    else if (submit) submit.textContent = text;
  };
  const authErrorMessage = (error) => {
    if (error?.status === 401) return initializing ? '当前口令不符合要求，请重新设置。' : '口令不正确，请重新输入。';
    if (error?.status === 409) return '控制面已完成初始化，请刷新页面后直接登录。';
    if (error?.status === 429) return '认证尝试次数过多，请稍后再试。';
    if (error?.status >= 500) return '控制面暂时无法完成认证，请稍后重试。';
    if (error?.code === 'network_error') return error.hint || '无法连接控制面，请检查服务状态。';
    return error?.hint || error?.message || '认证失败，请稍后重试。';
  };
  const showError = (text) => {
    if (!errorBox) return;
    errorBox.hidden = false;
    errorBox.innerHTML = `${icon('alert')}<span>${escapeHtml(text)}</span>`;
  };
  if (password.length < 8) {
    showError('口令至少需要 8 个字符');
    return;
  }
  if (initializing && password !== confirm) {
    showError('两次输入的口令不一致');
    return;
  }
  if (submit) {
    submit.disabled = true;
    setSubmitLabel(initializing ? '正在初始化…' : '正在登录…');
  }
  try {
    const session = initializing ? await api.initialize(password) : await api.login(password);
    state.auth = {
      ...state.auth,
      initialized: true,
      authenticated: Boolean(session?.authenticated),
      expiresAt: session?.expires_at || null,
      mode: 'password',
    };
    if (session?.authenticated) api.authStatus().then(applyAuthStatus).catch(() => {});
    document.getElementById('auth-screen')?.remove();
    const shell = document.querySelector('.app-shell');
    if (shell) shell.hidden = false;
    await enterApp();
  } catch (error) {
    showError(authErrorMessage(error));
  } finally {
    if (submit) {
      submit.disabled = false;
      setSubmitLabel(initializing ? '初始化并进入' : '登录控制面');
    }
  }
}

async function enterApp() {
  connectEvents();
  try {
    await refreshSnapshot({ rerender: false });
    state.loading = false;
    const shell = document.querySelector('.app-shell');
    if (shell) shell.hidden = false;
    render(false);
    syncChrome();
  } catch (error) {
    if (error.status === 401) {
      showAuthScreen('login', '会话已失效，请重新登录');
      return;
    }
    state.loading = false;
    app.innerHTML = `<section class="page"><div class="empty-state"><span class="empty-icon">${icon('alert')}</span><strong>无法读取控制面状态</strong><span>${escapeHtml(error.hint || error.message)}</span><button class="btn btn-primary" data-action="refresh-overview">重试</button></div></section>`;
  }
}

async function boot() {
  document.getElementById('icon-sprite').innerHTML = iconSprite();
  hydrateStaticIcons();
  applyTheme(state.themeChoice, true);
  try {
    const status = await api.authStatus();
    applyAuthStatus(status);
    if (!status.initialized) {
      showAuthScreen('initialize');
      return;
    }
    if (!status.authenticated) {
      showAuthScreen('login');
      return;
    }
    await enterApp();
  } catch (error) {
    showAuthScreen('login', error.hint || error.message || '无法连接控制面');
  }
}

function hydrateStaticIcons(root = document) {
  root.querySelectorAll('[data-icon]').forEach((element) => {
    if (!element.querySelector('svg')) element.innerHTML = icon(element.dataset.icon);
  });
}

function applyTheme(choice, silent = false) {
  state.themeChoice = choice;
  const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const actual = choice === 'system' ? (systemDark ? 'dark' : 'light') : choice;
  document.documentElement.dataset.theme = actual;
  localStorage.setItem('boxd-theme-v2', choice);
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) meta.content = actual === 'dark' ? '#071014' : '#eef5f2';
  const target = actual === 'dark' ? 'light' : 'dark';
  const themeIcon = document.getElementById('theme-icon');
  if (themeIcon) themeIcon.innerHTML = icon(target === 'dark' ? 'moon' : 'sun');
  const themeLabel = document.getElementById('theme-label');
  if (themeLabel) themeLabel.textContent = target === 'dark' ? '深色模式' : '浅色模式';
  const themeToggle = document.getElementById('theme-toggle');
  if (themeToggle) {
    const label = `切换到${target === 'dark' ? '深色' : '浅色'}模式`;
    themeToggle.title = label;
    themeToggle.setAttribute('aria-label', label);
    themeToggle.setAttribute('aria-pressed', actual === 'dark' ? 'true' : 'false');
  }
  const authThemeToggle = document.getElementById('auth-theme-toggle');
  if (authThemeToggle) {
    const label = `切换到${target === 'dark' ? '深色' : '浅色'}模式`;
    authThemeToggle.innerHTML = icon(target === 'dark' ? 'moon' : 'sun');
    authThemeToggle.title = label;
    authThemeToggle.setAttribute('aria-label', label);
    authThemeToggle.setAttribute('aria-pressed', actual === 'dark' ? 'true' : 'false');
  }
  if (!silent && !document.getElementById('auth-screen')) {
    showToast(actual === 'dark' ? '已切换深色主题' : '已切换浅色主题', '外观设置已保存到本地');
  }
}

function toggleTheme() {
  const actual = document.documentElement.dataset.theme;
  applyTheme(actual === 'dark' ? 'light' : 'dark');
  if (state.view === 'settings') render(true);
}

function showToast(title, message = '', iconName = 'check') {
  const toast = document.createElement('div');
  toast.className = 'toast';
  toast.innerHTML = `
    <span class="toast-icon">${icon(iconName)}</span>
    <span class="toast-copy"><strong>${escapeHtml(title)}</strong>${message ? `<span>${escapeHtml(message)}</span>` : ''}</span>
  `;
  toastRegion.appendChild(toast);
  window.setTimeout(() => {
    toast.classList.add('is-leaving');
    window.setTimeout(() => toast.remove(), 230);
  }, 2800);
}

function setView(view, options = {}) {
  const nextView = VIEW_META[view] ? view : 'overview';
  if (options.tab) {
    if (nextView === 'egress') state.egressTab = options.tab;
  }
  state.view = nextView;
  if (location.hash !== `#${nextView}`) history.replaceState(null, '', `#${nextView}`);
  document.body.classList.remove('sidebar-open');
  document.getElementById('sidebar-backdrop').hidden = true;
  render(false);
}

function render(preserveScroll = false) {
  const scrollTop = app.scrollTop;
  const meta = VIEW_META[state.view];
  document.getElementById('current-view-label').textContent = meta.label;
  document.title = `Boxd Control · ${meta.label}`;
  document.querySelectorAll('.nav-item').forEach((item) => {
    const active = item.dataset.view === state.view;
    item.classList.toggle('is-active', active);
    if (active) item.setAttribute('aria-current', 'page');
    else item.removeAttribute('aria-current');
  });
  const egressCount = document.querySelector('[data-view="egress"] .nav-count');
  if (egressCount) egressCount.textContent = String(state.nodes.length);

  const renderer = {
    overview: overviewView,
    inbounds: inboundsView,
    egress: egressView,
    routes: routesView,
    subscriptions: subscriptionsView,
    rulesets: rulesetsView,
    observability: observabilityView,
    logs: logsView,
    settings: settingsView,
  }[state.view];

  // Entry motion belongs to navigation only. Live snapshots, SSE state changes
  // and mutations replace the view contents in place without replaying it.
  app.classList.toggle('is-refreshing', preserveScroll);
  app.innerHTML = renderer();
  app.scrollTop = preserveScroll ? scrollTop : 0;
  if (state.view === 'logs') {
    requestAnimationFrame(() => {
      updateLogRuntimeMetrics();
      if (state.logAutoScroll) scrollLogConsole();
    });
  }
}

function pageHead({ eyebrow, title, description, actions = '' }) {
  return `
    <div class="page-head">
      <div class="page-head-copy">
        <div class="eyebrow">${escapeHtml(eyebrow)}</div>
        <h1>${escapeHtml(title)}</h1>
        <p>${escapeHtml(description)}</p>
      </div>
      ${actions ? `<div class="page-actions">${actions}</div>` : ''}
    </div>
  `;
}

function smoothPath(points) {
  if (!points.length) return '';
  let path = `M ${points[0].x.toFixed(2)} ${points[0].y.toFixed(2)}`;
  for (let index = 1; index < points.length; index += 1) {
    const previous = points[index - 1];
    const current = points[index];
    const midX = (previous.x + current.x) / 2;
    path += ` C ${midX.toFixed(2)} ${previous.y.toFixed(2)}, ${midX.toFixed(2)} ${current.y.toFixed(2)}, ${current.x.toFixed(2)} ${current.y.toFixed(2)}`;
  }
  return path;
}

function trafficChart() {
  const down = state.trafficHistory.down;
  const upload = state.trafficHistory.up;
  if (!state.traffic?.available || down.length < 2) {
    const sampling = Boolean(state.traffic?.available && down.length === 1);
    return `
      <div class="chart-empty" id="traffic-chart">
        <span class="empty-icon">${icon('activity')}</span>
        <strong>${sampling ? '已收到首个采样点' : '尚未收到实时流量采样'}</strong>
        <span>${sampling ? '等待下一次采样后绘制趋势' : '启动内核后会每 2 秒读取真实吞吐'}</span>
      </div>
    `;
  }
  const width = 720;
  const height = 181;
  const top = 8;
  const bottom = 174;
  const max = Math.max(0.2, ...down, ...upload) * 1.12;
  const toPoints = (values) => values.map((value, index) => ({
    x: (index / (values.length - 1)) * width,
    y: bottom - (value / max) * (bottom - top),
  }));
  const downPoints = toPoints(down);
  const upPoints = toPoints(upload);
  const downLine = smoothPath(downPoints);
  const upLine = smoothPath(upPoints);
  const area = `${downLine} L ${width} ${height} L 0 ${height} Z`;
  const grid = [0, 1, 2, 3, 4].map((index) => {
    const y = top + ((bottom - top) / 4) * index;
    return `<line class="chart-gridline" x1="0" y1="${y}" x2="${width}" y2="${y}"/>`;
  }).join('');
  const labels = ['实时采样', '', '', '', '现在'];
  const last = downPoints[downPoints.length - 1];
  return `
    <div class="chart-wrap" id="traffic-chart">
      <svg class="chart-svg" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none" role="img" aria-label="流量趋势图">
        <defs>
          <linearGradient id="traffic-area" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="var(--primary)" stop-opacity=".22"/>
            <stop offset="100%" stop-color="var(--primary)" stop-opacity="0"/>
          </linearGradient>
        </defs>
        ${grid}
        <path class="chart-area" d="${area}"/>
        <path class="chart-line-2" d="${upLine}"/>
        <path class="chart-line" d="${downLine}"/>
        <circle class="chart-point" cx="${last.x}" cy="${last.y}" r="4"/>
      </svg>
      <div class="chart-foot">${labels.map((label) => `<span>${label}</span>`).join('')}</div>
    </div>
  `;
}

function statusText(status) {
  return { online: '在线', degraded: '已逾期', paused: '已暂停', pending: '待首次同步', manual: '仅手动同步', ok: '已同步', ready: '就绪', idle: '待刷新', unknown: '已配置', error: '异常' }[status] || status;
}

const SUBSCRIPTION_STATE_META = {
  ok: { label: '已同步', tone: 'is-mint', dot: 'is-online' },
  degraded: { label: '已逾期', tone: 'is-amber', dot: 'is-warn' },
  paused: { label: '已暂停', tone: 'is-solid', dot: 'is-idle' },
  pending: { label: '待首次同步', tone: 'is-sky', dot: 'is-warn' },
  manual: { label: '仅手动同步', tone: 'is-solid', dot: 'is-idle' },
};

function subscriptionStateMeta(state) {
  return SUBSCRIPTION_STATE_META[state] || { label: statusText(state), tone: 'is-solid', dot: 'is-idle' };
}

function latencyClass(latency) {
  if (latency >= 180) return 'bad';
  if (latency >= 100) return 'warn';
  return '';
}

function latencyBars(latency) {
  const active = latency < 70 ? 4 : latency < 120 ? 3 : latency < 200 ? 2 : 1;
  const tone = latencyClass(latency);
  return `<span class="latency-bars ${tone}">${[1, 2, 3, 4].map((index) => `<i class="${index <= active ? 'on' : ''}"></i>`).join('')}</span>`;
}

function actionTag(action) {
  const config = {
    chain: { label: '链路', icon: 'link', tone: 'is-violet' },
    group: { label: '分组', icon: 'nodes', tone: 'is-sky' },
    direct: { label: '直连', icon: 'arrow-up-right', tone: 'is-mint' },
    reject: { label: '拒绝', icon: 'shield', tone: 'is-rose' },
  }[action] || { label: action, icon: 'route', tone: 'is-solid' };
  return `<span class="action-chip"><span class="tag ${config.tone}">${icon(config.icon)}${config.label}</span></span>`;
}

function overviewView() {
  const running = state.kernel.state === 'running';
  const trafficAvailable = Boolean(state.traffic.available);
  const unavailableTrafficHint = '内核未启动，未采集数据';
  const stateLabel = KERNEL_STATE_LABELS[state.kernel.state] || state.kernel.state;
  const kernelDot = state.kernel.state === 'failed' ? 'is-error' : ['starting', 'stopping'].includes(state.kernel.state) ? 'is-warn' : running ? 'is-online' : 'is-idle';
  const kernelTagTone = running ? 'is-mint' : state.kernel.state === 'failed' ? 'is-rose' : ['starting', 'stopping'].includes(state.kernel.state) ? 'is-amber' : 'is-solid';
  const snapshotClock = state.lastSnapshotAt ? formatClock(state.lastSnapshotAt) : '—';
  const defaultEgress = state.snapshot?.routes?.default_target || '直连';
  const healthySubscriptions = state.overview.health.filter((item) => item.state === 'ok').length;
  const statIcons = ['activity', 'download', 'shield', 'zap'];
  const statLiveIDs = ['live-connection-count', 'live-total-traffic', '', ''];
  const statDeltaIDs = ['live-connection-delta', 'live-traffic-delta', '', ''];
  const statDetailIDs = ['live-connection-detail', 'live-traffic-detail', '', ''];
  const stats = state.overview.stats.map((stat, index) => {
    const muted = stat.delta.includes('未运行') || stat.delta.includes('未采集');
    const trafficUnavailable = stat.value === '未采集';
    const value = trafficUnavailable ? '—' : stat.value;
    const delta = trafficUnavailable ? unavailableTrafficHint : stat.delta;
    return `
    <article class="stat-card tone-${stat.tone}">
      <div class="stat-top">
        <span class="stat-label">${escapeHtml(stat.label)}</span>
        <span class="stat-icon">${icon(statIcons[index])}</span>
      </div>
      <div class="stat-value-row">
        <strong class="stat-value"${statLiveIDs[index] ? ` id="${statLiveIDs[index]}"` : ''}>${escapeHtml(value)}</strong>
        <span class="stat-delta ${muted ? 'is-muted' : stat.tone === 'amber' ? 'is-warm' : ''}${trafficUnavailable ? ' is-hint' : ''}"${statDeltaIDs[index] ? ` id="${statDeltaIDs[index]}"` : ''}>${escapeHtml(delta)}</span>
      </div>
      <div class="stat-detail"><span${statDetailIDs[index] ? ` id="${statDetailIDs[index]}"` : ''}>${escapeHtml(stat.detail)}</span></div>
    </article>
  `;
  }).join('');

  const health = state.overview.health.map((item) => `
    <div class="health-item">
      <span class="health-logo">${escapeHtml(item.name.slice(0, 2).toUpperCase())}</span>
      <span class="health-copy"><strong>${escapeHtml(item.name)}</strong><span>${escapeHtml(item.meta)}</span></span>
      <span class="latency-value ${item.state === 'degraded' ? 'warn' : ''}">${escapeHtml(subscriptionStateMeta(item.state).label)}</span>
      <span class="health-state ${escapeHtml(item.state)}"></span>
    </div>
  `).join('');

  const quickActions = state.overview.quickActions.map((item) => `
    <button class="quick-action" data-tone="${item.tone}" data-view="${item.view}">
      ${icon(item.icon)}<span>${escapeHtml(item.label)}</span>
    </button>
  `).join('');

  const recentRows = state.connections.slice(0, 5).map((item) => `
    <tr>
      <td><div class="conn-target" title="${escapeHtml(item.target)} · ${escapeHtml(item.process || '—')}"><strong>${escapeHtml(item.target)}</strong><span>${escapeHtml(item.process || '—')}</span></div></td>
      <td title="${escapeHtml(item.outbound)}">${item.outbound === 'direct' || item.outbound === 'reject' ? `<span class="tag is-solid">${escapeHtml(item.outbound)}</span>` : escapeHtml(item.outbound)}</td>
      <td class="mono nowrap">${escapeHtml(item.upload)} / ${escapeHtml(item.download)}</td>
      <td class="mono nowrap muted">${escapeHtml(item.duration)}</td>
    </tr>
  `).join('');

  return `
    <section class="page overview-page">
      ${pageHead({
        eyebrow: 'CONTROL PLANE OVERVIEW',
        title: '控制平面总览',
        description: '从实时事件流到内核运行时，集中掌握控制面的运行状态、流量趋势与配置概况。',
        actions: `
          <button class="btn" data-action="refresh-overview">${icon('refresh')}刷新状态</button>
          <button class="btn btn-primary" data-view="logs">${icon('terminal')}运行日志</button>
        `,
      })}

      <section class="overview-status-strip ${state.sseConnected ? 'is-online' : 'is-warning'}" role="status" aria-label="控制面运行状态">
        <div class="overview-health-main">
          <span class="overview-health-signal" aria-hidden="true"><i></i>${icon('radio')}</span>
          <span class="overview-health-copy">
            <span class="overview-health-eyebrow">CONTROL PLANE HEALTH</span>
            <strong>${state.sseConnected ? '控制面在线' : '实时通道重连中'}</strong>
            <span>${state.sseConnected ? `事件流已连接，最近快照更新于 ${snapshotClock}` : '正在恢复事件流连接，页面保留最近一次快照'}</span>
          </span>
        </div>
        <div class="overview-health-facts">
          <div class="overview-health-fact"><span>数据采样</span><strong class="dot-word" id="overview-sampling-status"><span class="status-dot ${trafficAvailable ? 'is-online' : 'is-idle'}"></span>${trafficAvailable ? '实时采样' : '等待启动'}</strong></div>
          <div class="overview-health-fact"><span>内核状态</span><strong class="dot-word"><span class="status-dot ${kernelDot}"></span>${escapeHtml(stateLabel)}</strong></div>
          <div class="overview-health-fact"><span>默认出口</span><strong title="${escapeHtml(defaultEgress)}">${escapeHtml(defaultEgress)}</strong></div>
          <div class="overview-health-fact"><span>快照时间</span><strong class="mono">${escapeHtml(snapshotClock)}</strong></div>
        </div>
      </section>

      <article class="card overview-runtime-priority" aria-labelledby="overview-runtime-title">
        <div class="overview-runtime-identity">
          <div class="overview-runtime-visual"><span></span><span></span><span class="kernel-core"></span></div>
          <div class="overview-runtime-copy">
            <div class="overview-runtime-heading">
              <h2 id="overview-runtime-title">内核运行时</h2>
              <span class="tag ${kernelTagTone}"><span class="status-dot ${kernelDot}"></span>${escapeHtml(stateLabel)}</span>
            </div>
            <p>${running ? '配置已同步，正在承载当前代理连接。' : '配置修改会持久化，启动内核后才会建立代理连接。'}</p>
            <span class="overview-runtime-state"><span class="status-dot ${kernelDot}"></span>${running ? `UPTIME · ${escapeHtml(state.kernel.uptime)}` : 'OFFLINE'}</span>
          </div>
        </div>
        <div class="overview-runtime-facts">
          <div><span>运行时</span><strong>${escapeHtml(kernelVersionLabel())}</strong></div>
          <div><span>配置版本</span><code>${escapeHtml(shortRevision(state.kernel.revision || '—'))}</code></div>
          <div><span>启动时间</span><strong>${escapeHtml(state.kernel.startedAt || '—')}</strong></div>
          <div><span>路由模式</span><strong>${escapeHtml(String(state.routeMode).toUpperCase())}</strong></div>
        </div>
        <div class="overview-runtime-actions">
          ${running
            ? `<button class="btn btn-sm" data-action="stop-kernel">${icon('stop')}停止内核</button><button class="btn btn-sm" data-action="restart-kernel">${icon('power')}重启内核</button>`
            : `<button class="btn btn-sm btn-primary" data-action="start-kernel">${icon('play')}启动内核</button>`}
        </div>
      </article>

      <div class="stats-grid">${stats}</div>

      <div class="overview-main-grid">
        <article class="card overview-traffic-card">
          <div class="overview-card-head">
            <div class="overview-card-heading">
              <span class="overview-head-icon tone-sky">${icon('activity')}</span>
              <div><h2>实时流量</h2><p>来自内核运行时的 2 秒采样</p></div>
            </div>
            <span class="tag ${trafficAvailable ? 'is-mint' : 'is-solid'}" id="overview-traffic-state">${trafficAvailable ? '采样中' : '等待采样'}</span>
          </div>
          <div class="overview-traffic-rates">
            <div class="overview-rate"><span>下行速率</span><div class="overview-rate-line"><strong id="live-down-rate">${trafficAvailable ? formatRate(state.traffic.download_rate) : '—'}</strong><small class="overview-rate-hint" id="live-down-hint"${trafficAvailable ? ' hidden' : ''}>${unavailableTrafficHint}</small></div></div>
            <div class="overview-rate"><span>上行速率</span><div class="overview-rate-line"><strong id="live-up-rate">${trafficAvailable ? formatRate(state.traffic.upload_rate) : '—'}</strong><small class="overview-rate-hint" id="live-up-hint"${trafficAvailable ? ' hidden' : ''}>${unavailableTrafficHint}</small></div></div>
            <div class="overview-traffic-legend"><span class="legend-pill"><i class="legend-dot"></i>下行</span><span class="legend-pill"><i class="legend-dot sky"></i>上行</span></div>
          </div>
          ${trafficChart()}
        </article>

        <article class="card overview-connections-card">
          <div class="overview-card-head">
            <div class="overview-card-heading">
              <span class="overview-head-icon tone-mint">${icon('link')}</span>
              <div><h2>活跃连接</h2><p>来自内核的当前连接快照</p></div>
            </div>
            <div class="overview-card-actions"><span class="tag is-solid">${state.connections.length} 条</span><button class="link-button" data-view="observability">查看全部 ${icon('chevron-right')}</button></div>
          </div>
          ${recentRows ? `
            <div class="table-wrap" role="region" aria-label="近期连接" tabindex="0">
              <table class="data-table overview-connections-table">
                <thead><tr><th>目标</th><th>出口</th><th>上下行</th><th>时长</th></tr></thead>
                <tbody>${recentRows}</tbody>
              </table>
            </div>` : `
            <div class="overview-empty">
              <span class="overview-empty-icon">${icon('link')}</span>
              <span><strong>当前没有活动连接</strong><span>${running ? '连接产生后会实时显示在这里。' : '启动内核并产生流量后，可在这里查看连接详情。'}</span></span>
            </div>`}
        </article>
      </div>

      <div class="overview-bottom-grid">
        <article class="card overview-subscriptions-card">
          <div class="overview-card-head">
            <div class="overview-card-heading">
              <span class="overview-head-icon tone-amber">${icon('rss')}</span>
              <div><h2>订阅同步</h2><p>节点来源与最近同步结果</p></div>
            </div>
            <button class="link-button" data-view="subscriptions">管理 ${icon('chevron-right')}</button>
          </div>
          <div class="overview-sub-summary">
            <div><strong>${state.overview.health.length}</strong><span>订阅源</span></div>
            <div><strong>${healthySubscriptions}</strong><span>同步正常</span></div>
          </div>
          <div class="health-list">${health || `<div class="empty-state"><strong>还没有订阅</strong><span>添加订阅后会显示同步状态</span></div>`}</div>
        </article>
        <article class="card overview-actions-card">
          <div class="overview-card-head">
            <div class="overview-card-heading">
              <span class="overview-head-icon tone-violet">${icon('sparkles')}</span>
              <div><h2>快捷操作</h2><p>常用控制面入口</p></div>
            </div>
          </div>
          <div class="quick-grid">${quickActions}</div>
        </article>
      </div>
    </section>
  `;
}

function inboundCard(item) {
  const disabled = !item.enabled;
  const live = disabled ? false : state.kernel.state === 'running';
  const stateLabel = disabled ? '已停用' : live ? '运行中' : '已配置 · 内核未启动';
  return `
    <article class="card inbound-card ${disabled ? 'is-disabled' : ''}">
      <div class="inbound-top">
        <span class="inbound-icon">${icon(item.kind === 'tun' ? 'layers' : 'inbound')}</span>
        <div class="inbound-title">
          <h3>${escapeHtml(item.name)}</h3>
          <p>${item.kind === 'tun' ? `透明代理入口 · ${escapeHtml(item.stack || '未配置')} 栈` : 'HTTP / SOCKS5 混合监听'}</p>
        </div>
        <span class="tag ${disabled ? 'is-solid' : live ? 'is-mint' : 'is-amber'}"><span class="status-dot ${disabled ? 'is-idle' : live ? 'is-online' : 'is-warn'}"></span>${stateLabel}</span>
        <button class="switch ${item.enabled ? 'is-on' : ''}" data-action="toggle-inbound" data-id="${item.id}" role="switch" aria-checked="${item.enabled}" aria-label="切换 ${escapeHtml(item.name)}"></button>
      </div>
      <div class="inbound-list">
        <div class="inbound-row"><span>监听地址</span><code title="${escapeHtml(item.listen)}">${escapeHtml(item.listen)}</code></div>
        <div class="inbound-row"><span>UDP 转发</span><strong>${item.udp ? '已启用' : '未启用'}</strong></div>
        <div class="inbound-row"><span>${item.kind === 'tun' ? 'MTU' : '认证'}</span><strong>${item.kind === 'tun' ? `${item.mtu || '—'} · ${escapeHtml(item.stack || '—')}` : item.auth ? '需要认证' : '无认证'}</strong></div>
        <div class="inbound-row"><span>当前连接</span><strong>${state.kernel.state === 'running' ? `${item.connections} 条` : '内核未运行'}</strong></div>
      </div>
      <div class="inbound-footer">
        <div class="inbound-traffic">
          <span title="本次运行内由真实连接快照聚合，内核停止后显示未采集">${icon('download')}观测 ${escapeHtml(item.traffic)}</span>
          <span>${icon('activity')}速率 ${escapeHtml(item.rate || '—')}</span>
        </div>
        <div class="inbound-actions">
          <button class="btn btn-sm" data-action="inbound-config" data-id="${item.id}" aria-label="配置入口 ${escapeHtml(item.name)}">${icon('sliders')}配置</button>
          <button class="btn btn-sm btn-danger" data-action="delete-inbound" data-id="${item.id}" aria-label="删除入口 ${escapeHtml(item.name)}">${icon('trash')}删除</button>
        </div>
      </div>
    </article>
  `;
}

function flowCard() {
  const mixed = state.inbounds.find((item) => item.enabled && item.kind === 'mixed');
  const steps = [
    { icon: 'terminal', title: '客户端请求', meta: mixed?.listen || '未启用 Mixed 入口' },
    { icon: 'inbound', title: 'Mixed 入口', meta: 'HTTP / SOCKS5' },
    { icon: 'route', title: '路由决策', meta: `${state.routeRules.length} 条路由规则` },
    { icon: 'nodes', title: '出口编排', meta: state.snapshot?.routes?.default_target || '直连' },
    { icon: 'globe', title: '目标网络', meta: 'IPv4 / IPv6' },
  ];
  return `
    <div class="card flow-card" aria-label="请求路径">
      ${steps.map((step, index) => `
        ${index ? `<span class="flow-line"></span>` : ''}
        <div class="flow-step">
          <span class="flow-node">${icon(step.icon)}</span>
          <span class="flow-copy"><strong>${step.title}</strong><span>${step.meta}</span></span>
        </div>
      `).join('')}
    </div>
  `;
}

function inboundsView() {
  const running = state.inbounds.filter((item) => item.enabled).length;
  const kernelRunning = state.kernel.state === 'running';
  const recentInboundLogs = state.logs.filter((item) => item.source === 'inbound').slice(0, 4);
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'INBOUND LISTENERS',
        title: '入口管理',
        description: `管理本机监听入口、透明代理与请求接入策略。当前 ${running} 个入口已启用，内核${kernelRunning ? '运行中' : '未启动'}。`,
        actions: `
          <button class="btn" data-action="refresh-overview">${icon('refresh')}刷新状态</button>
          <button class="btn btn-primary" data-action="new-inbound">${icon('plus')}新增入口</button>
        `,
      })}
      ${flowCard()}
      <div style="height:14px"></div>
      <div class="inbound-grid">
        ${state.inbounds.map(inboundCard).join('')}
      </div>
      <div class="two-grid">
        <article class="card">
          <div class="card-head">
            <div class="card-head-copy"><h2>入口运行摘要</h2><p>监听器配置与当前连接</p></div>
            <span class="tag is-solid">${state.sseConnected ? 'SSE 快照' : '最近快照'}</span>
          </div>
          <div class="mini-stats">
            <div class="mini-stat"><span>已启用入口</span><strong>${running}</strong></div>
            <div class="mini-stat"><span>内核状态</span><strong>${escapeHtml(KERNEL_STATE_LABELS[state.kernel.state] || state.kernel.state)}</strong></div>
            <div class="mini-stat"><span>活动连接</span><strong>${kernelRunning ? state.connections.length : '—'}</strong></div>
          </div>
          <div class="bar-stack" aria-hidden="true">
            <span style="width:${state.inbounds.length ? (running / state.inbounds.length) * 100 : 0}%;background:linear-gradient(90deg,var(--mint),var(--sky))"></span>
          </div>
          <div class="chart-foot" style="padding-left:0;margin-top:10px"><span>${running} / ${state.inbounds.length} 已启用</span><span>内核 ${kernelRunning ? '运行中' : '未启动'}</span></div>
        </article>
        <article class="card">
          <div class="card-head">
            <div class="card-head-copy"><h2>最近入口事件</h2><p>来自应用日志的入口配置变更</p></div>
            <button class="link-button" data-view="logs">查看日志 ${icon('chevron-right')}</button>
          </div>
          <div class="event-list">
            ${recentInboundLogs.map((item) => `
              <div class="event-line">
                <span class="event-time">${escapeHtml(item.time)}</span>
                <span class="event-dot"></span>
                <span>${escapeHtml(item.message)}</span>
              </div>
            `).join('') || '<div class="empty-state"><strong>暂无入口事件</strong><span>入口配置变更会记录在这里</span></div>'}
          </div>
        </article>
      </div>
    </section>
  `;
}

function egressView() {
  const tabs = [
    { id: 'nodes', label: '节点', count: state.nodes.length },
    { id: 'groups', label: '策略组', count: state.groups.length },
    { id: 'chains', label: '链路', count: state.chains.length },
  ];
  const actions = state.egressTab === 'nodes'
    ? `<button class="btn" data-action="test-all-nodes" ${state.nodeTesting || (state.nodeTestKind === 'url' && state.kernel.state !== 'running') ? 'disabled title="URL Test 需要先启动内核"' : ''}>${icon('zap')}全部 ${escapeHtml(nodeTestMethod().label)}</button><button class="btn btn-primary" data-action="import-node">${icon('link')}导入节点</button>`
    : state.egressTab === 'groups'
      ? `<button class="btn btn-primary" data-action="add-group">${icon('plus')}新建策略组</button>`
      : `<button class="btn btn-primary" data-action="add-chain">${icon('plus')}新建链路</button>`;
  const panel = state.egressTab === 'nodes' ? nodesPanel() : state.egressTab === 'groups' ? groupsPanel() : chainsPanel();
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'EGRESS ORCHESTRATION',
        title: '出口编排',
        description: '管理节点、策略组与多跳链路，所有变更在应用后由控制面生成新的运行时配置。',
        actions,
      })}
      <div class="tabs">
        ${tabs.map((tab) => `<button class="tab ${state.egressTab === tab.id ? 'is-active' : ''}" data-action="egress-tab" data-tab="${tab.id}">${tab.label}<span class="tab-count">${tab.count}</span></button>`).join('')}
      </div>
      ${panel}
    </section>
  `;
}

function filteredNodes() {
  const query = state.nodeQuery.trim().toLowerCase();
  return state.nodes.filter((item) => {
    if (!query) return true;
    return [item.id, item.name, item.server, item.group, item.protocol].join(' ').toLowerCase().includes(query);
  });
}

function nodeRows() {
  const filtered = filteredNodes();
  if (!filtered.length) {
    return `<tr><td colspan="9"><div class="empty-state"><span class="empty-icon">${icon('search')}</span><strong>没有匹配的节点</strong><span>尝试搜索节点名称、协议或服务器地址</span></div></td></tr>`;
  }
  const testDisabled = state.nodeTesting || (state.nodeTestKind === 'url' && state.kernel.state !== 'running');
  const method = nodeTestMethod();
  return filtered.map((item) => `
    <tr class="${state.selectedNodeIDs.has(item.id) ? 'is-selected' : ''}">
      <td class="select-cell"><input class="check-input" type="checkbox" data-node-select="${escapeHtml(item.id)}" aria-label="选择 ${escapeHtml(item.name)}" ${state.selectedNodeIDs.has(item.id) ? 'checked' : ''}></td>
      <td>
        <div class="node-name" title="${escapeHtml(item.name)} · ${escapeHtml(item.server)}:${item.port}">
          <div class="cell-primary"><strong>${escapeHtml(item.name)}</strong><span>${escapeHtml(item.server)}:${item.port}</span></div>
        </div>
      </td>
      <td title="${escapeHtml(item.protocol)}"><span class="tag is-solid">${escapeHtml(item.protocol)}</span></td>
      <td class="muted" title="${escapeHtml(item.group)}">${escapeHtml(item.group)}</td>
      <td>${item.lastTestError ? `<span class="tag is-rose" title="${escapeHtml(item.lastTestError)}">测速失败</span>` : item.latency > 0 ? `<div class="latency-cell"><span class="latency-number ${latencyClass(item.latency)}">${item.latency} ms</span>${latencyBars(item.latency)}</div>` : '<span class="tag">未测速</span>'}</td>
      <td>${item.udp ? `<span class="tag is-mint">UDP</span>` : `<span class="tag">—</span>`}</td>
      <td class="mono nowrap" title="本次运行内由真实连接快照聚合；速率 ${escapeHtml(item.rate || '未采集')}">${escapeHtml(item.traffic)}</td>
      <td><span class="dot-word" title="${escapeHtml(item.lastTestKind ? `${nodeTestMethod(item.lastTestKind).label}${item.lastTestError ? `：${item.lastTestError}` : ''}` : '尚未测速')}"><span class="status-dot ${item.lastTestError ? 'is-error' : item.latency > 0 ? 'is-online' : 'is-idle'}"></span>${item.lastTestError ? '不可达' : item.latency > 0 ? '可达' : '未测速'}</span></td>
      <td><div class="cell-actions"><button class="btn btn-sm btn-icon-only" data-action="test-node" data-id="${escapeHtml(item.id)}" title="使用 ${escapeHtml(method.label)} 测速" aria-label="使用 ${escapeHtml(method.label)} 测速 ${escapeHtml(item.name)}" ${testDisabled ? 'disabled' : ''}>${icon('zap')}</button><button class="btn btn-sm btn-icon-only" data-action="edit-node" data-id="${escapeHtml(item.id)}" title="编辑节点" aria-label="编辑节点 ${escapeHtml(item.name)}">${icon('sliders')}</button><button class="btn btn-sm btn-icon-only" data-action="node-detail" data-id="${escapeHtml(item.id)}" title="查看详情" aria-label="查看节点详情 ${escapeHtml(item.name)}">${icon('more')}</button><button class="btn btn-sm btn-danger" data-action="delete-node" data-id="${escapeHtml(item.id)}" title="删除节点" aria-label="删除节点 ${escapeHtml(item.name)}">${icon('trash')}删除</button></div></td>
    </tr>
  `).join('');
}

function nodesPanel() {
  const filtered = filteredNodes();
  const selectedCount = state.selectedNodeIDs.size;
  const allSelected = filtered.length > 0 && filtered.every((item) => state.selectedNodeIDs.has(item.id));
  const method = nodeTestMethod();
  return `
    <article class="card">
      <div class="toolbar">
        <div class="toolbar-left">
          <label class="filter-input">${icon('search')}<input type="search" value="${escapeHtml(state.nodeQuery)}" placeholder="搜索节点、协议、服务器" data-filter="nodes" aria-label="搜索节点"></label>
          <span class="soft-count">${state.nodes.length} 个节点 · ${state.groups.length} 个策略组</span>
        </div>
        <div class="toolbar-right node-test-toolbar">
          <span class="soft-count">测速方式</span>
          <select class="select small" data-setting="node-test-kind" aria-label="节点测速方式">
            ${NODE_TEST_METHODS.map((item) => `<option value="${item.id}" ${state.nodeTestKind === item.id ? 'selected' : ''}>${escapeHtml(item.label)}</option>`).join('')}
          </select>
          ${state.nodeTestKind === 'url' ? `<input class="input node-test-url" id="node-test-url" type="url" value="${escapeHtml(state.nodeTestURL)}" placeholder="默认 generate_204" aria-label="URL Test 目标">` : `<span class="soft-count">${escapeHtml(method.hint)}</span>`}
          <span class="tag ${selectedCount ? 'is-sky' : ''}">${selectedCount} 已选</span>
          <button class="btn btn-sm" data-action="test-selected-nodes" ${selectedCount && !state.nodeTesting ? '' : 'disabled'}>${icon('zap')}${state.nodeTesting ? '测速中…' : '测速选中'}</button>
        </div>
      </div>
      <div class="table-wrap" role="region" aria-label="节点列表" tabindex="0">
        <table class="data-table nodes-table">
          <thead><tr><th class="select-cell"><input class="check-input" type="checkbox" data-node-select-all aria-label="选择当前列表全部节点" ${allSelected ? 'checked' : ''}></th><th>节点</th><th>协议</th><th>分组</th><th>延迟</th><th>UDP</th><th>流量</th><th>状态</th><th class="is-right">操作</th></tr></thead>
          <tbody id="nodes-body">${nodeRows()}</tbody>
        </table>
      </div>
    </article>
  `;
}

function groupsPanel() {
  const selectorCount = state.groups.filter((group) => group.kind !== 'urltest').length;
  const urlTestCount = state.groups.filter((group) => group.kind === 'urltest').length;
  const cards = state.groups.map((group) => {
    const selected = byId(group.selected);
    const interval = Number(group.interval_seconds || group.interval || 0) || (group.kind === 'urltest' ? 180 : 0);
    return `
      <article class="card group-card">
        <div class="group-top">
          <span class="group-icon">${icon(group.kind === 'urltest' ? 'gauge' : 'nodes')}</span>
          <div class="group-top-actions">
            <span class="tag ${group.kind === 'urltest' ? 'is-violet' : 'is-sky'}">${group.kind === 'urltest' ? 'URL TEST' : 'SELECTOR'}</span>
            <button class="btn btn-sm btn-danger" data-action="delete-group" data-id="${escapeHtml(group.id)}" title="删除策略组" aria-label="删除策略组 ${escapeHtml(group.name)}">${icon('trash')}删除</button>
          </div>
        </div>
        <div class="group-title"><h3>${escapeHtml(group.name)}</h3><p>${escapeHtml(group.description)}</p></div>
        <div class="group-meta">
          <span class="tag is-solid">${group.members.length} 个成员</span>
          ${group.kind === 'urltest' ? `<span class="tag is-violet">${interval}s 检测${Number(group.interval_seconds || group.interval || 0) ? '' : '（默认）'}</span>` : '<span class="tag is-solid">手动切换</span>'}
        </div>
        <div class="group-current"><span class="status-dot is-idle"></span><span>当前选择</span><strong>${escapeHtml(selected?.name || group.selected || '未选择')}</strong></div>
        <div class="group-members">
          ${group.members.map((memberId) => {
            const member = byId(memberId);
            return `<button class="member-chip ${memberId === group.selected ? 'is-selected' : ''}" data-action="select-member" data-group="${group.id}" data-node="${memberId}" title="设为当前出口">${escapeHtml(member?.name || memberId)}</button>`;
          }).join('')}
        </div>
        <div class="group-footer">
          <span class="latency-value">${selected?.latency > 0 ? `${selected.latency} ms` : '未测速'}</span>
          <div class="group-footer-actions">
            <button class="btn btn-sm" data-action="test-node" data-id="${selected?.id || ''}" ${selected && !(state.nodeTestKind === 'url' && state.kernel.state !== 'running') ? '' : 'disabled title="请先选择节点；URL Test 需要内核运行中"'}>${icon('zap')}测速</button>
            <button class="btn btn-sm" data-action="edit-group" data-id="${escapeHtml(group.id)}">${icon('sliders')}编辑</button>
          </div>
        </div>
      </article>
    `;
  }).join('');
  return `
    <div class="egress-group-grid">${cards || `<article class="card"><div class="empty-state"><span class="empty-icon">${icon('layers')}</span><strong>还没有策略组</strong><span>先导入节点，再创建 Selector 或 URL Test 策略组</span><button class="btn btn-primary" data-action="add-group">${icon('plus')}新建策略组</button></div></article>`}</div>
    <div style="height:14px"></div>
    <article class="card">
      <div class="card-head">
        <div class="card-head-copy"><h2>策略组行为</h2><p>策略组决定规则命中后如何选择具体出口</p></div>
        <span class="tag is-solid">${selectorCount} Selector · ${urlTestCount} URL Test</span>
      </div>
      <div class="three-grid">
        <div class="info-block"><span class="tag is-sky">SELECTOR</span><strong>${selectorCount} 个手动选择组</strong><p>规则命中后使用策略组当前选中的成员；未选择时由内核使用首个成员。</p></div>
        <div class="info-block"><span class="tag is-violet">URL TEST</span><strong>${urlTestCount} 个自动测速组</strong><p>检测间隔和 URL 取自当前策略组配置；未配置时由控制面使用默认值。</p></div>
        <div class="info-block"><span class="tag is-amber">MEASUREMENT</span><strong>延迟仅来自真实测速</strong><p>页面不会生成模拟延迟；未执行 URL Test、TCP Ping 或 ICMP Ping 时显示“未测速”。</p></div>
      </div>
    </article>
  `;
}

function chainsPanel() {
  return `
    <div class="chain-list">
      ${state.chains.length ? state.chains.map((chain) => `
        <article class="card chain-card">
          <div class="chain-main">
            <div class="chain-copy">
              <h3>${escapeHtml(chain.name)}</h3>
              <p>${escapeHtml(chain.description)}</p>
              <span>${chain.usedBy === '未绑定' ? '未被路由规则引用' : `被 “${escapeHtml(chain.usedBy)}” 引用`}</span>
            </div>
            <div class="chain-rail">
              ${chain.hops.map((hopId, index) => {
                const hop = byId(hopId);
                const nodeName = hop?.name || hopId;
                const serverLabel = hop?.server ? `${hop.protocol ? `${String(hop.protocol).toUpperCase()} · ` : ''}${hop.server}${hop.port ? `:${hop.port}` : ''}` : '节点信息缺失';
                return `
                  ${index ? `<span class="chain-arrow">${icon('chevron-right')}</span>` : ''}
                  <div class="hop" title="第 ${index + 1} 跳 · ${escapeHtml(nodeName)}">
                    <span class="hop-node">${escapeHtml(hop?.protocol ? String(hop.protocol).slice(0, 2).toUpperCase() : pad(index + 1))}</span>
                    <span class="hop-copy"><strong title="${escapeHtml(nodeName)}">${escapeHtml(nodeName)}</strong><span title="${escapeHtml(serverLabel)}">${escapeHtml(serverLabel)}</span></span>
                  </div>
                `;
              }).join('')}
              <span class="chain-arrow">${icon('chevron-right')}</span>
              <span class="flow-node">${icon('globe')}</span>
            </div>
            <div class="chain-meta">
              <span class="tag is-solid">${chain.hops.length} 跳</span>
              <button class="btn btn-sm" data-action="edit-chain" data-id="${escapeHtml(chain.id)}" aria-label="编辑链路 ${escapeHtml(chain.name)}">${icon('sliders')}编辑</button>
              <button class="btn btn-sm btn-danger" data-action="delete-chain" data-id="${escapeHtml(chain.id)}" aria-label="删除链路 ${escapeHtml(chain.name)}">${icon('trash')}删除</button>
            </div>
          </div>
        </article>
      `).join('') : `<article class="card"><div class="empty-state"><span class="empty-icon">${icon('link')}</span><strong>还没有多跳链路</strong><span>链路需要至少一个节点，并按顺序串联</span><button class="btn btn-primary" data-action="add-chain">${icon('plus')}新建链路</button></div></article>`}
    </div>
    <div style="height:14px"></div>
    <article class="card">
      <div class="card-head"><div class="card-head-copy"><h2>多跳链路</h2><p>链路按顺序将流量转发至下一跳，可用于跨区域中转与固定出口编排。</p></div><span class="tag is-violet">${icon('link')}${state.chains.length} 条链路</span></div>
      <div class="two-grid">
        <div class="info-block"><span class="tag is-mint">ROUTING</span><strong>规则引用链路</strong><p>路由规则可直接将目标流量绑定到链路。链路节点变更后无需修改上游规则。</p></div>
        <div class="info-block"><span class="tag is-amber">HOPS</span><strong>顺序固定</strong><p>流量按编辑器中的顺序逐跳转发，最多 4 跳；实际连通性以单独节点测速为准。</p></div>
      </div>
    </article>
  `;
}

function routesView() {
  const modes = [
    { id: 'rules', title: '规则模式', desc: '按规则表匹配流量并选择出口', tag: '推荐' },
    { id: 'global', title: '全局代理', desc: '所有流量通过默认出口转发', tag: '覆盖' },
    { id: 'direct', title: '直连模式', desc: '绕过代理出口，全部直接连接', tag: '旁路' },
  ];
  const rules = state.routeRules.map((rule, index) => `
    <tr>
      <td><div class="route-rule-order"><button class="order-button" data-action="move-rule" data-id="${escapeHtml(rule.id)}" data-direction="up" title="上移规则" aria-label="上移规则 ${escapeHtml(rule.name)}" ${index === 0 ? 'disabled' : ''}>${icon('chevron-up')}</button><span class="mono faint">${pad(index + 1)}</span><button class="order-button" data-action="move-rule" data-id="${escapeHtml(rule.id)}" data-direction="down" title="下移规则" aria-label="下移规则 ${escapeHtml(rule.name)}" ${index === state.routeRules.length - 1 ? 'disabled' : ''}>${icon('chevron-down')}</button></div></td>
      <td><div class="cell-primary" title="${escapeHtml(rule.name)} · ${escapeHtml(rule.type)}"><strong>${escapeHtml(rule.name)}</strong><span>${escapeHtml(rule.type)}</span></div></td>
      <td><div class="condition-text" title="${escapeHtml(rule.condition)}">${escapeHtml(rule.condition)}</div></td>
      <td>${actionTag(rule.action)}</td>
      <td class="muted nowrap" title="${escapeHtml(rule.target)}">${escapeHtml(rule.target)}</td>
      <td class="mono nowrap">${escapeHtml(rule.hits)}</td>
      <td><button class="switch switch-sm ${rule.enabled ? 'is-on' : ''}" data-action="toggle-rule" data-id="${rule.id}" role="switch" aria-checked="${rule.enabled}" aria-label="切换 ${escapeHtml(rule.name)}"></button></td>
      <td><div class="cell-actions"><button class="btn btn-sm btn-icon-only" data-action="edit-rule" data-id="${rule.id}" title="编辑规则" aria-label="编辑规则 ${escapeHtml(rule.name)}">${icon('sliders')}</button><button class="btn btn-sm btn-danger" data-action="delete-rule" data-id="${rule.id}" title="删除规则" aria-label="删除规则 ${escapeHtml(rule.name)}">${icon('trash')}删除</button></div></td>
    </tr>
  `).join('');
  const routePresets = state.snapshot?.routes?.presets || { bypass_lan: false, china_direct: false, ad_block: false };
  const presets = [
    { id: 'bypass_lan', icon: 'shield', name: '局域网直连', desc: '私有地址绕过代理出口' },
    { id: 'china_direct', icon: 'globe', name: '中国大陆直连', desc: '使用 geosite-cn 与 geoip-cn 规则集' },
    { id: 'ad_block', icon: 'stop', name: '广告拦截', desc: '使用 geosite-ads 规则集拒绝请求' },
  ];
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'ROUTING & POLICY',
        title: '路由策略',
        description: '控制流量如何匹配规则并选择出口。规则按从上到下的顺序执行，命中后停止匹配。',
        actions: `<button class="btn btn-primary" data-action="add-rule">${icon('plus')}新建规则</button>`,
      })}
      <div class="mode-grid">
        ${modes.map((mode) => `
          <button class="mode-card ${state.routeMode === mode.id ? 'is-active' : ''}" data-action="route-mode" data-mode="${mode.id}">
            <span class="mode-radio"></span>
            <span class="mode-copy"><strong>${mode.title}</strong><span>${mode.desc}</span></span>
            <span class="tag ${state.routeMode === mode.id ? 'is-mint' : 'is-solid'} mode-tag">${mode.tag}</span>
          </button>
        `).join('')}
      </div>
      <div class="side-grid">
        <article class="card">
          <div class="card-head">
            <div class="card-head-copy"><h2>规则表</h2><p>按优先级从高到低匹配 · ${state.routeMode === 'rules' ? '规则模式生效中' : '当前模式将覆盖规则表'}</p></div>
            <div class="card-actions"><button class="btn btn-sm" data-action="add-rule">${icon('plus')}添加</button></div>
          </div>
          <div class="table-wrap" role="region" aria-label="路由规则列表" tabindex="0">
            <table class="data-table routes-table">
              <thead><tr><th></th><th>规则</th><th>条件</th><th>动作</th><th>目标</th><th>命中统计</th><th>启用</th><th></th></tr></thead>
              <tbody>${rules || `<tr><td colspan="8"><div class="empty-state"><span class="empty-icon">${icon('route')}</span><strong>还没有路由规则</strong><span>新建规则后，流量会按列表顺序匹配</span><button class="btn btn-primary" data-action="add-rule">${icon('plus')}新建规则</button></div></td></tr>`}</tbody>
            </table>
          </div>
        </article>
        <div class="stack">
          <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>默认出口</h2><p>未命中规则时使用</p></div></div>
            <div class="default-outbound">
              <span class="inbound-icon" style="width:30px;height:30px;border-radius:8px">${icon('nodes')}</span>
              <select class="select small" data-setting="default-outbound" aria-label="默认出口">
                ${state.groups.length ? `<optgroup label="策略组">${state.groups.map((group) => `<option value="group:${group.id}" ${state.snapshot?.routes?.default_egress?.kind === 'group' && state.snapshot?.routes?.default_egress?.id === group.id ? 'selected' : ''}>${escapeHtml(group.name)}</option>`).join('')}</optgroup>` : ''}
                ${state.chains.length ? `<optgroup label="链路">${state.chains.map((chain) => `<option value="chain:${chain.id}" ${state.snapshot?.routes?.default_egress?.kind === 'chain' && state.snapshot?.routes?.default_egress?.id === chain.id ? 'selected' : ''}>${escapeHtml(chain.name)}</option>`).join('')}</optgroup>` : ''}
                ${state.nodes.length ? `<optgroup label="节点">${state.nodes.map((node) => `<option value="node:${node.id}" ${state.snapshot?.routes?.default_egress?.kind === 'node' && state.snapshot?.routes?.default_egress?.id === node.id ? 'selected' : ''}>${escapeHtml(node.name)}</option>`).join('')}</optgroup>` : ''}
                <option value="direct:" ${state.snapshot?.routes?.default_egress?.kind === 'direct' ? 'selected' : ''}>直连</option>
              </select>
            </div>
          </article>
          <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>路由预设</h2><p>内置策略组合，提交后立即生成新配置</p></div></div>
            <div class="preset-list">
              ${presets.map((preset) => `<button class="preset-card ${routePresets[preset.id] ? 'is-selected' : ''}" data-action="toggle-preset" data-key="${preset.id}" role="switch" aria-checked="${Boolean(routePresets[preset.id])}"><span class="preset-icon">${icon(preset.icon)}</span><span class="preset-copy"><strong>${preset.name}</strong><span>${preset.desc}</span></span><span class="switch switch-sm ${routePresets[preset.id] ? 'is-on' : ''}" aria-hidden="true"></span></button>`).join('')}
            </div>
          </article>
          <article class="card card-tight">
            <div class="system-list">
              <div class="system-row"><span>规则总数</span><strong>${state.routeRules.length} 条</strong></div>
              <div class="system-row"><span>规则模式</span><strong>${escapeHtml(state.routeMode)}</strong></div>
              <div class="system-row"><span>默认出口</span><strong>${escapeHtml(state.snapshot?.routes?.default_target || '直连')}</strong></div>
            </div>
          </article>
        </div>
      </div>
    </section>
  `;
}

function intervalLabel(seconds) {
  if (!seconds) return '未设置';
  if (seconds >= 86400) return `每 ${seconds / 86400} 天`;
  if (seconds >= 3600) return `每 ${seconds / 3600} 小时`;
  return `每 ${Math.round(seconds / 60)} 分钟`;
}

function schedulerProgressPercent() {
  const intervalSeconds = Number(state.scheduler?.interval_seconds) || 0;
  const nextRun = state.scheduler?.next_run_at ? new Date(state.scheduler.next_run_at).getTime() : 0;
  const now = state.lastSnapshotAt?.getTime() || Date.now();
  if (!state.flags.autoRefresh || !state.scheduler?.running || intervalSeconds <= 0 || !nextRun) return 0;
  const remaining = Math.max(0, nextRun - now);
  return Math.max(0, Math.min(100, 100 - (remaining / (intervalSeconds * 1000)) * 100));
}

function subscriptionCard(item) {
  const paused = !item.enabled;
  const stateMeta = subscriptionStateMeta(item.state);
  const footerText = paused
    ? '调度已暂停'
    : item.state === 'pending'
      ? '尚未执行首次同步'
      : item.etag && item.etag !== '—'
        ? `ETag ${item.etag}`
        : '远端未返回 ETag';
  return `
    <article class="card sub-card">
      <div class="sub-head">
        <span class="sub-logo">${escapeHtml(item.name.slice(0, 2).toUpperCase())}</span>
        <div class="sub-title"><h3>${escapeHtml(item.name)}</h3><p>${escapeHtml(item.provider)} · ${item.nodes} 个节点</p></div>
        <button class="switch ${item.enabled ? 'is-on' : ''}" data-action="toggle-subscription" data-id="${item.id}" role="switch" aria-checked="${item.enabled}" aria-label="切换 ${escapeHtml(item.name)}"></button>
      </div>
      <div class="sub-url"><code>${escapeHtml(item.url)}</code><span class="tag is-solid" title="出于安全考虑，Web API 不返回订阅令牌原文">已脱敏</span></div>
      <div class="sub-metrics">
        <div class="sub-metric"><span>同步状态</span><strong class="sub-state ${escapeHtml(item.state)}">${escapeHtml(stateMeta.label)}</strong></div>
        <div class="sub-metric"><span>刷新间隔</span><strong>${intervalLabel(item.interval)}</strong></div>
        <div class="sub-metric"><span>最近同步</span><strong>${escapeHtml(item.lastRefresh)}</strong></div>
      </div>
      <div class="sub-footer">
        <span class="dot-word"><span class="status-dot ${stateMeta.dot}"></span><span class="muted" style="font-size:9px">${escapeHtml(footerText)}</span></span>
        <div class="cell-actions"><button class="btn btn-sm" data-action="refresh-subscription" data-id="${item.id}" aria-label="立即刷新订阅 ${escapeHtml(item.name)}" ${paused ? 'disabled' : ''}>${icon('refresh')}立即刷新</button><button class="btn btn-sm" data-action="edit-subscription" data-id="${item.id}" aria-label="编辑订阅 ${escapeHtml(item.name)}">${icon('sliders')}编辑</button><button class="btn btn-sm btn-danger" data-action="delete-subscription" data-id="${item.id}" title="删除订阅" aria-label="删除订阅 ${escapeHtml(item.name)}">${icon('trash')}删除</button></div>
      </div>
    </article>
  `;
}

function subscriptionsView() {
  const schedulerRunning = Boolean(state.scheduler?.running);
  const autoRefresh = Boolean(state.flags.autoRefresh);
  const progress = schedulerProgressPercent();
  const schedulerInterval = Number(state.scheduler?.interval_seconds) || 0;
  const schedulerLabel = schedulerRunning ? (autoRefresh ? '调度器运行中' : '自动刷新已关闭') : '调度器未运行';
  const nextRunLabel = autoRefresh
    ? state.scheduler?.next_run_at ? formatDateTime(state.scheduler.next_run_at) : '等待调度'
    : '等待开启自动刷新';
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'SUBSCRIPTIONS',
        title: '订阅管理',
        description: '从订阅源同步节点与策略配置。刷新由控制面调度器执行，并记录到运行日志。',
        actions: `<button class="btn" data-action="refresh-all-subs">${icon('refresh')}同步全部</button><button class="btn" data-action="import-node">${icon('link')}导入单节点</button><button class="btn btn-primary" data-action="add-subscription">${icon('plus')}添加订阅</button>`,
      })}
      <div class="sub-grid">${state.subscriptions.length ? state.subscriptions.map(subscriptionCard).join('') : `<article class="card"><div class="empty-state"><span class="empty-icon">${icon('rss')}</span><strong>还没有订阅</strong><span>添加订阅或直接导入单个节点链接</span><button class="btn btn-primary" data-action="add-subscription">${icon('plus')}添加订阅</button></div></article>`}</div>
      <div class="two-grid">
        <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>下次计划刷新</h2><p>订阅调度器运行状态</p></div><span class="tag ${schedulerRunning && autoRefresh ? 'is-mint' : 'is-amber'}"><span class="status-dot ${schedulerRunning && autoRefresh ? 'is-online' : 'is-warn'}"></span>${escapeHtml(schedulerLabel)}</span></div>
          <div class="scheduler-card" style="padding:0 0 12px">
            <span class="scheduler-icon">${icon('clock')}</span>
            <div class="scheduler-copy"><strong>${escapeHtml(nextRunLabel)}</strong><span>${state.scheduler?.last_run_at ? `上次扫描 ${formatDateTime(state.scheduler.last_run_at)}` : '尚未执行扫描'}${state.scheduler?.last_error ? ` · ${escapeHtml(state.scheduler.last_error)}` : ''}</span></div>
            <span class="tag is-sky">${schedulerInterval ? `每 ${schedulerInterval}s` : '间隔未读取'}</span>
          </div>
          <div class="progress-track"><span style="width:${progress.toFixed(1)}%"></span></div>
          <div class="chart-foot" style="padding-left:0;margin-top:9px"><span>${state.subscriptions.filter((item) => item.enabled).length} 个启用订阅</span><span>${schedulerRunning ? (autoRefresh ? `本轮进度 ${Math.round(progress)}%` : '自动刷新已关闭') : '调度器已停止'}</span></div>
        </article>
        <article class="card">
          <div class="card-head"><div class="card-head-copy"><h2>同步策略</h2><p>统一控制订阅自动刷新、失败重试与节点冲突处理</p></div></div>
          <div class="setting-row"><div class="setting-copy"><strong>自动刷新</strong><span>按每个订阅的间隔自动同步</span></div><button class="switch switch-sm ${state.flags.autoRefresh ? 'is-on' : ''}" data-action="toggle-setting" data-key="autoRefresh" role="switch" aria-checked="${state.flags.autoRefresh}" aria-label="自动刷新订阅"></button></div>
          <div class="setting-row"><div class="setting-copy"><strong>失败重试</strong><span>拉取失败后按指数退避重试</span></div><select class="select small" data-setting="subscription-max-attempts" aria-label="订阅拉取最大尝试次数">${[1, 2, 3, 5].map((value) => `<option value="${value}" ${state.flags.subscriptionMaxAttempts === value ? 'selected' : ''}>最多 ${value} 次</option>`).join('')}</select></div>
          <div class="setting-row"><div class="setting-copy"><strong>更新冲突</strong><span>远端不再返回的订阅节点如何处理</span></div><select class="select small" data-setting="subscription-conflict-policy" aria-label="订阅更新冲突策略"><option value="keep_local" ${state.flags.subscriptionConflictPolicy === 'keep_local' ? 'selected' : ''}>保留本地副本</option><option value="follow_remote" ${state.flags.subscriptionConflictPolicy === 'follow_remote' ? 'selected' : ''}>跟随订阅删除</option></select></div>
          <div class="form-hint" style="margin-top:10px">策略保存后立即生效；自动刷新关闭时仍可手动“同步全部”或单独刷新。</div>
        </article>
      </div>
    </section>
  `;
}

function rulesetRows() {
  const query = state.rulesetQuery.trim().toLowerCase();
  const filtered = state.ruleSets.filter((item) => !query || [item.name, item.description, item.format, item.sha].join(' ').toLowerCase().includes(query));
  if (!filtered.length) return `<tr><td colspan="8"><div class="empty-state"><span class="empty-icon">${icon('search')}</span><strong>没有匹配的规则集</strong><span>尝试其他名称或 SHA 前缀</span></div></td></tr>`;
  return filtered.map((item) => `
    <tr class="${item.builtin ? '' : 'custom-ruleset'}">
      <td><div class="ruleset-name" title="${escapeHtml(item.name)} · ${escapeHtml(item.description)}"><span class="ruleset-icon">${icon(item.builtin ? 'database' : 'sparkles')}</span><div class="cell-primary"><strong>${escapeHtml(item.name)}</strong><span>${escapeHtml(item.description)}</span></div></div></td>
      <td><span class="tag ${item.builtin ? 'is-sky' : 'is-violet'}">${item.builtin ? '内置' : '自定义'}</span></td>
      <td><span class="tag is-solid">${item.format}</span></td>
      <td class="mono nowrap">${escapeHtml(item.size)}</td>
      <td class="muted nowrap">${escapeHtml(item.updated)}</td>
      <td class="mono faint nowrap" title="${escapeHtml(item.sha)}">${escapeHtml(item.sha)}</td>
      <td><span class="dot-word"><span class="status-dot ${item.status === 'ready' && item.enabled ? 'is-online' : 'is-idle'}"></span>${item.enabled ? statusText(item.status) : '已停用'}</span></td>
      <td><div class="cell-actions"><button class="switch switch-sm ${item.enabled ? 'is-on' : ''}" data-action="toggle-ruleset" data-name="${escapeHtml(item.name)}" role="switch" aria-checked="${item.enabled}" aria-label="切换 ${escapeHtml(item.name)}" ${item.builtin ? 'disabled title="内置规则集不可修改"' : ''}></button><button class="btn btn-sm btn-icon-only" data-action="refresh-ruleset" data-name="${escapeHtml(item.name)}" title="刷新规则集" aria-label="刷新规则集 ${escapeHtml(item.name)}">${icon('refresh')}</button><button class="btn btn-sm btn-icon-only" data-action="ruleset-detail" data-name="${escapeHtml(item.name)}" title="查看详情" aria-label="查看规则集详情 ${escapeHtml(item.name)}">${icon('more')}</button>${item.builtin ? '' : `<button class="btn btn-sm btn-danger" data-action="delete-ruleset" data-name="${escapeHtml(item.name)}" title="删除规则集" aria-label="删除规则集 ${escapeHtml(item.name)}">${icon('trash')}删除</button>`}</div></td>
    </tr>
  `).join('');
}

function rulesetsView() {
  const readyCount = state.ruleSets.filter((item) => item.status === 'ready').length;
  const errorCount = state.ruleSets.filter((item) => item.status === 'error').length;
  const enabledCount = state.ruleSets.filter((item) => item.enabled).length;
  const shaCount = state.ruleSets.filter((item) => item.sha && item.sha !== '未记录').length;
  const latestRuleSet = [...state.ruleSets]
    .filter((item) => item.updated_at)
    .sort((left, right) => new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime())[0];
  const verificationLabel = errorCount ? `${errorCount} 个异常` : readyCount ? `${readyCount} / ${state.ruleSets.length} 最近刷新成功` : '尚未执行刷新';
  const refreshDetail = errorCount ? `${errorCount} 个最近刷新失败` : readyCount ? `${readyCount} 个最近刷新成功` : '尚无刷新记录';
  const verificationTone = errorCount ? 'is-rose' : readyCount === state.ruleSets.length && state.ruleSets.length > 0 ? 'is-mint' : 'is-amber';
  const verificationDot = errorCount ? 'is-error' : readyCount === state.ruleSets.length && state.ruleSets.length > 0 ? 'is-online' : 'is-warn';
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'RULE SETS',
        title: '规则集管理',
        description: '维护内置与自定义规则集，供路由规则引用。规则集更新后由控制面生成新的配置 revision。',
        actions: `<button class="btn" data-action="refresh-all-rulesets">${icon('refresh')}更新全部</button><button class="btn btn-primary" data-action="new-ruleset">${icon('plus')}新建规则集</button>`,
      })}
      <div class="obs-metrics">
        <div class="obs-metric"><span>规则集总数</span><strong>${state.ruleSets.length}</strong><small>${state.ruleSets.filter((item) => item.builtin).length} 内置 / ${state.ruleSets.filter((item) => !item.builtin).length} 自定义</small></div>
        <div class="obs-metric"><span>已启用</span><strong>${enabledCount}</strong><small>参与路由匹配</small></div>
        <div class="obs-metric"><span>刷新状态</span><strong>${readyCount}</strong><small>${refreshDetail}</small></div>
        <div class="obs-metric"><span>最近更新</span><strong>${escapeHtml(latestRuleSet?.updated || '从未刷新')}</strong><small>最近一次成功刷新</small></div>
      </div>
      <div class="side-grid">
        <article class="card">
          <div class="toolbar">
            <div class="toolbar-left"><label class="filter-input">${icon('search')}<input type="search" value="${escapeHtml(state.rulesetQuery)}" placeholder="搜索规则集或 SHA" data-filter="rulesets" aria-label="搜索规则集"></label><span class="soft-count">${state.ruleSets.length} 个规则集</span></div>
            <div class="toolbar-right"><span class="tag ${verificationTone}"><span class="status-dot ${verificationDot}"></span>${escapeHtml(verificationLabel)}</span></div>
          </div>
          <div class="table-wrap" role="region" aria-label="规则集列表" tabindex="0">
            <table class="data-table ruleset-table">
              <thead><tr><th>规则集</th><th>来源</th><th>格式</th><th>大小</th><th>更新时间</th><th>SHA</th><th>状态</th><th class="is-right">操作</th></tr></thead>
              <tbody id="rulesets-body">${rulesetRows()}</tbody>
            </table>
          </div>
        </article>
        <div class="stack">
          <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>更新策略</h2><p>控制内置资源与自定义源</p></div></div>
            <div class="setting-row"><div class="setting-copy"><strong>刷新方式</strong><span>当前版本由管理员手动触发规则集刷新</span></div><span class="tag is-solid">手动</span></div>
            <div class="setting-row"><div class="setting-copy"><strong>摘要记录</strong><span>刷新成功时记录远端内容的 SHA-256</span></div><span class="tag ${shaCount === state.ruleSets.length && state.ruleSets.length > 0 ? 'is-mint' : 'is-solid'}">${shaCount} / ${state.ruleSets.length} 已记录</span></div>
            <div class="setting-row"><div class="setting-copy"><strong>失败策略</strong><span>刷新失败记录错误，不替换当前已应用配置</span></div><span class="tag is-solid">保留当前配置</span></div>
          </article>
          <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>当前 revision</h2><p>控制面编译当前配置后的运行时版本</p></div></div>
            <div class="code-block"><span>${escapeHtml(shortRevision(state.kernel.revision || '—'))}</span><button class="icon-button" data-action="copy" data-copy="${escapeHtml(state.kernel.revision || '')}" title="复制 revision" aria-label="复制 revision">${icon('copy')}</button></div>
            <div class="system-list" style="margin-top:9px"><div class="system-row"><span>规则模式</span><strong>${escapeHtml(state.routeMode)}</strong></div><div class="system-row"><span>生效规则</span><strong>${enabledCount} 个规则集</strong></div></div>
          </article>
        </div>
      </div>
    </section>
  `;
}

function connectionRows() {
  const query = state.connQuery.trim().toLowerCase();
  const filtered = state.connections.filter((item) => !query || [item.target, item.id, item.outbound, item.rule, item.process].join(' ').toLowerCase().includes(query));
  if (!filtered.length) return `<tr><td colspan="8"><div class="empty-state"><span class="empty-icon">${icon('activity')}</span><strong>${state.connections.length ? '没有匹配的连接' : state.kernel.state === 'running' ? '当前没有活动连接' : '内核未运行'}</strong><span>${state.connections.length ? '尝试调整筛选条件' : state.kernel.state === 'running' ? '新连接会实时显示在这里' : '启动内核后才会产生连接快照'}</span></div></td></tr>`;
  return filtered.map((item) => `
    <tr>
      <td><div class="conn-target" title="${escapeHtml(item.target)} · ${escapeHtml(item.source || item.id)}"><strong>${escapeHtml(item.target)}</strong><span>${escapeHtml(item.source || item.id)}</span></div></td>
      <td title="${escapeHtml(item.inbound)}"><span class="tag is-solid">${escapeHtml(item.inbound)}</span></td>
      <td title="${escapeHtml(item.outbound)}">${escapeHtml(item.outbound)}</td>
      <td class="muted" title="${escapeHtml(item.rule)}">${escapeHtml(item.rule)}</td>
      <td class="mono" title="${escapeHtml(item.process || '—')}">${escapeHtml(item.process || '—')}</td>
      <td class="mono nowrap">${escapeHtml(item.upload)} / ${escapeHtml(item.download)}</td>
      <td class="mono nowrap muted">${escapeHtml(item.duration)}</td>
      <td><div class="cell-actions"><button class="btn btn-sm btn-danger" data-action="close-connection" data-id="${item.id}" title="关闭连接" aria-label="关闭连接 ${escapeHtml(item.target)}" ${state.kernel.state === 'running' ? '' : 'disabled'}>${icon('close')}关闭</button></div></td>
    </tr>
  `).join('');
}



function logStats() {
  const app = state.logs.filter((item) => item.channel === 'app').length;
  const kernel = state.logs.filter((item) => item.channel === 'kernel').length;
  const warn = state.logs.filter((item) => item.level === 'warn').length;
  const error = state.logs.filter((item) => item.level === 'error').length;
  return { app, kernel, warn, error, total: state.logs.length };
}

function filteredLogs() {
  const query = state.logQuery.trim().toLowerCase();
  return state.logs.filter((item) => {
    if (state.logSource !== 'all' && item.channel !== state.logSource) return false;
    if (state.logLevel !== 'all' && item.level !== state.logLevel) return false;
    if (query && ![item.channel, item.level, item.source, item.message].join(' ').toLowerCase().includes(query)) return false;
    return true;
  });
}

function logLineText(item) {
  return `${item.time} [${item.channel.toUpperCase()}] ${item.level.toUpperCase()} ${item.source} ${item.message}`;
}

function logLines() {
  const filtered = filteredLogs();
  if (!filtered.length) return `<div class="empty-state"><span class="empty-icon">${icon('terminal')}</span><strong>暂无匹配日志</strong><span>调整来源、级别或关键词筛选</span></div>`;
  return filtered.map((item, index) => `
    <div class="log-line ${index === filtered.length - 1 ? 'is-new' : ''}" data-action="copy-log" data-log-id="${escapeHtml(item.id)}" role="button" tabindex="0" title="点击复制完整日志">
      <span class="log-time">${escapeHtml(item.time)}</span>
      <span class="log-channel ${item.channel}">${item.channel === 'app' ? 'APP' : 'KERNEL'}</span>
      <span class="log-level ${item.level}">${escapeHtml(item.level)}</span>
      <span class="log-source">${escapeHtml(item.source)}</span>
      <span class="log-message">${escapeHtml(item.message)}</span>
    </div>
  `).join('');
}

function logsView() {
  const stats = logStats();
  const filtered = filteredLogs();
  const streamState = !state.sseConnected ? 'RETRY' : state.logPaused ? 'PAUSED' : 'LIVE';
  const appStreamLive = state.sseConnected;
  const kernelStreamLive = state.sseConnected && state.kernel.state === 'running';
  const activeSourceCount = Number(appStreamLive) + Number(kernelStreamLive);
  const sourceOptions = [
    { id: 'all', label: `全部 ${stats.total}` },
    { id: 'app', label: `应用 ${stats.app}` },
    { id: 'kernel', label: `内核 ${stats.kernel}` },
  ];
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'STREAMING LOGS',
        title: '应用与内核日志',
        description: '流式查看控制面应用日志与 sing-box 内核日志。支持双流合并、来源筛选、级别过滤和自动滚动。',
        actions: `<span class="tag ${state.sseConnected && !state.logPaused ? 'is-mint' : 'is-amber'}"><span class="status-dot ${state.sseConnected && !state.logPaused ? 'is-online' : 'is-warn'}"></span>${!state.sseConnected ? '事件流重连中' : state.logPaused ? '显示已暂停' : '正在接收'}</span><button class="btn" data-action="toggle-log-pause">${icon(state.logPaused ? 'play' : 'stop')}${state.logPaused ? '继续查看' : '暂停显示'}</button>`,
      })}
      <div class="obs-metrics">
        <div class="obs-metric"><span>流状态</span><strong>${streamState}</strong><small>SSE · /api/v1/events</small></div>
        <div class="obs-metric"><span>应用日志</span><strong id="log-app-count">${stats.app}</strong><small>control plane</small></div>
        <div class="obs-metric"><span>内核日志</span><strong id="log-kernel-count">${stats.kernel}</strong><small>sing-box runtime</small></div>
        <div class="obs-metric"><span>告警 / 错误</span><strong id="log-alert-count">${stats.warn + stats.error}</strong><small>${stats.error} 个错误</small></div>
      </div>
      <div class="side-grid logs-layout">
        <article class="card log-stream-card">
          <div class="log-toolbar log-toolbar-wrap">
            <div class="toolbar-left">
              <span class="segmented log-source-tabs" aria-label="日志来源">
                ${sourceOptions.map((source) => `<button data-action="log-source" data-source="${source.id}" class="${state.logSource === source.id ? 'is-active' : ''}">${source.label}</button>`).join('')}
              </span>
              <label class="filter-input log-search">${icon('search')}<input type="search" value="${escapeHtml(state.logQuery)}" placeholder="搜索来源、级别或消息" data-filter="logs" aria-label="搜索日志"></label>
            </div>
            <div class="toolbar-right">
              <select class="select small" data-setting="log-level" aria-label="日志级别">
                <option value="all" ${state.logLevel === 'all' ? 'selected' : ''}>所有级别</option>
                <option value="info" ${state.logLevel === 'info' ? 'selected' : ''}>INFO</option>
                <option value="warn" ${state.logLevel === 'warn' ? 'selected' : ''}>WARN</option>
                <option value="error" ${state.logLevel === 'error' ? 'selected' : ''}>ERROR</option>
                <option value="debug" ${state.logLevel === 'debug' ? 'selected' : ''}>DEBUG</option>
              </select>
              <button class="btn btn-sm ${state.logAutoScroll ? 'is-selected' : ''}" data-action="toggle-log-autoscroll">${icon('chevron-down')}自动滚动</button>
              <button class="btn btn-sm" data-action="clear-logs">${icon('close')}清空</button>
            </div>
          </div>
          <div class="log-console log-console-xl" id="logs-console" role="log" aria-label="应用与内核日志" tabindex="0">${logLines()}</div>
          <div class="log-console-footer">
            <span class="dot-word"><span class="status-dot ${state.sseConnected ? 'is-online' : 'is-warn'}"></span>${!state.sseConnected ? '事件流重连中' : state.logPaused ? '显示已暂停 · 仍在接收' : '流式接收中'}</span>
            <span class="log-copy-help">${icon('copy')}点击日志行复制</span>
            <span id="log-match-count">显示 ${filtered.length} / ${state.logs.length} 行</span>
            <span id="log-buffer-count">缓冲区 ${state.logs.length} / 500</span>
            <button class="link-button" data-action="scroll-log-bottom">跳到底部 ${icon('chevron-down')}</button>
          </div>
        </article>
        <div class="stack">
          <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>数据源</h2><p>控制面与内核日志流状态</p></div><span class="tag ${activeSourceCount === 2 ? 'is-mint' : 'is-amber'}">${activeSourceCount} / 2 在线</span></div>
            <button class="log-source-card ${state.logSource === 'app' ? 'is-active' : ''}" data-action="log-source" data-source="app">
              <span class="log-source-icon app">${icon('terminal')}</span>
              <span class="log-source-copy"><strong>应用日志</strong><span>control plane · structured</span></span>
              <span class="log-source-count" id="log-source-app-count">${stats.app}</span>
              <span class="status-dot ${state.sseConnected ? 'is-online' : 'is-warn'}" title="${state.sseConnected ? '应用日志流已连接' : '应用日志流重连中'}"></span>
            </button>
            <button class="log-source-card ${state.logSource === 'kernel' ? 'is-active' : ''}" data-action="log-source" data-source="kernel">
              <span class="log-source-icon kernel">${icon('cpu')}</span>
              <span class="log-source-copy"><strong>内核日志</strong><span>${escapeHtml(kernelVersionLabel())} · runtime</span></span>
              <span class="log-source-count" id="log-source-kernel-count">${stats.kernel}</span>
              <span class="status-dot ${kernelStreamLive ? 'is-online' : 'is-idle'}" title="${kernelStreamLive ? '内核运行中，日志事件实时接入' : '内核未运行，仅显示历史日志'}"></span>
            </button>
          </article>
          <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>流状态</h2><p>当前日志管道参数</p></div></div>
            <div class="system-list">
              <div class="system-row"><span>传输通道</span><strong>SSE</strong></div>
              <div class="system-row"><span>连接状态</span><strong>${state.sseConnected ? '已连接' : '重连中'}</strong></div>
              <div class="system-row"><span>文本编码</span><strong>UTF-8</strong></div>
              <div class="system-row"><span>本地缓冲上限</span><strong>500 行</strong></div>
            </div>
          </article>
          <article class="card">
            <div class="card-head"><div class="card-head-copy"><h2>快捷筛选</h2><p>快速定位异常日志</p></div></div>
            <div class="log-quick-actions">
              <button class="log-quick-filter" data-action="log-level-quick" data-level="error"><span class="tag is-rose">ERROR</span><strong>${stats.error}</strong></button>
              <button class="log-quick-filter" data-action="log-level-quick" data-level="warn"><span class="tag is-amber">WARN</span><strong>${stats.warn}</strong></button>
              <button class="log-quick-filter" data-action="log-level-quick" data-level="debug"><span class="tag is-violet">DEBUG</span><strong>${state.logs.filter((item) => item.level === 'debug').length}</strong></button>
              <button class="log-quick-filter" data-action="log-level-quick" data-level="all"><span class="tag is-solid">全部</span><strong>${stats.total}</strong></button>
            </div>
          </article>
        </div>
      </div>
    </section>
  `;
}

function observabilityView() {
  const available = Boolean(state.traffic.available);
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'OBSERVABILITY',
        title: '运行观测',
        description: '查看内核当前连接与吞吐采样。应用和内核日志位于独立的流式日志页面。',
        actions: `<span class="tag ${state.sseConnected ? 'is-mint' : 'is-amber'}"><span class="status-dot ${state.sseConnected ? 'is-online' : 'is-warn'}"></span>${state.sseConnected ? 'LIVE' : 'RECONNECTING'} · ${state.lastSnapshotAt ? formatClock(state.lastSnapshotAt) : '—'}</span><button class="btn btn-primary" data-view="logs">${icon('terminal')}流式日志</button>`,
      })}
      <div class="obs-metrics">
        <div class="obs-metric"><span>活动连接</span><strong id="live-connection-count">${available ? state.connections.length : '—'}</strong><small>${available ? '来自内核连接快照' : '内核未运行'}</small></div>
        <div class="obs-metric"><span>下行速率</span><strong id="live-down-rate">${available ? formatRate(state.traffic.download_rate) : '未采集'}</strong><small>${available ? '实时采样' : '启动内核后采样'}</small></div>
        <div class="obs-metric"><span>上行速率</span><strong id="live-up-rate">${available ? formatRate(state.traffic.upload_rate) : '未采集'}</strong><small>${available ? '实时采样' : '启动内核后采样'}</small></div>
        <div class="obs-metric"><span>本次运行流量</span><strong id="live-total-traffic">${available ? formatBytes((state.traffic.upload_total || 0) + (state.traffic.download_total || 0)) : '未采集'}</strong><small>${available ? `上行 ${formatBytes(state.traffic.upload_total)}` : '启动内核后采样'}</small></div>
      </div>
      ${connectionsPanel()}
    </section>
  `;
}

function connectionsPanel() {
  return `
    <article class="card">
      <div class="toolbar">
        <div class="toolbar-left"><label class="filter-input">${icon('search')}<input type="search" value="${escapeHtml(state.connQuery)}" placeholder="筛选目标、进程或出口" data-filter="connections" aria-label="筛选连接"></label><span class="soft-count" id="conn-count">${state.traffic.available ? `${state.connections.length} 条连接` : '内核未运行'}</span></div>
        <div class="toolbar-right"><button class="btn btn-sm btn-danger" data-action="close-all-connections" ${state.kernel.state === 'running' ? '' : 'disabled title="需要先启动内核"'}>${icon('close')}全部关闭</button></div>
      </div>
      <div class="table-wrap" role="region" aria-label="活动连接列表" tabindex="0">
        <table class="data-table connections-table">
          <thead><tr><th>目标</th><th>入口</th><th>出口</th><th>命中规则</th><th>进程</th><th>上下行</th><th>时长</th><th class="is-right">操作</th></tr></thead>
          <tbody id="connections-body">${connectionRows()}</tbody>
        </table>
      </div>
    </article>
  `;
}

const DNS_SLOT_LABELS = { domestic: '国内 DNS', foreign: '国外 DNS', default: '默认 DNS' };
const DNS_PROVIDERS = [
  { id: 'alidns', label: '阿里 DNS' },
  { id: 'tencent', label: '腾讯 DNS' },
  { id: 'cloudflare', label: 'Cloudflare' },
  { id: 'google', label: 'Google' },
  { id: 'quad9', label: 'Quad9' },
  { id: 'custom', label: '自定义' },
];
const DNS_PROTOCOLS = [
  { id: 'doh', label: 'DoH', hint: 'https://dns.example.com/dns-query' },
  { id: 'dot', label: 'DoT', hint: 'tls://dns.example.com 或 1.1.1.1:853' },
  { id: 'udp', label: 'UDP', hint: '223.5.5.5 或 1.1.1.1:53' },
];
const DNS_DETOURS = [
  { id: 'direct', label: '直连' },
  { id: 'route_default', label: '跟随路由出口' },
  { id: 'egress', label: '指定出口' },
];

function dnsProviderLabel(value) {
  return DNS_PROVIDERS.find((provider) => provider.id === value)?.label || value || '未指定';
}

function dnsDetourLabel(server) {
  return DNS_DETOURS.find((detour) => detour.id === server?.detour)?.label || server?.detour || '跟随路由出口';
}

function dnsEgressLabel(ref) {
  if (!ref?.id) return '未选择出口';
  if (ref.kind === 'group') return groupById(ref.id)?.name || ref.id;
  if (ref.kind === 'chain') return state.chains.find((chain) => chain.id === ref.id)?.name || ref.id;
  if (ref.kind === 'node') return byId(ref.id)?.name || ref.id;
  return ref.id;
}

function dnsAddressHint(protocol = inputValue('form-dns-protocol')) {
  protocol = protocol || 'doh';
  return DNS_PROTOCOLS.find((item) => item.id === protocol)?.hint || '';
}

function syncDNSEditorFields() {
  const followField = document.getElementById('form-dns-follow');
  const core = document.getElementById('dns-editor-core');
  const egress = document.getElementById('form-dns-egress-field');
  const detour = inputValue('form-dns-detour');
  const follow = Boolean(followField) && directFlag('form-dns-follow');
  if (core) core.hidden = follow;
  if (egress) egress.hidden = detour !== 'egress';
  const hint = document.getElementById('form-dns-address-hint');
  if (hint) hint.textContent = dnsAddressHint();
  const address = document.getElementById('form-dns-address');
  if (address) address.placeholder = dnsAddressHint();
}

function openDNSEditor(key) {
  const server = state.dnsProfile[key];
  if (!server) return;
  const follow = key === 'default' && Boolean(server.follow_foreign);
  const egressValue = server.egress?.kind && server.egress?.id ? `${server.egress.kind}:${server.egress.id}` : '';
  const targetChoices = targetOptions(egressValue) || '<option value="">暂无可选节点、策略组或链路</option>';
  openModal({
    title: `编辑${DNS_SLOT_LABELS[key] || key}`,
    description: '修改服务商、协议、地址、出口与超时设置',
    body: `
      <div class="form-grid">
        <label class="field"><span>槽位状态</span><select class="select" id="form-dns-enabled"><option value="true" ${server.enabled ? 'selected' : ''}>启用</option><option value="false" ${server.enabled ? '' : 'selected'}>停用</option></select></label>
        ${key === 'default' ? `<label class="field"><span>解析来源</span><select class="select" id="form-dns-follow"><option value="false" ${follow ? '' : 'selected'}>独立 DNS 服务器</option><option value="true" ${follow ? 'selected' : ''}>沿用国外 DNS</option></select></label>` : ''}
      </div>
      <div class="dns-editor-core" id="dns-editor-core">
        <div class="form-grid">
          <label class="field"><span>服务商</span><select class="select" id="form-dns-provider">${DNS_PROVIDERS.map((provider) => `<option value="${provider.id}" ${(server.provider || 'custom') === provider.id ? 'selected' : ''}>${provider.label}</option>`).join('')}</select></label>
          <label class="field"><span>协议</span><select class="select" id="form-dns-protocol">${DNS_PROTOCOLS.map((protocol) => `<option value="${protocol.id}" ${(server.protocol || 'doh') === protocol.id ? 'selected' : ''}>${protocol.label}</option>`).join('')}</select></label>
          <label class="field full"><span>服务器地址</span><input class="input" id="form-dns-address" value="${escapeHtml(server.address || '')}" placeholder="${escapeHtml(dnsAddressHint(server.protocol))}" spellcheck="false" autocomplete="off"><small class="field-hint" id="form-dns-address-hint">${escapeHtml(dnsAddressHint(server.protocol))}</small></label>
          <label class="field"><span>出口方式</span><select class="select" id="form-dns-detour">${DNS_DETOURS.map((detour) => `<option value="${detour.id}" ${(server.detour || 'route_default') === detour.id ? 'selected' : ''}>${detour.label}</option>`).join('')}</select></label>
          <label class="field"><span>超时（秒）</span><input class="input" id="form-dns-timeout" type="number" min="1" max="60" value="${escapeHtml(server.timeout_seconds || 5)}"></label>
          <label class="field full" id="form-dns-egress-field" ${server.detour === 'egress' ? '' : 'hidden'}><span>指定出口</span><select class="select" id="form-dns-egress">${targetChoices}</select><small class="field-hint">可选择一个节点、策略组或多跳链路作为 DNS 查询出口。</small></label>
        </div>
        <div class="modal-callout" style="margin-top:13px">${icon('shield')}<span>保存时会执行控制面校验；地址必须符合所选协议，指定出口必须引用已存在的节点、策略组或链路。</span></div>
      </div>`,
    footer: `<button class="btn" data-action="modal-close">取消</button><button class="btn btn-primary" data-action="save-dns-slot" data-key="${escapeHtml(key)}">${icon('check')}保存修改</button>`,
  });
  syncDNSEditorFields();
}

function dnsCard(slot) {
  const enabled = slot.enabled;
  const config = state.dnsProfile[slot.key] || {};
  const detour = dnsDetourLabel(config);
  const egress = config.detour === 'egress' ? dnsEgressLabel(config.egress) : '';
  const probeLabel = slot.error
    ? `<span class="tag is-rose" title="${escapeHtml(slot.error)}">探测失败</span>`
    : slot.skipped
      ? `<span class="tag is-solid" title="${escapeHtml(slot.reason || '已跳过')}">${escapeHtml(slot.reason || '已跳过')}</span>`
      : slot.latency > 0
        ? `<span class="dns-latency" title="${escapeHtml(slot.tested_at ? `探测于 ${formatDateTime(slot.tested_at)}` : '真实主动探测')}">${slot.latency} ms</span>`
        : '<span class="dns-latency">未测速</span>';
  return `
    <div class="dns-card ${enabled ? '' : 'is-disabled'}">
      <div class="dns-head"><strong>${escapeHtml(slot.slot)}</strong><div class="dns-head-actions"><button class="btn btn-sm dns-edit" data-action="edit-dns-slot" data-key="${escapeHtml(slot.key)}" aria-label="编辑 ${escapeHtml(slot.slot)}">${icon('sliders')}编辑</button><button class="switch switch-sm ${enabled ? 'is-on' : ''}" data-action="toggle-dns-slot" data-key="${escapeHtml(slot.key)}" role="switch" aria-checked="${enabled}" aria-label="切换 ${escapeHtml(slot.slot)}"></button></div></div>
      <div class="dns-provider">${icon('globe')}${escapeHtml(dnsProviderLabel(config.provider || slot.provider))} <span class="tag">${escapeHtml(slot.protocol)}</span></div>
      <div class="dns-address" title="${escapeHtml(slot.address)}">${escapeHtml(slot.address || '未配置地址')}</div>
      <div class="dns-foot"><span title="${escapeHtml(egress || detour)}">${escapeHtml(egress ? `${detour} · ${egress}` : detour)}</span>${probeLabel}</div>
    </div>
  `;
}

function settingsView() {
  const kernelState = String(state.kernel.state || 'stopped');
  const kernelStateLabel = KERNEL_STATE_LABELS[kernelState] || kernelState;
  const kernelRunning = kernelState === 'running';
  const kernelStateDot = kernelState === 'failed' ? 'is-error' : ['starting', 'stopping'].includes(kernelState) ? 'is-warn' : kernelRunning ? 'is-online' : 'is-idle';
  const kernelTagTone = kernelRunning ? 'is-mint' : kernelState === 'failed' ? 'is-rose' : 'is-solid';
  const initialized = Boolean(state.auth.initialized);
  const authenticated = Boolean(state.auth.authenticated);
  const authLabel = !initialized ? '未初始化' : authenticated ? '会话有效' : '未认证';
  const authTone = authenticated ? 'is-mint' : 'is-amber';
  const modeLabel = state.auth.mode === 'password' ? '本地管理员口令 + 24 小时会话' : '本地模式';
  const sessionExpiry = state.auth.expiresAt ? formatDateTime(state.auth.expiresAt) : '—';
  const credentialUpdatedAt = state.auth.credentialUpdatedAt ? formatDateTime(state.auth.credentialUpdatedAt) : '—';
  return `
    <section class="page">
      ${pageHead({
        eyebrow: 'CONTROL PLANE SETTINGS',
        title: '设置',
        description: '管理访问安全、DNS 与外观；内核生命周期统一在总览页操作。',
        actions: `<button class="btn" data-action="refresh-overview">${icon('refresh')}刷新状态</button>`,
      })}
      <div class="settings-grid">
        <article class="card settings-card wide settings-security-card">
          <div class="card-head"><div class="card-head-copy"><h2>访问安全</h2><p>本地管理员与 API 访问控制</p></div><span class="tag is-mint">${icon('shield')}${initialized ? '已初始化' : '待初始化'}</span></div>
          <div class="settings-security-grid">
            <div class="setting-row"><div class="setting-copy"><strong>认证模式</strong><span>${escapeHtml(modeLabel)}</span></div><span class="tag ${authTone}">${icon('lock')}${escapeHtml(authLabel)}</span></div>
            <div class="setting-row"><div class="setting-copy"><strong>管理员口令</strong><span>${initialized ? `最近轮换于 ${escapeHtml(credentialUpdatedAt)}` : '尚未初始化管理员口令'}</span></div><button class="btn btn-sm" data-action="rotate-admin-password" ${initialized ? '' : 'disabled'}>${icon('refresh')}随机轮换</button></div>
            <div class="setting-row"><div class="setting-copy"><strong>API 认证</strong><span>除健康检查、认证状态和登录外，全部 API 需要会话</span></div><span class="tag is-solid">强制</span></div>
            <div class="setting-row"><div class="setting-copy"><strong>写请求来源校验</strong><span>Cookie 会话的写操作要求同源请求</span></div><span class="tag is-solid">启用</span></div>
            <div class="setting-row"><div class="setting-copy"><strong>当前会话</strong><span>${authenticated ? `有效期至 ${escapeHtml(sessionExpiry)}` : '当前没有有效会话'}</span></div><button class="btn btn-sm" data-action="logout">${icon('power')}退出登录</button></div>
          </div>
        </article>
        <article class="card settings-card wide">
          <div class="card-head"><div class="card-head-copy"><h2>DNS 配置</h2><p>国内、国外与默认三个逻辑槽位</p></div><div class="card-actions"><span class="tag is-sky">${escapeHtml(state.dnsProfile.strategy || '—')}</span><button class="switch switch-sm ${state.dnsProfile.cache ? 'is-on' : ''}" data-action="toggle-dns-cache" role="switch" aria-checked="${Boolean(state.dnsProfile.cache)}" aria-label="DNS 缓存"></button><button class="btn btn-sm" data-action="test-dns">${icon('zap')}DNS 测速</button><button class="btn btn-sm" data-action="flush-dns">${icon('refresh')}刷新缓存</button></div></div>
          <div class="dns-grid">${(state.dnsProfile.slots || []).map(dnsCard).join('')}</div>
          <div class="setting-row" style="margin-top:5px"><div class="setting-copy"><strong>DNS 解析策略</strong><span>控制 A/AAAA 记录的偏好顺序</span></div><select class="select small" data-setting="dns-strategy" aria-label="DNS 解析策略">
            ${['auto', 'prefer_ipv4', 'prefer_ipv6'].map((value) => `<option value="${value}" ${state.dnsProfile.strategy === value ? 'selected' : ''}>${value}</option>`).join('')}
          </select></div>
          <div class="modal-callout" style="margin-top:12px">${icon('zap')}<span>“DNS 测速”由控制面主机向各已启用槽位发起真实查询，记录完整网络往返耗时；未执行时显示“未测速”，失败时显示真实错误。</span></div>
        </article>
        <article class="card settings-card wide">
          <div class="card-head"><div class="card-head-copy"><h2>关于控制面</h2><p>控制面版本、访问方式与内核运行时概况</p></div><span class="tag ${state.sseConnected ? 'is-mint' : 'is-amber'}" title="${state.sseConnected ? 'SSE 实时事件流已连接' : 'SSE 实时事件流正在重连'}"><span class="status-dot ${state.sseConnected ? 'is-online' : 'is-warn'}"></span>${state.sseConnected ? '事件流 LIVE' : '事件流重连中'}</span></div>
          <div class="three-grid">
            <div class="info-block"><span class="tag is-mint">VERSION</span><strong>Boxd v2 Control Plane</strong><p>Web 控制台通过同源 HTTP API 与 SSE 接入控制面门面。</p></div>
            <div class="info-block"><span class="tag is-sky">API</span><strong>REST · SSE</strong><p>REST 提供配置与状态读写，SSE 持续推送日志、流量和运行时事件。</p></div>
            <div class="info-block"><span class="tag is-violet">ACCESS</span><strong>Local Control</strong><p>Web 端仅调用本机控制面 API，不直接暴露内核控制端口，也不提供 Shell 或终端执行。</p></div>
          </div>
          <section class="about-kernel-panel" aria-label="内核基本信息">
            <div class="about-panel-head">
              <span class="about-panel-icon">${icon('cpu')}</span>
              <div class="about-panel-copy"><strong>内核运行时</strong><span>生命周期由总览页统一管理，此处仅展示基本状态</span></div>
              <span class="tag ${kernelTagTone}"><span class="status-dot ${kernelStateDot}"></span>${escapeHtml(kernelStateLabel)}</span>
            </div>
            <div class="about-kernel-grid">
              <div class="system-list">
                <div class="system-row"><span>运行时版本</span><strong>${escapeHtml(kernelVersionLabel())}</strong></div>
                <div class="system-row"><span>配置 revision</span><code>${escapeHtml(shortRevision(state.kernel.revision || '—'))}</code></div>
                <div class="system-row"><span>启动时间</span><strong>${escapeHtml(state.kernel.startedAt || '—')}</strong></div>
              </div>
              <div class="system-list">
                <div class="system-row"><span>运行时长</span><strong>${escapeHtml(state.kernel.uptime || '—')}</strong></div>
                <div class="system-row"><span>路由模式</span><strong>${escapeHtml(String(state.routeMode || 'rules').toUpperCase())}</strong></div>
                <div class="system-row"><span>生命周期入口</span><strong>总览页</strong></div>
              </div>
            </div>
          </section>
        </article>
        <article class="card settings-card wide settings-preferences-card">
          <div class="card-head"><div class="card-head-copy"><h2>外观</h2><p>控制台主题与显示密度</p></div></div>
          <div class="settings-preferences-grid">
            <div class="settings-preference-group">
              <span class="settings-group-label">主题模式</span>
              <div class="theme-options">
                <button class="theme-option ${state.themeChoice === 'light' ? 'is-active' : ''}" data-action="set-theme-choice" data-theme-value="light"><span class="theme-swatch light"></span>浅色</button>
                <button class="theme-option ${state.themeChoice === 'dark' ? 'is-active' : ''}" data-action="set-theme-choice" data-theme-value="dark"><span class="theme-swatch"></span>深色</button>
                <button class="theme-option ${state.themeChoice === 'system' ? 'is-active' : ''}" data-action="set-theme-choice" data-theme-value="system"><span class="theme-swatch system"></span>跟随系统</button>
              </div>
            </div>
            <div class="settings-preference-group">
              <span class="settings-group-label">显示密度</span>
              <div class="setting-row"><div class="setting-copy"><strong>紧凑表格</strong><span>减少表格行高以显示更多数据</span></div><button class="switch switch-sm ${state.flags.compactTable ? 'is-on' : ''}" data-action="toggle-setting" data-key="compactTable" role="switch" aria-checked="${Boolean(state.flags.compactTable)}" aria-label="紧凑表格"></button></div>
              <div class="form-hint">主题偏好保存在当前浏览器；紧凑表格会写入控制面设置。</div>
            </div>
          </div>
        </article>
      </div>
    </section>
  `;
}

function openModal({ title, description = '', body = '', footer = '' }) {
  modalRoot.hidden = false;
  modalRoot.innerHTML = `
    <div class="modal" role="dialog" aria-modal="true" aria-label="${escapeHtml(title)}">
      <div class="modal-head">
        <div><h2>${escapeHtml(title)}</h2>${description ? `<p>${escapeHtml(description)}</p>` : ''}</div>
        <button class="icon-button" data-action="modal-close" aria-label="关闭">${icon('close')}</button>
      </div>
      <div class="modal-body">${body}</div>
      ${footer ? `<div class="modal-footer">${footer}</div>` : ''}
    </div>
  `;
  const focusTarget = modalRoot.querySelector('input, select, textarea, button');
  if (focusTarget) window.setTimeout(() => focusTarget.focus(), 30);
}

function closeModal() {
  modalRoot.hidden = true;
  modalRoot.innerHTML = '';
}

const NODE_LINK_PROTOCOLS = {
  ss: 'shadowsocks',
  vmess: 'vmess',
  vless: 'vless',
  trojan: 'trojan',
  hysteria2: 'hysteria2',
  hy2: 'hysteria2',
  socks: 'socks',
  socks5: 'socks',
  http: 'http',
  https: 'http',
};

const NODE_DEFAULT_PORTS = {
  shadowsocks: 8388,
  vmess: 443,
  vless: 443,
  trojan: 443,
  hysteria2: 443,
  socks: 1080,
  http: 8080,
};

const NODE_PROTOCOL_LABELS = {
  shadowsocks: 'SS',
  vmess: 'VMess',
  vless: 'VLESS',
  trojan: 'Trojan',
  hysteria2: 'HY2',
  socks: 'SOCKS',
  http: 'HTTP',
};

const NODE_TEST_METHODS = [
  { id: 'url', label: 'URL Test', hint: '经代理请求 HTTP 204' },
  { id: 'tcp', label: 'TCP Ping', hint: '连接节点服务器端口' },
  { id: 'icmp', label: 'ICMP Ping', hint: '主机网络 ICMP Echo' },
];

function nodeTestMethod(kind = state.nodeTestKind) {
  return NODE_TEST_METHODS.find((method) => method.id === kind) || NODE_TEST_METHODS[0];
}

function decodeBase64Text(value) {
  try {
    const normalized = String(value).replace(/-/g, '+').replace(/_/g, '/').replace(/\s+/g, '');
    const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4);
    const binary = atob(padded);
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return '';
  }
}

function decodeLinkName(hash) {
  const raw = String(hash || '').replace(/^#/, '');
  if (!raw) return '';
  try {
    return decodeURIComponent(raw).trim();
  } catch {
    return raw.trim();
  }
}

function parseHostPort(value, fallbackPort = 0) {
  const authority = String(value || '').trim().replace(/^\/+|\/+$/g, '');
  if (!authority) return null;
  let server;
  let portText = '';
  if (authority.startsWith('[')) {
    const closeIndex = authority.indexOf(']');
    if (closeIndex < 0) return null;
    server = authority.slice(1, closeIndex);
    portText = authority.slice(closeIndex + 1).replace(/^:/, '');
  } else if (authority.indexOf(':') === authority.lastIndexOf(':')) {
    const colonIndex = authority.lastIndexOf(':');
    server = colonIndex >= 0 ? authority.slice(0, colonIndex) : authority;
    portText = colonIndex >= 0 ? authority.slice(colonIndex + 1) : '';
  } else {
    server = authority;
  }
  const port = Number(portText) || fallbackPort;
  if (!server || !Number.isInteger(port) || port < 1 || port > 65535) return null;
  return { server, port };
}

function parseShadowsocksLink(link) {
  const body = link.slice(link.indexOf('://') + 3);
  const hashIndex = body.indexOf('#');
  const fragment = hashIndex >= 0 ? body.slice(hashIndex + 1) : '';
  let authority = hashIndex >= 0 ? body.slice(0, hashIndex) : body;
  const queryIndex = authority.indexOf('?');
  if (queryIndex >= 0) authority = authority.slice(0, queryIndex);
  let hostPort;
  const atIndex = authority.lastIndexOf('@');
  if (atIndex >= 0) {
    hostPort = authority.slice(atIndex + 1);
  } else {
    const decoded = decodeBase64Text(authority);
    const decodedAt = decoded.lastIndexOf('@');
    if (decodedAt < 0) return null;
    hostPort = decoded.slice(decodedAt + 1);
  }
  const endpoint = parseHostPort(hostPort, NODE_DEFAULT_PORTS.shadowsocks);
  if (!endpoint) return null;
  return {
    protocol: 'shadowsocks',
    server: endpoint.server,
    port: endpoint.port,
    name: decodeLinkName(fragment),
  };
}

function parseVmessLink(link) {
  const body = link.slice(link.indexOf('://') + 3).split('#')[0].trim();
  if (!body) return null;
  let payload = body;
  try {
    payload = decodeURIComponent(payload);
  } catch {
    payload = body;
  }
  const decoded = decodeBase64Text(payload);
  if (!decoded) return null;
  let data;
  try {
    data = JSON.parse(decoded);
  } catch {
    return null;
  }
  const server = String(data.add || '').trim();
  const port = Number(data.port);
  if (!server || !Number.isInteger(port) || port < 1 || port > 65535) return null;
  return {
    protocol: 'vmess',
    server,
    port,
    name: String(data.ps || '').trim(),
  };
}

function parseNodeLink(link) {
  const raw = String(link || '').trim();
  const separator = raw.indexOf('://');
  if (separator <= 0) return null;
  const scheme = raw.slice(0, separator).toLowerCase();
  const protocol = NODE_LINK_PROTOCOLS[scheme];
  if (!protocol) return null;
  if (scheme === 'ss') return parseShadowsocksLink(raw);
  if (scheme === 'vmess') return parseVmessLink(raw);
  let parsed;
  try {
    parsed = new URL(raw);
  } catch {
    return null;
  }
  const server = parsed.hostname.replace(/^\[|\]$/g, '').trim();
  const schemePort = scheme === 'https' ? 443 : scheme === 'http' ? 80 : 0;
  const port = Number(parsed.port) || schemePort || NODE_DEFAULT_PORTS[protocol] || 0;
  if (!server || !port) return null;
  return {
    protocol,
    server,
    port,
    name: decodeLinkName(parsed.hash),
  };
}

function protocolLabel(protocol) {
  return NODE_PROTOCOL_LABELS[protocol] || String(protocol || '').toUpperCase();
}

function renderNodeImportPreview(value) {
  const links = String(value || '').split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
  if (!links.length) {
    return `<div class="node-import-empty">${icon('link')}<span>粘贴分享链接后在这里预览解析结果</span></div>`;
  }
  const parsed = links.map(parseNodeLink);
  const valid = parsed.filter(Boolean);
  const invalid = parsed.length - valid.length;
  if (!valid.length) {
    return `<div class="node-import-empty is-error">${icon('alert')}<span>没有识别到可导入的节点链接，请检查协议与格式</span></div>`;
  }
  return `
    <div class="node-import-preview-head"><span>解析预览</span><span class="tag ${invalid ? 'is-amber' : 'is-mint'}">${valid.length} 个可导入${invalid ? ` · ${invalid} 个无效` : ''}</span></div>
    <div class="node-import-preview-list">
      ${valid.slice(0, 4).map((node) => `
        <div class="node-import-preview-row">
          <span class="node-import-protocol">${escapeHtml(protocolLabel(node.protocol))}</span>
          <span class="node-import-copy"><strong>${escapeHtml(node.name || '未命名节点')}</strong><span>${escapeHtml(node.server)}:${node.port}</span></span>
          <span class="tag is-mint">${icon('check')}可导入</span>
        </div>
      `).join('')}
    </div>
    ${valid.length > 4 ? `<div class="node-import-more">另有 ${valid.length - 4} 个节点将在导入后显示</div>` : ''}
  `;
}

function updateNodeImportPreview(value) {
  const preview = document.getElementById('node-import-preview');
  if (preview) preview.innerHTML = renderNodeImportPreview(value);
}

function openNodeImportModal() {
  openModal({
    title: '导入节点',
    description: '粘贴代理分享链接，单个或多个节点都可以导入',
    body: `
      <div class="modal-callout" style="margin-bottom:13px">${icon('sparkles')}<span>链接会提交到控制面解析并原子写入。单个节点也使用同一导入通道。</span></div>
      <div class="form-grid">
        <label class="field full"><span>节点分享链接</span><textarea class="textarea" id="node-import-links" rows="5" spellcheck="false" aria-label="节点分享链接"></textarea><small class="field-hint">支持 SS / VMess / VLESS / Trojan / Hysteria2 / SOCKS / HTTP，每行一个链接。</small></label>
        <label class="field full"><span>导入后加入策略组</span><select class="select" id="node-import-group"><option value="">暂不加入</option>${state.groups.map((group) => `<option value="${escapeHtml(group.id)}">${escapeHtml(group.name)} · ${escapeHtml(group.description)}</option>`).join('')}</select></label>
      </div>
      <div class="node-import-preview" id="node-import-preview">${renderNodeImportPreview('')}</div>
    `,
    footer: `<button class="btn" data-action="modal-close">取消</button><button class="btn btn-primary" data-action="do-import-node">${icon('link')}导入节点</button>`,
  });
}

async function openNodeEditor(id) {
  let node;
  try {
    node = await api.nodes.get(id);
  } catch (error) {
    if (error.status !== 401) showToast('无法加载节点', error.hint || error.message, 'alert');
    return;
  }
  const protocols = [...new Set(['shadowsocks', 'vmess', 'vless', 'trojan', 'hysteria2', 'socks', 'http', node.protocol].filter(Boolean))];
  const optionsText = JSON.stringify(node.options || {}, null, 2);
  openModal({
    title: `编辑 ${node.name}`,
    description: `${node.protocol} · ${node.server}:${node.port}`,
    body: `
      <input type="hidden" id="form-node-subscription" value="${escapeHtml(node.subscription_id || '')}">
      <div class="form-grid">
        <label class="field full"><span>节点名称</span><input class="input" id="form-node-name" value="${escapeHtml(node.name)}" maxlength="120" required></label>
        <label class="field"><span>协议</span><select class="select" id="form-node-protocol">${protocols.map((protocol) => `<option value="${escapeHtml(protocol)}" ${protocol === node.protocol ? 'selected' : ''}>${escapeHtml(protocolLabel(protocol))}</option>`).join('')}</select></label>
        <label class="field"><span>服务器</span><input class="input" id="form-node-server" value="${escapeHtml(node.server)}" required></label>
        <label class="field"><span>端口</span><input class="input" id="form-node-port" type="number" min="1" max="65535" value="${escapeHtml(node.port)}" required></label>
        <label class="field"><span>UDP 转发</span><select class="select" id="form-node-udp"><option value="true" ${node.supports_udp !== false ? 'selected' : ''}>启用</option><option value="false" ${node.supports_udp === false ? 'selected' : ''}>关闭</option></select></label>
        <label class="field full"><span>协议高级参数（JSON）</span><textarea class="textarea node-options-editor" id="form-node-options" rows="10" spellcheck="false">${escapeHtml(optionsText)}</textarea><small class="field-hint">保存前会校验 JSON。UUID、密码、Reality 公钥等协议参数在此维护。</small></label>
      </div>
      <div class="modal-callout" style="margin-top:13px">${icon('shield')}<span>编辑内容会先写入控制面并重新生成运行时配置。订阅节点在下次订阅刷新时仍可能被订阅源覆盖。</span></div>
    `,
    footer: `<button class="btn" data-action="modal-close">取消</button><button class="btn btn-primary" data-action="save-node" data-id="${escapeHtml(node.id)}">${icon('check')}保存修改</button>`,
  });
}

async function saveNodeEditor(id) {
  const name = String(document.getElementById('form-node-name')?.value || '').trim();
  const protocol = String(document.getElementById('form-node-protocol')?.value || '').trim();
  const server = String(document.getElementById('form-node-server')?.value || '').trim();
  const port = Number(document.getElementById('form-node-port')?.value || 0);
  const udp = document.getElementById('form-node-udp')?.value !== 'false';
  const subscriptionID = String(document.getElementById('form-node-subscription')?.value || '').trim();
  if (!name || !protocol || !server || !Number.isInteger(port) || port < 1 || port > 65535) {
    showToast('节点信息不完整', '名称、协议、服务器和 1-65535 端口都必须填写', 'alert');
    return;
  }
  let options = {};
  const optionsText = String(document.getElementById('form-node-options')?.value || '').trim();
  if (optionsText) {
    try {
      options = JSON.parse(optionsText);
    } catch (error) {
      showToast('高级参数不是有效 JSON', error.message, 'alert');
      return;
    }
    if (!options || Array.isArray(options) || typeof options !== 'object') {
      showToast('高级参数必须是 JSON 对象', '例如 {"uuid":"...","tls":{}}', 'alert');
      return;
    }
  }
  const saved = await mutate(() => api.nodes.update(id, {
    id,
    name,
    protocol,
    server,
    port,
    options,
    supports_udp: udp,
    subscription_id: subscriptionID,
  }), { title: '节点已更新', message: `${name} · ${server}:${port}`, icon: 'check' });
  if (saved) closeModal();
}

async function importNodeLinks() {
  const links = String(document.getElementById('node-import-links')?.value || '')
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
  if (!links.length) {
    showToast('请先粘贴节点链接', '每行输入一个分享链接', 'alert');
    return;
  }
  const groupId = document.getElementById('node-import-group')?.value || '';
  try {
    const result = await api.nodes.importLinks(links, groupId);
    closeModal();
    await refreshSnapshot({ rerender: false });
    state.egressTab = 'nodes';
    state.nodeQuery = '';
    setView('egress', { tab: 'nodes' });
    const groupName = groupId ? groupById(groupId)?.name : '';
    const detail = [
      `${result.added || 0} 新增`,
      `${result.updated || 0} 更新`,
      groupName ? `加入 ${groupName}` : '',
      result.group_warning ? `分组警告：${result.group_warning}` : '',
    ].filter(Boolean).join(' · ');
    showToast('节点导入完成', detail || '控制面已返回导入结果', 'link');
  } catch (error) {
    if (error.status !== 401) showToast('导入失败', error.hint || error.message, 'alert');
  }
}

function openConfirm({ title, description, note, confirmLabel = '确认', action, id = '' }) {
  const destructive = action.startsWith('do-delete') || action.startsWith('do-close');
  openModal({
    title,
    description,
    body: `<div class="modal-callout ${destructive ? 'danger' : ''}">${icon(destructive ? 'trash' : 'alert')}<span>${escapeHtml(note)}</span></div>`,
    footer: `<button class="btn" data-action="modal-close">取消</button><button class="btn ${destructive ? 'btn-danger' : 'btn-primary'}" data-action="${action}" ${id ? `data-id="${escapeHtml(id)}"` : ''}>${escapeHtml(confirmLabel)}</button>`,
  });
}

function openRotatedPasswordModal(password) {
  openModal({
    title: '新管理员口令已生成',
    description: '旧口令与全部旧会话已失效',
    body: `<div class="modal-callout danger">${icon('alert')}<span>这是唯一一次显示新口令。关闭弹窗后，服务端和浏览器都不会再次提供明文。</span></div>
      <div class="rotated-password"><code id="rotated-admin-password" aria-label="新管理员口令">${escapeHtml(password)}</code><button class="icon-button" data-action="copy-rotated-password" aria-label="复制新管理员口令" title="复制新管理员口令">${icon('copy')}</button></div>
      <div class="modal-callout" style="margin-top:12px">${icon('shield')}<span>口令由 32 位随机字符组成，可直接粘贴到密码管理器。不要通过聊天、邮件或截图传播。</span></div>`,
    footer: `<button class="btn btn-primary" data-action="modal-close">${icon('check')}我已安全保存</button>`,
  });
}

function pickerOrderMarkup(ids) {
  if (!ids.length) {
    return '<div class="picker-order-empty">从下方节点列表选择节点，选中后会按顺序加入链路</div>';
  }
  return ids.map((nodeID, index) => {
    const node = byId(nodeID);
    const name = node?.name || nodeID;
    const detail = node ? `${node.protocol} · ${node.server}:${node.port}` : '节点信息缺失';
    return `
      <div class="picker-order-item" data-picker-order-id="${escapeHtml(nodeID)}">
        <span class="picker-order-index">${index + 1}</span>
        <span class="picker-order-copy"><strong>${escapeHtml(name)}</strong><span>${escapeHtml(detail)}</span></span>
        <span class="picker-order-actions">
          <button class="picker-order-btn is-up" type="button" data-picker-move="up" aria-label="上移 ${escapeHtml(name)}" title="上移" ${index === 0 ? 'disabled' : ''}>${icon('chevron-down')}</button>
          <button class="picker-order-btn" type="button" data-picker-move="down" aria-label="下移 ${escapeHtml(name)}" title="下移" ${index === ids.length - 1 ? 'disabled' : ''}>${icon('chevron-down')}</button>
          <button class="picker-order-btn is-remove" type="button" data-picker-remove aria-label="移除 ${escapeHtml(name)}" title="移除">${icon('close')}</button>
        </span>
      </div>
    `;
  }).join('');
}

function nodePicker({ id, label, selected = [], max = 0, hint = '', ordered = false }) {
  const chosen = new Set(selected);
  const count = state.nodes.filter((node) => chosen.has(node.id)).length;
  const orderIDs = ordered ? selected.filter(Boolean) : [];
  const options = state.nodes.map((node) => {
    const checked = chosen.has(node.id);
    const latency = Number(node.latency) > 0 ? `${node.latency} ms` : '';
    return `
      <label class="node-option ${checked ? 'is-selected' : ''}">
        <input type="checkbox" value="${escapeHtml(node.id)}" data-picker-value ${checked ? 'checked' : ''}>
        <span class="node-option-copy"><strong>${escapeHtml(node.name)}</strong><span>${escapeHtml(node.protocol)} · ${escapeHtml(node.server)}:${node.port}</span></span>
        ${latency ? `<span class="node-option-latency">${latency}</span>` : ''}
      </label>
    `;
  }).join('');
  return `
    <div class="field full">
      <span>${escapeHtml(label)}</span>
      <div class="node-picker" id="${escapeHtml(id)}" data-node-picker data-max="${max}" data-picker-ordered="${ordered ? 'true' : 'false'}">
        <div class="node-picker-head"><span>已选 <strong data-picker-count>${count}</strong> 个节点</span><span>${escapeHtml(hint)}</span></div>
        ${ordered ? `<div class="picker-order"><div class="picker-order-head"><span>链路顺序</span><span>流量从上到下依次转发</span></div><div class="picker-order-list" data-picker-order>${pickerOrderMarkup(orderIDs)}</div></div>` : ''}
        <div class="node-picker-list">${options || '<div class="node-picker-empty">暂无可选节点</div>'}</div>
      </div>
    </div>
  `;
}

function pickerOrderIDs(picker) {
  return picker ? [...picker.querySelectorAll('[data-picker-order-id]')].map((element) => element.dataset.pickerOrderId) : [];
}

function pickerOptionInput(picker, nodeID) {
  return picker ? [...picker.querySelectorAll('[data-picker-value]')].find((input) => input.value === nodeID) : null;
}

function updatePickerCount(picker) {
  const countElement = picker?.querySelector('[data-picker-count]');
  if (countElement) countElement.textContent = String(picker.querySelectorAll('[data-picker-value]:checked').length);
}

function renderPickerOrder(picker, ids) {
  const order = picker?.querySelector('[data-picker-order]');
  if (order) order.innerHTML = pickerOrderMarkup(ids);
}

function pickerValues(id) {
  const picker = document.getElementById(id);
  if (picker?.dataset.pickerOrdered === 'true') return pickerOrderIDs(picker);
  return picker ? [...picker.querySelectorAll('[data-picker-value]:checked')].map((input) => input.value) : [];
}

function targetOptions(selected = '') {
  return [
    state.groups.length ? `<optgroup label="策略组">${state.groups.map((group) => `<option value="group:${group.id}" ${selected === `group:${group.id}` ? 'selected' : ''}>${escapeHtml(group.name)}</option>`).join('')}</optgroup>` : '',
    state.chains.length ? `<optgroup label="链路">${state.chains.map((chain) => `<option value="chain:${chain.id}" ${selected === `chain:${chain.id}` ? 'selected' : ''}>${escapeHtml(chain.name)}</option>`).join('')}</optgroup>` : '',
    state.nodes.length ? `<optgroup label="节点">${state.nodes.map((node) => `<option value="node:${node.id}" ${selected === `node:${node.id}` ? 'selected' : ''}>${escapeHtml(node.name)}</option>`).join('')}</optgroup>` : '',
  ].join('');
}

function describeRuleForForm(rule) {
  const fields = [
    ['domain_suffix', 'domain_suffixes'],
    ['domain_keyword', 'domain_keywords'],
    ['ip_cidr', 'ip_cidrs'],
    ['rule_set', 'rule_sets'],
    ['process_name', 'process_names'],
    ['domain', 'domains'],
  ];
  for (const [type, key] of fields) {
    if (Array.isArray(rule?.match?.[key]) && rule.match[key].length) {
      return { type, values: rule.match[key].join('\n') };
    }
  }
  return { type: 'domain_suffix', values: '' };
}

function openInboundConfig(id) {
  const inbound = state.inbounds.find((item) => item.id === id);
  if (!inbound) return;
  const [host, port] = String(inbound.listen || '').startsWith('[')
    ? [inbound.listen.slice(1, inbound.listen.indexOf(']')), inbound.listen.slice(inbound.listen.indexOf(']') + 2)]
    : incomingSplit(inbound.listen);
  openModal({
    title: `${inbound.name} 配置`,
    description: inbound.kind === 'tun' ? 'TUN 入口的高级参数暂由控制面默认值管理' : '保存后需要重启内核才会应用监听变更',
    body: `<div class="form-grid">
      <label class="field full"><span>入口名称</span><input class="input" id="form-name" value="${escapeHtml(inbound.name)}" maxlength="60" required></label>
      <label class="field"><span>启用状态</span><select class="select" id="form-enabled"><option value="true" ${inbound.enabled ? 'selected' : ''}>启用</option><option value="false" ${inbound.enabled ? '' : 'selected'}>停用</option></select></label>
      ${inbound.kind === 'tun' ? `<label class="field"><span>监听地址</span><input class="input" value="${escapeHtml(inbound.listen)}" disabled></label>` : `<label class="field"><span>监听地址</span><input class="input" id="form-host" value="${escapeHtml(host)}" required></label><label class="field"><span>监听端口</span><input class="input" id="form-port" type="number" min="1" max="65535" value="${escapeHtml(port)}" required></label><label class="field"><span>UDP 转发</span><select class="select" id="form-udp"><option value="true" ${inbound.udp ? 'selected' : ''}>启用</option><option value="false" ${inbound.udp ? '' : 'selected'}>关闭</option></select></label>`}
    </div>${inbound.kind === 'tun' ? `<div class="modal-callout" style="margin-top:13px">${icon('alert')}<span>当前 HTTP API 只开放 TUN 入口的启用/停用与名称；地址、MTU 和路由参数不在 Web 层修改。</span></div>` : ''}`,
    footer: `<button class="btn" data-action="modal-close">取消</button><button class="btn btn-primary" data-action="save-inbound" data-id="${escapeHtml(inbound.id)}">保存修改</button>`,
  });
}

function incomingSplit(value) {
  const text = String(value || '');
  const index = text.lastIndexOf(':');
  return index > 0 ? [text.slice(0, index), text.slice(index + 1)] : [text, ''];
}

function openFormModal(kind, id = '') {
  const rule = kind === 'rule' && id ? state.routeRules.find((item) => item.id === id) : null;
  const group = kind === 'group' && id ? groupById(id) : null;
  const chain = kind === 'chain' && id ? state.chains.find((item) => item.id === id) : null;
  const subscription = kind === 'subscription' && id ? state.subscriptions.find((item) => item.id === id) : null;
  const ruleForm = rule ? describeRuleForForm(rule) : { type: 'domain_suffix', values: '' };
  const subscriptionInterval = subscription?.interval || 21600;
  const intervalValues = [...new Set([0, 3600, 21600, 86400, subscriptionInterval])].sort((left, right) => left - right);
  const intervalLabels = { 0: '不自动刷新', 3600: '每 1 小时', 21600: '每 6 小时', 86400: '每天' };
  const intervalOptions = intervalValues.map((value) => `<option value="${value}" ${subscriptionInterval === value ? 'selected' : ''}>${intervalLabels[value] || `每 ${value} 秒`}</option>`).join('');
  const forms = {
    subscription: {
      title: subscription ? '编辑订阅' : '添加订阅', description: subscription ? '修改订阅信息；订阅地址留空时将保留当前地址与令牌' : '从远端订阅源同步节点；令牌会在所有视图中脱敏',
      body: `<div class="form-grid">
        <label class="field full"><span>订阅名称</span><input class="input" id="form-name" value="${escapeHtml(subscription?.name || '')}" placeholder="例如 AirNet Pro" maxlength="80" required></label>
        <label class="field full"><span>${subscription ? '新订阅地址（可选）' : '订阅地址'}</span><input class="input" id="form-url" placeholder="${subscription ? '留空则保持当前订阅地址' : 'https://example.com/sub?token=...'}" ${subscription ? '' : 'required'}>${subscription ? `<small class="field-hint">当前地址：<code>${escapeHtml(subscription.url)}</code>。留空不会覆盖已保存的令牌地址。</small>` : ''}</label>
        <label class="field"><span>刷新间隔</span><select class="select" id="form-interval">${intervalOptions}</select></label>
        <label class="field"><span>${subscription ? '订阅状态' : '首次保存'}</span><select class="select" id="form-enabled"><option value="true" ${subscription?.enabled !== false ? 'selected' : ''}>启用</option><option value="false" ${subscription?.enabled === false ? 'selected' : ''}>暂停</option></select></label>
      </div><div class="modal-callout" style="margin-top:13px">${icon('shield')}<span>${subscription ? '为避免令牌泄露，当前地址只显示脱敏结果。只有填写新地址时才会更新远端订阅源。' : '订阅地址可能包含访问令牌。后端只返回脱敏地址，令牌不会回显到 Web 页面。'}</span></div>`,
    },
    ruleset: {
      title: '新建规则集', description: '创建自定义规则集供路由规则引用',
      body: `<div class="form-grid">
        <label class="field full"><span>名称</span><input class="input" id="form-name" placeholder="例如 ai-domains" pattern="[a-zA-Z0-9._-]+" required></label>
        <label class="field full"><span>来源地址</span><input class="input" id="form-url" placeholder="https://example.com/rules.json" required></label>
        <label class="field"><span>格式</span><select class="select" id="form-format"><option value="source">source</option><option value="binary" selected>binary</option></select></label>
        <label class="field"><span>状态</span><select class="select" id="form-enabled"><option value="true" selected>启用</option><option value="false">停用</option></select></label>
      </div>`,
    },
    inbound: {
      title: '新增 Mixed 入口', description: '创建本机 HTTP / SOCKS5 混合监听入口',
      body: `<div class="form-grid">
        <label class="field full"><span>入口名称</span><input class="input" id="form-name" value="Mixed 2" maxlength="60" required></label>
        <label class="field"><span>监听地址</span><input class="input" id="form-host" value="::" required></label>
        <label class="field"><span>监听端口</span><input class="input" id="form-port" type="number" min="1" max="65535" value="1081" required></label>
        <label class="field full"><span>UDP 转发</span><select class="select" id="form-udp"><option value="true" selected>启用</option><option value="false">关闭</option></select></label>
      </div><div class="modal-callout" style="margin-top:13px">${icon('shield')}<span>入口会监听本机网络接口。保存后需启动内核才会实际建立监听。</span></div>`,
    },
    rule: {
      title: rule ? '编辑路由规则' : '新建路由规则', description: '规则按顺序匹配，命中后使用指定动作',
      body: `<div class="form-grid">
        <label class="field full"><span>规则名称</span><input class="input" id="form-rule-name" value="${escapeHtml(rule?.name || '')}" placeholder="例如 AI 服务" maxlength="80" required></label>
        <label class="field"><span>条件类型</span><select class="select" id="form-rule-type">${[
          ['domain_suffix', '域名后缀'], ['domain_keyword', '域名关键字'], ['domain', '完整域名'], ['ip_cidr', 'IP CIDR'], ['rule_set', '规则集'], ['process_name', '进程名'],
        ].map(([value, label]) => `<option value="${value}" ${ruleForm.type === value ? 'selected' : ''}>${label}</option>`).join('')}</select></label>
        <label class="field"><span>动作</span><select class="select" id="form-rule-action">${[
          ['egress', '转发到出口'], ['direct', '直连'], ['reject', '拒绝'],
        ].map(([value, label]) => `<option value="${value}" ${(rule?.route_action || 'egress') === value ? 'selected' : ''}>${label}</option>`).join('')}</select></label>
        <label class="field full"><span>匹配值</span><textarea class="textarea" id="form-rule-values" rows="4" placeholder="每行一个值，例如 openai.com" required>${escapeHtml(ruleForm.values)}</textarea></label>
        <label class="field full"><span>出口目标</span><select class="select" id="form-rule-target" ${rule?.route_action === 'direct' || rule?.route_action === 'reject' ? 'disabled' : ''}>${targetOptions(rule?.egress?.kind && rule?.egress?.id ? `${rule.egress.kind}:${rule.egress.id}` : '')}</select></label>
      </div>${rule ? `<div class="modal-callout danger" style="margin-top:13px">${icon('alert')}<span>删除规则会立即生成新的运行时配置，正在使用该规则的连接不受影响。</span></div>` : ''}`,
    },
    group: {
      title: group ? '编辑策略组' : '新建策略组', description: 'Selector 手动选择出口，URL Test 按成员自动测速',
      body: `<div class="form-grid">
        <label class="field full"><span>策略组名称</span><input class="input" id="form-name" value="${escapeHtml(group?.name || '')}" placeholder="例如 智能选择" maxlength="60" required></label>
        <label class="field"><span>类型</span><select class="select" id="form-group-kind"><option value="selector" ${(group?.kind || 'selector') === 'selector' ? 'selected' : ''}>Selector</option><option value="urltest" ${group?.kind === 'urltest' ? 'selected' : ''}>URL Test</option></select></label>
        <label class="field"><span>测速间隔（秒）</span><input class="input" id="form-interval" type="number" min="30" value="${group?.interval_seconds || group?.interval || 180}"></label>
        ${nodePicker({ id: 'form-members', label: '成员节点', selected: group?.members || [], hint: '勾选要加入策略组的节点' })}
      </div>`,
    },
    chain: {
      title: chain ? '编辑多跳链路' : '新建多跳链路', description: '按选择顺序串联节点，最多 4 跳',
      body: `<div class="form-grid">
        <label class="field full"><span>链路名称</span><input class="input" id="form-name" value="${escapeHtml(chain?.name || '')}" placeholder="例如 亚太中转" maxlength="60" required></label>
        ${nodePicker({ id: 'form-hops', label: '节点顺序', selected: chain?.hops || [], max: 4, ordered: true, hint: '先勾选节点，再用箭头调整顺序，最多 4 跳' })}
      </div>`,
    },
  };
  const form = forms[kind];
  if (!form) return;
  openModal({
    ...form,
    footer: `<button class="btn" data-action="modal-close">取消</button>${rule ? `<button class="btn btn-danger" data-action="delete-rule" data-id="${escapeHtml(rule.id)}">${icon('trash')}删除</button>` : ''}${group ? `<button class="btn btn-danger" data-action="delete-group" data-id="${escapeHtml(group.id)}">${icon('trash')}删除</button>` : ''}<button class="btn btn-primary" data-action="modal-submit" data-kind="${kind}" data-id="${escapeHtml(id)}">${id ? '保存修改' : '提交'}</button>`,
  });
}

function inputValue(id) {
  return String(document.getElementById(id)?.value || '').trim();
}

function directFlag(id) {
  return inputValue(id) === 'true';
}

async function submitForm(kind, id, button) {
  const run = async () => {
    if (kind === 'subscription') {
      const url = inputValue('form-url');
      const payload = {
        name: inputValue('form-name'),
        interval_seconds: Number(inputValue('form-interval')),
        enabled: directFlag('form-enabled'),
      };
      if (!id) return api.subscriptions.create({ ...payload, url });
      if (url) payload.url = url;
      return api.subscriptions.patch(id, payload);
    }
    if (kind === 'ruleset') {
      return api.rulesets.create({
        name: inputValue('form-name'),
        url: inputValue('form-url'),
        format: inputValue('form-format'),
        enabled: directFlag('form-enabled'),
      });
    }
    if (kind === 'inbound') {
      return api.inbounds.create({
        name: inputValue('form-name'),
        kind: 'mixed',
        enabled: true,
        listen: { host: inputValue('form-host'), port: Number(inputValue('form-port')) },
        udp: directFlag('form-udp'),
      });
    }
    if (kind === 'group') {
      const members = pickerValues('form-members');
      const existing = groupById(id);
      const selected = existing?.selected && members.includes(existing.selected) ? existing.selected : members[0] || '';
      return api.groups.save({
        id,
        name: inputValue('form-name'),
        kind: inputValue('form-group-kind'),
        members,
        selected,
        url: existing?.url || '',
        interval_seconds: Number(inputValue('form-interval')) || 0,
      });
    }
    if (kind === 'chain') {
      const hops = pickerValues('form-hops').map((nodeId) => ({ kind: 'node', id: nodeId }));
      return api.chains.save({ id, name: inputValue('form-name'), hops });
    }
    if (kind === 'rule') {
      const type = inputValue('form-rule-type');
      const values = inputValue('form-rule-values').split(/\r?\n|,/).map((value) => value.trim()).filter(Boolean);
      const match = {};
      const matchKey = {
        domain_suffix: 'domain_suffixes', domain_keyword: 'domain_keywords', domain: 'domains',
        ip_cidr: 'ip_cidrs', rule_set: 'rule_sets', process_name: 'process_names',
      }[type];
      match[matchKey] = values;
      const routeAction = inputValue('form-rule-action');
      const target = inputValue('form-rule-target');
      const [targetKind, targetID] = target.split(':');
      return api.routes.saveRule({
        id,
        name: inputValue('form-rule-name'),
        match,
        action: routeAction,
        egress: routeAction === 'egress' && targetKind && targetID ? { kind: targetKind, id: targetID } : { kind: routeAction, id: '' },
      });
    }
    return null;
  };
  if (kind === 'group' && pickerValues('form-members').length === 0) {
    showToast('无法保存策略组', '至少选择一个成员节点', 'alert');
    return;
  }
  if (kind === 'chain' && pickerValues('form-hops').length === 0) {
    showToast('无法保存链路', '至少选择一个节点', 'alert');
    return;
  }
  if (button) button.disabled = true;
  try {
    await run();
    closeModal();
    await refreshSnapshot({ rerender: false });
    render(true);
    showToast('已写入控制面', '配置已持久化并应用到运行时');
  } catch (error) {
    if (error.status !== 401) showToast('提交失败', error.hint || error.message, 'alert');
  } finally {
    if (button) button.disabled = false;
  }
}

function openAbout() {
  openModal({
    title: 'Boxd Control Plane',
    description: '控制面 Web 接入信息',
    body: `
      <div class="about-strip" style="margin-bottom:13px"><div><strong>当前范围：控制平面及以下能力</strong><span>Web 控制台已接入真实控制面；桌面应用等上层宿主暂不启用</span></div><span class="tag ${state.sseConnected ? 'is-mint' : 'is-amber'}">${state.sseConnected ? '事件流 LIVE' : '事件流重连中'}</span></div>
      <div class="system-list">
        <div class="system-row"><span>前端</span><strong>原生 HTML / CSS / ES Module</strong></div>
        <div class="system-row"><span>API</span><strong>同源 REST · /api/v1</strong></div>
        <div class="system-row"><span>实时事件</span><strong>SSE · /api/v1/events</strong></div>
        <div class="system-row"><span>后端运行时</span><strong>${escapeHtml(kernelVersionLabel())}</strong></div>
        <div class="system-row"><span>配置 revision</span><code>${escapeHtml(shortRevision(state.kernel.revision))}</code></div>
        <div class="system-row"><span>访问地址</span><strong>${escapeHtml(location.host)}</strong></div>
      </div>
    `,
    footer: `<button class="btn btn-primary" data-action="modal-close">关闭</button>`,
  });
}

function openNotifications() {
  const entries = state.logs.filter((item) => item.level === 'warn' || item.level === 'error').slice(-8).reverse();
  openModal({
    title: '最近告警',
    description: '来自应用日志与内核日志的 WARN / ERROR 事件',
    body: entries.length ? `<div class="event-list">${entries.map((item) => `
      <div class="event-line"><span class="event-time">${escapeHtml(item.time)}</span><span class="event-dot ${item.level === 'error' ? 'error' : 'warn'}"></span><span><strong>${escapeHtml(item.channel.toUpperCase())}</strong> · ${escapeHtml(item.message)}</span></div>
    `).join('')}</div>` : `<div class="empty-state"><span class="empty-icon">${icon('check')}</span><strong>当前没有告警</strong><span>新的 WARN / ERROR 日志会实时出现在这里</span></div>`,
    footer: `<button class="btn" data-action="modal-close">关闭</button><button class="btn btn-primary" data-view="logs">${icon('terminal')}打开日志</button>`,
  });
}

function updateNodeRows() {
  const body = document.getElementById('nodes-body');
  if (body) body.innerHTML = nodeRows();
}

function updateRulesetRows() {
  const body = document.getElementById('rulesets-body');
  if (body) body.innerHTML = rulesetRows();
}

function updateConnectionRows() {
  const body = document.getElementById('connections-body');
  if (body) body.innerHTML = connectionRows();
  const count = document.getElementById('conn-count');
  if (count) count.textContent = state.traffic.available ? `${state.connections.length} 条连接` : '内核未运行';
}

function scrollLogConsole() {
  const consoleElement = document.getElementById('logs-console');
  if (consoleElement) consoleElement.scrollTop = consoleElement.scrollHeight;
}

function updateLogRuntimeMetrics() {
  const stats = logStats();
  const filtered = filteredLogs();
  const values = {
    'log-app-count': stats.app,
    'log-kernel-count': stats.kernel,
    'log-alert-count': stats.warn + stats.error,
    'log-source-app-count': stats.app,
    'log-source-kernel-count': stats.kernel,
    'log-match-count': `显示 ${filtered.length} / ${stats.total} 行`,
    'log-buffer-count': `缓冲区 ${stats.total} / 500`,
  };
  Object.entries(values).forEach(([id, value]) => {
    const element = document.getElementById(id);
    if (element) element.textContent = value;
  });
}

function updateLogConsole() {
  const consoleElement = document.getElementById('logs-console');
  if (!consoleElement) return;
  consoleElement.innerHTML = logLines();
  updateLogRuntimeMetrics();
  if (state.logAutoScroll) scrollLogConsole();
}

async function mutate(action, success = null) {
  try {
    const result = await action();
    await refreshSnapshot({ rerender: false });
    render(true);
    if (success) showToast(success.title, success.message || '', success.icon || 'check');
    return result;
  } catch (error) {
    if (error.status !== 401) showToast('操作失败', error.hint || error.message, 'alert');
    return null;
  }
}

async function measureNode(id) {
  const node = byId(id);
  if (!node) return;
  await measureNodes([id], state.nodeTestKind);
}

async function measureAllNodes() {
  if (!state.nodes.length) {
    showToast('没有可测速的节点', '请先导入节点', 'alert');
    return;
  }
  await measureNodes(state.nodes.map((node) => node.id), state.nodeTestKind);
}

async function measureSelectedNodes() {
  const ids = [...state.selectedNodeIDs];
  if (!ids.length) {
    showToast('请先选择节点', '使用表格左侧复选框选择一个或多个节点', 'alert');
    return;
  }
  await measureNodes(ids, state.nodeTestKind);
}

async function measureNodes(ids, kind = state.nodeTestKind) {
  const uniqueIDs = [...new Set(ids.filter(Boolean))];
  if (!uniqueIDs.length) return;
  const method = nodeTestMethod(kind);
  if (kind === 'url' && state.kernel.state !== 'running') {
    showToast('URL Test 需要先启动内核', 'ICMP Ping 与 TCP Ping 不依赖内核运行状态', 'alert');
    return;
  }
  if (state.nodeTesting) {
    showToast('测速正在进行', '请等待当前批次完成', 'clock');
    return;
  }
  state.nodeTesting = true;
  render(true);
  let succeeded = 0;
  let failed = 0;
  try {
    for (let offset = 0; offset < uniqueIDs.length; offset += 100) {
      const chunk = uniqueIDs.slice(offset, offset + 100);
      const payload = await api.nodes.test(chunk, kind, kind === 'url' ? state.nodeTestURL.trim() : '');
      for (const result of payload.results || []) {
        const node = byId(result.node_id);
        if (!node) continue;
        node.lastTestKind = result.kind;
        if (result.error) {
          failed += 1;
          node.latency = 0;
          node.lastTestError = result.error;
        } else {
          succeeded += 1;
          node.latency = Number(result.latency_ms) || 0;
          node.lastTestError = '';
        }
      }
    }
    render(true);
    showToast(`${method.label} 完成`, failed ? `${succeeded} 成功 · ${failed} 失败` : `${succeeded} 个节点全部可达`, failed ? 'alert' : 'check');
  } catch (error) {
    if (error.status !== 401) showToast(`${method.label} 失败`, error.hint || error.message, 'alert');
  } finally {
    state.nodeTesting = false;
    render(true);
  }
}

function currentDNSProfile() {
  return {
    domestic: { ...(state.dnsProfile.domestic || {}) },
    foreign: { ...(state.dnsProfile.foreign || {}) },
    default: { ...(state.dnsProfile.default || {}) },
    strategy: state.dnsProfile.strategy,
    cache: Boolean(state.dnsProfile.cache),
  };
}

async function saveDNSSlot(key, button) {
  const profile = currentDNSProfile();
  const server = profile[key];
  if (!server) return;
  const followForeign = key === 'default' && directFlag('form-dns-follow');
  if (followForeign && !profile.foreign.enabled) {
    showToast('无法保存默认 DNS', '沿用国外 DNS 前需要先启用国外 DNS 槽位', 'alert');
    return;
  }
  const detour = inputValue('form-dns-detour') || 'route_default';
  const next = {
    ...server,
    enabled: directFlag('form-dns-enabled'),
    provider: inputValue('form-dns-provider') || 'custom',
    protocol: inputValue('form-dns-protocol') || 'doh',
    address: inputValue('form-dns-address'),
    detour,
    timeout_seconds: Number(inputValue('form-dns-timeout')) || 5,
    follow_foreign: followForeign,
  };
  if (followForeign) {
    delete next.egress;
  } else if (detour === 'egress') {
    const [kind, id] = inputValue('form-dns-egress').split(':');
    if (!kind || !id) {
      showToast('无法保存 DNS 出口', '请选择一个节点、策略组或链路', 'alert');
      return;
    }
    next.egress = { kind, id };
  } else {
    delete next.egress;
  }
  if (button) button.disabled = true;
  try {
    const saved = await mutate(() => api.dns.patch({ [key]: next }), {
      title: `${DNS_SLOT_LABELS[key] || key} 已更新`,
      message: '配置已持久化到控制面',
      icon: 'globe',
    });
    if (saved) closeModal();
  } finally {
    if (button) button.disabled = false;
  }
}

async function closeConnection(id) {
  const connection = state.connections.find((item) => item.id === id);
  const numericID = connection?.numeric_id ?? Number(id);
  if (!Number.isSafeInteger(numericID) || numericID <= 0) {
    showToast('连接 ID 无效', '请刷新连接列表后重试', 'alert');
    return;
  }
  await mutate(() => api.connections.close(String(numericID)), {
    title: '连接已关闭',
    message: connection?.target || String(numericID),
    icon: 'stop',
  });
}

async function handleAction(actionEl) {
  const action = actionEl.dataset.action;
  const id = actionEl.dataset.id;
  switch (action) {
    case 'toggle-sidebar': {
      const open = document.body.classList.toggle('sidebar-open');
      document.getElementById('sidebar-backdrop').hidden = !open;
      break;
    }
    case 'toggle-theme':
      toggleTheme();
      break;
    case 'set-theme-choice':
      applyTheme(actionEl.dataset.themeValue);
      render(true);
      break;
    case 'notifications':
      openNotifications();
      break;
    case 'about':
      openAbout();
      break;
    case 'profile':
      openModal({
        title: '本地管理员会话',
        description: '控制面认证状态',
        body: `<div class="system-list">
          <div class="system-row"><span>访问地址</span><code>${escapeHtml(location.host)}</code></div>
          <div class="system-row"><span>认证方式</span><strong>本地管理员口令</strong></div>
          <div class="system-row"><span>会话状态</span><strong>${state.auth.authenticated ? '会话有效' : state.auth.initialized ? '未认证' : '未初始化'}</strong></div>
          <div class="system-row"><span>会话有效期</span><strong>${state.auth.expiresAt ? formatDateTime(state.auth.expiresAt) : '—'}</strong></div>
          <div class="system-row"><span>SSE</span><strong>${state.sseConnected ? '已连接' : '重连中'}</strong></div>
          <div class="system-row"><span>最近快照</span><strong>${state.lastSnapshotAt ? formatDateTime(state.lastSnapshotAt) : '—'}</strong></div>
        </div>`,
        footer: `<button class="btn" data-action="modal-close">关闭</button><button class="btn btn-danger" data-action="logout">${icon('power')}退出登录</button>`,
      });
      break;
    case 'rotate-admin-password':
      openConfirm({
        title: '轮换管理员口令？',
        description: '新口令由服务端使用安全随机数生成，不接受自定义输入。',
        note: '旧口令会立即失效，所有现有管理员会话都会被撤销；当前页面会自动获得一个新会话。新口令只在下一步显示一次。',
        confirmLabel: '生成新口令',
        action: 'do-rotate-admin-password',
      });
      break;
    case 'do-rotate-admin-password': {
      const rotation = await api.rotatePassword();
      applyAuthStatus({
        initialized: true,
        authenticated: true,
        mode: 'password',
        expires_at: rotation.expires_at,
        credential_updated_at: rotation.credential_updated_at,
        session_count: 1,
      });
      openRotatedPasswordModal(rotation.password);
      break;
    }
    case 'copy-rotated-password': {
      const password = document.getElementById('rotated-admin-password')?.textContent || '';
      if (!password) break;
      const copied = await copyTextToClipboard(password);
      showToast(copied ? '新口令已复制' : '复制失败', copied ? '请妥善保存，关闭弹窗后无法再次查看' : '浏览器未授予剪贴板权限', copied ? 'copy' : 'alert');
      break;
    }
    case 'logout':
      try {
        await api.logout();
      } catch {
        // Local session is cleared by the browser regardless of server response.
      }
      stopEvents();
      showAuthScreen('login', '已退出本地管理员会话');
      break;
    case 'refresh-overview':
      refreshSnapshot({ rerender: true, quiet: false }).then(() => showToast('数据已刷新', '已读取控制面最新快照')).catch(() => {});
      break;
    case 'start-kernel':
      if (actionEl.disabled) break;
      actionEl.disabled = true;
      await mutate(() => api.kernel.start(), { title: '内核已启动', message: 'sing-box 运行时已进入 running 状态', icon: 'play' });
      actionEl.disabled = false;
      break;
    case 'stop-kernel':
      await mutate(() => api.kernel.stop(), { title: '内核已停止', message: '活动连接已中断', icon: 'stop' });
      break;
    case 'restart-kernel':
      openConfirm({ title: '重启内核？', description: '活动连接会短暂中断。', note: '控制面会重新编译配置并重启 sing-box。失败时会返回真实错误。', confirmLabel: '确认重启', action: 'do-restart-kernel' });
      break;
    case 'do-restart-kernel':
      closeModal();
      await mutate(() => api.kernel.restart(), { title: '内核已重启', message: '新的运行时状态已载入', icon: 'power' });
      break;
    case 'toggle-inbound': {
      const inbound = state.inbounds.find((item) => item.id === id);
      if (!inbound) break;
      await mutate(() => api.inbounds.save(id, { enabled: !inbound.enabled }), {
        title: `${inbound.name} 已${inbound.enabled ? '停用' : '启用'}`,
        message: '入口配置已写入控制面',
        icon: 'inbound',
      });
      break;
    }
    case 'inbound-config':
      openInboundConfig(id);
      break;
    case 'save-inbound': {
      const inbound = state.inbounds.find((item) => item.id === id);
      if (!inbound) break;
      const patch = { name: inputValue('form-name'), enabled: directFlag('form-enabled') };
      if (inbound.kind !== 'tun') {
        patch.listen = { host: inputValue('form-host'), port: Number(inputValue('form-port')) };
        patch.udp = directFlag('form-udp');
      }
      await mutate(() => api.inbounds.save(id, patch), { title: '入口已保存', message: inbound.name, icon: 'inbound' });
      closeModal();
      break;
    }
    case 'new-inbound':
      openFormModal('inbound');
      break;
    case 'delete-inbound': {
      const inbound = state.inbounds.find((item) => item.id === id);
      if (!inbound) break;
      openConfirm({ title: '删除入口？', description: inbound.name, note: '删除后会移除该监听入口；内核运行时会重新生成运行时配置。', confirmLabel: '删除入口', action: 'do-delete-inbound', id });
      break;
    }
    case 'do-delete-inbound':
      if (!id) break;
      closeModal();
      await mutate(() => api.inbounds.remove(id), { title: '入口已删除', message: id, icon: 'trash' });
      break;
    case 'egress-tab':
      state.egressTab = actionEl.dataset.tab;
      render(true);
      break;
    case 'select-member': {
      const group = groupById(actionEl.dataset.group);
      if (!group) break;
      await mutate(() => api.groups.select(group.id, actionEl.dataset.node), {
        title: '出口已切换',
        message: `${group.name} → ${byId(actionEl.dataset.node)?.name || actionEl.dataset.node}`,
        icon: 'nodes',
      });
      break;
    }
    case 'test-node':
      await measureNode(id);
      break;
    case 'test-all-nodes':
      await measureAllNodes();
      break;
    case 'test-selected-nodes':
      await measureSelectedNodes();
      break;
    case 'edit-node':
      await openNodeEditor(id);
      break;
    case 'save-node':
      await saveNodeEditor(id);
      break;
    case 'node-detail': {
      const node = byId(id);
      if (!node) break;
      const method = nodeTestMethod();
      openModal({
        title: node.name,
        description: `${node.protocol} · ${node.server}:${node.port}`,
        body: `<div class="system-list"><div class="system-row"><span>节点 ID</span><code>${escapeHtml(node.id)}</code></div><div class="system-row"><span>策略组</span><strong>${escapeHtml(node.group)}</strong></div><div class="system-row"><span>延迟</span><strong>${node.lastTestError ? '测速失败' : node.latency > 0 ? `${node.latency} ms · ${escapeHtml(nodeTestMethod(node.lastTestKind).label)}` : '未测速'}</strong></div><div class="system-row"><span>UDP 转发</span><strong>${node.udp ? '已启用' : '未启用'}</strong></div><div class="system-row"><span>订阅引用</span><code>${escapeHtml(node.subscription_id || '手动导入')}</code></div></div>${node.lastTestError ? `<div class="modal-callout danger" style="margin-top:12px">${icon('alert')}<span>${escapeHtml(node.lastTestError)}</span></div>` : ''}`,
        footer: `<button class="btn" data-action="modal-close">关闭</button><button class="btn btn-danger" data-action="delete-node" data-id="${escapeHtml(node.id)}">${icon('trash')}删除</button><button class="btn" data-action="edit-node" data-id="${escapeHtml(node.id)}">${icon('sliders')}编辑</button><button class="btn btn-primary" data-action="test-node" data-id="${escapeHtml(node.id)}" ${state.nodeTesting || (state.nodeTestKind === 'url' && state.kernel.state !== 'running') ? 'disabled' : ''}>${icon('zap')}${escapeHtml(method.label)}</button>`,
      });
      break;
    }
    case 'delete-node': {
      const node = byId(id);
      if (!node) break;
      closeModal();
      openConfirm({ title: '删除节点？', description: node.name, note: '如果节点仍被策略组、链路或路由引用，控制面会拒绝删除并返回具体原因。', confirmLabel: '删除节点', action: 'do-delete-node', id });
      break;
    }
    case 'do-delete-node':
      if (!id) break;
      closeModal();
      await mutate(() => api.nodes.remove(id), { title: '节点已删除', message: id, icon: 'stop' });
      break;
    case 'import-node':
      openNodeImportModal();
      break;
    case 'do-import-node':
      importNodeLinks();
      break;
    case 'add-group':
      openFormModal('group');
      break;
    case 'edit-group':
      openFormModal('group', id);
      break;
    case 'delete-group': {
      const group = groupById(id);
      if (!group) break;
      openConfirm({ title: '删除策略组？', description: group.name, note: '如果策略组仍被链路或路由引用，控制面会拒绝删除并返回具体原因。', confirmLabel: '删除策略组', action: 'do-delete-group', id });
      break;
    }
    case 'do-delete-group':
      if (!id) break;
      closeModal();
      await mutate(() => api.groups.remove(id), { title: '策略组已删除', message: id, icon: 'stop' });
      break;
    case 'add-chain':
      openFormModal('chain');
      break;
    case 'edit-chain':
      openFormModal('chain', id);
      break;
    case 'delete-chain': {
      const chain = state.chains.find((item) => item.id === id);
      if (!chain) break;
      openConfirm({ title: '删除链路？', description: chain.name, note: '如果链路仍被路由规则或默认出口引用，控制面会拒绝删除并返回具体原因。', confirmLabel: '删除链路', action: 'do-delete-chain', id });
      break;
    }
    case 'do-delete-chain':
      if (!id) break;
      closeModal();
      await mutate(() => api.chains.remove(id), { title: '链路已删除', message: id, icon: 'trash' });
      break;
    case 'toggle-rule': {
      const rule = state.routeRules.find((item) => item.id === id);
      if (!rule) break;
      await mutate(() => api.routes.patchRule(id, { disabled: rule.enabled }), {
        title: `规则已${rule.enabled ? '停用' : '启用'}`,
        message: rule.name,
        icon: 'route',
      });
      break;
    }
    case 'route-mode':
      await mutate(() => api.routes.mode(actionEl.dataset.mode), {
        title: '路由模式已切换',
        message: { rules: '规则模式', global: '全局代理', direct: '直连模式' }[actionEl.dataset.mode],
        icon: 'route',
      });
      break;
    case 'toggle-preset': {
      const key = actionEl.dataset.key;
      const presets = { ...(state.snapshot?.routes?.presets || {}) };
      presets[key] = !presets[key];
      await mutate(() => api.routes.presets(presets), {
        title: `预设已${presets[key] ? '启用' : '停用'}`,
        message: key,
        icon: 'sparkles',
      });
      break;
    }
    case 'move-rule': {
      const index = state.routeRules.findIndex((item) => item.id === id);
      const target = actionEl.dataset.direction === 'up' ? index - 1 : index + 1;
      if (index < 0 || target < 0 || target >= state.routeRules.length) break;
      const next = [...state.routeRules];
      [next[index], next[target]] = [next[target], next[index]];
      const payload = next.map((rule) => ({
        id: rule.id,
        name: rule.name,
        disabled: Boolean(rule.disabled),
        match: rule.match,
        action: rule.action,
        egress: rule.egress,
      }));
      await mutate(() => api.routes.replaceRules(payload), {
        title: '规则顺序已更新',
        message: `${next[target].name} 已${actionEl.dataset.direction === 'up' ? '上移' : '下移'}`,
        icon: 'route',
      });
      break;
    }
    case 'add-rule':
      openFormModal('rule');
      break;
    case 'edit-rule':
      openFormModal('rule', id);
      break;
    case 'delete-rule':
      closeModal();
      openConfirm({ title: '删除路由规则？', description: state.routeRules.find((item) => item.id === id)?.name || id, note: '删除后会立即生成新的运行时配置。', confirmLabel: '删除规则', action: 'do-delete-rule', id });
      break;
    case 'do-delete-rule':
      if (!id) break;
      closeModal();
      await mutate(() => api.routes.removeRule(id), { title: '规则已删除', message: id, icon: 'stop' });
      break;
    case 'add-subscription':
      openFormModal('subscription');
      break;
    case 'edit-subscription':
      openFormModal('subscription', id);
      break;
    case 'toggle-subscription': {
      const sub = state.subscriptions.find((item) => item.id === id);
      if (!sub) break;
      await mutate(() => api.subscriptions.patch(id, { enabled: !sub.enabled }), {
        title: `订阅已${sub.enabled ? '暂停' : '启用'}`,
        message: sub.name,
        icon: 'rss',
      });
      break;
    }
    case 'refresh-subscription': {
      const sub = state.subscriptions.find((item) => item.id === id);
      if (!sub) break;
      actionEl.disabled = true;
      actionEl.innerHTML = `${icon('refresh')}刷新中`;
      await mutate(() => api.subscriptions.refresh(id), {
        title: '订阅刷新完成',
        message: `${sub.name} · 结果以服务端返回为准`,
        icon: 'refresh',
      });
      break;
    }
    case 'refresh-all-subs': {
      actionEl.disabled = true;
      await mutate(() => api.subscriptions.refreshAll(), { title: '订阅同步完成', message: '已按服务端调度结果更新节点', icon: 'refresh' });
      actionEl.disabled = false;
      break;
    }
    case 'delete-subscription': {
      const sub = state.subscriptions.find((item) => item.id === id);
      if (!sub) break;
      openConfirm({ title: '删除订阅？', description: sub.name, note: '会同时删除该订阅导入的节点；被策略组或规则引用时服务端会拒绝。', confirmLabel: '删除订阅', action: 'do-delete-subscription', id });
      break;
    }
    case 'do-delete-subscription':
      if (!id) break;
      closeModal();
      await mutate(() => api.subscriptions.remove(id), { title: '订阅已删除', message: id, icon: 'stop' });
      break;
    case 'copy': {
      const text = actionEl.dataset.copy || '';
      const copied = await copyTextToClipboard(text);
      showToast(copied ? '已复制到剪贴板' : '复制失败', copied ? text.length > 38 ? `${text.slice(0, 38)}…` : text : '浏览器未授予剪贴板权限', copied ? 'check' : 'alert');
      break;
    }
    case 'copy-log': {
      const entry = state.logs.find((item) => item.id === actionEl.dataset.logId);
      if (!entry) {
        showToast('日志已刷新', '目标日志已不在当前缓冲区', 'alert');
        break;
      }
      const copied = await copyTextToClipboard(logLineText(entry));
      showToast(copied ? '日志已复制' : '日志复制失败', copied ? `${entry.channel.toUpperCase()} · ${entry.time}` : '浏览器未授予剪贴板权限', copied ? 'copy' : 'alert');
      break;
    }
    case 'toggle-ruleset': {
      const ruleset = state.ruleSets.find((item) => item.name === actionEl.dataset.name);
      if (!ruleset) break;
      await mutate(() => api.rulesets.patch(ruleset.name, { enabled: !ruleset.enabled }), {
        title: `规则集已${ruleset.enabled ? '停用' : '启用'}`,
        message: ruleset.name,
        icon: 'rules',
      });
      break;
    }
    case 'refresh-ruleset': {
      const ruleset = state.ruleSets.find((item) => item.name === actionEl.dataset.name);
      if (!ruleset) break;
      actionEl.disabled = true;
      await mutate(() => api.rulesets.refresh(ruleset.name), { title: '规则集已刷新', message: ruleset.name, icon: 'refresh' });
      break;
    }
    case 'refresh-all-rulesets': {
      actionEl.disabled = true;
      await mutate(() => api.rulesets.refreshAll(), { title: '规则集更新完成', message: '已刷新全部远端规则集', icon: 'refresh' });
      actionEl.disabled = false;
      break;
    }
    case 'new-ruleset':
      openFormModal('ruleset');
      break;
    case 'ruleset-detail': {
      const ruleset = state.ruleSets.find((item) => item.name === actionEl.dataset.name);
      if (!ruleset) break;
      openModal({
        title: ruleset.name,
        description: ruleset.description,
        body: `<div class="system-list"><div class="system-row"><span>来源</span><strong>${ruleset.builtin ? '内置资源' : '自定义源'}</strong></div><div class="system-row"><span>URL</span><code>${escapeHtml(ruleset.url || '—')}</code></div><div class="system-row"><span>格式</span><strong>${escapeHtml(ruleset.format)}</strong></div><div class="system-row"><span>大小</span><strong>${escapeHtml(ruleset.size)}</strong></div><div class="system-row"><span>SHA</span><code>${escapeHtml(ruleset.sha)}</code></div><div class="system-row"><span>更新时间</span><strong>${escapeHtml(ruleset.updated)}</strong></div><div class="system-row"><span>引用状态</span><strong>${ruleset.enabled ? '参与路由匹配' : '未启用'}</strong></div></div>${ruleset.error ? `<div class="modal-callout danger" style="margin-top:12px">${icon('alert')}<span>${escapeHtml(ruleset.error)}</span></div>` : ''}`,
        footer: `<button class="btn" data-action="modal-close">关闭</button>${ruleset.builtin ? '' : `<button class="btn btn-danger" data-action="delete-ruleset" data-name="${escapeHtml(ruleset.name)}">${icon('trash')}删除</button>`}<button class="btn btn-primary" data-action="refresh-ruleset" data-name="${escapeHtml(ruleset.name)}">${icon('refresh')}刷新规则集</button>`,
      });
      break;
    }
    case 'delete-ruleset': {
      const name = actionEl.dataset.name;
      closeModal();
      openConfirm({ title: '删除规则集？', description: name, note: '仍被路由规则引用时服务端会拒绝删除。', confirmLabel: '删除规则集', action: 'do-delete-ruleset', id: name });
      break;
    }
    case 'do-delete-ruleset':
      if (!id) break;
      closeModal();
      await mutate(() => api.rulesets.remove(id), { title: '规则集已删除', message: id, icon: 'stop' });
      break;
    case 'close-connection':
      closeConnection(id);
      break;
    case 'close-all-connections':
      openConfirm({ title: '关闭全部活动连接？', description: '当前所有请求会话将被中断。', note: '控制面会调用 sing-box 连接管理器关闭全部连接。', confirmLabel: '全部关闭', action: 'do-close-all-connections' });
      break;
    case 'do-close-all-connections':
      closeModal();
      await mutate(() => api.connections.closeAll(), { title: '全部连接已关闭', message: '服务端返回实际关闭数量', icon: 'stop' });
      break;
    case 'do-close-detail-connection':
      closeModal();
      closeConnection(id);
      break;
    case 'toggle-log-pause':
      state.logPaused = !state.logPaused;
      render(true);
      if (!state.logPaused) updateLogConsole();
      showToast(state.logPaused ? '日志显示已暂停' : '日志显示已恢复', state.logPaused ? 'SSE 仍在后台接收' : '已显示缓冲中的最新日志');
      break;
    case 'log-source':
      state.logSource = actionEl.dataset.source || 'all';
      render(true);
      break;
    case 'log-level-quick':
      state.logLevel = actionEl.dataset.level || 'all';
      render(true);
      break;
    case 'toggle-log-autoscroll':
      state.logAutoScroll = !state.logAutoScroll;
      render(true);
      showToast(state.logAutoScroll ? '自动滚动已开启' : '自动滚动已关闭');
      break;
    case 'scroll-log-bottom':
      state.logAutoScroll = true;
      render(true);
      scrollLogConsole();
      break;
    case 'clear-logs':
      state.logs = [];
      render(true);
      showToast('本地日志缓冲已清空', '服务端日志环不会被清空');
      break;
    case 'edit-dns-slot':
      openDNSEditor(actionEl.dataset.key);
      break;
    case 'save-dns-slot':
      await saveDNSSlot(actionEl.dataset.key, actionEl);
      break;
    case 'toggle-dns-slot': {
      const key = actionEl.dataset.key;
      const profile = currentDNSProfile();
      if (!profile[key]) break;
      profile[key].enabled = !profile[key].enabled;
      await mutate(() => api.dns.updateProfile(profile), {
        title: `DNS 槽位已${profile[key].enabled ? '启用' : '停用'}`,
        message: key,
        icon: 'globe',
      });
      break;
    }
    case 'toggle-dns-cache':
      await mutate(() => api.dns.patch({ cache: !state.dnsProfile.cache }), {
        title: `DNS 缓存已${state.dnsProfile.cache ? '关闭' : '开启'}`,
        message: '配置已写入控制面',
        icon: 'globe',
      });
      break;
    case 'flush-dns':
      await mutate(() => api.dns.flush(), { title: 'DNS 缓存已刷新', message: '运行态缓存已清理', icon: 'refresh' });
      break;
    case 'test-dns':
      await mutate(() => api.dns.test(), { title: 'DNS 测速完成', message: '已记录各启用槽位的真实网络往返耗时', icon: 'zap' });
      break;
    case 'toggle-setting': {
      const key = actionEl.dataset.key;
      if (key === 'compactTable') {
        const next = !state.flags.compactTable;
        const result = await mutate(() => api.settings.update({ compactTable: next ? 'on' : 'off' }), {
          title: '设置已保存',
          message: `紧凑表格 ${next ? '已开启' : '已关闭'}`,
          icon: 'check',
        });
        if (result !== null) document.body.classList.toggle('is-compact', next);
      }
      if (key === 'autoRefresh') {
        const next = !state.flags.autoRefresh;
        await mutate(() => api.settings.update({ autoRefresh: next ? 'on' : 'off' }), {
          title: `自动刷新已${next ? '开启' : '关闭'}`,
          message: next ? '调度器会按订阅间隔自动同步' : '后台不再自动同步，手动同步仍可用',
          icon: 'refresh',
        });
      }
      break;
    }
    case 'modal-close':
      closeModal();
      break;
    case 'modal-submit':
      submitForm(actionEl.dataset.kind, actionEl.dataset.id || '', actionEl);
      break;
    default:
      break;
  }
}

document.addEventListener('click', (event) => {
  const orderControl = event.target.closest('[data-picker-move], [data-picker-remove]');
  if (orderControl) {
    const picker = orderControl.closest('[data-node-picker]');
    const row = orderControl.closest('[data-picker-order-id]');
    if (!picker || !row) return;
    const ids = pickerOrderIDs(picker);
    const index = ids.indexOf(row.dataset.pickerOrderId);
    if (orderControl.hasAttribute('data-picker-remove')) {
      const input = pickerOptionInput(picker, row.dataset.pickerOrderId);
      if (input) {
        input.checked = false;
        input.closest('.node-option')?.classList.remove('is-selected');
      }
      ids.splice(index, 1);
      renderPickerOrder(picker, ids);
      updatePickerCount(picker);
      return;
    }
    const target = orderControl.dataset.pickerMove === 'up' ? index - 1 : index + 1;
    if (index < 0 || target < 0 || target >= ids.length) return;
    [ids[index], ids[target]] = [ids[target], ids[index]];
    renderPickerOrder(picker, ids);
    const movedRow = [...picker.querySelectorAll('[data-picker-order-id]')].find((element) => element.dataset.pickerOrderId === row.dataset.pickerOrderId);
    movedRow?.querySelector(`[data-picker-move="${orderControl.dataset.pickerMove}"]`)?.focus();
    return;
  }

  const viewElement = event.target.closest('[data-view]');
  if (viewElement && !event.target.closest('[data-action]')) {
    if (!modalRoot.hidden) closeModal();
    setView(viewElement.dataset.view, { tab: viewElement.dataset.tab });
    return;
  }

  const actionElement = event.target.closest('[data-action]');
  if (actionElement) {
    handleAction(actionElement).catch((error) => {
      if (error?.status !== 401) showToast('操作失败', error?.hint || error?.message || '未知错误', 'alert');
    });
    return;
  }

  if (event.target.classList.contains('sidebar-backdrop')) {
    document.body.classList.remove('sidebar-open');
    document.getElementById('sidebar-backdrop').hidden = true;
  }
  if (event.target === modalRoot) closeModal();
});

document.addEventListener('input', (event) => {
  if (event.target.id === 'node-import-links') {
    updateNodeImportPreview(event.target.value);
    return;
  }
  if (event.target.id === 'node-test-url') {
    state.nodeTestURL = event.target.value;
    return;
  }
  const filter = event.target.dataset.filter;
  if (filter === 'nodes') {
    state.nodeQuery = event.target.value;
    updateNodeRows();
  }
  if (filter === 'rulesets') {
    state.rulesetQuery = event.target.value;
    updateRulesetRows();
  }
  if (filter === 'connections') {
    state.connQuery = event.target.value;
    updateConnectionRows();
  }
  if (filter === 'logs') {
    state.logQuery = event.target.value;
    updateLogConsole();
  }
});

document.addEventListener('change', async (event) => {
  const picker = event.target.closest('[data-node-picker]');
  if (picker && event.target.matches('[data-picker-value]')) {
    const max = Number(picker.dataset.max || 0);
    if (max && picker.querySelectorAll('[data-picker-value]:checked').length > max) {
      event.target.checked = false;
      showToast('超出选择上限', `最多选择 ${max} 个节点`, 'alert');
      return;
    }
    event.target.closest('.node-option')?.classList.toggle('is-selected', event.target.checked);
    if (picker.dataset.pickerOrdered === 'true') {
      const ids = pickerOrderIDs(picker).filter((nodeID) => nodeID !== event.target.value);
      if (event.target.checked) ids.push(event.target.value);
      renderPickerOrder(picker, ids);
    }
    updatePickerCount(picker);
    return;
  }
  const setting = event.target.dataset.setting;
  if (setting === 'node-test-kind') {
    state.nodeTestKind = event.target.value;
    state.selectedNodeIDs = new Set(state.selectedNodeIDs);
    render(true);
    return;
  }
  if (event.target.dataset.nodeSelectAll !== undefined) {
    const filtered = filteredNodes();
    if (event.target.checked) {
      filtered.forEach((node) => state.selectedNodeIDs.add(node.id));
    } else {
      filtered.forEach((node) => state.selectedNodeIDs.delete(node.id));
    }
    render(true);
    return;
  }
  if (event.target.dataset.nodeSelect) {
    const nodeID = event.target.dataset.nodeSelect;
    if (event.target.checked) state.selectedNodeIDs.add(nodeID);
    else state.selectedNodeIDs.delete(nodeID);
    render(true);
    return;
  }
  if (setting === 'log-level') {
    state.logLevel = event.target.value;
    render(true);
    return;
  }
  if (setting === 'default-outbound') {
    const [kind, refID = ''] = String(event.target.value).split(':');
    await mutate(() => api.routes.defaultEgress(kind, refID), {
      title: '默认出口已更新',
      message: event.target.selectedOptions[0]?.textContent || '',
      icon: 'route',
    });
    return;
  }
  if (setting === 'dns-strategy') {
    await mutate(() => api.dns.patch({ strategy: event.target.value }), {
      title: 'DNS 策略已更新',
      message: event.target.value,
      icon: 'globe',
    });
    return;
  }
  if (setting === 'subscription-max-attempts') {
    await mutate(() => api.settings.update({ subscriptionMaxAttempts: event.target.value }), {
      title: '订阅重试策略已更新',
      message: `最多尝试 ${event.target.value} 次`,
      icon: 'refresh',
    });
    return;
  }
  if (setting === 'subscription-conflict-policy') {
    await mutate(() => api.settings.update({ subscriptionConflictPolicy: event.target.value }), {
      title: '订阅冲突策略已更新',
      message: event.target.value === 'follow_remote' ? '远端移除的节点将被删除' : '远端移除的节点将保留本地副本',
      icon: 'rss',
    });
    return;
  }
  if (event.target.id === 'form-dns-detour' || event.target.id === 'form-dns-follow' || event.target.id === 'form-dns-protocol') {
    syncDNSEditorFields();
    return;
  }
  if (event.target.id === 'form-rule-action') {
    const target = document.getElementById('form-rule-target');
    if (target) target.disabled = event.target.value !== 'egress';
  }
});

document.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' || event.key === ' ') {
    const logRow = event.target.closest('[data-action="copy-log"]');
    if (logRow) {
      event.preventDefault();
      handleAction(logRow).catch((error) => {
        if (error?.status !== 401) showToast('操作失败', error?.hint || error?.message || '未知错误', 'alert');
      });
      return;
    }
  }
  if (event.key === 'Escape') {
    if (!modalRoot.hidden) closeModal();
    document.body.classList.remove('sidebar-open');
    document.getElementById('sidebar-backdrop').hidden = true;
  }
});

document.getElementById('sidebar-backdrop').addEventListener('click', () => {
  document.body.classList.remove('sidebar-open');
  document.getElementById('sidebar-backdrop').hidden = true;
});

window.addEventListener('hashchange', () => {
  const view = location.hash.slice(1);
  if (VIEW_META[view] && view !== state.view) {
    state.view = view;
    render(false);
  }
});

window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if (state.themeChoice === 'system') applyTheme('system', true);
});

document.getElementById('icon-sprite').innerHTML = iconSprite();
hydrateStaticIcons();
applyTheme(state.themeChoice, true);
boot();
window.setInterval(syncSidebarStatus, 1000);

window.addEventListener('boxd:unauthorized', () => {
  stopEvents();
  showAuthScreen('login', '会话已失效，请重新登录');
});

window.addEventListener('beforeunload', stopEvents);
