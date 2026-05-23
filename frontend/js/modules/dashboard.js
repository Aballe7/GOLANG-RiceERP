// dashboard.js — Modules.Dashboard

// ── Widget Registry ────────────────────────────────────────────────────────
const DASHBOARD_WIDGETS = [
  { id: 'lowStockAlert',     label: 'Low Stock Alert',          icon: 'bi-exclamation-triangle-fill', group: 'Alerts',          default: true },
  { id: 'kpiPayables',       label: 'KPI: Payables Due',        icon: 'bi-file-earmark-text',         group: 'KPI Cards',       default: true },
  { id: 'kpiReceivables',    label: 'KPI: Receivables',         icon: 'bi-cash-coin',                 group: 'KPI Cards',       default: true },
  { id: 'kpiOpenOrders',     label: 'KPI: Open Orders',         icon: 'bi-bag',                       group: 'KPI Cards',       default: true },
  { id: 'kpiInventory',      label: 'KPI: Inventory',           icon: 'bi-boxes',                     group: 'KPI Cards',       default: true },
  { id: 'kpiMilling',        label: 'KPI: Milling',             icon: 'bi-gear-wide-connected',       group: 'KPI Cards',       default: true },
  { id: 'quickActions',      label: 'Quick Actions',            icon: 'bi-lightning-charge-fill',     group: 'Sections',        default: true },
  { id: 'warehouseStock',    label: 'Warehouse Stock Summary',  icon: 'bi-building',                  group: 'Sections',        default: true },
  { id: 'recentPayments',    label: 'Recent AP Payments',       icon: 'bi-cash-stack',                group: 'Recent Activity', default: true },
  { id: 'recentCollections', label: 'Recent Collections',       icon: 'bi-cash-coin',                 group: 'Recent Activity', default: true },
];

const DASHBOARD_PREFS_KEY = 'ricemill_dashboard_prefs_v2';

// ── Prefs Helpers ──────────────────────────────────────────────────────────
function buildDefaultPrefs() {
  return Object.fromEntries(DASHBOARD_WIDGETS.map(w => [w.id, w.default]));
}

function loadDashboardPrefs() {
  try {
    const raw = localStorage.getItem(DASHBOARD_PREFS_KEY);
    if (!raw) return buildDefaultPrefs();
    const saved  = JSON.parse(raw);
    const merged = buildDefaultPrefs();
    Object.keys(saved).forEach(id => { if (id in merged) merged[id] = !!saved[id]; });
    return merged;
  } catch (_) {
    return buildDefaultPrefs();
  }
}

function saveDashboardPrefs(prefs) {
  try { localStorage.setItem(DASHBOARD_PREFS_KEY, JSON.stringify(prefs)); } catch (_) {}
}

function isWidgetVisible(prefs, id) { return prefs[id] !== false; }

// ── Customize Panel ────────────────────────────────────────────────────────
function openCustomizePanel() {
  renderCustomizePanel(loadDashboardPrefs());
  const el = document.getElementById('dashboardCustomizePanel');
  (bootstrap.Offcanvas.getInstance(el) || new bootstrap.Offcanvas(el)).show();
}

function renderCustomizePanel(prefs) {
  const groups = {};
  DASHBOARD_WIDGETS.forEach(w => {
    if (!groups[w.group]) groups[w.group] = [];
    groups[w.group].push(w);
  });

  const html = Object.entries(groups).map(([groupName, widgets], idx) => `
    ${idx > 0 ? '<hr class="my-2">' : ''}
    <div class="mb-3">
      <div class="text-uppercase text-muted small fw-bold mb-2" style="letter-spacing:.04em;">${groupName}</div>
      ${widgets.map(w => `
        <div class="form-check form-switch mb-2">
          <input class="form-check-input" type="checkbox" role="switch" id="dbw_${w.id}" ${prefs[w.id] !== false ? 'checked' : ''}>
          <label class="form-check-label small" for="dbw_${w.id}">
            <i class="bi ${w.icon} me-1 text-warning"></i>${w.label}
          </label>
        </div>`).join('')}
    </div>`).join('');

  document.getElementById('dashboardWidgetToggles').innerHTML = html;
}

function applyDashboardPrefs() {
  const prefs = {};
  DASHBOARD_WIDGETS.forEach(w => {
    const el = document.getElementById('dbw_' + w.id);
    prefs[w.id] = el ? el.checked : w.default;
  });
  saveDashboardPrefs(prefs);
  const panel = bootstrap.Offcanvas.getInstance(document.getElementById('dashboardCustomizePanel'));
  if (panel) panel.hide();
  loadDashboardView();
}

