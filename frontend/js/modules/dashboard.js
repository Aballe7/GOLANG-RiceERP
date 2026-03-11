// dashboard.js — Modules.Dashboard

const DASH_WIDGETS = [
  { id: 'w-kpis',     label: 'KPI Strip',         icon: 'bi-bar-chart-line',  def: true },
  { id: 'w-layers',   label: 'Layer Houses',       icon: 'bi-egg-fill',        def: true },
  { id: 'w-growers',  label: 'Grower Houses',      icon: 'bi-moisture',        def: true },
  { id: 'w-eggs',     label: 'Egg Inventory',      icon: 'bi-egg',             def: true },
  { id: 'w-birds',    label: 'Population Summary', icon: 'bi-houses-fill',     def: true },
  { id: 'w-actions',  label: 'Quick Actions',      icon: 'bi-lightning-fill',  def: true },
  { id: 'w-sales',    label: 'Recent Sales',       icon: 'bi-bag-check',       def: true },
  { id: 'w-alerts',   label: 'Alerts',             icon: 'bi-bell-fill',       def: true },
];
const DASH_PREF_KEY = 'dashboard_widgets_v2';

function dashLoadPrefs() {
  try { return JSON.parse(localStorage.getItem(DASH_PREF_KEY)) || {}; } catch { return {}; }
}
function dashSavePrefs(p) { localStorage.setItem(DASH_PREF_KEY, JSON.stringify(p)); }
function dashApplyVisibility() {
  const prefs = dashLoadPrefs();
  DASH_WIDGETS.forEach(w => {
    const el = document.getElementById(w.id);
    if (!el) return;
    const vis = prefs.hasOwnProperty(w.id) ? prefs[w.id] : w.def;
    el.classList.toggle('widget-hidden', !vis);
  });
}
function dashToggleWidget(id, vis) {
  const p = dashLoadPrefs(); p[id] = vis; dashSavePrefs(p);
  dashApplyVisibility();
  const card = document.getElementById('dash-toggle-card-' + id);
  if (card) card.classList.toggle('opacity-50', !vis);
}
function dashResetWidgets() {
  localStorage.removeItem(DASH_PREF_KEY);
  dashApplyVisibility();
  dashBuildToggles();
}
function dashBuildToggles() {
  const prefs = dashLoadPrefs();
  const c = document.getElementById('dashWidgetToggles');
  if (!c) return;
  c.innerHTML = DASH_WIDGETS.map(w => {
    const vis = prefs.hasOwnProperty(w.id) ? prefs[w.id] : w.def;
    return `
    <div class="col-sm-6 col-md-4">
      <div class="card border shadow-sm h-100 ${vis ? '' : 'opacity-50'}" id="dash-toggle-card-${w.id}">
        <div class="card-body py-2 px-3 d-flex align-items-center gap-3">
          <i class="bi ${w.icon} fs-5 text-primary"></i>
          <span class="flex-grow-1 fw-bold small">${w.label}</span>
          <div class="form-check form-switch mb-0">
            <input class="form-check-input" type="checkbox" role="switch"
                   id="dash-toggle-${w.id}" ${vis ? 'checked' : ''}
                   onchange="dashToggleWidget('${w.id}', this.checked)">
          </div>
        </div>
      </div>
    </div>`;
  }).join('');
}

