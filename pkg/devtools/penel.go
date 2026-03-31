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

    /* Header */
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
    .logo { display: flex; align-items: center; gap: 0.6rem; font-weight: 900; font-size: 1rem; letter-spacing: -0.02em; }
    .logo-badge {
      background: linear-gradient(135deg, var(--accent), var(--accent2));
      color: #fff; font-size: 0.65rem; font-weight: 700;
      padding: 0.15rem 0.5rem; border-radius: 100px; letter-spacing: 0.05em;
    }
    .header-actions { display: flex; align-items: center; gap: 0.75rem; }
    .btn-reload {
      background: var(--surface2); border: 1px solid var(--border); color: var(--text);
      padding: 0.4rem 1rem; border-radius: 6px; font-size: 0.8rem;
      cursor: pointer; font-family: var(--mono); transition: border-color 0.2s;
    }
    .btn-reload:hover { border-color: var(--accent); color: var(--accent); }
    .hmr-indicator { display: flex; align-items: center; gap: 0.4rem; font-size: 0.75rem; color: var(--green); font-family: var(--mono); }
    .hmr-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); box-shadow: 0 0 6px var(--green); }

    /* Layout */
    .main { display: grid; grid-template-columns: 210px 1fr; }

    /* Sidebar */
    aside {
      border-right: 1px solid var(--border); padding: 0.75rem 0;
      background: var(--surface); overflow-y: auto; max-height: calc(100vh - 56px);
    }
    .nav-group { margin-bottom: 0.15rem; }
    .nav-label {
      font-size: 0.6rem; font-weight: 700; letter-spacing: 0.12em;
      color: var(--muted); padding: 0.5rem 1.25rem 0.2rem; text-transform: uppercase;
    }
    .nav-item {
      display: flex; align-items: center; gap: 0.6rem;
      padding: 0.45rem 1.25rem; font-size: 0.82rem; color: var(--muted);
      cursor: pointer; border-left: 2px solid transparent; transition: all 0.15s;
    }
    .nav-item:hover { color: var(--text); background: var(--surface2); }
    .nav-item.active { color: var(--accent); border-left-color: var(--accent); background: rgba(0,210,255,0.05); }
    .nav-item .icon { font-size: 0.95rem; width: 18px; text-align: center; }

    /* Content */
    .content { padding: 1.5rem; overflow-y: auto; max-height: calc(100vh - 56px); }
    .panel { display: none; }
    .panel.active { display: block; animation: fadeIn 0.2s ease; }
    @keyframes fadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }

    .panel-title {
      font-size: 1.25rem; font-weight: 700; margin-bottom: 1.25rem;
      display: flex; align-items: center; gap: 0.5rem;
    }
    .panel-title small {
      font-size: 0.75rem; font-weight: 400; color: var(--muted);
      font-family: var(--mono); background: var(--surface2);
      padding: 0.2rem 0.6rem; border-radius: 4px;
    }

    /* Routes Panel */
    .route-list { display: flex; flex-direction: column; gap: 0.5rem; }
    .route-card {
      background: var(--surface); border: 1px solid var(--border); border-radius: 8px;
      padding: 0.875rem 1.25rem; display: grid; grid-template-columns: 60px 1fr 1fr;
      align-items: center; gap: 1rem; transition: border-color 0.15s; cursor: pointer;
    }
    .route-card:hover { border-color: var(--accent); }
    .method-badge {
      font-family: var(--mono); font-size: 0.7rem; font-weight: 700;
      padding: 0.2rem 0.5rem; border-radius: 4px; text-align: center;
    }
    .method-page { background: rgba(0,210,255,0.15); color: var(--accent); }
    .method-api  { background: rgba(174,130,255,0.15); color: var(--purple); }
    .route-pattern { font-family: var(--mono); font-size: 0.875rem; }
    .route-file { font-family: var(--mono); font-size: 0.75rem; color: var(--muted); text-align: right; }

    /* Performance Panel */
    .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem; margin-bottom: 1.5rem; }
    .metric-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1.25rem; }
    .metric-label { font-size: 0.75rem; color: var(--muted); margin-bottom: 0.5rem; text-transform: uppercase; letter-spacing: 0.08em; }
    .metric-value { font-size: 2rem; font-weight: 800; font-family: var(--mono); letter-spacing: -0.03em; }
    .metric-value.green { color: var(--green); }
    .metric-value.yellow { color: var(--yellow); }
    .metric-value.red { color: var(--red); }
    .metric-value.blue { color: var(--accent); }
    .metric-unit { font-size: 0.8rem; color: var(--muted); margin-top: 0.2rem; }

    /* Log Panel */
    .log-container {
      background: var(--surface); border: 1px solid var(--border); border-radius: 8px;
      padding: 1rem; font-family: var(--mono); font-size: 0.8rem;
      height: 420px; overflow-y: auto; display: flex; flex-direction: column; gap: 0.25rem;
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

    /* Config Panel */
    .config-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem; }
    .config-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
    .config-card-header {
      padding: 0.75rem 1.25rem; border-bottom: 1px solid var(--border);
      font-size: 0.8rem; font-weight: 600; color: var(--muted);
      text-transform: uppercase; letter-spacing: 0.08em;
    }
    .config-row {
      display: flex; justify-content: space-between; align-items: center;
      padding: 0.6rem 1.25rem; border-bottom: 1px solid var(--border); font-size: 0.85rem;
    }
    .config-row:last-child { border-bottom: none; }
    .config-key { color: var(--muted); font-family: var(--mono); }
    .config-val { font-family: var(--mono); font-weight: 600; }
    .config-val.true  { color: var(--green); }
    .config-val.false { color: var(--red); }
    .config-val.num   { color: var(--accent); }
    .config-val.str   { color: var(--yellow); }

    /* Request log */
    .req-list { display: flex; flex-direction: column; gap: 0.5rem; }
    .req-row {
      display: grid; grid-template-columns: 80px 60px 1fr 80px 80px;
      gap: 0.75rem; align-items: center; padding: 0.6rem 1rem;
      background: var(--surface); border: 1px solid var(--border); border-radius: 6px;
      font-family: var(--mono); font-size: 0.78rem; transition: border-color 0.15s;
    }
    .req-row:hover { border-color: #333; }
    .req-method { font-weight: 700; color: var(--accent); }
    .req-status { font-weight: 700; }
    .req-status.s2 { color: var(--green); }
    .req-status.s4 { color: var(--yellow); }
    .req-status.s5 { color: var(--red); }
    .req-path { color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .req-dur { color: var(--muted); text-align: right; }
    .req-time { color: var(--muted); font-size: 0.7rem; text-align: right; }

    /* Empty states */
    .empty { text-align: center; padding: 4rem 2rem; color: var(--muted); }
    .empty-icon { font-size: 3rem; margin-bottom: 1rem; }
    .empty-text { font-size: 0.9rem; }

    /* ══════ NEW PANEL STYLES ══════ */

    /* Generic form elements */
    .form-group { margin-bottom: 1rem; }
    .form-label { display: block; font-size: 0.72rem; color: var(--muted); margin-bottom: 0.35rem; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 600; }
    .form-input, .form-textarea {
      width: 100%; background: var(--surface2); border: 1px solid var(--border);
      color: var(--text); padding: 0.55rem 0.9rem; border-radius: 6px;
      font-family: var(--mono); font-size: 0.82rem; outline: none; transition: border-color 0.2s;
    }
    .form-input:focus, .form-textarea:focus { border-color: var(--accent); }
    .form-textarea { resize: vertical; min-height: 70px; line-height: 1.5; }
    .char-count { font-size: 0.68rem; color: var(--muted); text-align: right; margin-top: 0.2rem; font-family: var(--mono); }
    .char-count.warn { color: var(--yellow); }
    .char-count.over { color: var(--red); }

    /* Buttons */
    .btn {
      display: inline-flex; align-items: center; gap: 0.4rem;
      padding: 0.5rem 1.2rem; border-radius: 6px; font-size: 0.8rem;
      font-family: var(--mono); cursor: pointer; border: 1px solid var(--border);
      transition: all 0.2s; background: var(--surface2); color: var(--text);
    }
    .btn:hover { border-color: var(--accent); color: var(--accent); }
    .btn-primary {
      background: linear-gradient(135deg, var(--accent), var(--accent2));
      color: #fff; border: none; font-weight: 600;
    }
    .btn-primary:hover { opacity: 0.9; transform: translateY(-1px); }
    .action-bar { display: flex; gap: 0.75rem; margin-top: 1.25rem; flex-wrap: wrap; }

    /* Section label */
    .section-label { font-size: 0.72rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.08em; font-weight: 600; margin: 1.5rem 0 0.75rem; }
    .section-label:first-child { margin-top: 0; }

    /* Chips / Tags */
    .chip-list { display: flex; flex-wrap: wrap; gap: 0.4rem; margin-bottom: 0.75rem; }
    .chip {
      display: inline-flex; align-items: center; gap: 0.35rem;
      padding: 0.25rem 0.7rem; background: rgba(0,210,255,0.1);
      border: 1px solid rgba(0,210,255,0.2); border-radius: 100px;
      font-size: 0.78rem; font-family: var(--mono); color: var(--accent);
    }
    .chip-x { cursor: pointer; font-size: 0.9rem; line-height: 1; opacity: 0.6; }
    .chip-x:hover { opacity: 1; color: var(--red); }
    .chip-add { display: flex; gap: 0.5rem; }
    .chip-add input { flex: 1; }
    .chip-add button { white-space: nowrap; }

    /* ── Colors Panel ── */
    .theme-presets { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-bottom: 1.25rem; }
    .theme-btn {
      padding: 0.4rem 1rem; border-radius: 100px; font-size: 0.78rem;
      cursor: pointer; border: 2px solid var(--border); background: var(--surface);
      color: var(--text); font-family: var(--font); transition: all 0.2s;
    }
    .theme-btn:hover { border-color: var(--accent); }
    .theme-btn.active { border-color: var(--accent); background: rgba(0,210,255,0.1); color: var(--accent); }
    .color-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.6rem; }
    .color-row {
      display: flex; align-items: center; gap: 0.6rem;
      padding: 0.45rem 0.7rem; background: var(--surface);
      border: 1px solid var(--border); border-radius: 6px;
    }
    .color-row label { flex: 1; font-size: 0.78rem; color: var(--muted); text-transform: capitalize; }
    .color-row input[type="color"] { width: 30px; height: 30px; border: none; border-radius: 4px; cursor: pointer; background: none; padding: 0; }
    .color-row input[type="text"] {
      width: 76px; background: var(--surface2); border: 1px solid var(--border);
      color: var(--text); padding: 0.25rem 0.4rem; border-radius: 4px;
      font-family: var(--mono); font-size: 0.72rem; text-align: center;
    }
    .theme-preview {
      background: var(--surface); border: 1px solid var(--border);
      border-radius: 8px; padding: 1.5rem; margin-top: 1.25rem;
    }
    .preview-mock { border-radius: 8px; padding: 1.5rem; transition: all 0.3s; }
    .preview-mock h3 { margin-bottom: 0.5rem; font-size: 1.1rem; }
    .preview-mock p { margin-bottom: 1rem; font-size: 0.85rem; opacity: 0.8; }
    .preview-btns { display: flex; gap: 0.5rem; }
    .preview-pbtn {
      padding: 0.4rem 1rem; border-radius: 6px; font-size: 0.8rem;
      border: none; cursor: pointer; font-family: var(--font); font-weight: 600;
    }
    .css-output {
      background: var(--surface); border: 1px solid var(--border); border-radius: 8px;
      padding: 1rem; font-family: var(--mono); font-size: 0.75rem;
      line-height: 1.7; color: var(--green); margin-top: 1rem;
      max-height: 200px; overflow-y: auto; white-space: pre;
    }

    /* ── SEO Panel ── */
    .seo-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
    .seo-col { min-width: 0; }
    .serp-preview {
      background: #fff; border-radius: 8px; padding: 1rem 1.25rem;
      margin-top: 1rem; border: 1px solid var(--border);
    }
    .serp-title { color: #1a0dab; font-size: 1.1rem; font-family: arial, sans-serif; margin-bottom: 0.15rem; }
    .serp-url { color: #006621; font-size: 0.82rem; font-family: arial, sans-serif; margin-bottom: 0.3rem; }
    .serp-desc { color: #545454; font-size: 0.82rem; font-family: arial, sans-serif; line-height: 1.45; }
    .meta-output {
      background: var(--surface); border: 1px solid var(--border); border-radius: 8px;
      padding: 1rem; font-family: var(--mono); font-size: 0.72rem;
      line-height: 1.8; color: var(--text); margin-top: 1rem;
      white-space: pre-wrap; word-break: break-all; max-height: 220px; overflow-y: auto;
    }

    /* ── WASM Panel ── */
    .wasm-options { display: flex; flex-direction: column; gap: 0.75rem; }
    .wasm-option {
      display: flex; align-items: flex-start; gap: 1rem;
      padding: 1rem 1.25rem; background: var(--surface);
      border: 2px solid var(--border); border-radius: 8px;
      cursor: pointer; transition: all 0.2s;
    }
    .wasm-option:hover { border-color: #333; }
    .wasm-option.active { border-color: var(--accent); background: rgba(0,210,255,0.05); }
    .wasm-option input[type="radio"] { margin-top: 0.25rem; accent-color: var(--accent); }
    .wasm-opt-body { flex: 1; }
    .wasm-opt-title { font-weight: 700; font-size: 0.95rem; margin-bottom: 0.2rem; }
    .wasm-opt-desc { font-size: 0.78rem; color: var(--muted); line-height: 1.5; }
    .wasm-status {
      display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 0.75rem; margin-top: 1.25rem;
    }
    .wasm-stat {
      background: var(--surface); border: 1px solid var(--border);
      border-radius: 8px; padding: 0.75rem 1rem;
    }
    .wasm-stat-label { font-size: 0.7rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.06em; }
    .wasm-stat-val { font-family: var(--mono); font-size: 1rem; font-weight: 700; margin-top: 0.25rem; }
    .code-block {
      background: var(--surface); border: 1px solid var(--border);
      border-radius: 8px; overflow: hidden; margin-top: 1rem;
    }
    .code-header {
      display: flex; justify-content: space-between; align-items: center;
      padding: 0.5rem 1rem; border-bottom: 1px solid var(--border);
      font-size: 0.72rem; color: var(--muted); font-family: var(--mono);
    }
    .code-content {
      padding: 1rem; font-family: var(--mono); font-size: 0.78rem;
      line-height: 1.6; overflow-x: auto; white-space: pre; color: var(--text);
    }

    /* ── Hooks Panel ── */
    .hook-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0.75rem; margin-bottom: 1.25rem; }
    .hook-card {
      background: var(--surface); border: 1px solid var(--border);
      border-radius: 8px; padding: 0.9rem; cursor: pointer; transition: all 0.2s;
    }
    .hook-card:hover { border-color: var(--accent); }
    .hook-card.active { border-color: var(--accent); background: rgba(0,210,255,0.05); }
    .hook-card h4 { font-family: var(--mono); font-size: 0.85rem; color: var(--accent); margin-bottom: 0.2rem; }
    .hook-card p { font-size: 0.75rem; color: var(--muted); line-height: 1.4; }
    .hook-demo {
      background: var(--surface); border: 1px solid var(--border);
      border-radius: 8px; padding: 1.25rem; margin-top: 1rem;
    }
    .demo-label { font-size: 0.72rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.08em; margin-bottom: 0.75rem; }
    .demo-counter { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
    .demo-value { font-size: 2.5rem; font-weight: 800; font-family: var(--mono); color: var(--accent); min-width: 60px; text-align: center; }
    .demo-btn {
      width: 36px; height: 36px; border-radius: 50%; border: 1px solid var(--border);
      background: var(--surface2); color: var(--text); font-size: 1.2rem;
      cursor: pointer; display: flex; align-items: center; justify-content: center; transition: all 0.2s;
    }
    .demo-btn:hover { border-color: var(--accent); color: var(--accent); }
    .demo-effect-log {
      font-family: var(--mono); font-size: 0.72rem; color: var(--muted);
      max-height: 100px; overflow-y: auto; border-top: 1px solid var(--border);
      padding-top: 0.75rem; margin-top: 0.75rem;
    }
    .state-inspector {
      background: var(--surface); border: 1px solid var(--border);
      border-radius: 8px; padding: 1rem; margin-top: 1rem;
    }
    .state-entry {
      display: flex; justify-content: space-between; padding: 0.35rem 0;
      border-bottom: 1px solid var(--border); font-family: var(--mono); font-size: 0.78rem;
    }
    .state-entry:last-child { border-bottom: none; }
    .state-key { color: var(--purple); }
    .state-val { color: var(--green); }

    /* ── About / Landing Panel ── */
    .about-hero {
      text-align: center; padding: 2.5rem 2rem;
      background: linear-gradient(135deg, rgba(0,210,255,0.08), rgba(123,47,247,0.08));
      border-radius: 12px; margin-bottom: 1.5rem; border: 1px solid var(--border);
    }
    .about-hero h2 { font-size: 1.6rem; margin-bottom: 0.4rem; }
    .about-hero p { color: var(--muted); font-size: 0.9rem; }
    .social-links { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0.75rem; margin-bottom: 1.5rem; }
    .social-link {
      display: flex; align-items: center; gap: 0.75rem; padding: 1rem;
      background: var(--surface); border: 1px solid var(--border); border-radius: 8px;
      text-decoration: none; color: var(--text); transition: all 0.2s; cursor: pointer;
    }
    .social-link:hover { border-color: var(--accent); transform: translateY(-2px); }
    .social-icon { font-size: 1.5rem; }
    .social-name { font-weight: 600; font-size: 0.88rem; }
    .social-handle { font-size: 0.78rem; color: var(--muted); font-family: var(--mono); }
    .landing-form { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
    .landing-preview {
      margin-top: 1rem; border: 1px solid var(--border); border-radius: 8px;
      overflow: hidden; background: #fff;
    }
    .landing-preview iframe { width: 100%; height: 400px; border: none; }

    /* Toast */
    .toast {
      position: fixed; bottom: 2rem; right: 2rem;
      background: var(--surface2); border: 1px solid var(--green);
      padding: 0.75rem 1.5rem; border-radius: 8px;
      font-size: 0.82rem; font-family: var(--mono); color: var(--green);
      transform: translateY(80px); opacity: 0; transition: all 0.3s ease; z-index: 1000;
      pointer-events: none;
    }
    .toast.show { transform: translateY(0); opacity: 1; }

    /* Sparkline chart */
    .sparkline { display: flex; align-items: flex-end; gap: 2px; height: 60px; margin-top: 1rem; }
    .spark-bar { flex: 1; background: var(--accent); border-radius: 2px 2px 0 0; min-height: 2px; transition: height 0.3s; opacity: 0.7; }
    .spark-bar:last-child { opacity: 1; }

    /* Scrollbar */
    ::-webkit-scrollbar { width: 6px; }
    ::-webkit-scrollbar-track { background: var(--bg); }
    ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
    ::-webkit-scrollbar-thumb:hover { background: var(--muted); }

    @media (max-width: 768px) {
      .main { grid-template-columns: 1fr; }
      aside { display: none; }
      .color-grid { grid-template-columns: 1fr; }
      .seo-grid { grid-template-columns: 1fr; }
      .landing-form { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <div class="logo">
      <span>NexGo</span>
      <span class="logo-badge">DEV TOOLS</span>
    </div>
    <div class="header-actions">
      <div class="hmr-indicator" id="hmrStatus">
        <div class="hmr-dot"></div>
        <span>HMR connected</span>
      </div>
      <button class="btn-reload" onclick="triggerReload()">Reload</button>
    </div>
  </header>

  <div class="main">
    <aside>
      <div class="nav-group">
        <div class="nav-label">Inspect</div>
        <div class="nav-item active" onclick="showPanel('routes', this)">
          <span class="icon">R</span> Routes
        </div>
        <div class="nav-item" onclick="showPanel('requests', this)">
          <span class="icon">N</span> Requests
        </div>
        <div class="nav-item" onclick="showPanel('logs', this)">
          <span class="icon">L</span> Logs
        </div>
      </div>
      <div class="nav-group">
        <div class="nav-label">Design</div>
        <div class="nav-item" onclick="showPanel('colors', this)">
          <span class="icon">C</span> Colors & Themes
        </div>
        <div class="nav-item" onclick="showPanel('seo', this)">
          <span class="icon">S</span> SEO Manager
        </div>
      </div>
      <div class="nav-group">
        <div class="nav-label">Tools</div>
        <div class="nav-item" onclick="showPanel('wasm', this)">
          <span class="icon">W</span> Go WASM
        </div>
        <div class="nav-item" onclick="showPanel('hooks', this)">
          <span class="icon">H</span> Hooks & State
        </div>
      </div>
      <div class="nav-group">
        <div class="nav-label">System</div>
        <div class="nav-item" onclick="showPanel('perf', this)">
          <span class="icon">P</span> Performance
        </div>
        <div class="nav-item" onclick="showPanel('config', this)">
          <span class="icon">G</span> Config
        </div>
      </div>
      <div class="nav-group">
        <div class="nav-label">About</div>
        <div class="nav-item" onclick="showPanel('landing', this)">
          <span class="icon">@</span> Landing Page
        </div>
      </div>
    </aside>

    <div class="content">

      <!-- ════ Routes Panel ════ -->
      <div class="panel active" id="panel-routes">
        <div class="panel-title">Routes <small id="routeCount">loading...</small></div>
        <div class="route-list" id="routeList">
          <div class="empty"><div class="empty-icon">...</div><div class="empty-text">Loading routes...</div></div>
        </div>
      </div>

      <!-- ════ Requests Panel ════ -->
      <div class="panel" id="panel-requests">
        <div class="panel-title">Live Requests <small id="reqCount">0 total</small></div>
        <div class="req-list" id="reqList">
          <div class="empty"><div class="empty-icon">~</div><div class="empty-text">No requests yet - browse your app to see them here</div></div>
        </div>
      </div>

      <!-- ════ Logs Panel ════ -->
      <div class="panel" id="panel-logs">
        <div class="panel-title">Server Logs</div>
        <div class="log-container" id="logContainer">
          <div class="log-empty">Logs will appear here when the server handles requests</div>
        </div>
      </div>

      <!-- ════ Colors & Themes Panel ════ -->
      <div class="panel" id="panel-colors">
        <div class="panel-title">Colors & Themes</div>
        <div class="section-label">Theme Presets</div>
        <div class="theme-presets" id="themePresets"></div>
        <div class="section-label">Custom Colors</div>
        <div class="color-grid" id="colorGrid"></div>
        <div class="section-label">Live Preview</div>
        <div class="theme-preview">
          <div class="preview-mock" id="colorPreview">
            <h3>Sample Heading</h3>
            <p>This is a preview of your color theme applied to a sample card. Customize the colors above to see changes in real time.</p>
            <div class="preview-btns">
              <button class="preview-pbtn" id="prevBtnPrimary">Primary</button>
              <button class="preview-pbtn" id="prevBtnSecondary">Secondary</button>
            </div>
          </div>
        </div>
        <div class="section-label">CSS Output</div>
        <div class="css-output" id="cssOutput"></div>
        <div class="action-bar">
          <button class="btn btn-primary" onclick="copyCSSVars()">Copy CSS Variables</button>
          <button class="btn" onclick="resetColors()">Reset to Default</button>
        </div>
      </div>

      <!-- ════ SEO Manager Panel ════ -->
      <div class="panel" id="panel-seo">
        <div class="panel-title">SEO Manager</div>
        <div class="seo-grid">
          <div class="seo-col">
            <div class="form-group">
              <label class="form-label">Page Title</label>
              <input class="form-input" id="seoTitle" type="text" placeholder="My Awesome Site" oninput="updateSEO()">
              <div class="char-count" id="seoTitleCount">0 / 60</div>
            </div>
            <div class="form-group">
              <label class="form-label">Meta Description</label>
              <textarea class="form-input form-textarea" id="seoDesc" placeholder="A brief description of your page..." oninput="updateSEO()"></textarea>
              <div class="char-count" id="seoDescCount">0 / 160</div>
            </div>
            <div class="form-group">
              <label class="form-label">Canonical URL</label>
              <input class="form-input" id="seoUrl" type="text" placeholder="https://example.com/page" oninput="updateSEO()">
            </div>
            <div class="form-group">
              <label class="form-label">OG Image URL</label>
              <input class="form-input" id="seoOgImage" type="text" placeholder="https://example.com/og-image.png" oninput="updateSEO()">
            </div>
          </div>
          <div class="seo-col">
            <div class="section-label" style="margin-top:0">Keywords</div>
            <div class="chip-list" id="keywordList"></div>
            <div class="chip-add">
              <input class="form-input" id="keywordInput" type="text" placeholder="Add keyword..." onkeydown="if(event.key==='Enter')addKeyword()">
              <button class="btn" onclick="addKeyword()">Add</button>
            </div>

            <div class="section-label">Google SERP Preview</div>
            <div class="serp-preview">
              <div class="serp-title" id="serpTitle">Page Title</div>
              <div class="serp-url" id="serpUrl">https://example.com</div>
              <div class="serp-desc" id="serpDesc">Your meta description will appear here...</div>
            </div>
          </div>
        </div>
        <div class="section-label">Generated Meta Tags</div>
        <div class="meta-output" id="metaOutput">&lt;!-- Fill in the fields above to generate meta tags --&gt;</div>
        <div class="action-bar">
          <button class="btn btn-primary" onclick="copyMetaTags()">Copy Meta Tags</button>
          <button class="btn" onclick="clearSEO()">Clear All</button>
        </div>
      </div>

      <!-- ════ Go WebAssembly Panel ════ -->
      <div class="panel" id="panel-wasm">
        <div class="panel-title">Go WebAssembly</div>
        <div class="section-label">Rendering Mode</div>
        <div class="wasm-options" id="wasmOptions">
          <div class="wasm-option active" onclick="setWasmMode('js', this)">
            <input type="radio" name="wasmMode" checked>
            <div class="wasm-opt-body">
              <div class="wasm-opt-title">JavaScript Only</div>
              <div class="wasm-opt-desc">Standard client-side JS for all interactions. Smallest bundle size and fastest initial load. Best for most websites and apps.</div>
            </div>
          </div>
          <div class="wasm-option" onclick="setWasmMode('wasm', this)">
            <input type="radio" name="wasmMode">
            <div class="wasm-opt-body">
              <div class="wasm-opt-title">Go WebAssembly</div>
              <div class="wasm-opt-desc">Compile Go code to WebAssembly and run it in the browser. Access Go standard library, goroutines, and type safety in the browser.</div>
            </div>
          </div>
          <div class="wasm-option" onclick="setWasmMode('hybrid', this)">
            <input type="radio" name="wasmMode">
            <div class="wasm-opt-body">
              <div class="wasm-opt-title">Hybrid (JS + Go WASM)</div>
              <div class="wasm-opt-desc">Use JavaScript for UI and Go WASM for computation-heavy tasks. Best of both worlds - fast UI with powerful Go backend in browser.</div>
            </div>
          </div>
        </div>

        <div class="section-label">Status</div>
        <div class="wasm-status">
          <div class="wasm-stat">
            <div class="wasm-stat-label">Mode</div>
            <div class="wasm-stat-val" id="wasmModeDisplay" style="color:var(--accent)">JavaScript</div>
          </div>
          <div class="wasm-stat">
            <div class="wasm-stat-label">WASM Support</div>
            <div class="wasm-stat-val" id="wasmSupport" style="color:var(--green)">Checking...</div>
          </div>
          <div class="wasm-stat">
            <div class="wasm-stat-label">Est. Bundle</div>
            <div class="wasm-stat-val" id="wasmSize" style="color:var(--yellow)">~5 KB</div>
          </div>
          <div class="wasm-stat">
            <div class="wasm-stat-label">Runtime</div>
            <div class="wasm-stat-val" id="wasmRuntime" style="color:var(--purple)">V8/SpiderMonkey</div>
          </div>
        </div>

        <div class="section-label">Quick Start Code</div>
        <div id="wasmCodeBlock"></div>
      </div>

      <!-- ════ Hooks & State Panel ════ -->
      <div class="panel" id="panel-hooks">
        <div class="panel-title">Hooks & State</div>
        <div class="section-label">Available Hooks</div>
        <div class="hook-cards">
          <div class="hook-card active" onclick="showHookSnippet('useState')">
            <h4>useState</h4>
            <p>Create reactive state variables with automatic UI updates</p>
          </div>
          <div class="hook-card" onclick="showHookSnippet('useEffect')">
            <h4>useEffect</h4>
            <p>Run side effects when dependencies change</p>
          </div>
          <div class="hook-card" onclick="showHookSnippet('useMemo')">
            <h4>useMemo</h4>
            <p>Memoize expensive computations for performance</p>
          </div>
          <div class="hook-card" onclick="showHookSnippet('useRef')">
            <h4>useRef</h4>
            <p>Hold mutable references to DOM elements</p>
          </div>
          <div class="hook-card" onclick="showHookSnippet('useLazy')">
            <h4>useLazy</h4>
            <p>Lazy load components and modules on demand</p>
          </div>
        </div>

        <div class="section-label">Live Demo - Counter with Hooks</div>
        <div class="hook-demo">
          <div class="demo-label">useState + useEffect</div>
          <div class="demo-counter">
            <button class="demo-btn" onclick="demoDecrement()">-</button>
            <div class="demo-value" id="demoCount">0</div>
            <button class="demo-btn" onclick="demoIncrement()">+</button>
            <button class="demo-btn" onclick="demoReset()" style="font-size:0.7rem;width:auto;padding:0 0.75rem;border-radius:6px">Reset</button>
          </div>
          <div class="demo-effect-log" id="demoEffectLog"></div>
        </div>

        <div class="section-label">State Inspector</div>
        <div class="state-inspector" id="stateInspector">
          <div style="color:var(--muted);text-align:center;padding:0.5rem">Use the demo above to register state</div>
        </div>

        <div class="section-label">Code Snippet</div>
        <div id="hookCodeBlock"></div>
      </div>

      <!-- ════ Performance Panel ════ -->
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
            <div class="metric-value green" id="perfAvg">--</div>
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
        <div class="section-label">Requests per Second (last 30s)</div>
        <div class="sparkline" id="perfSparkline"></div>
      </div>

      <!-- ════ Config Panel ════ -->
      <div class="panel" id="panel-config">
        <div class="panel-title">Configuration</div>
        <div class="config-grid" id="configGrid">
          <div class="empty"><div class="empty-icon">...</div><div class="empty-text">Loading config...</div></div>
        </div>
      </div>

      <!-- ════ Landing Page Panel ════ -->
      <div class="panel" id="panel-landing">
        <div class="panel-title">Landing Page Builder</div>
        <div class="about-hero">
          <h2>salmanfaris.dev</h2>
          <p>NexGo Framework - Build fast web apps with Go</p>
        </div>
        <div class="social-links">
          <a class="social-link" href="https://salmanfaris.dev" target="_blank">
            <span class="social-icon">W</span>
            <div>
              <div class="social-name">Website</div>
              <div class="social-handle">salmanfaris.dev</div>
            </div>
          </a>
          <a class="social-link" href="https://instagram.com/salmanfaris.dev" target="_blank">
            <span class="social-icon">IG</span>
            <div>
              <div class="social-name">Instagram</div>
              <div class="social-handle">@salmanfaris.dev</div>
            </div>
          </a>
          <a class="social-link" href="https://github.com/salmanfaris22" target="_blank">
            <span class="social-icon">GH</span>
            <div>
              <div class="social-name">GitHub</div>
              <div class="social-handle">salmanfaris22</div>
            </div>
          </a>
        </div>

        <div class="section-label">Generate Landing Page</div>
        <div class="landing-form">
          <div class="form-group">
            <label class="form-label">Site Name</label>
            <input class="form-input" id="landName" type="text" value="salmanfaris.dev" oninput="updateLandingPreview()">
          </div>
          <div class="form-group">
            <label class="form-label">Tagline</label>
            <input class="form-input" id="landTagline" type="text" value="Developer & Creator" oninput="updateLandingPreview()">
          </div>
          <div class="form-group">
            <label class="form-label">Instagram Handle</label>
            <input class="form-input" id="landIG" type="text" value="@salmanfaris.dev" oninput="updateLandingPreview()">
          </div>
          <div class="form-group">
            <label class="form-label">GitHub Username</label>
            <input class="form-input" id="landGH" type="text" value="salmanfaris22" oninput="updateLandingPreview()">
          </div>
          <div class="form-group">
            <label class="form-label">Website URL</label>
            <input class="form-input" id="landURL" type="text" value="https://salmanfaris.dev" oninput="updateLandingPreview()">
          </div>
          <div class="form-group">
            <label class="form-label">Bio</label>
            <textarea class="form-input form-textarea" id="landBio" oninput="updateLandingPreview()">Full-stack developer building with Go and NexGo. Passionate about web performance and developer tools.</textarea>
          </div>
        </div>
        <div class="action-bar">
          <button class="btn btn-primary" onclick="copyLandingHTML()">Copy Landing Page HTML</button>
          <button class="btn" onclick="previewLanding()">Preview</button>
        </div>
        <div class="landing-preview" id="landingPreview" style="display:none">
          <iframe id="landingFrame" sandbox="allow-same-origin"></iframe>
        </div>
      </div>

    </div>
  </div>

  <div class="toast" id="toast"></div>

  <script>
    // ══════════════════════════════════════
    //  NexGo DevTools - Core
    // ══════════════════════════════════════

    // Utility: escape HTML
    function esc(s) {
      var d = document.createElement('div');
      d.appendChild(document.createTextNode(String(s)));
      return d.innerHTML;
    }

    // Toast notification
    function showToast(msg) {
      var t = document.getElementById('toast');
      t.textContent = msg;
      t.classList.add('show');
      clearTimeout(t._tid);
      t._tid = setTimeout(function() { t.classList.remove('show'); }, 2000);
    }

    // Copy to clipboard
    function copyText(text) {
      if (navigator.clipboard) {
        navigator.clipboard.writeText(text).then(function() {
          showToast('Copied to clipboard!');
        });
      } else {
        var ta = document.createElement('textarea');
        ta.value = text;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
        showToast('Copied to clipboard!');
      }
    }

    // Safe localStorage
    function lsGet(key, fallback) {
      try { var v = localStorage.getItem('nexgo_' + key); return v ? JSON.parse(v) : fallback; }
      catch(e) { return fallback; }
    }
    function lsSet(key, val) {
      try { localStorage.setItem('nexgo_' + key, JSON.stringify(val)); } catch(e) {}
    }

    // ═══ State ═══
    var state = { routes: [], requests: [], logs: [] };

    // ═══ Panel Switching ═══
    function showPanel(name, el) {
      document.querySelectorAll('.panel').forEach(function(p) { p.classList.remove('active'); });
      document.querySelectorAll('.nav-item').forEach(function(n) { n.classList.remove('active'); });
      document.getElementById('panel-' + name).classList.add('active');
      if (el) el.classList.add('active');
    }

    // ═══ Routes ═══
    function loadRoutes() {
      fetch('/_nexgo/routes').then(function(res) { return res.json(); }).then(function(routes) {
        state.routes = routes || [];
        renderRoutes();
        document.getElementById('perfRoutes').textContent = state.routes.length;
      }).catch(function(e) {
        console.error('Failed to load routes:', e);
      });
    }

    function renderRoutes() {
      var list = document.getElementById('routeList');
      var count = document.getElementById('routeCount');
      var routes = state.routes;
      count.textContent = routes.length + ' route' + (routes.length !== 1 ? 's' : '');
      if (!routes.length) {
        list.innerHTML = '<div class="empty"><div class="empty-icon">--</div><div class="empty-text">No routes found in pages/</div></div>';
        return;
      }
      list.innerHTML = routes.map(function(r) {
        var isApi = r.type === 'api';
        var badge = isApi
          ? '<span class="method-badge method-api">API</span>'
          : '<span class="method-badge method-page">PAGE</span>';
        var file = r.file ? r.file.replace(/.*\/pages\//, 'pages/') : '';
        return '<div class="route-card" data-route="' + esc(r.pattern) + '">' +
          badge +
          '<span class="route-pattern">' + esc(r.pattern) + '</span>' +
          '<span class="route-file">' + esc(file) + '</span>' +
        '</div>';
      }).join('');
    }

    // Route click delegation
    document.addEventListener('click', function(e) {
      var card = e.target.closest('.route-card');
      if (card && card.dataset.route) {
        window.open(card.dataset.route, '_blank');
      }
    });

    // ═══ Request Tracking ═══
    var reqTotal = 0, reqErrors = 0, reqTotalMs = 0;
    var rpsHistory = [];

    function addRequest(method, status, path, ms) {
      reqTotal++;
      reqTotalMs += ms;
      if (status >= 400) reqErrors++;

      var entry = { method: method, status: status, path: path, ms: ms, time: new Date() };
      state.requests.unshift(entry);
      if (state.requests.length > 200) state.requests.pop();

      // Track RPS
      rpsHistory.push(Date.now());
      while (rpsHistory.length && rpsHistory[0] < Date.now() - 30000) rpsHistory.shift();

      renderRequests();
      updatePerfMetrics();
      addLog(status >= 500 ? 'error' : status >= 400 ? 'warn' : 'ok', method, status, path, ms);
    }

    function renderRequests() {
      var list = document.getElementById('reqList');
      var count = document.getElementById('reqCount');
      count.textContent = reqTotal + ' total';
      if (!state.requests.length) {
        list.innerHTML = '<div class="empty"><div class="empty-icon">~</div><div class="empty-text">No requests yet</div></div>';
        return;
      }
      list.innerHTML = state.requests.slice(0, 50).map(function(r) {
        var sc = r.status >= 500 ? 's5' : r.status >= 400 ? 's4' : 's2';
        var t = r.time.toLocaleTimeString('en', { hour12: false });
        return '<div class="req-row">' +
          '<span class="req-method">' + esc(r.method) + '</span>' +
          '<span class="req-status ' + sc + '">' + r.status + '</span>' +
          '<span class="req-path">' + esc(r.path) + '</span>' +
          '<span class="req-dur">' + r.ms + 'ms</span>' +
          '<span class="req-time">' + t + '</span>' +
        '</div>';
      }).join('');
    }

    // ═══ Logs ═══
    function addLog(level, method, status, path, ms) {
      var container = document.getElementById('logContainer');
      var empty = container.querySelector('.log-empty');
      if (empty) empty.remove();
      var t = new Date().toLocaleTimeString('en', { hour12: false });
      var sc = status >= 500 ? 'e5' : status >= 400 ? 'e4' : '';
      var entry = document.createElement('div');
      entry.className = 'log-entry';
      entry.innerHTML =
        '<span class="log-time">' + t + '</span>' +
        '<span class="log-level ' + esc(level) + '">' + esc(level.toUpperCase()) + '</span>' +
        '<span class="log-status ' + sc + '">' + status + '</span>' +
        '<span class="log-path">' + esc(method) + ' ' + esc(path) + '</span>' +
        '<span class="log-msg" style="color:var(--muted)">' + ms + 'ms</span>';
      container.insertBefore(entry, container.firstChild);
      while (container.children.length > 500) container.removeChild(container.lastChild);
    }

    // ═══ Performance Metrics ═══
    function updatePerfMetrics() {
      document.getElementById('perfTotal').textContent = reqTotal;
      document.getElementById('perfAvg').textContent = reqTotal ? Math.round(reqTotalMs / reqTotal) : '--';
      var errPct = reqTotal ? Math.round((reqErrors / reqTotal) * 100) : 0;
      document.getElementById('perfErr').textContent = errPct + '%';
      document.getElementById('perfErr').className = 'metric-value ' + (errPct > 10 ? 'red' : errPct > 0 ? 'yellow' : 'green');
    }

    // Sparkline
    function updateSparkline() {
      var el = document.getElementById('perfSparkline');
      if (!el) return;
      var now = Date.now();
      var buckets = [];
      for (var i = 0; i < 30; i++) {
        var start = now - (30 - i) * 1000;
        var end = start + 1000;
        var c = 0;
        for (var j = 0; j < rpsHistory.length; j++) {
          if (rpsHistory[j] >= start && rpsHistory[j] < end) c++;
        }
        buckets.push(c);
      }
      var max = Math.max.apply(null, buckets) || 1;
      el.innerHTML = buckets.map(function(v) {
        var h = Math.max(2, (v / max) * 60);
        return '<div class="spark-bar" style="height:' + h + 'px"></div>';
      }).join('');
    }
    setInterval(updateSparkline, 1000);

    // ═══ Config ═══
    function loadConfig() {
      var cfg = {
        Server: { port: 3000, host: 'localhost', mode: 'development' },
        Build: { compression: true, minify: true, hotReload: true },
        Paths: { pages: 'pages/', static: 'static/', layouts: 'layouts/', components: 'components/' }
      };
      var grid = document.getElementById('configGrid');
      grid.innerHTML = Object.keys(cfg).map(function(section) {
        var values = cfg[section];
        var rows = Object.keys(values).map(function(k) {
          var v = values[k];
          var cls = typeof v === 'boolean' ? v.toString() : typeof v === 'number' ? 'num' : 'str';
          return '<div class="config-row">' +
            '<span class="config-key">' + esc(k) + '</span>' +
            '<span class="config-val ' + cls + '">' + esc(JSON.stringify(v)) + '</span>' +
          '</div>';
        }).join('');
        return '<div class="config-card">' +
          '<div class="config-card-header">' + esc(section) + '</div>' +
          rows +
        '</div>';
      }).join('');
    }

    // ══════════════════════════════════════
    //  Colors & Themes
    // ══════════════════════════════════════

    var colorKeys = ['primary','secondary','accent','background','surface','text','muted','border','success','warning','error'];

    var colorThemes = {
      Light:    { primary:'#2563eb',secondary:'#7c3aed',accent:'#06b6d4',background:'#ffffff',surface:'#f8fafc',text:'#0f172a',muted:'#64748b',border:'#e2e8f0',success:'#22c55e',warning:'#eab308',error:'#ef4444' },
      Dark:     { primary:'#3b82f6',secondary:'#8b5cf6',accent:'#22d3ee',background:'#0f172a',surface:'#1e293b',text:'#f1f5f9',muted:'#94a3b8',border:'#334155',success:'#4ade80',warning:'#facc15',error:'#f87171' },
      Ocean:    { primary:'#0284c7',secondary:'#0891b2',accent:'#06b6d4',background:'#0c4a6e',surface:'#164e63',text:'#e0f2fe',muted:'#7dd3fc',border:'#155e75',success:'#34d399',warning:'#fbbf24',error:'#fb7185' },
      Sunset:   { primary:'#ea580c',secondary:'#dc2626',accent:'#f59e0b',background:'#1c1917',surface:'#292524',text:'#fef3c7',muted:'#a8a29e',border:'#44403c',success:'#a3e635',warning:'#fbbf24',error:'#ef4444' },
      Forest:   { primary:'#16a34a',secondary:'#059669',accent:'#84cc16',background:'#052e16',surface:'#14532d',text:'#dcfce7',muted:'#86efac',border:'#166534',success:'#4ade80',warning:'#fde047',error:'#f87171' },
      Midnight: { primary:'#6366f1',secondary:'#8b5cf6',accent:'#a78bfa',background:'#020617',surface:'#0f172a',text:'#e2e8f0',muted:'#64748b',border:'#1e293b',success:'#34d399',warning:'#fbbf24',error:'#fb7185' }
    };

    var currentColors = {};
    var activeTheme = 'Dark';

    function initColors() {
      // Load saved or default
      currentColors = lsGet('colors', Object.assign({}, colorThemes.Dark));
      activeTheme = lsGet('activeTheme', 'Dark');
      renderThemePresets();
      renderColorGrid();
      updateColorPreview();
      updateCSSOutput();
    }

    function renderThemePresets() {
      var el = document.getElementById('themePresets');
      el.innerHTML = Object.keys(colorThemes).map(function(name) {
        var cls = name === activeTheme ? 'theme-btn active' : 'theme-btn';
        return '<button class="' + cls + '" onclick="applyTheme(\'' + name + '\')">' + name + '</button>';
      }).join('');
    }

    function renderColorGrid() {
      var el = document.getElementById('colorGrid');
      el.innerHTML = colorKeys.map(function(key) {
        var val = currentColors[key] || '#000000';
        return '<div class="color-row">' +
          '<label>' + key + '</label>' +
          '<input type="color" value="' + esc(val) + '" onchange="setColor(\'' + key + '\', this.value)" data-key="' + key + '">' +
          '<input type="text" value="' + esc(val) + '" onchange="setColor(\'' + key + '\', this.value)" data-key="' + key + '-hex">' +
        '</div>';
      }).join('');
    }

    function applyTheme(name) {
      activeTheme = name;
      currentColors = Object.assign({}, colorThemes[name]);
      lsSet('colors', currentColors);
      lsSet('activeTheme', activeTheme);
      renderThemePresets();
      renderColorGrid();
      updateColorPreview();
      updateCSSOutput();
    }

    function setColor(key, val) {
      currentColors[key] = val;
      activeTheme = 'Custom';
      lsSet('colors', currentColors);
      lsSet('activeTheme', activeTheme);
      // Sync picker and hex
      var picker = document.querySelector('input[data-key="' + key + '"]');
      var hex = document.querySelector('input[data-key="' + key + '-hex"]');
      if (picker) picker.value = val;
      if (hex) hex.value = val;
      renderThemePresets();
      updateColorPreview();
      updateCSSOutput();
    }

    function updateColorPreview() {
      var c = currentColors;
      var el = document.getElementById('colorPreview');
      el.style.background = c.background || '#0f172a';
      el.style.color = c.text || '#f1f5f9';
      el.style.border = '1px solid ' + (c.border || '#334155');
      var btn1 = document.getElementById('prevBtnPrimary');
      var btn2 = document.getElementById('prevBtnSecondary');
      if (btn1) { btn1.style.background = c.primary || '#3b82f6'; btn1.style.color = '#fff'; }
      if (btn2) { btn2.style.background = c.secondary || '#8b5cf6'; btn2.style.color = '#fff'; }
    }

    function updateCSSOutput() {
      var c = currentColors;
      var lines = [':root {'];
      colorKeys.forEach(function(key) {
        if (c[key]) lines.push('  --color-' + key + ': ' + c[key] + ';');
      });
      lines.push('}');
      document.getElementById('cssOutput').textContent = lines.join('\n');
    }

    function copyCSSVars() {
      copyText(document.getElementById('cssOutput').textContent);
    }

    function resetColors() {
      applyTheme('Dark');
    }

    // ══════════════════════════════════════
    //  SEO Manager
    // ══════════════════════════════════════

    var seoKeywords = lsGet('seo_keywords', []);

    function initSEO() {
      var saved = lsGet('seo_data', {});
      if (saved.title) document.getElementById('seoTitle').value = saved.title;
      if (saved.desc) document.getElementById('seoDesc').value = saved.desc;
      if (saved.url) document.getElementById('seoUrl').value = saved.url;
      if (saved.ogImage) document.getElementById('seoOgImage').value = saved.ogImage;
      renderKeywords();
      updateSEO();
    }

    function updateSEO() {
      var title = document.getElementById('seoTitle').value;
      var desc = document.getElementById('seoDesc').value;
      var url = document.getElementById('seoUrl').value;
      var ogImage = document.getElementById('seoOgImage').value;

      // Char counts
      var tc = document.getElementById('seoTitleCount');
      tc.textContent = title.length + ' / 60';
      tc.className = 'char-count' + (title.length > 60 ? ' over' : title.length > 50 ? ' warn' : '');

      var dc = document.getElementById('seoDescCount');
      dc.textContent = desc.length + ' / 160';
      dc.className = 'char-count' + (desc.length > 160 ? ' over' : desc.length > 140 ? ' warn' : '');

      // SERP Preview
      document.getElementById('serpTitle').textContent = title || 'Page Title';
      document.getElementById('serpUrl').textContent = url || 'https://example.com';
      document.getElementById('serpDesc').textContent = desc || 'Your meta description will appear here...';

      // Meta tags
      var meta = [];
      if (title) meta.push('<title>' + esc(title) + '</title>');
      if (title) meta.push('<meta name="title" content="' + esc(title) + '">');
      if (desc) meta.push('<meta name="description" content="' + esc(desc) + '">');
      if (seoKeywords.length) meta.push('<meta name="keywords" content="' + esc(seoKeywords.join(', ')) + '">');
      if (url) meta.push('<link rel="canonical" href="' + esc(url) + '">');
      meta.push('');
      meta.push('<!-- Open Graph -->');
      if (title) meta.push('<meta property="og:title" content="' + esc(title) + '">');
      if (desc) meta.push('<meta property="og:description" content="' + esc(desc) + '">');
      if (url) meta.push('<meta property="og:url" content="' + esc(url) + '">');
      if (ogImage) meta.push('<meta property="og:image" content="' + esc(ogImage) + '">');
      meta.push('<meta property="og:type" content="website">');
      meta.push('');
      meta.push('<!-- Twitter -->');
      meta.push('<meta name="twitter:card" content="summary_large_image">');
      if (title) meta.push('<meta name="twitter:title" content="' + esc(title) + '">');
      if (desc) meta.push('<meta name="twitter:description" content="' + esc(desc) + '">');
      if (ogImage) meta.push('<meta name="twitter:image" content="' + esc(ogImage) + '">');

      document.getElementById('metaOutput').textContent = meta.join('\n');

      // Save
      lsSet('seo_data', { title: title, desc: desc, url: url, ogImage: ogImage });
    }

    function addKeyword() {
      var input = document.getElementById('keywordInput');
      var kw = input.value.trim();
      if (kw && seoKeywords.indexOf(kw) === -1) {
        seoKeywords.push(kw);
        input.value = '';
        lsSet('seo_keywords', seoKeywords);
        renderKeywords();
        updateSEO();
      }
    }

    function removeKeyword(idx) {
      seoKeywords.splice(idx, 1);
      lsSet('seo_keywords', seoKeywords);
      renderKeywords();
      updateSEO();
    }

    function renderKeywords() {
      var el = document.getElementById('keywordList');
      if (!seoKeywords.length) {
        el.innerHTML = '<span style="color:var(--muted);font-size:0.8rem">No keywords added yet</span>';
        return;
      }
      el.innerHTML = seoKeywords.map(function(kw, i) {
        return '<span class="chip">' + esc(kw) +
          '<span class="chip-x" onclick="removeKeyword(' + i + ')">x</span></span>';
      }).join('');
    }

    function copyMetaTags() {
      copyText(document.getElementById('metaOutput').textContent);
    }

    function clearSEO() {
      document.getElementById('seoTitle').value = '';
      document.getElementById('seoDesc').value = '';
      document.getElementById('seoUrl').value = '';
      document.getElementById('seoOgImage').value = '';
      seoKeywords = [];
      lsSet('seo_keywords', []);
      lsSet('seo_data', {});
      renderKeywords();
      updateSEO();
    }

    // ══════════════════════════════════════
    //  Go WebAssembly
    // ══════════════════════════════════════

    var wasmMode = lsGet('wasm_mode', 'js');

    function initWasm() {
      // Check WASM support
      var supported = typeof WebAssembly === 'object';
      document.getElementById('wasmSupport').textContent = supported ? 'Supported' : 'Not Supported';
      document.getElementById('wasmSupport').style.color = supported ? 'var(--green)' : 'var(--red)';

      // Apply saved mode
      setWasmMode(wasmMode);
    }

    function setWasmMode(mode, el) {
      wasmMode = mode;
      lsSet('wasm_mode', mode);

      // Update UI
      document.querySelectorAll('.wasm-option').forEach(function(opt) { opt.classList.remove('active'); });
      document.querySelectorAll('.wasm-option input[type="radio"]').forEach(function(r) { r.checked = false; });
      if (el) {
        el.classList.add('active');
        el.querySelector('input[type="radio"]').checked = true;
      } else {
        var opts = document.querySelectorAll('.wasm-option');
        var idx = mode === 'js' ? 0 : mode === 'wasm' ? 1 : 2;
        opts[idx].classList.add('active');
        opts[idx].querySelector('input[type="radio"]').checked = true;
      }

      // Mode display
      var names = { js: 'JavaScript', wasm: 'Go WebAssembly', hybrid: 'Hybrid (JS + WASM)' };
      var sizes = { js: '~5 KB', wasm: '~2.5 MB', hybrid: '~1.5 MB' };
      var runtimes = { js: 'V8/SpiderMonkey', wasm: 'Go WASM Runtime', hybrid: 'JS + WASM Runtime' };
      document.getElementById('wasmModeDisplay').textContent = names[mode] || mode;
      document.getElementById('wasmSize').textContent = sizes[mode] || '--';
      document.getElementById('wasmRuntime').textContent = runtimes[mode] || '--';

      renderWasmCode(mode);
    }

    function renderWasmCode(mode) {
      var el = document.getElementById('wasmCodeBlock');
      var code = '';
      var title = '';

      if (mode === 'js') {
        title = 'main.js';
        code = '// JavaScript Only Mode\n' +
          '// All client-side logic runs as standard JS\n\n' +
          'document.addEventListener("DOMContentLoaded", function() {\n' +
          '  console.log("NexGo app ready");\n\n' +
          '  // Use NexGo hooks for state management\n' +
          '  var [count, setCount] = NexGo.useState("count", 0);\n\n' +
          '  document.getElementById("btn").onclick = function() {\n' +
          '    setCount(count + 1);\n' +
          '  };\n' +
          '});';
      } else if (mode === 'wasm') {
        title = 'main.go (compiles to WASM)';
        code = 'package main\n\n' +
          'import (\n' +
          '    "fmt"\n' +
          '    "syscall/js"\n' +
          ')\n\n' +
          'func main() {\n' +
          '    // Register Go functions for browser use\n' +
          '    js.Global().Set("goAdd", js.FuncOf(goAdd))\n' +
          '    js.Global().Set("goFetch", js.FuncOf(goFetch))\n\n' +
          '    fmt.Println("Go WASM initialized")\n' +
          '    select {} // Keep alive\n' +
          '}\n\n' +
          'func goAdd(this js.Value, args []js.Value) interface{} {\n' +
          '    return args[0].Int() + args[1].Int()\n' +
          '}\n\n' +
          '// Build: GOOS=js GOARCH=wasm go build -o main.wasm';
      } else {
        title = 'hybrid setup';
        code = '// Hybrid Mode: JS for UI, Go WASM for compute\n\n' +
          '// 1. Load WASM module\n' +
          'var go = new Go();\n' +
          'WebAssembly.instantiateStreaming(\n' +
          '  fetch("/static/main.wasm"), go.importObject\n' +
          ').then(function(result) {\n' +
          '  go.run(result.instance);\n' +
          '  console.log("WASM loaded");\n\n' +
          '  // 2. Use Go functions from JS\n' +
          '  var sum = goAdd(10, 20); // Runs in Go/WASM\n' +
          '  document.getElementById("result").textContent = sum;\n' +
          '});\n\n' +
          '// 3. UI stays in JS for fast interactions\n' +
          'document.getElementById("btn").onclick = function() {\n' +
          '  // Light UI work in JS, heavy compute in WASM\n' +
          '};';
      }

      el.innerHTML = '<div class="code-block">' +
        '<div class="code-header"><span>' + esc(title) + '</span>' +
        '<button class="btn" style="padding:0.2rem 0.6rem;font-size:0.7rem" onclick="copyText(this.closest(\'.code-block\').querySelector(\'.code-content\').textContent)">Copy</button></div>' +
        '<div class="code-content">' + esc(code) + '</div></div>';
    }

    // ══════════════════════════════════════
    //  Hooks & State (Mini Library + Demo)
    // ══════════════════════════════════════

    var _hooksState = {};
    var _hooksEffects = {};
    var _effectLog = [];

    function useState(key, initial) {
      if (!(key in _hooksState)) {
        _hooksState[key] = initial;
      }
      var k = key;
      return [_hooksState[k], function(val) {
        var prev = _hooksState[k];
        _hooksState[k] = typeof val === 'function' ? val(prev) : val;
        renderHooksState();
      }];
    }

    function useEffect(key, fn, deps) {
      var prev = _hooksEffects[key];
      var changed = !prev || !prev.deps || deps.length !== prev.deps.length ||
        deps.some(function(d, i) { return d !== prev.deps[i]; });
      if (changed) {
        if (prev && typeof prev.cleanup === 'function') prev.cleanup();
        var cleanup = fn();
        _hooksEffects[key] = { deps: deps.slice(), cleanup: cleanup };
        var t = new Date().toLocaleTimeString('en', { hour12: false });
        _effectLog.unshift('[' + t + '] Effect "' + key + '" triggered');
        if (_effectLog.length > 15) _effectLog.pop();
        renderEffectLog();
      }
    }

    function useMemo(key, fn, deps) {
      var prev = _hooksEffects['memo_' + key];
      var changed = !prev || !prev.deps || deps.length !== prev.deps.length ||
        deps.some(function(d, i) { return d !== prev.deps[i]; });
      if (changed) {
        var val = fn();
        _hooksEffects['memo_' + key] = { deps: deps.slice(), value: val };
        return val;
      }
      return prev.value;
    }

    function useRef(initial) {
      return { current: initial };
    }

    function useLazy(loader) {
      var loaded = false;
      var result = null;
      return {
        load: function() {
          if (!loaded) {
            result = loader();
            loaded = true;
          }
          return result;
        },
        loaded: function() { return loaded; }
      };
    }

    // Demo
    var demoCountState = useState('count', 0);
    var demoNameState = useState('name', 'NexGo');

    function demoIncrement() {
      demoCountState = useState('count', 0);
      demoCountState[1](_hooksState.count + 1);
      document.getElementById('demoCount').textContent = _hooksState.count;
      useEffect('countLogger', function() {
        return undefined;
      }, [_hooksState.count]);
    }

    function demoDecrement() {
      demoCountState = useState('count', 0);
      demoCountState[1](Math.max(0, _hooksState.count - 1));
      document.getElementById('demoCount').textContent = _hooksState.count;
      useEffect('countLogger', function() {
        return undefined;
      }, [_hooksState.count]);
    }

    function demoReset() {
      _hooksState.count = 0;
      document.getElementById('demoCount').textContent = '0';
      renderHooksState();
      var t = new Date().toLocaleTimeString('en', { hour12: false });
      _effectLog.unshift('[' + t + '] State "count" reset to 0');
      if (_effectLog.length > 15) _effectLog.pop();
      renderEffectLog();
    }

    function renderHooksState() {
      var el = document.getElementById('stateInspector');
      if (!el) return;
      var keys = Object.keys(_hooksState);
      if (!keys.length) {
        el.innerHTML = '<div style="color:var(--muted);text-align:center;padding:0.5rem">No state registered</div>';
        return;
      }
      el.innerHTML = keys.map(function(k) {
        return '<div class="state-entry">' +
          '<span class="state-key">' + esc(k) + '</span>' +
          '<span class="state-val">' + esc(JSON.stringify(_hooksState[k])) + '</span>' +
        '</div>';
      }).join('');
    }

    function renderEffectLog() {
      var el = document.getElementById('demoEffectLog');
      if (!el) return;
      el.innerHTML = _effectLog.map(function(line) {
        return '<div>' + esc(line) + '</div>';
      }).join('');
    }

    // Hook code snippets
    var hookSnippets = {
      useState: {
        title: 'useState - Reactive State',
        code: '// Create reactive state\n' +
          'var result = NexGo.useState("counter", 0);\n' +
          'var count = result[0];    // current value\n' +
          'var setCount = result[1]; // setter function\n\n' +
          '// Update state\n' +
          'setCount(5);           // set directly\n' +
          'setCount(function(prev) { return prev + 1; }); // functional update\n\n' +
          '// State persists across renders\n' +
          'console.log(count); // always current value'
      },
      useEffect: {
        title: 'useEffect - Side Effects',
        code: '// Run effect when dependencies change\n' +
          'NexGo.useEffect("logger", function() {\n' +
          '  console.log("Count changed to:", count);\n\n' +
          '  // Return cleanup function (optional)\n' +
          '  return function() {\n' +
          '    console.log("Cleaning up previous effect");\n' +
          '  };\n' +
          '}, [count]); // dependency array\n\n' +
          '// Runs only when count changes\n' +
          '// Cleanup runs before next effect'
      },
      useMemo: {
        title: 'useMemo - Memoized Computation',
        code: '// Memoize expensive calculations\n' +
          'var expensive = NexGo.useMemo("filtered", function() {\n' +
          '  return items.filter(function(item) {\n' +
          '    return item.score > threshold;\n' +
          '  }).sort(function(a, b) {\n' +
          '    return b.score - a.score;\n' +
          '  });\n' +
          '}, [items, threshold]);\n\n' +
          '// Only recomputes when items or threshold change\n' +
          '// Returns cached result otherwise'
      },
      useRef: {
        title: 'useRef - Mutable References',
        code: '// Create a mutable reference\n' +
          'var inputRef = NexGo.useRef(null);\n\n' +
          '// Attach to DOM element\n' +
          'inputRef.current = document.getElementById("my-input");\n\n' +
          '// Access the element directly\n' +
          'inputRef.current.focus();\n' +
          'inputRef.current.value = "Hello";\n\n' +
          '// Ref changes do NOT trigger re-renders\n' +
          '// Great for DOM access and timers'
      },
      useLazy: {
        title: 'useLazy - Lazy Loading',
        code: '// Lazy load heavy modules\n' +
          'var chart = NexGo.useLazy(function() {\n' +
          '  return import("/static/js/chart-library.js");\n' +
          '});\n\n' +
          '// Load only when needed\n' +
          'document.getElementById("show-chart").onclick = function() {\n' +
          '  chart.load().then(function(module) {\n' +
          '    module.render(data);\n' +
          '  });\n' +
          '};\n\n' +
          '// Check if loaded\n' +
          'if (chart.loaded()) {\n' +
          '  // Already available, use immediately\n' +
          '}'
      }
    };

    function showHookSnippet(name) {
      document.querySelectorAll('.hook-card').forEach(function(c) { c.classList.remove('active'); });
      // Find and activate the card
      document.querySelectorAll('.hook-card').forEach(function(c) {
        if (c.querySelector('h4').textContent === name) c.classList.add('active');
      });

      var snippet = hookSnippets[name];
      if (!snippet) return;

      var el = document.getElementById('hookCodeBlock');
      el.innerHTML = '<div class="code-block">' +
        '<div class="code-header"><span>' + esc(snippet.title) + '</span>' +
        '<button class="btn" style="padding:0.2rem 0.6rem;font-size:0.7rem" onclick="copyText(this.closest(\'.code-block\').querySelector(\'.code-content\').textContent)">Copy</button></div>' +
        '<div class="code-content">' + esc(snippet.code) + '</div></div>';
    }

    // ══════════════════════════════════════
    //  Landing Page Builder
    // ══════════════════════════════════════

    function generateLandingHTML() {
      var name = document.getElementById('landName').value || 'My Site';
      var tagline = document.getElementById('landTagline').value || 'Developer & Creator';
      var ig = document.getElementById('landIG').value || '@salmanfaris.dev';
      var gh = document.getElementById('landGH').value || 'salmanfaris22';
      var url = document.getElementById('landURL').value || 'https://salmanfaris.dev';
      var bio = document.getElementById('landBio').value || '';

      var igHandle = ig.replace('@', '');

      var html = '<!DOCTYPE html>\n' +
        '<html lang="en">\n' +
        '<head>\n' +
        '  <meta charset="UTF-8">\n' +
        '  <meta name="viewport" content="width=device-width, initial-scale=1.0">\n' +
        '  <title>' + esc(name) + '</title>\n' +
        '  <style>\n' +
        '    * { box-sizing: border-box; margin: 0; padding: 0; }\n' +
        '    body {\n' +
        '      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;\n' +
        '      min-height: 100vh;\n' +
        '      display: flex; align-items: center; justify-content: center;\n' +
        '      background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);\n' +
        '      color: #f1f5f9;\n' +
        '    }\n' +
        '    .container {\n' +
        '      text-align: center; padding: 3rem 2rem;\n' +
        '      max-width: 480px; width: 100%;\n' +
        '    }\n' +
        '    .avatar {\n' +
        '      width: 100px; height: 100px; border-radius: 50%;\n' +
        '      background: linear-gradient(135deg, #06b6d4, #8b5cf6);\n' +
        '      margin: 0 auto 1.5rem;\n' +
        '      display: flex; align-items: center; justify-content: center;\n' +
        '      font-size: 2.5rem; font-weight: 900; color: #fff;\n' +
        '    }\n' +
        '    h1 { font-size: 1.8rem; margin-bottom: 0.4rem; }\n' +
        '    .tagline { color: #94a3b8; margin-bottom: 1rem; }\n' +
        '    .bio { color: #cbd5e1; font-size: 0.95rem; line-height: 1.6; margin-bottom: 2rem; max-width: 360px; margin-left: auto; margin-right: auto; }\n' +
        '    .links { display: flex; flex-direction: column; gap: 0.75rem; }\n' +
        '    .link {\n' +
        '      display: block; padding: 0.85rem 1.5rem;\n' +
        '      background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1);\n' +
        '      border-radius: 12px; color: #f1f5f9;\n' +
        '      text-decoration: none; font-weight: 500;\n' +
        '      transition: all 0.2s;\n' +
        '    }\n' +
        '    .link:hover { background: rgba(255,255,255,0.1); transform: translateY(-2px); }\n' +
        '    .footer { margin-top: 2rem; font-size: 0.8rem; color: #475569; }\n' +
        '    .footer a { color: #06b6d4; text-decoration: none; }\n' +
        '  </style>\n' +
        '</head>\n' +
        '<body>\n' +
        '  <div class="container">\n' +
        '    <div class="avatar">' + esc(name.charAt(0).toUpperCase()) + '</div>\n' +
        '    <h1>' + esc(name) + '</h1>\n' +
        '    <p class="tagline">' + esc(tagline) + '</p>\n' +
        (bio ? '    <p class="bio">' + esc(bio) + '</p>\n' : '') +
        '    <div class="links">\n' +
        '      <a class="link" href="https://instagram.com/' + esc(igHandle) + '" target="_blank">Instagram @' + esc(igHandle) + '</a>\n' +
        '      <a class="link" href="https://github.com/' + esc(gh) + '" target="_blank">GitHub @' + esc(gh) + '</a>\n' +
        '      <a class="link" href="' + esc(url) + '" target="_blank">Website</a>\n' +
        '    </div>\n' +
        '    <div class="footer">Built with <a href="https://github.com/salmanfaris22/nexgo">NexGo</a></div>\n' +
        '  </div>\n' +
        '</body>\n' +
        '</html>';

      return html;
    }

    function copyLandingHTML() {
      copyText(generateLandingHTML());
    }

    function previewLanding() {
      var preview = document.getElementById('landingPreview');
      var frame = document.getElementById('landingFrame');
      preview.style.display = 'block';
      var html = generateLandingHTML();
      frame.srcdoc = html;
    }

    function updateLandingPreview() {
      var preview = document.getElementById('landingPreview');
      if (preview.style.display !== 'none') {
        previewLanding();
      }
    }

    // ══════════════════════════════════════
    //  HMR (Hot Module Reload)
    // ══════════════════════════════════════

    function initHMR() {
      var es = new EventSource('/_nexgo/hmr');
      var status = document.getElementById('hmrStatus');

      es.onopen = function() {
        status.innerHTML = '<div class="hmr-dot"></div><span>HMR connected</span>';
        status.style.color = 'var(--green)';
      };

      es.onmessage = function(e) {
        var msg = JSON.parse(e.data);
        if (msg.type === 'reload') {
          addLog('ok', 'HMR', 200, '/reload', 0);
          loadRoutes();
        }
      };

      es.onerror = function() {
        status.innerHTML = '<div class="hmr-dot" style="background:var(--red);box-shadow:0 0 6px var(--red)"></div><span>Reconnecting...</span>';
        status.style.color = 'var(--red)';
      };
    }

    // ══════════════════════════════════════
    //  Fetch Interception
    // ══════════════════════════════════════

    var origFetch = window.fetch;
    window.fetch = function() {
      var args = arguments;
      var url = typeof args[0] === 'string' ? args[0] : (args[0] && args[0].url ? args[0].url : '');
      if (url.indexOf('/_nexgo/') === 0) return origFetch.apply(window, args);

      var method = (args[1] && args[1].method) ? args[1].method : 'GET';
      var t0 = performance.now();
      return origFetch.apply(window, args).then(function(res) {
        var ms = Math.round(performance.now() - t0);
        addRequest(method, res.status, url, ms);
        return res;
      }).catch(function(e) {
        addRequest(method, 0, url, Math.round(performance.now() - t0));
        throw e;
      });
    };

    // ══════════════════════════════════════
    //  Manual Reload
    // ══════════════════════════════════════

    function triggerReload() {
      fetch('/_nexgo/reload').then(function() { loadRoutes(); });
    }

    // ══════════════════════════════════════
    //  Keyboard Shortcuts
    // ══════════════════════════════════════

    var panelOrder = ['routes','requests','logs','colors','seo','wasm','hooks','perf','config','landing'];
    document.addEventListener('keydown', function(e) {
      // Alt+1..0 to switch panels
      if (e.altKey && e.key >= '1' && e.key <= '9') {
        e.preventDefault();
        var idx = parseInt(e.key) - 1;
        if (idx < panelOrder.length) {
          var navItems = document.querySelectorAll('.nav-item');
          showPanel(panelOrder[idx], navItems[idx]);
        }
      }
      if (e.altKey && e.key === '0') {
        e.preventDefault();
        var navItems = document.querySelectorAll('.nav-item');
        showPanel(panelOrder[9], navItems[9]);
      }
    });

    // ══════════════════════════════════════
    //  Initialize Everything
    // ══════════════════════════════════════

    loadRoutes();
    loadConfig();
    initHMR();
    initColors();
    initSEO();
    initWasm();
    showHookSnippet('useState');
    renderHooksState();

    addLog('ok', 'SYS', 200, '/devtools loaded', 0);
  </script>
</body>
</html>`