function resetDashboardPrefs() {
  const defaults = buildDefaultPrefs();
  saveDashboardPrefs(defaults);
  renderCustomizePanel(defaults);
}

// ── Segment Renderers ──────────────────────────────────────────────────────

// lowOITW  : OITW rows where on_hand < min_stock (already filtered)
// itemsMap : { item_code → OITM record }
// whsMap   : { whs_code → OWHS record }
function renderLowStockAlert(lowOITW, itemsMap, whsMap) {
  if (lowOITW.length === 0) {
    return `
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-body d-flex align-items-center gap-3 py-3">
          <div class="rounded-3 p-2 bg-success bg-opacity-10 flex-shrink-0">
            <i class="bi bi-check-circle-fill fs-4 text-success"></i>
          </div>
          <div>
            <div class="fw-bold small">Low Stock Alert</div>
            <div class="text-muted small">All per-warehouse stock levels are within acceptable range.</div>
          </div>
          <a href="#" onclick="navigate('#/inventory/warehouses');return false;"
             class="ms-auto small text-warning text-decoration-none">View Warehouses →</a>
        </div>
      </div>`;
  }

  // Group by category
  const byCategory = {};
  lowOITW.forEach(r => {
    const item = itemsMap[r.item_code] || {};
    const cat  = item.category_name || 'Uncategorized';
    if (!byCategory[cat]) byCategory[cat] = [];
    byCategory[cat].push(r);
  });

  const rows = Object.entries(byCategory).map(([cat, catRows]) => {
    const itemRows = catRows.map(r => {
      const item    = itemsMap[r.item_code] || {};
      const whs     = whsMap[r.whs_code]   || {};
      const onHand  = r.on_hand  || 0;
      const commited = r.is_commited || 0;
      const onOrder  = r.on_order   || 0;
      const avail   = onHand - commited + onOrder;
      const isZero  = onHand === 0;
      const unit    = item.invntry_uom || '';
      return `
        <tr>
          <td class="ps-3 py-1">
            <a href="#" onclick="navigate('#/inventory/warehouses');return false;"
               class="text-warning fw-semibold text-decoration-none small">${item.item_name || r.item_code}</a>
          </td>
          <td class="py-1 small text-muted">${whs.whs_name || r.whs_code}</td>
          <td class="py-1 text-center small ${isZero ? 'text-danger fw-bold' : 'text-secondary'}">${formatNumber(onHand)} ${unit}</td>
          <td class="py-1 text-center small ${avail < 0 ? 'text-danger fw-bold' : 'text-muted'}">${formatNumber(avail)}</td>
          <td class="pe-3 py-1 text-center small text-muted">min: ${formatNumber(r.min_stock || 0)}</td>
        </tr>`;
    }).join('');
    return `
      <tr class="table-warning">
        <td colspan="5" class="ps-3 py-1">
          <span class="fw-bold small text-uppercase text-warning-emphasis" style="letter-spacing:.04em;">${cat}</span>
        </td>
      </tr>
      ${itemRows}`;
  }).join('');

  return `
    <div class="card border-warning shadow-sm mb-4">
      <div class="card-header bg-warning bg-opacity-10 border-warning d-flex align-items-center gap-2 py-2"
           style="cursor:pointer;" onclick="this.nextElementSibling.classList.toggle('d-none');this.querySelector('.ls-chevron').classList.toggle('bi-chevron-down');this.querySelector('.ls-chevron').classList.toggle('bi-chevron-up');">
        <i class="bi bi-exclamation-triangle-fill text-warning"></i>
        <span class="fw-bold">Low Stock Alert</span>
        <span class="badge bg-warning text-dark ms-1">${lowOITW.length} warehouse record${lowOITW.length > 1 ? 's' : ''}</span>
        <a href="#" onclick="navigate('#/inventory/warehouses');return false;event.stopPropagation();"
           class="ms-auto small text-warning text-decoration-none me-2">View Warehouses →</a>
        <i class="bi bi-chevron-down ls-chevron text-warning"></i>
      </div>
      <div class="d-none" style="max-height:280px;overflow-y:auto;">
        <table class="table table-sm table-hover mb-0">
          <thead class="table-light">
            <tr>
              <th class="ps-3 small">Item</th>
              <th class="small">Warehouse</th>
              <th class="text-center small">On Hand</th>
              <th class="text-center small">Available</th>
              <th class="pe-3 text-center small">Min Stock</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>`;
}