function dashOpenHousePanel(id, type, num, count, name) {
  const isL = type === 'Layer';
  const color = isL ? '#0d6efd' : '#ffc107';
  document.getElementById('hpHeader').style.background = color;
  document.getElementById('hpTitle').textContent = (isL ? 'Layer' : 'Grower') + ' House';
  document.getElementById('hpSubtitle').textContent = name;
  document.getElementById('hpNum').textContent = (isL ? 'L' : 'G') + '-' + num;
  document.getElementById('hpCount').textContent = parseInt(count).toLocaleString();
  document.getElementById('hpBatch').textContent = name;
  const badge = document.getElementById('hpBadge');
  badge.textContent = isL ? 'Layer' : 'Grower';
  badge.className = 'badge ' + (isL ? 'bg-primary' : 'bg-warning text-dark');
  const logBtn = document.getElementById('hpLogBtn');
  logBtn.onclick = () => { navigate('#/operations/log/' + id); bootstrap.Offcanvas.getInstance(document.getElementById('housePanel'))?.hide(); };
  logBtn.className = 'btn text-start w-100 ' + (isL ? 'btn-primary' : 'btn-warning');
  const histBtn = document.getElementById('hpHistBtn');
  histBtn.onclick = () => { navigate('#/operations/history/' + id); bootstrap.Offcanvas.getInstance(document.getElementById('housePanel'))?.hide(); };
  document.getElementById('hpLogInfo').innerHTML = (isL ? [
    '<i class="bi bi-egg-fill text-warning me-1"></i> Egg harvest by size (Pewee → Jumbo)',
    '<i class="bi bi-x-circle text-danger me-1"></i> Rejects: Cracked/Dirty, Double Yolk',
    '<i class="bi bi-bag-fill text-success me-1"></i> Feed consumed & type',
    '<i class="bi bi-hospital text-info me-1"></i> Vaccine administered',
    '<i class="bi bi-heartbreak text-danger me-1"></i> Mortality count',
  ] : [
    '<i class="bi bi-bag-fill text-success me-1"></i> Feed consumed & type',
    '<i class="bi bi-hospital text-info me-1"></i> Vaccine / Medicine',
    '<i class="bi bi-heartbreak text-danger me-1"></i> Mortality count',
  ]).map(i => `<li>${i}</li>`).join('');
  new bootstrap.Offcanvas(document.getElementById('housePanel')).show();
}

Modules.Dashboard = {
  async load() {
    showLoading();
    const data = await api.GetDashboard();
    if (!data) {
      showView(`<div class="alert alert-warning m-4">Failed to load dashboard data.</div>`);
      return;
    }
    showView(renderDashboard(data));
    dashApplyVisibility();
    dashBuildToggles();
    document.getElementById('customizeModal')?.addEventListener('show.bs.modal', dashBuildToggles);
  }
};

