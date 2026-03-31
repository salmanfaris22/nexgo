package devtools

// DevToolsHTML is the dev tools panel served at /_nexgo/devtools
const DevToolsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>NexGo Dev Tools</title>
  <style>
    @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600;700&family=Outfit:wght@400;600;700;900&display=swap');

    :root {
      --bg: #080808;
      --surface: #0f0f0f;
      --surface2: #161616;
      --border: #1e1e1e;
      --text: #e8e8e8;
      --muted: #555;
      --accent: #00d2ff;
      --accent2: #7b2ff7;
      --green: #00e676;
      --yellow: #ffd740;
      --red: #ff4757;
      --purple: #ae82ff;
      --orange: #ff9100;
      --mono: 'JetBrains Mono', monospace;
      --font: 'Outfit', system-ui, sans-serif;
    }

    * { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: var(--font);
      background: var(--bg);
      color: var(--text);
      min-height: 100vh;
      display: grid;
      grid-template-rows: 56px 1fr;
    }

    /* ── Header ── */
    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 1.5rem;
      border-bottom: 1px solid var(--border);
      background: var(--surface);
      position: sticky;
      top: 0;
      z-index: 100;
    }
    .logo {
      display: flex;
      align-items: center;
      gap: 0.6rem;
      font-weight: 900;
      font-size: 1rem;
      letter-spacing: -0.02em;
    }
    .logo-badge {
      background: linear-gradient(135deg, var(--accent), var(--accent2));
      color: #fff;
      font-size: 0.65rem;
      font-weight: 700;
      padding: 0.15rem 0.5rem;
      border-radius: 100px;
      letter-spacing: 0.05em;
    }
    .header-actions {
      display: flex;
      align-items: center;
      gap: 0.75rem;
    }
    .status-dot {
      width: 8px; height: 8px;
      border-radius: 50%;
      background: var(--green);
      box-shadow: 0 0 8px var(--green);
      animation: pulse 2s infinite;
    }
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.4; }
    }
    .status-text { font-size: 0.8rem; color: var(--muted); font-family: var(--mono); }
    .btn-reload {
      background: var(--surface2);
      border: 1px solid var(--border);
      color: var(--text);
      padding: 0.4rem 1rem;
      border-radius: 6px;
      font-size: 0.8rem;
      cursor: pointer;
      font-family: var(--mono);
      transition: border-color 0.2s;
    }
    .btn-reload:hover { border-color: var(--accent); color: var(--accent); }

    /* ── Layout ── */
    .main {
      display: grid;
      grid-template-columns: 200px 1fr;
    }

    /* ── Sidebar ── */
    aside {
      border-right: 1px solid var(--border);
      padding: 1rem 0;
      background: var(--surface);
    }
    .nav-group { margin-bottom: 0.25rem; }
    .nav-label {
      font-size: 0.65rem;
      font-weight: 700;
      letter-spacing: 0.12em;
      color: var(--muted);
      padding: 0.5rem 1.25rem 0.25rem;
      text-transform: uppercase;
    }
    .nav-item {
      display: flex;
      align-items: center;
      gap: 0.6rem;
      padding: 0.55rem 1.25rem;
      font-size: 0.85rem;
      color: var(--muted);
      cursor: pointer;
      border-left: 2px solid transparent;
      transition: all 0.15s;
    }
    .nav-item:hover { color: var(--text); background: var(--surface2); }
    .nav-item.active {
      color: var(--accent);
      border-left-color: var(--accent);
      background: rgba(0, 210, 255, 0.05);
    }
    .nav-item .icon { font-size: 1rem; width: 18px; text-align: center; }

    /* ── Content ── */
    .content {
      padding: 1.5rem;
      overflow-y: auto;
      max-height: calc(100vh - 56px);
    }

    /* ── Panels ── */
    .panel { display: none; }
    .panel.active { display: block; }

    .panel-title {
      font-size: 1.25rem;
      font-weight: 700;
      margin-bottom: 1.25rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .panel-title small {
      font-size: 0.75rem;
      font-weight: 400;
      color: var(--muted);
      font-family: var(--mono);
      background: var(--surface2);
      padding: 0.2rem 0.6rem;
      border-radius: 4px;
    }

    /* ── Routes Panel ── */
    .route-list { display: flex; flex-direction: column; gap: 0.5rem; }
    .route-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 0.875rem 1.25rem;
      display: grid;
      grid-template-columns: 60px 1fr 1fr;
      align-items: center;
      gap: 1rem;
      transition: border-color 0.15s;
    }
    .route-card:hover { border-color: var(--accent); }
    .method-badge {
      font-family: var(--mono);
      font-size: 0.7rem;
      font-weight: 700;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      text-align: center;
    }
    .method-page { background: rgba(0,210,255,0.15); color: var(--accent); }
    .method-api  { background: rgba(174,130,255,0.15); color: var(--purple); }
    .route-pattern {
      font-family: var(--mono);
      font-size: 0.875rem;
    }
    .route-file {
      font-family: var(--mono);
      font-size: 0.75rem;
      color: var(--muted);
      text-align: right;
    }

    /* ── Performance Panel ── */
    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 1rem;
      margin-bottom: 1.5rem;
    }
    .metric-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1.25rem;
    }
    .metric-label { font-size: 0.75rem; color: var(--muted); margin-bottom: 0.5rem; text-transform: uppercase; letter-spacing: 0.08em; }
    .metric-value { font-size: 2rem; font-weight: 800; font-family: var(--mono); letter-spacing: -0.03em; }
    .metric-value.green { color: var(--green); }
    .metric-value.yellow { color: var(--yellow); }
    .metric-value.blue { color: var(--accent); }
    .metric-unit { font-size: 0.8rem; color: var(--muted); margin-top: 0.2rem; }

    /* ── Log Panel ── */
    .log-container {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1rem;
      font-family: var(--mono);
      font-size: 0.8rem;
      height: 420px;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
    }
    .log-entry { display: flex; gap: 1rem; align-items: baseline; }
    .log-time { color: var(--muted); min-width: 80px; }
    .log-level { min-width: 50px; font-weight: 600; }
    .log-level.info  { color: var(--accent); }
    .log-level.warn  { color: var(--yellow); }
    .log-level.error { color: var(--red); }
    .log-level.ok    { color: var(--green); }
    .log-msg { color: var(--text); }
    .log-path { color: var(--purple); }
    .log-status { color: var(--green); font-weight: 700; }
    .log-status.e4 { color: var(--yellow); }
    .log-status.e5 { color: var(--red); }
    .log-empty { color: var(--muted); text-align: center; padding: 2rem; font-size: 0.85rem; }

    /* ── Config Panel ── */
    .config-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 1rem;
    }
    .config-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 8px;
      overflow: hidden;
    }
    .config-card-header {
      padding: 0.75rem 1.25rem;
      border-bottom: 1px solid var(--border);
      font-size: 0.8rem;
      font-weight: 600;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
    .config-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0.6rem 1.25rem;
      border-bottom: 1px solid var(--border);
      font-size: 0.85rem;
    }
    .config-row:last-child { border-bottom: none; }
    .config-key { color: var(--muted); font-family: var(--mono); }
    .config-val { font-family: var(--mono); font-weight: 600; }
    .config-val.true  { color: var(--green); }
    .config-val.false { color: var(--red); }
    .config-val.num   { color: var(--accent); }
    .config-val.str   { color: var(--yellow); }

    /* ── Request log ── */
    .req-list { display: flex; flex-direction: column; gap: 0.5rem; }
    .req-row {
      display: grid;
      grid-template-columns: 80px 60px 1fr 80px 80px;
      gap: 0.75rem;
      align-items: center;
      padding: 0.6rem 1rem;
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 6px;
      font-family: var(--mono);
      font-size: 0.78rem;
      transition: border-color 0.15s;
    }
    .req-row:hover { border-color: var(--border); border-color: #333; }
    .req-method { font-weight: 700; color: var(--accent); }
    .req-status { font-weight: 700; }
    .req-status.s2 { color: var(--green); }
    .req-status.s4 { color: var(--yellow); }
    .req-status.s5 { color: var(--red); }
    .req-path { color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .req-dur { color: var(--muted); text-align: right; }
    .req-time { color: var(--muted); font-size: 0.7rem; text-align: right; }

    /* ── Empty states ── */
    .empty {
      text-align: center;
      padding: 4rem 2rem;
      color: var(--muted);
    }
    .empty-icon { font-size: 3rem; margin-bottom: 1rem; }
    .empty-text { font-size: 0.9rem; }

    /* ── HMR badge ── */
    .hmr-indicator {
      display: flex;
      align-items: center;
      gap: 0.4rem;
      font-size: 0.75rem;
      color: var(--green);
      font-family: var(--mono);
    }
    .hmr-dot {
      width: 6px; height: 6px;
      border-radius: 50%;
      background: var(--green);
      box-shadow: 0 0 6px var(--green);
    }
  </style>
</head>
<body>
  <header>
    <div class="logo">
      <span>⚡ NexGo</span>
      <span class="logo-badge">DEV TOOLS</span>
    </div>
    <div class="header-actions">
      <div class="hmr-indicator" id="hmrStatus">
        <div class="hmr-dot"></div>
        <span>HMR connected</span>
      </div>
      <button class="btn-reload" onclick="triggerReload()">↺ Reload</button>
    </div>
  </header>

  <div class="main">
    <aside>
      <div class="nav-group">
        <div class="nav-label">Inspect</div>
        <div class="nav-item active" onclick="showPanel('routes', this)">
          <span class="icon">📁</span> Routes
        </div>
        <div class="nav-item" onclick="showPanel('requests', this)">
          <span class="icon">🌐</span> Requests
        </div>
        <div class="nav-item" onclick="showPanel('logs', this)">
          <span class="icon">📋</span> Logs
        </div>
      </div>
      <div class="nav-group">
        <div class="nav-label">System</div>
        <div class="nav-item" onclick="showPanel('perf', this)">
          <span class="icon">⚡</span> Performance
        </div>
        <div class="nav-item" onclick="showPanel('config', this)">
          <span class="icon">⚙️</span> Config
        </div>
      </div>
    </aside>

    <div class="content">

      <!-- Routes Panel -->
      <div class="panel active" id="panel-routes">
        <div class="panel-title">
          Routes
          <small id="routeCount">loading...</small>
        </div>
        <div class="route-list" id="routeList">
          <div class="empty">
            <div class="empty-icon">📁</div>
            <div class="empty-text">Loading routes...</div>
          </div>
        </div>
      </div>

      <!-- Requests Panel -->
      <div class="panel" id="panel-requests">
        <div class="panel-title">
          Live Requests
          <small id="reqCount">0 total</small>
        </div>
        <div class="req-list" id="reqList">
          <div class="empty">
            <div class="empty-icon">🌐</div>
            <div class="empty-text">No requests yet — browse your app to see them here</div>
          </div>
        </div>
      </div>

      <!-- Logs Panel -->
      <div class="panel" id="panel-logs">
        <div class="panel-title">Server Logs</div>
        <div class="log-container" id="logContainer">
          <div class="log-empty">Logs will appear here when the server handles requests</div>
        </div>
      </div>

      <!-- Performance Panel -->
      <div class="panel" id="panel-perf">
        <div class="panel-title">Performance</div>
        <div class="metrics-grid">
          <div class="metric-card">
            <div class="metric-label">Total Requests</div>
            <div class="metric-value blue" id="perfTotal">0</div>
            <div class="metric-unit">since server start</div>
          </div>
          <div class="metric-card">
            <div class="metric-label">Avg Response</div>
            <div class="metric-value green" id="perfAvg">—</div>
            <div class="metric-unit">milliseconds</div>
          </div>
          <div class="metric-card">
            <div class="metric-label">Error Rate</div>
            <div class="metric-value yellow" id="perfErr">0%</div>
            <div class="metric-unit">4xx + 5xx</div>
          </div>
          <div class="metric-card">
            <div class="metric-label">Active Routes</div>
            <div class="metric-value blue" id="perfRoutes">0</div>
            <div class="metric-unit">pages + api</div>
          </div>
        </div>
      </div>

      <!-- Config Panel -->
      <div class="panel" id="panel-config">
        <div class="panel-title">Configuration</div>
        <div class="config-grid" id="configGrid">
          <div class="empty">
            <div class="empty-icon">⚙️</div>
            <div class="empty-text">Loading config...</div>
          </div>
        </div>
      </div>

    </div>
  </div>

  <script>
    // ── State ──
    const state = {
      routes: [],
      requests: [],
      logs: [],
    };

    // ── Panel switching ──
    function showPanel(name, el) {
      document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
      document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
      document.getElementById('panel-' + name).classList.add('active');
      el.classList.add('active');
    }

    // ── Load routes ──
    async function loadRoutes() {
      try {
        const res = await fetch('/_nexgo/routes');
        const routes = await res.json();
        state.routes = routes || [];
        renderRoutes();
        document.getElementById('perfRoutes').textContent = state.routes.length;
      } catch(e) {
        console.error('Failed to load routes:', e);
      }
    }

    function renderRoutes() {
      const list = document.getElementById('routeList');
      const count = document.getElementById('routeCount');
      const routes = state.routes;

      count.textContent = routes.length + ' route' + (routes.length !== 1 ? 's' : '');

      if (!routes.length) {
        list.innerHTML = '<div class="empty"><div class="empty-icon">📭</div><div class="empty-text">No routes found in pages/</div></div>';
        return;
      }

      list.innerHTML = routes.map(r => {
        const isApi = r.type === 'api';
        const badge = isApi
          ? '<span class="method-badge method-api">API</span>'
          : '<span class="method-badge method-page">PAGE</span>';

        // Shorten file path
        const file = r.file ? r.file.replace(/.*\/pages\//, 'pages/') : '';


		

        return '<div class="route-card" onclick="window.open('${r.pattern}','_blank')" style="cursor:pointer">
            ${badge}
            <span class="route-pattern">${r.pattern}</span>
            <span class="route-file">${file}</span>
          </div>';
      }).join('');
    }

    // ── Request tracking ──
    let reqTotal = 0, reqErrors = 0, reqTotalMs = 0;

    function addRequest(method, status, path, ms) {
      reqTotal++;
      reqTotalMs += ms;
      if (status >= 400) reqErrors++;

      const entry = { method, status, path, ms, time: new Date() };
      state.requests.unshift(entry);
      if (state.requests.length > 200) state.requests.pop();

      renderRequests();
      updatePerfMetrics();
      addLog(status >= 500 ? 'error' : status >= 400 ? 'warn' : 'ok', method, status, path, ms);
    }

    function renderRequests() {
      const list = document.getElementById('reqList');
      const count = document.getElementById('reqCount');
      count.textContent = reqTotal + ' total';

      if (!state.requests.length) {
        list.innerHTML = '<div class="empty"><div class="empty-icon">🌐</div><div class="empty-text">No requests yet</div></div>';
        return;
      }

      list.innerHTML = state.requests.slice(0, 50).map(r => {
        const sc = r.status >= 500 ? 's5' : r.status >= 400 ? 's4' : 's2';
        const t = r.time.toLocaleTimeString('en', { hour12: false });
        return '<div class="req-row">
          <span class="req-method">${r.method}</span>
          <span class="req-status ${sc}">${r.status}</span>
          <span class="req-path">${r.path}</span>
          <span class="req-dur">${r.sms}ms</span>
          <span class="req-time">${t}</span>
        </div>';
      }).join('');
    }

    // ── Logs ──
    function addLog(level, method, status, path, ms) {
      const container = document.getElementById('logContainer');
      const empty = container.querySelector('.log-empty');
      if (empty) empty.remove();

      const t = new Date().toLocaleTimeString('en', { hour12: false });
      const sc = status >= 500 ? 'e5' : status >= 400 ? 'e4' : '';
      const entry = document.createElement('div');
      entry.className = 'log-entry';
      entry.innerHTML = '
        <span class="log-time">${t}</span>
        <span class="log-level ${level}">${level.toUpperCase()}</span>
        <span class="log-status ${sc}">${status}</span>
        <span class="log-path">${method} ${path}</span>
        <span class="log-msg" style="color:var(--muted)">${ms}ms</span>';
      container.insertBefore(entry, container.firstChild);

      // Cap at 500 entries
      while (container.children.length > 500) {
        container.removeChild(container.lastChild);
      }
    }

    // ── Perf metrics ──
    function updatePerfMetrics() {
      document.getElementById('perfTotal').textContent = reqTotal;
      document.getElementById('perfAvg').textContent = reqTotal ? Math.round(reqTotalMs / reqTotal) : '—';
      const errPct = reqTotal ? Math.round((reqErrors / reqTotal) * 100) : 0;
      document.getElementById('perfErr').textContent = errPct + '%';
      document.getElementById('perfErr').className = 'metric-value ' + (errPct > 10 ? 'red' : errPct > 0 ? 'yellow' : 'green');
    }

    // ── Config display ──
    function loadConfig() {
      // We'll show static known config from the page
      const cfg = {
        Server: {
          port: window.__NEXGO_PORT__ || 3000,
          host: window.__NEXGO_HOST__ || 'localhost',
          mode: 'development',
        },
        Build: {
          compression: true,
          minify: true,
          hotReload: true,
        },
        Paths: {
          pages: 'pages/',
          static: 'static/',
          layouts: 'layouts/',
          components: 'components/',
        }
      };

      const grid = document.getElementById('configGrid');
      grid.innerHTML = Object.entries(cfg).map(([section, values]) => '
        <div class="config-card">
          <div class="config-card-header">${section}</div>
          ${Object.entries(values).map(([k, v]) => {
            let cls = typeof v === 'boolean' ? v.toString() : typeof v === 'number' ? 'num' : 'str';
            return '<div class="config-row">
              <span class="config-key">${k}</span>
              <span class="config-val ${cls}">${JSON.stringify(v)}</span>
            </div>';
          }).join('')}
        </div>'
      ).join('');
    }

    // ── Hot reload via SSE ──
    function initHMR() {
      const es = new EventSource('/_nexgo/hmr');
      const status = document.getElementById('hmrStatus');

      es.onopen = () => {
        status.innerHTML = '<div class="hmr-dot"></div><span>HMR connected</span>';
        status.style.color = 'var(--green)';
      };

      es.onmessage = (e) => {
        const msg = JSON.parse(e.data);
        if (msg.type === 'reload') {
          addLog('ok', 'HMR', 200, '/reload', 0);
          // Don't reload devtools itself, just refresh route list
          loadRoutes();
        }
      };

      es.onerror = () => {
        status.innerHTML = '<div class="hmr-dot" style="background:var(--red);box-shadow:0 0 6px var(--red)"></div><span>Reconnecting...</span>';
        status.style.color = 'var(--red)';
      };
    }

    // ── Intercept fetch to track requests ──
    const origFetch = window.fetch;
    window.fetch = async function(...args) {
      const url = typeof args[0] === 'string' ? args[0] : args[0]?.url || '';
      if (url.startsWith('/_nexgo/')) return origFetch(...args);

      const method = args[1]?.method || 'GET';
      const t0 = performance.now();
      try {
        const res = await origFetch(...args);
        const ms = Math.round(performance.now() - t0);
        addRequest(method, res.status, url, ms);
        return res;
      } catch(e) {
        addRequest(method, 0, url, Math.round(performance.now() - t0));
        throw e;
      }
    };

    // ── Manual reload ──
    async function triggerReload() {
      await fetch('/_nexgo/reload');
      loadRoutes();
    }

    // ── Init ──
    loadRoutes();
    loadConfig();
    initHMR();

    // Simulate a startup log entry
    addLog('ok', 'SYS', 200, '/devtools loaded', 0);
  </script>
</body>
</html>`