function renderKpiPayables(apOpen, apTotal) {
  return `
    <div class="col-sm-6 col-xl-3">
      <div class="card border-0 shadow-sm h-100" onclick="navigate('#/purchasing/ap-invoices')" style="cursor:pointer">
        <div class="card-body d-flex align-items-center gap-3">
          <div class="rounded-3 p-3 bg-warning bg-opacity-10 flex-shrink-0">
            <i class="bi bi-file-earmark-text fs-3 text-warning"></i>
          </div>
          <div>
            <div class="small text-muted fw-bold text-uppercase">Payables Due</div>
            <div class="fw-bold fs-5">${formatCurrency(apTotal)}</div>
            <div class="text-muted small">${apOpen.length} open invoice${apOpen.length !== 1 ? 's' : ''}</div>
          </div>
        </div>
      </div>
    </div>`;
}

function renderKpiReceivables(arOpen, arTotal) {
  return `
    <div class="col-sm-6 col-xl-3">
      <div class="card border-0 shadow-sm h-100" onclick="navigate('#/sales/ar-invoices')" style="cursor:pointer">
        <div class="card-body d-flex align-items-center gap-3">
          <div class="rounded-3 p-3 bg-success bg-opacity-10 flex-shrink-0">
            <i class="bi bi-cash-coin fs-3 text-success"></i>
          </div>
          <div>
            <div class="small text-muted fw-bold text-uppercase">Receivables</div>
            <div class="fw-bold fs-5">${formatCurrency(arTotal)}</div>
            <div class="text-muted small">${arOpen.length} open invoice${arOpen.length !== 1 ? 's' : ''}</div>
          </div>
        </div>
      </div>
    </div>`;
}

function renderKpiOpenOrders(openSOs) {
  return `
    <div class="col-sm-6 col-xl-3">
      <div class="card border-0 shadow-sm h-100" onclick="navigate('#/sales/orders')" style="cursor:pointer">
        <div class="card-body d-flex align-items-center gap-3">
          <div class="rounded-3 p-3 bg-primary bg-opacity-10 flex-shrink-0">
            <i class="bi bi-bag fs-3 text-primary"></i>
          </div>
          <div>
            <div class="small text-muted fw-bold text-uppercase">Open Orders</div>
            <div class="fw-bold fs-5">${openSOs.length}</div>
            <div class="text-muted small">sales order${openSOs.length !== 1 ? 's' : ''}</div>
          </div>
        </div>
      </div>
    </div>`;
}

function renderKpiInventory(allItems, lowCount) {
  return `
    <div class="col-sm-6 col-xl-3">
      <div class="card border-0 shadow-sm h-100" onclick="navigate('#/inventory/stock')" style="cursor:pointer">
        <div class="card-body d-flex align-items-center gap-3">
          <div class="rounded-3 p-3 ${lowCount > 0 ? 'bg-danger bg-opacity-10' : 'bg-info bg-opacity-10'} flex-shrink-0">
            <i class="bi bi-boxes fs-3 ${lowCount > 0 ? 'text-danger' : 'text-info'}"></i>
          </div>
          <div>
            <div class="small text-muted fw-bold text-uppercase">Inventory</div>
            <div class="fw-bold fs-5">${allItems.length} items</div>
            <div class="${lowCount > 0 ? 'text-danger' : 'text-muted'} small">
              ${lowCount > 0 ? `${lowCount} low stock` : 'All levels OK'}
            </div>
          </div>
        </div>
      </div>
    </div>`;
}

function renderKpiMilling(inProgressMOs) {
  return `
    <div class="col-sm-6 col-xl-3">
      <div class="card border-0 shadow-sm h-100" onclick="navigate('#/production/milling-orders')" style="cursor:pointer">
        <div class="card-body d-flex align-items-center gap-3">
          <div class="rounded-3 p-3 bg-warning bg-opacity-10 flex-shrink-0">
            <i class="bi bi-gear-wide-connected fs-3 text-warning"></i>
          </div>
          <div>
            <div class="small text-muted fw-bold text-uppercase">Milling</div>
            <div class="fw-bold fs-5">${inProgressMOs.length}</div>
            <div class="text-muted small">in progress</div>
          </div>
        </div>
      </div>
    </div>`;
}