function renderDashboard(d) {
  const today = new Date().toLocaleDateString('en-PH', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });

  // ── KPI strip ─────────────────────────────────────────────────────────────
  function kpiClass(val, good, warn) {
    return val <= good ? 'text-success' : val <= warn ? 'text-warning' : 'text-danger';
  }
  const hdp     = d.hdp          ?? 0;
  const fcr     = d.fcr          ?? 0;
  const mort    = d.mortality_rate ?? 0;
  const reject  = d.reject_rate  ?? 0;

  const kpiStrip = `
  <div class="col-12 widget" id="w-kpis">
    <div class="row g-2">
      ${[
        { label: 'Hen-Day Production', id: 'kpiHDP',    val: hdp.toFixed(1) + '%',   sub: 'Last 30 days avg',
          cls: hdp >= 80 ? 'text-success' : hdp >= 60 ? 'text-warning' : 'text-danger' },
        { label: 'Feed Conv. Ratio',   id: 'kpiFCR',    val: fcr > 0 ? fcr.toFixed(2) : '—', sub: 'kg feed / kg eggs',
          cls: fcr === 0 ? 'text-muted' : fcr <= 1.8 ? 'text-success' : fcr <= 2.2 ? 'text-warning' : 'text-danger' },
        { label: 'Mortality Rate',     id: 'kpiMort',   val: mort.toFixed(2) + '%',  sub: 'Last 30 days',
          cls: mort <= 0.5 ? 'text-success' : mort <= 1 ? 'text-warning' : 'text-danger' },
        { label: 'Reject Rate',        id: 'kpiReject', val: reject.toFixed(2) + '%', sub: 'Cracked + DY',
          cls: reject <= 3 ? 'text-success' : reject <= 6 ? 'text-warning' : 'text-danger' },
      ].map(k => `
        <div class="col-6 col-md-3">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body py-2 px-3">
              <div class="text-muted" style="font-size:10px;letter-spacing:1px;text-transform:uppercase;">${k.label}</div>
              <div class="fw-bold fs-4 mt-1 ${k.cls}">${k.val}</div>
              <div class="text-muted" style="font-size:11px;">${k.sub}</div>
            </div>
          </div>
        </div>`).join('')}
    </div>
  </div>`;

  // ── Layer house tiles ──────────────────────────────────────────────────────
  const layers  = (d.flock_tiles || []).filter(f => f.house_type === 'Layer');
  const growers = (d.flock_tiles || []).filter(f => f.house_type === 'Grower');

  function houseTile(f) {
    return `
    <div class="house-tile border rounded text-center bg-white shadow-sm"
         style="width:72px;cursor:pointer;padding:6px 4px;"
         onclick="dashOpenHousePanel(${f.id},'${f.house_type}','${f.house_number}',${f.current_count},${JSON.stringify(f.name)})">
      <div class="fw-bold ${f.house_type === 'Layer' ? 'text-primary' : 'text-warning'}" style="font-size:12px;">${f.house_type === 'Layer' ? 'L' : 'G'}-${f.house_number}</div>
      <i class="bi bi-circle-fill text-success" style="font-size:7px;"></i>
      <div class="text-muted" style="font-size:10px;">${f.current_count.toLocaleString()}</div>
    </div>`;
  }

  const layerTilesSection = `
  <div class="col-lg-8 widget" id="w-layers">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-header bg-white border-0 py-2 d-flex justify-content-between align-items-center">
        <span class="fw-bold"><i class="bi bi-egg-fill text-primary me-2"></i>Layer Houses</span>
        <span class="badge bg-primary">${layers.length}</span>
      </div>
      <div class="card-body pb-3">
        <div class="d-flex flex-wrap gap-2">
          ${layers.length ? layers.map(houseTile).join('') : '<p class="text-muted small my-2">No layer houses yet.</p>'}
        </div>
      </div>
    </div>
  </div>`;

  const growerTilesSection = `
  <div class="col-lg-4 widget" id="w-growers">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-header bg-white border-0 py-2 d-flex justify-content-between align-items-center">
        <span class="fw-bold"><i class="bi bi-moisture text-warning me-2"></i>Grower Houses</span>
        <span class="badge bg-warning text-dark">${growers.length}</span>
      </div>
      <div class="card-body pb-3">
        <div class="d-flex flex-wrap gap-2">
          ${growers.length ? growers.map(houseTile).join('') : '<p class="text-muted small my-2">No grower houses yet.</p>'}
        </div>
      </div>
    </div>
  </div>`;

  // ── Egg inventory ──────────────────────────────────────────────────────────
  const inv = d.egg_inventory || {};
  const EGG_ORDER = ['Jumbo', 'Extra Large', 'Large', 'Medium', 'Small', 'Pullet', 'Peewee', 'Double Yolk', 'Cracked/Dirty'];
  const invEntries = EGG_ORDER.filter(s => (inv[s] ?? 0) > 0);
  let totalTrays = 0;
  invEntries.forEach(s => { totalTrays += (inv[s] ?? 0) / 30; });
  const eggSection = `
  <div class="col-md-4 widget" id="w-eggs">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-header bg-warning bg-opacity-10 border-0 py-2 d-flex justify-content-between">
        <span class="fw-bold small"><i class="bi bi-egg-fill text-warning me-1"></i> Egg Inventory</span>
        <span class="badge bg-warning text-dark">${totalTrays.toFixed(1)} trays</span>
      </div>
      <ul class="list-group list-group-flush">
        ${invEntries.length ? invEntries.map(s => {
          const pcs = inv[s] ?? 0;
          const trays = Math.floor(pcs / 30), rem = pcs % 30;
          const label = trays > 0 ? `${trays}t${rem ? ' + ' + rem + 'pc' : ''}` : `${pcs} pcs`;
          const danger = s === 'Cracked/Dirty' || s === 'Double Yolk';
          return `<li class="list-group-item d-flex justify-content-between align-items-center py-1">
            <small class="fw-bold ${danger ? 'text-danger' : ''}">${s}</small>
            <span class="text-muted small">${label}</span>
          </li>`;
        }).join('') : '<li class="list-group-item text-muted small text-center py-3">No egg inventory yet.</li>'}
      </ul>
      <div class="card-footer bg-white border-0 py-2">
        <button class="btn btn-warning btn-sm w-100 fw-bold" onclick="navigate('#/sales/orders/new')">
          <i class="bi bi-cart-plus me-1"></i> New Sales Order
        </button>
      </div>
    </div>
  </div>`;

  // ── Population summary ─────────────────────────────────────────────────────
  const popSection = `
  <div class="col-md-4 widget" id="w-birds">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-header bg-white border-0 py-2 fw-bold small">
        <i class="bi bi-houses-fill text-secondary me-1"></i> Population Summary
      </div>
      <div class="card-body">
        <div class="text-center mb-3">
          <div class="text-muted" style="font-size:11px;text-transform:uppercase;letter-spacing:1px;">Total Birds</div>
          <div class="fw-bold" style="font-size:2.2rem;">${(d.total_birds ?? 0).toLocaleString()}</div>
        </div>
        <div class="row g-2 text-center">
          <div class="col-6">
            <div class="p-2 rounded bg-primary bg-opacity-10">
              <div class="fw-bold text-primary fs-5">${d.layer_flocks ?? 0}</div>
              <div style="font-size:11px;" class="text-muted">Layer Houses</div>
            </div>
          </div>
          <div class="col-6">
            <div class="p-2 rounded bg-warning bg-opacity-10">
              <div class="fw-bold text-warning fs-5">${d.grower_flocks ?? 0}</div>
              <div style="font-size:11px;" class="text-muted">Grower Houses</div>
            </div>
          </div>
          <div class="col-6">
            <div class="p-2 rounded bg-success bg-opacity-10">
              <div class="fw-bold text-success fs-5">${(d.layer_birds ?? 0).toLocaleString()}</div>
              <div style="font-size:11px;" class="text-muted">Layers</div>
            </div>
          </div>
          <div class="col-6">
            <div class="p-2 rounded bg-info bg-opacity-10">
              <div class="fw-bold text-info fs-5">${(d.grower_birds ?? 0).toLocaleString()}</div>
              <div style="font-size:11px;" class="text-muted">Growers</div>
            </div>
          </div>
        </div>
      </div>
      <div class="card-footer bg-white border-0 py-2">
        <button class="btn btn-outline-secondary btn-sm w-100" onclick="navigate('#/flocks')">
          <i class="bi bi-list-ul me-1"></i> View All Flocks
        </button>
      </div>
    </div>
  </div>`;

  // ── Quick actions ──────────────────────────────────────────────────────────
  const actionsSection = `
  <div class="col-md-4 widget" id="w-actions">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-header bg-white border-0 py-2 fw-bold small">
        <i class="bi bi-lightning-fill text-warning me-1"></i> Quick Actions
      </div>
      <div class="card-body d-flex flex-column gap-2">
        <button class="btn btn-outline-success btn-sm text-start" onclick="navigate('#/sales')">
          <i class="bi bi-receipt me-2"></i> Sales
        </button>
        <button class="btn btn-outline-primary btn-sm text-start" onclick="navigate('#/purchasing')">
          <i class="bi bi-cart me-2"></i> Purchasing
        </button>
        <button class="btn btn-outline-secondary btn-sm text-start" onclick="navigate('#/operations')">
          <i class="bi bi-journal-text me-2"></i> Daily Logs
        </button>
        <button class="btn btn-outline-info btn-sm text-start" onclick="navigate('#/flocks')">
          <i class="bi bi-houses me-2"></i> Flocks
        </button>
        <button class="btn btn-outline-dark btn-sm text-start" onclick="navigate('#/accounting')">
          <i class="bi bi-bank me-2"></i> Accounting
        </button>
        ${currentUser && currentUser.role === 'Admin' ? `
        <hr class="my-1">
        <button class="btn btn-outline-secondary btn-sm text-start" id="dashBackupBtn" onclick="dashboardBackup()">
          <i class="bi bi-cloud-upload me-2"></i> Backup Database
        </button>` : ''}
      </div>
    </div>
  </div>`;

  // ── Recent sales ───────────────────────────────────────────────────────────
  const recentOrders = d.recent_orders || [];
  const salesSection = `
  <div class="col-md-4 widget" id="w-sales">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-header bg-white border-0 py-2 fw-bold small">
        <i class="bi bi-bag-check text-success me-1"></i> Recent Sales
      </div>
      <ul class="list-group list-group-flush">
        ${recentOrders.length ? recentOrders.map(o => {
          const statusCls = o.payment_status === 'Paid' ? 'bg-success' : o.payment_status === 'Partial' ? 'bg-warning text-dark' : o.payment_status === 'Void' ? 'bg-secondary' : 'bg-danger';
          return `<li class="list-group-item py-2">
            <div class="d-flex justify-content-between align-items-start">
              <div>
                <div class="fw-bold" style="font-size:12px;">${o.customer_name_snapshot || '—'}</div>
                <div class="text-muted font-monospace" style="font-size:10px;">${o.sales_order_number}</div>
              </div>
              <div class="text-end ms-2">
                <div class="fw-bold text-success" style="font-size:12px;">${formatCurrency(o.grand_total)}</div>
                <span class="badge ${statusCls}" style="font-size:9px;">${o.payment_status}</span>
              </div>
            </div>
          </li>`;
        }).join('') : '<li class="list-group-item text-muted small text-center py-3">No recent sales.</li>'}
      </ul>
      <div class="card-footer bg-white border-0 py-2">
        <button class="btn btn-outline-success btn-sm w-100" onclick="navigate('#/sales')">
          <i class="bi bi-list-ul me-1"></i> All Sales
        </button>
      </div>
    </div>
  </div>`;

  // ── Alerts ─────────────────────────────────────────────────────────────────
  const unlogged    = d.unlogged_houses || [];
  const lowFeed     = d.low_feed_alerts || [];
  const alertsSection = `
  <div class="col-12 widget" id="w-alerts">
    ${unlogged.length ? `
    <div class="alert alert-warning d-flex align-items-start gap-2 mb-2">
      <i class="bi bi-exclamation-circle-fill mt-1"></i>
      <div>
        <strong>Unlogged Houses Today:</strong> ${unlogged.join(', ')}
        <button class="btn btn-sm btn-link p-0 ms-2" onclick="navigate('#/operations')">Log Now &rarr;</button>
      </div>
    </div>` : ''}
    ${lowFeed.length ? `
    <div class="alert alert-danger d-flex align-items-start gap-2 mb-2">
      <i class="bi bi-fuel-pump-fill mt-1"></i>
      <div>
        <strong>Low Feed Stock:</strong>
        <ul class="mb-0 mt-1">
          ${lowFeed.map(f => `<li>${f.name}: <strong>${f.total_sacks} sacks</strong></li>`).join('')}
        </ul>
        <button class="btn btn-sm btn-link p-0 mt-1" onclick="navigate('#/purchasing')">Go to Purchasing &rarr;</button>
      </div>
    </div>` : ''}
    ${!unlogged.length && !lowFeed.length ? '<div class="alert alert-success mb-2 py-2 small"><i class="bi bi-check-circle me-1"></i>No alerts today.</div>' : ''}
  </div>`;

  // ── AR/AP balance ──────────────────────────────────────────────────────────
  const arBalance = d.open_ar_balance ?? 0;
  const apBalance = d.open_ap_balance ?? 0;
  const balanceRow = `
  <div class="col-sm-6 col-md-3">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-body py-3">
        <div class="d-flex align-items-center gap-2 mb-1">
          <i class="bi bi-arrow-down-circle text-success fs-5"></i>
          <span class="text-muted small fw-bold">Open AR</span>
        </div>
        <div class="fs-5 fw-semibold text-success">${formatCurrency(arBalance)}</div>
      </div>
    </div>
  </div>
  <div class="col-sm-6 col-md-3">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-body py-3">
        <div class="d-flex align-items-center gap-2 mb-1">
          <i class="bi bi-arrow-up-circle text-danger fs-5"></i>
          <span class="text-muted small fw-bold">Open AP</span>
        </div>
        <div class="fs-5 fw-semibold text-danger">${formatCurrency(apBalance)}</div>
      </div>
    </div>
  </div>
  <div class="col-sm-6 col-md-3">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-body py-3">
        <div class="d-flex align-items-center gap-2 mb-1">
          <i class="bi bi-egg text-warning fs-5"></i>
          <span class="text-muted small fw-bold">Today's Eggs</span>
        </div>
        <div class="fs-5 fw-semibold">${(d.today_eggs ?? 0).toLocaleString()} pcs</div>
      </div>
    </div>
  </div>
  <div class="col-sm-6 col-md-3">
    <div class="card border-0 shadow-sm h-100">
      <div class="card-body py-3">
        <div class="d-flex align-items-center gap-2 mb-1">
          <i class="bi bi-heartbreak text-danger fs-5"></i>
          <span class="text-muted small fw-bold">Today's Mortality</span>
        </div>
        <div class="fs-5 fw-semibold ${d.today_mortality > 0 ? 'text-danger' : ''}">${(d.today_mortality ?? 0).toLocaleString()}</div>
      </div>
    </div>
  </div>`;

  return `
<div class="container-fluid py-3">
  <style>
    .house-tile { transition: .15s; }
    .house-tile:hover { background: #eef2ff !important; border-color: #6c8ebf !important; transform: translateY(-2px); }
    .widget-hidden { display: none !important; }
    .widget { transition: opacity .2s; }
  </style>

  <!-- Top bar -->
  <div class="d-flex align-items-center justify-content-between mb-3">
    <div>
      <h5 class="fw-bold mb-0"><i class="bi bi-speedometer2 me-2 text-primary"></i>Dashboard</h5>
      <small class="text-muted">${today}</small>
    </div>
    <button class="btn btn-outline-secondary btn-sm" data-bs-toggle="modal" data-bs-target="#customizeModal">
      <i class="bi bi-sliders me-1"></i> Customize
    </button>
  </div>

  <!-- Widget grid -->
  <div class="row g-3" id="dashWidgetGrid">
    <!-- KPI strip -->
    ${kpiStrip}

    <!-- Summary bar -->
    <div class="col-12">
      <div class="row g-2">${balanceRow}</div>
    </div>

    <!-- Alerts -->
    ${alertsSection}

    <!-- Houses -->
    ${layerTilesSection}
    ${growerTilesSection}

    <!-- Egg + Population + Actions -->
    ${eggSection}
    ${popSection}
    ${actionsSection}

    <!-- Recent Sales -->
    ${salesSection}
  </div>
</div>

<!-- House offcanvas -->
<div class="offcanvas offcanvas-end" tabindex="-1" id="housePanel">
  <div class="offcanvas-header text-white" id="hpHeader" style="background:#0d6efd;">
    <div>
      <h5 class="offcanvas-title fw-bold mb-0" id="hpTitle">House Details</h5>
      <small id="hpSubtitle" class="opacity-75"></small>
    </div>
    <button type="button" class="btn-close btn-close-white" data-bs-dismiss="offcanvas"></button>
  </div>
  <div class="offcanvas-body">
    <div class="d-flex align-items-center justify-content-between mb-3">
      <div>
        <h3 class="fw-bold mb-0" id="hpNum">--</h3>
        <span class="badge" id="hpBadge">--</span>
      </div>
      <div class="text-end">
        <p class="small text-muted mb-0">Bird Population</p>
        <h4 class="fw-bold" id="hpCount">--</h4>
      </div>
    </div>
    <ul class="list-group list-group-flush mb-4">
      <li class="list-group-item d-flex justify-content-between px-0">
        <span class="text-muted small">Batch Name</span>
        <span class="fw-bold small" id="hpBatch">--</span>
      </li>
    </ul>
    <h6 class="fw-bold text-uppercase text-muted mb-2" style="font-size:11px;">Quick Actions</h6>
    <div class="d-grid gap-2">
      <button id="hpLogBtn" class="btn btn-primary text-start">
        <i class="bi bi-journal-plus me-2"></i> Record Daily Log
      </button>
      <button id="hpHistBtn" class="btn btn-outline-secondary text-start">
        <i class="bi bi-clock-history me-2"></i> View History
      </button>
    </div>
    <div class="mt-4 p-3 rounded-3 small" style="background:#f8f9fa;border:1px solid #e9ecef;">
      <p class="fw-bold mb-2" style="font-size:11px;text-transform:uppercase;color:#999;">This log records:</p>
      <ul class="mb-0 ps-3" id="hpLogInfo" style="line-height:1.9;"></ul>
    </div>
  </div>
</div>

<!-- Customize modal -->
<div class="modal fade" id="customizeModal" tabindex="-1">
  <div class="modal-dialog modal-lg">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title fw-bold"><i class="bi bi-sliders me-2 text-primary"></i>Customize Dashboard</h5>
        <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
      </div>
      <div class="modal-body">
        <p class="text-muted small mb-3">Toggle widgets on or off. Changes are saved automatically.</p>
        <div class="row g-3" id="dashWidgetToggles"></div>
      </div>
      <div class="modal-footer justify-content-between">
        <button class="btn btn-outline-secondary btn-sm" onclick="dashResetWidgets()">
          <i class="bi bi-arrow-counterclockwise me-1"></i> Reset to Default
        </button>
        <button class="btn btn-primary" data-bs-dismiss="modal">
          <i class="bi bi-check-lg me-1"></i> Done
        </button>
      </div>
    </div>
  </div>
</div>`;
}


// ── Dashboard quick backup ─────────────────────────────────────────────────────

async function dashboardBackup() {
  const btn = document.getElementById('dashBackupBtn');
  const reset = () => {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-cloud-upload me-2"></i> Backup Database';
    }
  };
  if (btn) {
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Backing up…';
  }

  const resp = await window.go.app.App.CreateBackup();
  reset();

  if (!resp || !resp.ok) {
    const msg = (resp && resp.message) ? resp.message : 'Unknown error — resp: ' + JSON.stringify(resp);
    console.error('[backup]', msg);
    toast('Backup failed — navigate to Backup module for details', 'danger');
    return;
  }
  const fname = resp.data?.file_name || 'done';
  const fpath = resp.data?.path || '';
  toast(`Backup saved: ${fname}${fpath ? ' → ' + fpath : ''}`, 'success');
}