function renderQuickActions() {
  return `
    <div class="col-6 col-md-3">
      <a href="#" onclick="navigate('#/purchasing/purchases/new');return false;"
        class="card border-0 shadow-sm text-decoration-none text-dark h-100">
        <div class="card-body text-center py-4">
          <i class="bi bi-cart-plus fs-2 text-warning mb-2 d-block"></i>
          <div class="fw-semibold small">New Purchase Order</div>
        </div>
      </a>
    </div>
    <div class="col-6 col-md-3">
      <a href="#" onclick="navigate('#/sales/orders/new');return false;"
        class="card border-0 shadow-sm text-decoration-none text-dark h-100">
        <div class="card-body text-center py-4">
          <i class="bi bi-bag-plus fs-2 text-primary mb-2 d-block"></i>
          <div class="fw-semibold small">New Sales Order</div>
        </div>
      </a>
    </div>
    <div class="col-6 col-md-3">
      <a href="#" onclick="navigate('#/purchasing/ap-payments/new');return false;"
        class="card border-0 shadow-sm text-decoration-none text-dark h-100">
        <div class="card-body text-center py-4">
          <i class="bi bi-cash-stack fs-2 text-danger mb-2 d-block"></i>
          <div class="fw-semibold small">Record AP Payment</div>
        </div>
      </a>
    </div>
    <div class="col-6 col-md-3">
      <a href="#" onclick="navigate('#/sales/collections/new');return false;"
        class="card border-0 shadow-sm text-decoration-none text-dark h-100">
        <div class="card-body text-center py-4">
          <i class="bi bi-cash-coin fs-2 text-success mb-2 d-block"></i>
          <div class="fw-semibold small">Record Collection</div>
        </div>
      </a>
    </div>
    <div class="col-6 col-md-3">
      <a href="#" onclick="navigate('#/production/milling-orders/new');return false;"
        class="card border-0 shadow-sm text-decoration-none text-dark h-100">
        <div class="card-body text-center py-4">
          <i class="bi bi-gear-wide-connected fs-2 text-warning mb-2 d-block"></i>
          <div class="fw-semibold small">New Milling Order</div>
        </div>
      </a>
    </div>`;
}

// whsList  : OWHS records
// oitwRows : all OITW records
function renderWarehouseStock(whsList, oitwRows) {
  if (whsList.length === 0) return '';

  const cards = whsList.map(w => {
    const whsRows  = oitwRows.filter(r => r.whs_code === w.whs_code);
    const onHand   = whsRows.reduce((s, r) => s + (r.on_hand || 0), 0);
    const commited = whsRows.reduce((s, r) => s + (r.is_commited || 0), 0);
    const onOrder  = whsRows.reduce((s, r) => s + (r.on_order || 0), 0);
    const avail    = onHand - commited + onOrder;
    const lowCnt   = whsRows.filter(r => (r.min_stock || 0) > 0 && (r.on_hand || 0) < r.min_stock).length;
    const inactive = w.inactive === 'Y';
    return `
      <div class="col-sm-6 col-xl-3">
        <div class="card border-0 shadow-sm h-100" onclick="navigate('#/inventory/warehouses')" style="cursor:pointer;">
          <div class="card-body py-3">
            <div class="d-flex align-items-start justify-content-between mb-2">
              <div>
                <div class="fw-bold small">${w.whs_name}</div>
                <div class="text-muted small font-monospace">${w.whs_code}</div>
              </div>
              <div class="d-flex flex-column align-items-end gap-1">
                <span class="badge ${inactive ? 'bg-secondary' : 'bg-success-subtle text-success border border-success-subtle'} small">
                  ${inactive ? 'Inactive' : 'Active'}
                </span>
                ${lowCnt > 0 ? `<span class="badge bg-danger">${lowCnt} low</span>` : ''}
              </div>
            </div>
            <div class="border-top pt-2 d-flex gap-3">
              <div>
                <div class="text-muted" style="font-size:.7rem;">ON HAND</div>
                <div class="fw-bold small">${formatNumber(onHand)}</div>
              </div>
              <div>
                <div class="text-muted" style="font-size:.7rem;">AVAILABLE</div>
                <div class="fw-bold small ${avail < 0 ? 'text-danger' : ''}">${formatNumber(avail)}</div>
              </div>
              <div>
                <div class="text-muted" style="font-size:.7rem;">ITEMS</div>
                <div class="fw-bold small">${whsRows.length}</div>
              </div>
            </div>
          </div>
        </div>
      </div>`;
  }).join('');

  return `
    <div class="fw-semibold text-muted small text-uppercase mb-2">
      <i class="bi bi-building me-1"></i>Warehouse Stock
      <span class="text-muted fw-normal normal-case ms-1" style="font-size:.7rem;text-transform:none;">
        Available = On Hand − Committed + On Order
      </span>
    </div>
    <div class="row g-3 mb-4">${cards}</div>`;
}

function renderRecentPayments(allPayments, fullWidth = false) {
  const colClass  = fullWidth ? 'col-12' : 'col-md-6';
  const recent    = allPayments.slice(0, 5);
  const payRows   = recent.length === 0
    ? `<tr><td colspan="4" class="text-center text-muted py-3 small">No payments yet.</td></tr>`
    : recent.map(p => `
        <tr style="cursor:pointer" onclick="navigate('#/purchasing/ap-payments/${p.id}')">
          <td class="fw-semibold small">${p.payment_number || '—'}</td>
          <td class="text-muted small">${formatDate(p.date)}</td>
          <td class="small">${p.supplier_name_snapshot || '—'}</td>
          <td class="text-end fw-bold text-warning small">${formatCurrency(p.total_amount || 0)}</td>
        </tr>`).join('');
  return `
    <div class="${colClass}">
      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-cash-stack me-2 text-warning"></i>AP Payments</span>
          <a href="#" onclick="navigate('#/purchasing/ap-payments');return false;" class="small text-warning text-decoration-none">View all →</a>
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Payment #</th><th>Date</th><th>Supplier</th><th class="text-end">Amount</th></tr>
            </thead>
            <tbody>${payRows}</tbody>
          </table>
        </div>
      </div>
    </div>`;
}

function renderRecentCollections(allCollections, fullWidth = false) {
  const colClass = fullWidth ? 'col-12' : 'col-md-6';
  const recent   = allCollections.slice(0, 5);
  const colRows  = recent.length === 0
    ? `<tr><td colspan="4" class="text-center text-muted py-3 small">No collections yet.</td></tr>`
    : recent.map(c => `
        <tr style="cursor:pointer" onclick="navigate('#/sales/collections/${c.id}')">
          <td class="fw-semibold small">${c.collection_number || '—'}</td>
          <td class="text-muted small">${formatDate(c.date)}</td>
          <td class="small">${c.customer_name_snapshot || '—'}</td>
          <td class="text-end fw-bold text-success small">${formatCurrency(c.total_amount || 0)}</td>
        </tr>`).join('');
  return `
    <div class="${colClass}">
      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-cash-coin me-2 text-success"></i>Collections</span>
          <a href="#" onclick="navigate('#/sales/collections');return false;" class="small text-success text-decoration-none">View all →</a>
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Collection #</th><th>Date</th><th>Customer</th><th class="text-end">Amount</th></tr>
            </thead>
            <tbody>${colRows}</tbody>
          </table>
        </div>
      </div>
    </div>`;
}

function renderEmptyDashboard() {
  return `
    <div class="text-center py-5">
      <i class="bi bi-layout-wtf display-4 text-muted mb-3 d-block"></i>
      <h6 class="text-muted">All widgets are hidden</h6>
      <p class="text-muted small mb-3">Use Customize to show widgets on your dashboard.</p>
      <button class="btn btn-warning btn-sm" onclick="openCustomizePanel()">
        <i class="bi bi-layout-wtf me-1"></i>Customize Dashboard
      </button>
    </div>`;
}

// ── Module Entry ───────────────────────────────────────────────────────────
Modules.Dashboard = {
  async load() {
    showLoading();
    await loadDashboardView();
  }
};

// ── Orchestrator ───────────────────────────────────────────────────────────
async function loadDashboardView() {
  const prefs = loadDashboardPrefs();

  const [
    openAP, partialAP, openAR, partialAR,
    salesOrders, payments, collections,
    items, millingOrders,
    oitwResult, whsResult,
  ] = await Promise.allSettled([
    api.ListAPInvoices(null, 'Open'),
    api.ListAPInvoices(null, 'Partial'),
    api.ListARInvoices(null, 'Open'),
    api.ListARInvoices(null, 'Partial'),
    api.ListSalesOrders(),
    api.ListAPPayments(),
    api.ListCollections(),
    api.ListItemMasters(),
    api.ListMillingOrders(),
    api.ListAllWarehouseStock(),
    api.ListWarehouses(),
  ]);

  const apOpen         = [...(openAP.value || []), ...(partialAP.value || [])];
  const arOpen         = [...(openAR.value || []), ...(partialAR.value || [])];
  const allSOs         = salesOrders.value || [];
  const allPayments    = payments.value || [];
  const allCollections = collections.value || [];
  const allItems       = items.value || [];
  const allMOs         = millingOrders.value || [];
  const allOITW        = oitwResult.value || [];
  const whsList        = whsResult.value || [];

  const inProgressMOs = allMOs.filter(m => m.status === 'In Progress');
  const openSOs       = allSOs.filter(s => s.status === 'Open' || s.status === 'Submitted');
  const apTotal       = apOpen.reduce((s, i) => s + Math.max(0, (i.doc_total || 0) - (i.amount_paid_stored || 0)), 0);
  const arTotal       = arOpen.reduce((s, i) => s + Math.max(0, (i.doc_total || 0) - (i.amount_paid_stored || 0)), 0);

  // OITW-powered low stock: per-warehouse check against OITW.min_stock
  const itemsMap = Object.fromEntries(allItems.map(i => [i.item_code, i]));
  const whsMap   = Object.fromEntries(whsList.map(w => [w.whs_code, w]));
  const lowOITW  = allOITW.filter(r => (r.min_stock ?? 0) > 0 && (r.on_hand ?? 0) < r.min_stock);

  // ── Visibility flags ───────────────────────────────────────────────────
  const showLowStock          = isWidgetVisible(prefs, 'lowStockAlert');
  const showWarehouseStock    = isWidgetVisible(prefs, 'warehouseStock');
  const showRecentPayments    = isWidgetVisible(prefs, 'recentPayments');
  const showRecentCollections = isWidgetVisible(prefs, 'recentCollections');
  const showQuickActions      = isWidgetVisible(prefs, 'quickActions');

  const visibleKpiCards = [
    isWidgetVisible(prefs, 'kpiPayables')    ? renderKpiPayables(apOpen, apTotal)           : '',
    isWidgetVisible(prefs, 'kpiReceivables') ? renderKpiReceivables(arOpen, arTotal)        : '',
    isWidgetVisible(prefs, 'kpiOpenOrders')  ? renderKpiOpenOrders(openSOs)                 : '',
    isWidgetVisible(prefs, 'kpiInventory')   ? renderKpiInventory(allItems, lowOITW.length) : '',
    isWidgetVisible(prefs, 'kpiMilling')     ? renderKpiMilling(inProgressMOs)              : '',
  ].join('');

  const showKpiSection    = visibleKpiCards.trim().length > 0;
  const showRecentSection = showRecentPayments || showRecentCollections;
  const anyVisible        = showLowStock || showKpiSection || showQuickActions || showWarehouseStock || showRecentSection;

  const paymentsFullWidth    = showRecentPayments && !showRecentCollections;
  const collectionsFullWidth = showRecentCollections && !showRecentPayments;

  const dateStr = new Date().toLocaleDateString('en-PH', {
    weekday: 'long', year: 'numeric', month: 'long', day: 'numeric'
  });

  showView(`
    <div class="container-fluid px-4 py-3">
      <div class="d-flex align-items-start justify-content-between mb-4">
        <div>
          <h4 class="fw-bold mb-0"><i class="bi bi-speedometer2 me-2 text-warning"></i>Dashboard</h4>
          <div class="text-muted small">${dateStr}</div>
        </div>
        <button class="btn btn-sm btn-outline-secondary" onclick="openCustomizePanel()">
          <i class="bi bi-layout-wtf me-1"></i>Customize
        </button>
      </div>

      ${showLowStock ? renderLowStockAlert(lowOITW, itemsMap, whsMap) : ''}

      ${showKpiSection ? `<div class="row g-3 mb-4">${visibleKpiCards}</div>` : ''}

      ${showQuickActions ? `
        <div class="fw-semibold text-muted small text-uppercase mb-2">Quick Actions</div>
        <div class="row g-3 mb-4">${renderQuickActions()}</div>` : ''}

      ${showWarehouseStock ? renderWarehouseStock(whsList, allOITW) : ''}

      ${showRecentSection ? `
        <div class="fw-semibold text-muted small text-uppercase mb-2">Recent Activity</div>
        <div class="row g-3">
          ${showRecentPayments    ? renderRecentPayments(allPayments, paymentsFullWidth)          : ''}
          ${showRecentCollections ? renderRecentCollections(allCollections, collectionsFullWidth) : ''}
        </div>` : ''}

      ${!anyVisible ? renderEmptyDashboard() : ''}
    </div>
  `);
}
