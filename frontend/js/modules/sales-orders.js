// sales-orders.js

// ── Sales Orders ──────────────────────────────────────────────────────────────

let _soList   = [];
let _soDetail = null;

async function loadSalesOrdersList(tabBar) {
  _soList = await api.ListSalesOrders() || [];
  if (!_soList) {
    showView(wrapSales(tabBar, `<div class="alert alert-warning">Failed to load sales orders.</div>`));
    return;
  }

  // Pre-compute effective status for filtering/display
  _soList.forEach(o => { o._eff = _soEffectiveStatus(o); });

  const statusOpts = [
    { v: 'Draft,Unpaid,Partial', l: 'Active (Default)' },
    { v: 'Draft',          l: 'Draft' },
    { v: 'Unpaid,Partial', l: 'Unpaid / Partial' },
    { v: 'Unpaid',         l: 'Unpaid' },
    { v: 'Partial',        l: 'Partial' },
    { v: 'Paid',           l: 'Paid' },
    { v: 'Cancelled',      l: 'Cancelled' },
    { v: '',               l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    <!-- Header: title + New button -->
    <div class="d-flex align-items-center justify-content-between mb-3">
      <h5 class="fw-bold mb-0">Sales Orders</h5>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/sales/orders/new')">
        <i class="bi bi-plus-lg me-1"></i>New Sales Order
      </button>
    </div>

    <!-- KPI summary strip (updated by soFilter) -->
    <div class="row g-3 mb-3">
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-bag me-1"></i>Total Value</div>
            <div class="fw-bold fs-5 text-dark" id="so-kpi-total">—</div>
            <div class="text-muted" style="font-size:.7rem" id="so-kpi-total-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-truck me-1 text-primary"></i>Delivered</div>
            <div class="fw-bold fs-5 text-primary" id="so-kpi-delivered">—</div>
            <div class="text-muted" style="font-size:.7rem" id="so-kpi-delivered-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-hourglass-split me-1 text-warning"></i>Outstanding</div>
            <div class="fw-bold fs-5 text-warning" id="so-kpi-outstanding">—</div>
            <div class="text-muted" style="font-size:.7rem" id="so-kpi-outstanding-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-files me-1 text-danger"></i>Open Orders</div>
            <div class="fw-bold fs-5 text-danger" id="so-kpi-open">—</div>
            <div class="text-muted" style="font-size:.7rem" id="so-kpi-open-sub"></div>
          </div>
        </div>
      </div>
    </div>

${_filterBar(statusOpts, 'Draft,Unpaid,Partial', 'soFilter()', {
      doc:  _soList.map(o => o.sales_order_number),
      cust: _soList.map(o => o.customer_name_snapshot || (o.customer && o.customer.name) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>SO #</th><th>Date</th><th>Customer</th><th>Amount</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="so-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  soFilter();
}


window.soFilter = function() {
  const filtered = _applyFilters(_soList, {
    docField: 'sales_order_number', amountField: 'grand_total', statusField: '_eff',
    defaultStatuses: ['Draft','Unpaid','Partial'],
  });

  // ── KPI computation (all non-Cancelled, date-filtered) ────────────────────
  const dateFrom = _sfVal('sf-date-from');
  const dateTo   = _sfVal('sf-date-to');
  const activeSO = _soList.filter(o => {
    if (o.doc_status === 'Cancelled') return false;
    const d = (o.date || '').slice(0, 10);
    if (dateFrom && d < dateFrom) return false;
    if (dateTo   && d > dateTo)   return false;
    return true;
  });
  const soTotal       = activeSO.reduce((s, o) => s + (o.grand_total      ?? 0), 0);
  const soDelivered   = activeSO.reduce((s, o) => s + (o.amount_delivered  ?? 0), 0);
  const soCollected   = activeSO.reduce((s, o) => s + (o.amount_collected  ?? 0), 0);
  const soOutstanding = soTotal - soCollected;
  const openOrders    = activeSO.filter(o => o.doc_status === 'Open' || o.doc_status === 'Draft');
  const _kpi = id => document.getElementById(id);
  if (_kpi('so-kpi-total'))          _kpi('so-kpi-total').textContent          = formatCurrency(soTotal);
  if (_kpi('so-kpi-total-sub'))      _kpi('so-kpi-total-sub').textContent      = `${activeSO.length} order${activeSO.length !== 1 ? 's' : ''}`;
  if (_kpi('so-kpi-delivered'))      _kpi('so-kpi-delivered').textContent      = formatCurrency(soDelivered);
  if (_kpi('so-kpi-delivered-sub'))  _kpi('so-kpi-delivered-sub').textContent  = soTotal > 0 ? `${Math.round(soDelivered / soTotal * 100)}% of total` : '';
  if (_kpi('so-kpi-outstanding'))    _kpi('so-kpi-outstanding').textContent    = formatCurrency(soOutstanding);
  if (_kpi('so-kpi-outstanding-sub')) _kpi('so-kpi-outstanding-sub').textContent = soTotal > 0 ? `${Math.round(soOutstanding / soTotal * 100)}% uncollected` : '';
  if (_kpi('so-kpi-open'))           _kpi('so-kpi-open').textContent           = openOrders.length;
  if (_kpi('so-kpi-open-sub'))       _kpi('so-kpi-open-sub').textContent       = openOrders.length > 0
    ? formatCurrency(openOrders.reduce((s, o) => s + Math.max(0, (o.grand_total ?? 0) - (o.amount_collected ?? 0)), 0)) + ' due'
    : activeSO.length > 0 ? 'All closed/paid' : '';

  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No sales orders match the filter.</td></tr>`
    : filtered.map(o => `
        <tr>
          <td class="fw-semibold">${o.sales_order_number || '—'}</td>
          <td>${formatDate(o.date)}</td>
          <td>${o.customer_name_snapshot || (o.customer && o.customer.name) || '—'}</td>
          <td>${formatCurrency(o.grand_total ?? 0)}</td>
          <td>${salesStatusBadge(o._eff)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary me-1" onclick="navigate('#/sales/orders/${o.id}')">
              <i class="bi bi-eye"></i>
            </button>
            ${o.doc_status === 'Open' ? `
              <button class="btn btn-sm btn-outline-primary" title="Create Delivery Order" onclick="newDOFromSO(${o.id})">
                <i class="bi bi-truck"></i>
              </button>` : ''}
          </td>
        </tr>`).join('');
  const el = document.getElementById('so-tbody');
  if (el) el.innerHTML = rows;
};

// ── Sales Order Detail ────────────────────────────────────────────────────────

async function loadSODetail(id) {
  showLoading();
  const so = await api.GetSalesOrder(id);
  if (!so) { showView(`<div class="alert alert-warning m-4">Sales order not found.</div>`); return; }
  _soDetail = so;

  const items     = so.items || [];
  const deliveries = so.deliveries || [];
  const arInvoices = so.ar_invoices || [];
  const draftDO   = deliveries.find(d => d.status === 'Draft');

  const itemRows = items.map(i => `
    <tr>
      <td class="ps-4 fw-semibold">${i.sku}</td>
      <td class="text-center text-muted">${i.unit}</td>
      <td class="text-center">${formatNumber(i.quantity)}</td>
      <td class="text-center ${i.open_qty > 0.001 ? 'text-warning fw-semibold' : 'text-muted'}">${formatNumber(i.open_qty ?? i.quantity)}</td>
      <td class="text-center">${formatCurrency(i.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.line_total)}</td>
    </tr>`).join('') || `<tr><td colspan="6" class="text-center text-muted py-3">No items.</td></tr>`;

  const doRows = deliveries.map(d => {
    const doTotal = (d.items || []).reduce((s, i) => s + ((i.quantity_delivered || 0) * (i.price_per_unit || 0)), 0);
    return `
    <tr>
      <td class="ps-4 fw-semibold">
        <a href="#" onclick="navigate('#/sales/delivery-orders/${d.id}');return false;" class="text-decoration-none">${d.delivery_number}</a>
      </td>
      <td class="text-muted small">${formatDate(d.date)}</td>
      <td>${salesStatusBadge(d.status)}</td>
      <td class="text-end">${formatCurrency(doTotal)}</td>
      <td class="text-end pe-3">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/delivery-orders/${d.id}')">View</button>
      </td>
    </tr>`;
  }).join('') || `<tr><td colspan="5" class="text-center text-muted py-3">No delivery orders.</td></tr>`;

  const invRows = arInvoices.map(i => {
    const bal = Math.max(0, (i.total_amount || 0) - (i.amount_collected || 0));
    return `<tr>
      <td class="ps-4 fw-semibold">
        <a href="#" onclick="navigate('#/sales/ar-invoices/${i.id}');return false;" class="text-decoration-none">${i.invoice_number}</a>
      </td>
      <td class="text-muted small">${formatDate(i.date)}</td>
      <td>${formatCurrency(i.total_amount || 0)}</td>
      <td class="fw-bold ${bal <= 0.005 ? 'text-success' : 'text-danger'}">${bal <= 0.005 ? 'Paid' : formatCurrency(bal)}</td>
      <td>${salesStatusBadge(i.status)}</td>
    </tr>`;
  }).join('') || `<tr><td colspan="5" class="text-center text-muted py-3">No AR invoices.</td></tr>`;

  const isDraft     = so.doc_status === 'Draft';
  const isOpen      = so.doc_status === 'Open';
  const isCancelled = so.doc_status === 'Cancelled';
  const canCancel   = !isCancelled && so.doc_status !== 'Closed' && deliveries.length === 0 && arInvoices.length === 0;
  const effStatus   = _soEffectiveStatus(so);
  const custName    = so.customer_name_snapshot || (so.customer && so.customer.name) || '—';

  // Fulfillment progress percentages
  const _gt = so.grand_total || 0;
  const _pDel = _gt > 0 ? Math.min(100, Math.round((so.amount_delivered || 0) / _gt * 100)) : 0;
  const _pInv = _gt > 0 ? Math.min(100, Math.round((so.amount_invoiced  || 0) / _gt * 100)) : 0;
  const _pCol = _gt > 0 ? Math.min(100, Math.round((so.amount_collected || 0) / _gt * 100)) : 0;
  const _fulfillmentBar = `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-body py-3 px-4">
        <div class="small text-muted fw-bold mb-2"><i class="bi bi-bar-chart-steps me-2"></i>FULFILLMENT PROGRESS</div>
        <div class="d-flex align-items-start flex-nowrap gap-0 overflow-auto">
          <div class="flex-fill text-center px-2" style="min-width:90px">
            <div class="text-muted" style="font-size:.6rem;text-transform:uppercase;letter-spacing:.05em">Ordered</div>
            <div class="fw-bold text-dark mt-1 small">${formatCurrency(_gt)}</div>
            <div class="progress my-1" style="height:4px"><div class="progress-bar bg-secondary" style="width:100%"></div></div>
            <div class="text-muted" style="font-size:.65rem">100%</div>
          </div>
          <div class="text-muted align-self-center pb-4 px-1" style="font-size:.8rem">›</div>
          <div class="flex-fill text-center px-2" style="min-width:90px">
            <div class="text-muted" style="font-size:.6rem;text-transform:uppercase;letter-spacing:.05em">Delivered</div>
            <div class="fw-bold text-primary mt-1 small">${formatCurrency(so.amount_delivered || 0)}</div>
            <div class="progress my-1" style="height:4px"><div class="progress-bar bg-primary" style="width:${_pDel}%"></div></div>
            <div class="text-muted" style="font-size:.65rem">${_pDel}%</div>
          </div>
          <div class="text-muted align-self-center pb-4 px-1" style="font-size:.8rem">›</div>
          <div class="flex-fill text-center px-2" style="min-width:90px">
            <div class="text-muted" style="font-size:.6rem;text-transform:uppercase;letter-spacing:.05em">Invoiced</div>
            <div class="fw-bold text-info mt-1 small">${formatCurrency(so.amount_invoiced || 0)}</div>
            <div class="progress my-1" style="height:4px"><div class="progress-bar bg-info" style="width:${_pInv}%"></div></div>
            <div class="text-muted" style="font-size:.65rem">${_pInv}%</div>
          </div>
          <div class="text-muted align-self-center pb-4 px-1" style="font-size:.8rem">›</div>
          <div class="flex-fill text-center px-2" style="min-width:90px">
            <div class="text-muted" style="font-size:.6rem;text-transform:uppercase;letter-spacing:.05em">Collected</div>
            <div class="fw-bold text-success mt-1 small">${formatCurrency(so.amount_collected || 0)}</div>
            <div class="progress my-1" style="height:4px"><div class="progress-bar bg-success" style="width:${_pCol}%"></div></div>
            <div class="text-muted" style="font-size:.65rem">${_pCol}%</div>
          </div>
        </div>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/orders')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-bag me-2 text-primary"></i>${so.sales_order_number}</h4>
          <small class="text-muted">Sales Order · ${formatDate(so.date)}</small>
        </div>
        ${salesStatusBadge(effStatus)}
        <button class="btn btn-sm btn-outline-secondary" onclick="printSO()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('sales',${so.id},'SO',${so.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
        ${isDraft ? `
          <button class="btn btn-sm btn-primary" onclick="submitSO(${so.id})">
            <i class="bi bi-check-lg me-1"></i>Submit
          </button>
          <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/orders/${so.id}/edit')">
            <i class="bi bi-pencil me-1"></i>Edit
          </button>` : ''}
        ${isOpen ? `
          ${draftDO
            ? `<button class="btn btn-sm btn-warning" onclick="navigate('#/sales/delivery-orders/${draftDO.id}')">
                 <i class="bi bi-truck me-1"></i>View Draft DO (${draftDO.delivery_number})
               </button>`
            : `<button class="btn btn-sm btn-outline-primary" onclick="newDOFromSO(${so.id})">
                 <i class="bi bi-truck me-1"></i>Create Delivery Order
               </button>`}
          <button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromSO(${so.id})">
            <i class="bi bi-file-earmark-plus me-1"></i>Create AR Invoice
          </button>` : ''}
        ${canCancel ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelSO(${so.id})">Cancel</button>` : ''}
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted fw-bold">CUSTOMER</div><div class="fw-semibold">${custName}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">DATE</div><div>${formatDate(so.date)}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">TERMS</div><div>${so.terms || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">PAYMENT METHOD</div><div>${so.payment_method || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">GRAND TOTAL</div><div class="fw-bold text-primary">${formatCurrency(so.grand_total)}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">AMOUNT COLLECTED</div><div class="fw-bold text-success">${formatCurrency(so.amount_collected || 0)}</div></div>
            ${so.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${so.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      ${_fulfillmentBar}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-box-seam me-2 text-primary"></i>Line Items</span></div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Item</th><th class="text-center">Unit</th><th class="text-center">Ordered</th><th class="text-center">Open Qty</th><th class="text-center">Price/Unit</th><th class="text-end pe-4">Total</th></tr></thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="5" class="text-end pe-3 ps-4">Grand Total</td><td class="text-end pe-4 text-primary fs-5">${formatCurrency(so.grand_total)}</td></tr></tfoot>
          </table>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-truck me-2"></i>Delivery Orders</span>
          ${isOpen ? (draftDO
            ? `<button class="btn btn-sm btn-outline-warning" onclick="navigate('#/sales/delivery-orders/${draftDO.id}')">View Draft</button>`
            : `<button class="btn btn-sm btn-outline-primary" onclick="newDOFromSO(${so.id})">+ Create DO</button>`) : ''}
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">DO #</th><th>Date</th><th>Status</th><th class="text-end">Total</th><th></th></tr></thead>
            <tbody>${doRows}</tbody>
          </table>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices</span>
          ${isOpen ? `<button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromSO(${so.id})">+ Create AR Invoice</button>` : ''}
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Invoice #</th><th>Date</th><th>Total</th><th>Balance</th><th>Status</th></tr></thead>
            <tbody>${invRows}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}

function _soEffectiveStatus(o) {
  if (o.doc_status === 'Draft')     return 'Draft';
  if (o.doc_status === 'Cancelled') return 'Cancelled';
  if (o.doc_status === 'Closed')    return o.payment_status === 'Paid' ? 'Paid' : 'Closed';
  return o.payment_status || 'Unpaid';
}

async function submitSO(id) {
  if (!confirm('Submit this sales order? It will become Open and cannot be edited.')) return;
  const r = await api.SubmitSalesOrder(id);
  if (r) await loadSODetail(id);
}

async function cancelSO(id) {
  if (!confirm('Cancel this sales order?')) return;
  const r = await api.CancelSalesOrder(id);
  if (r) await loadSODetail(id);
}

// kept for backward compat from list row
async function voidSO(id) { return cancelSO(id); }

function newDOFromSO(soId) {
  window._newDOPrefillSOId = soId;
  navigate('#/sales/delivery-orders/new');
}

function newARInvoiceFromSO(soId) {
  window._newARInvoicePrefillSOId = soId;
  navigate('#/sales/ar-invoices/new');
}

function newCollectionFromARInv(invId) {
  window._newCollectionPrefillInvId = invId;
  navigate('#/sales/collections/new');
}

// ── New Sales Order Form ───────────────────────────────────────────────────────

let _soItems          = []; // ItemMaster[] from item_master db
let _soCategories     = []; // ItemCategory[] from item_categories db
let _soCustomerPrices = {}; // {name_unit: price}
let _soCustomers      = [];
let _soLineCounter    = 0;
let _soUoMGroups      = []; // UoMGroup[] for unit dropdowns

function soOnCategoryChange(sel) {
  const row       = sel.closest('tr');
  const itemInput = row.querySelector('.so-item');
  if (itemInput) {
    itemInput.value = '';
    const unitSel = row.querySelector('.so-unit');
    if (unitSel) unitSel.innerHTML = '';
    soSuggestItem(itemInput);
  }
}

function soSuggestItem(input) {
  const row  = input.closest('tr');
  const cat  = row.querySelector('.so-cat')?.value || '';
  const term = input.value.toLowerCase();
  const pool = cat ? _soItems.filter(i => String(i.itms_grp_cod) === cat) : _soItems;
  const matches = term
    ? pool.filter(i =>
        (i.item_name || '').toLowerCase().includes(term) ||
        (i.item_code || '').toLowerCase().includes(term)
      ).slice(0, 15)
    : pool.slice(0, 15);

  const sugDiv = input.parentElement?.querySelector('.so-suggestions');
  if (!sugDiv) return;
  if (matches.length === 0) { sugDiv.style.display = 'none'; return; }

  sugDiv.innerHTML = matches.map(i =>
    `<button type="button" class="list-group-item list-group-item-action py-1 px-2 small"
             data-item-name="${i.item_name}"
             onmousedown="soSelectSuggestion(this)">
       <span class="fw-semibold">${i.item_name}</span>
       <small class="text-muted ms-2">${i.item_code}</small>
     </button>`
  ).join('');
  const rect = input.getBoundingClientRect();
  sugDiv.style.top   = rect.bottom + 'px';
  sugDiv.style.left  = rect.left + 'px';
  sugDiv.style.width = rect.width + 'px';
  sugDiv.style.display = 'block';
}

function soSelectSuggestion(btn) {
  const wrapper = btn.closest('.position-relative');
  const input   = wrapper?.querySelector('.so-item');
  const sugDiv  = wrapper?.querySelector('.so-suggestions');
  if (!input) return;
  input.value = btn.dataset.itemName;
  if (sugDiv) sugDiv.style.display = 'none';
  soOnItemSelect(input);
}

function soHideSuggestions(input) {
  setTimeout(() => {
    const sugDiv = input.parentElement?.querySelector('.so-suggestions');
    if (sugDiv) sugDiv.style.display = 'none';
  }, 200);
}

function soOnItemSelect(input) {
  const row  = input.closest('tr');
  const item = _soItems.find(i => i.item_name === input.value);
  if (item) soPopulateUnitDropdown(row, item, '');
  soUpdateRow(input);
  soApplyPriceToRow(row);
}

async function loadNewSOForm() {
  showLoading();
  const [customers, items, cats, groups] = await Promise.all([
    api.ListCustomers(false), api.ListSalesItems(), api.ListItemCategoriesForModule('sales'), api.ListUoMGroups(),
  ]);
  _soCustomers      = customers || [];
  _soItems          = items     || [];
  _soCategories     = cats      || [];
  _soUoMGroups      = groups    || [];
  _soCustomerPrices = {};
  _soLineCounter    = 0;

  const today = new Date().toISOString().slice(0, 10);

  const custOptions = _soCustomers.map(c => {
    const hasPg = c.price_group_id ? ' ★' : '';
    const type  = c.customer_type === 'Account' ? ' (Account)' : '';
    return `<option value="${c.id}"
      data-name="${c.name}"
      data-address="${c.delivery_address || c.address || ''}"
      data-contact="${c.contact_number || ''}"
      data-pgid="${c.price_group_id || ''}"
    >${c.name}${type}${hasPg}</option>`;
  }).join('');

  showView(`
    <div class="container py-4" style="max-width:900px;">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/orders')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div>
          <h4 class="fw-bold mb-0"><i class="bi bi-receipt me-2 text-primary"></i>New Sales Order</h4>
          <small class="text-muted">Fill in customer details and add order line items below</small>
        </div>
      </div>

      <!-- Date -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Order Date</label>
              <input type="date" id="soDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-8">
              <label class="form-label small fw-bold">Invoice # <span class="text-muted fw-normal">(auto-generated on save)</span></label>
              <input type="text" class="form-control bg-light" disabled placeholder="SO-YYYYMMDD-XXXX">
            </div>
          </div>
        </div>
      </div>

      <!-- Customer -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-light border-0 py-3">
          <h6 class="fw-bold mb-0"><i class="bi bi-person me-2"></i>Customer</h6>
        </div>
        <div class="card-body">
          <div class="mb-3">
            <label class="form-label small fw-bold">
              Existing Account <span class="text-muted fw-normal">(optional — leave blank for walk-in)</span>
              <span id="soPriceBadge" class="badge ms-2 rounded-pill d-none"
                    style="background:#dcfce7;color:#16a34a;font-size:11px;">
                <i class="bi bi-tag-fill me-1"></i>Price list loaded
              </span>
            </label>
            <select id="soCustomerSelect" class="form-select" onchange="soFillCustomer(this)">
              <option value="">— Walk-in / New Customer —</option>
              ${custOptions}
            </select>
          </div>
          <div class="row g-3">
            <div class="col-md-5">
              <label class="form-label small fw-bold">Customer Name <span class="text-danger">*</span></label>
              <input type="text" id="soCustName" class="form-control" placeholder="Full name" required>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Contact Number</label>
              <input type="text" id="soCustContact" class="form-control" placeholder="09XX XXX XXXX">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Terms</label>
              <select id="soTerms" class="form-select">
                <option value="COD">COD</option>
                <option value="7 days">7 days</option>
                <option value="15 days">15 days</option>
                <option value="30 days">30 days</option>
              </select>
            </div>
            <div class="col-12">
              <label class="form-label small fw-bold">Delivery Address</label>
              <input type="text" id="soCustAddress" class="form-control" placeholder="Street, Barangay, Municipality">
            </div>
          </div>
        </div>
      </div>

      <!-- Line Items -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-primary bg-opacity-10 border-0 py-3 d-flex justify-content-between align-items-center">
          <h6 class="fw-bold mb-0"><i class="bi bi-box-seam me-2 text-primary"></i>Order Items</h6>
          <button type="button" class="btn btn-sm btn-outline-primary" onclick="soAddRow()">
            <i class="bi bi-plus-lg me-1"></i> Add Item
          </button>
        </div>
        <div class="card-body p-0">
          <div class="table-responsive">
            <table class="table mb-0">
              <thead class="table-light">
                <tr>
                  <th class="ps-3" style="width:18%;">Category</th>
                  <th style="width:24%;">Item</th>
                  <th style="width:10%;" class="text-center">Unit</th>
                  <th style="width:12%;" class="text-end">Qty</th>
                  <th style="width:16%;" class="text-end">Price / Unit (₱)</th>
                  <th style="width:15%;" class="text-end">Line Total</th>
                  <th style="width:5%;"></th>
                </tr>
              </thead>
              <tbody id="soItemsBody"></tbody>
              <tfoot>
                <tr class="table-light">
                  <td colspan="5" class="text-end fw-bold pe-3">Grand Total</td>
                  <td class="fw-bold text-primary fs-5" id="soGrandTotal">₱0.00</td>
                  <td></td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>

      <!-- Payment -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-light border-0 py-3">
          <h6 class="fw-bold mb-0"><i class="bi bi-cash-coin me-2"></i>Payment</h6>
        </div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">Method</label>
              <select id="soPayMethod" class="form-select">
                <option value="Cash">Cash</option>
                <option value="GCash">GCash</option>
                <option value="Bank Transfer">Bank Transfer</option>
                <option value="Cheque">Cheque</option>
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Status</label>
              <select id="soPayStatus" class="form-select" onchange="soToggleDueDate()">
                <option value="Unpaid" selected>Unpaid</option>
                <option value="Partial">Partial</option>
                <option value="Paid">Paid</option>
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Amount Paid (₱)</label>
              <input type="number" step="0.01" id="soAmountPaid" class="form-control" placeholder="0.00" min="0" value="0">
            </div>
            <div class="col-md-3" id="soDueDateGroup">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="soDueDate" class="form-control">
            </div>
          </div>
        </div>
      </div>

      <!-- Notes -->
      <div class="card shadow-sm border-0 mb-4">
        <div class="card-body">
          <label class="form-label small fw-bold">Remarks / Notes</label>
          <textarea id="soNotes" class="form-control" rows="2" placeholder="Delivery instructions, special requests…"></textarea>
        </div>
      </div>

      <div class="d-flex gap-2">
        <button class="btn btn-primary btn-lg fw-bold flex-grow-1" onclick="submitNewSOForm()">
          <i class="bi bi-floppy me-2"></i>Save Order
        </button>
        <button class="btn btn-outline-secondary btn-lg" onclick="navigate('#/sales/orders')">Cancel</button>
      </div>
    </div>
  `);

  soAddRow(); // start with one empty row
  soToggleDueDate();
}

function soFillCustomer(sel) {
  const opt = sel.options[sel.selectedIndex];
  document.getElementById('soCustName').value    = opt.dataset.name    || '';
  document.getElementById('soCustAddress').value = opt.dataset.address || '';
  document.getElementById('soCustContact').value = opt.dataset.contact || '';

  _soCustomerPrices = {};
  document.getElementById('soPriceBadge').classList.add('d-none');

  const pgId = opt.dataset.pgid ? parseInt(opt.dataset.pgid, 10) : null;
  if (!pgId) return;

  api.GetPriceGroup(pgId).then(pg => {
    if (!pg || !pg.items) return;
    pg.items.forEach(item => {
      _soCustomerPrices[`${item.egg_size}|${item.unit}`] = item.price;
    });
    if (Object.keys(_soCustomerPrices).length > 0) {
      const badge = document.getElementById('soPriceBadge');
      badge.innerHTML = `<i class="bi bi-tag-fill me-1"></i>${pg.name}`;
      badge.classList.remove('d-none');
      // Apply prices to existing rows
      document.querySelectorAll('#soItemsBody tr').forEach(row => soApplyPriceToRow(row));
    }
  });
}

// Populates the unit dropdown for a SO line row.
// Shows all UoMs in the item's UoM group (or falls back to the item's sales/inventory unit string).
// Embeds data-uom-entry on each option for DO conversion.
function soPopulateUnitDropdown(row, item, preUnit) {
  const sel = row.querySelector('.so-unit');
  if (!sel) return;
  const units = item?.ugp_entry
    ? getUoMGroupUnits(item.ugp_entry, _soUoMGroups)
    : [];

  const defaultCode = preUnit || item?.sales_uom?.uom_code || item?.invntry_uom || '';
  if (units.length === 0 && defaultCode) {
    units.push({ code: defaultCode, name: defaultCode, entry: item?.s_uom_entry || item?.i_uom_entry || 0 });
  }
  sel.innerHTML = units.map(u =>
    `<option value="${u.code}" data-uom-entry="${u.entry || 0}"${u.code === defaultCode ? ' selected' : ''}>${u.name || u.code}</option>`
  ).join('');
}

function getUoMGroupUnits(ugpEntry, uomGroups) {
  const group = (uomGroups || []).find(g => g.ugp_entry === ugpEntry);
  if (!group) return [];
  const units = [];
  if (group.base_unit?.uom_code) {
    units.push({ code: group.base_unit.uom_code, name: group.base_unit.uom_name, entry: group.base_uom || 0 });
  }
  (group.lines || []).forEach(l => {
    if (l.unit?.uom_code && !units.some(u => u.code === l.unit.uom_code)) {
      units.push({ code: l.unit.uom_code, name: l.unit.uom_name, entry: l.uom_entry || 0 });
    }
  });
  return units;
}

// Resolve price for a UoM code using ITM9 data or auto-calculation from group lines.
// Returns a number price or null if item is unknown.
function resolveUoMPrice(item, uomCode, uomGroups) {
  if (!item) return null;
  const group = (uomGroups || []).find(g => g.ugp_entry === item.ugp_entry);
  const baseCode = group?.base_unit?.uom_code || item.invntry_uom || '';
  // Base UoM — use avg_price directly
  if (!uomCode || uomCode === baseCode) return item.avg_price || 0;
  // Check explicit ITM9 price first
  const line = (group?.lines || []).find(l => l.unit?.uom_code === uomCode);
  if (line) {
    const explicit = (item.uom_prices || []).find(p => p.uom_entry === line.uom_entry);
    if (explicit && explicit.price > 0) {
      return explicit.factor !== 0
        ? explicit.price * (1 - explicit.factor / 100)
        : explicit.price;
    }
    // Auto-calculate from conversion factor
    if (line.alt_qty > 0) {
      return (item.avg_price || 0) * (line.base_qty / line.alt_qty);
    }
  }
  return item.avg_price || 0;
}

function soOnUnitChange(sel) {
  const row      = sel.closest('tr');
  const itemName = row.querySelector('.so-item')?.value;
  const item     = (_soItems || []).find(i => i.item_name === itemName);
  const price    = resolveUoMPrice(item, sel.value, _soUoMGroups);
  if (price !== null) {
    const priceEl = row.querySelector('.so-price');
    if (priceEl) { priceEl.value = price.toFixed(2); soUpdateRow(priceEl); }
  }
}

function soAddRow(preCategory = '', preItem = '', preUnit = '', preQty = '', prePrice = '') {
  const tbody = document.getElementById('soItemsBody');
  if (!tbody) return;
  _soLineCounter++;

  const catOptions = _soCategories.map(c =>
    `<option value="${c.itms_grp_cod}"${String(c.itms_grp_cod) === preCategory ? ' selected' : ''}>${c.itms_grp_nam}</option>`
  ).join('');

  const tr = document.createElement('tr');
  tr.innerHTML = `
    <td class="ps-3">
      <select class="form-select form-select-sm so-cat" onchange="soOnCategoryChange(this)">
        <option value="">— All —</option>
        ${catOptions}
      </select>
    </td>
    <td>
      <div class="position-relative">
        <input type="text" class="form-control form-control-sm so-item"
               value="${preItem}" placeholder="Type to search…"
               autocomplete="off"
               oninput="soSuggestItem(this)" onfocus="soSuggestItem(this)" onblur="soHideSuggestions(this)" required>
        <div class="so-suggestions list-group"
             style="position:fixed;z-index:9999;display:none;max-height:200px;overflow-y:auto;"></div>
      </div>
    </td>
    <td>
      <select class="form-select form-select-sm text-center so-unit" onchange="soOnUnitChange(this)"></select>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end so-qty"
             value="${preQty}" placeholder="0" min="0.001" step="0.001" required oninput="soUpdateRow(this)">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end so-price"
             value="${prePrice}" placeholder="0.00" min="0" step="0.01" required oninput="soUpdateRow(this)">
    </td>
    <td class="fw-bold so-line-total text-primary">₱0.00</td>
    <td>
      <button type="button" class="btn btn-sm btn-outline-danger py-0"
              onclick="this.closest('tr').remove(); soUpdateGrandTotal();">
        <i class="bi bi-x"></i>
      </button>
    </td>`;
  tbody.appendChild(tr);
  // Populate unit dropdown (handles both new rows and pre-filled rows from existing records)
  if (preItem) {
    const item = _soItems.find(i => i.item_name === preItem);
    if (item) {
      soPopulateUnitDropdown(tr, item, preUnit);
    } else if (preUnit) {
      const sel = tr.querySelector('.so-unit');
      if (sel) sel.innerHTML = `<option value="${preUnit}" selected>${preUnit}</option>`;
    }
  }
  soApplyPriceToRow(tr);
  const recalcEl = tr.querySelector('.so-qty');
  if (recalcEl) soUpdateRow(recalcEl);
}

function soApplyPriceToRow(row) {
  const sku     = row.querySelector('.so-item')?.value;
  const unit    = row.querySelector('.so-unit')?.value;
  const priceEl = row.querySelector('.so-price');
  if (!sku || !priceEl) return;

  const key = `${sku}|${unit}`;
  if (_soCustomerPrices[key] !== undefined) {
    priceEl.value = Number(_soCustomerPrices[key]).toFixed(2);
    priceEl.style.background = '#f0fdf4';
  } else {
    const item = _soItems.find(i => i.item_name === sku);
    if (item?.avg_price) priceEl.value = Number(item.avg_price).toFixed(2);
    priceEl.style.background = '';
  }
  soUpdateRow(priceEl);
}

function soUpdateRow(el) {
  const row   = el.closest('tr');
  const qty   = parseFloat(row.querySelector('.so-qty')?.value)   || 0;
  const price = parseFloat(row.querySelector('.so-price')?.value) || 0;
  row.querySelector('.so-line-total').textContent = formatCurrency(qty * price);
  soUpdateGrandTotal();
}

function soUpdateGrandTotal() {
  let grand = 0;
  document.querySelectorAll('#soItemsBody tr').forEach(row => {
    const qty   = parseFloat(row.querySelector('.so-qty')?.value)   || 0;
    const price = parseFloat(row.querySelector('.so-price')?.value) || 0;
    grand += qty * price;
  });
  document.getElementById('soGrandTotal').textContent = formatCurrency(grand);
}

function soToggleDueDate() {
  const status = document.getElementById('soPayStatus').value;
  const show   = status === 'Unpaid' || status === 'Partial';
  document.getElementById('soDueDateGroup').style.display = show ? '' : 'none';
}

async function submitNewSOForm() {
  const customerIdRaw = document.getElementById('soCustomerSelect').value;
  const custName      = document.getElementById('soCustName').value.trim();
  if (!custName) { toast('Customer name is required.', 'warning'); return; }

  // Collect items
  const items = [];
  let valid = true;
  document.querySelectorAll('#soItemsBody tr').forEach(row => {
    const sku   = row.querySelector('.so-item')?.value;
    const unit  = row.querySelector('.so-unit')?.value;
    const qty   = parseFloat(row.querySelector('.so-qty')?.value)   || 0;
    const price = parseFloat(row.querySelector('.so-price')?.value) || 0;
    if (!sku) { valid = false; return; }
    const unitSel  = row.querySelector('.so-unit');
    const uomEntry = parseInt(unitSel?.selectedOptions[0]?.dataset.uomEntry || 0);
    items.push({ sku, unit, uom_entry: uomEntry, quantity: qty, price_per_unit: price, line_total: qty * price });
  });
  if (!valid) { toast('Please select an item for all rows.', 'warning'); return; }
  if (items.length === 0) { toast('Add at least one item.', 'warning'); return; }

  const dueDateVal = document.getElementById('soDueDate').value;
  const payStatus  = document.getElementById('soPayStatus').value;

  const payload = {
    date:             document.getElementById('soDate').value,
    customer_id:      customerIdRaw ? parseInt(customerIdRaw, 10) : null,
    customer_name:    custName,
    customer_addr:    document.getElementById('soCustAddress').value.trim(),
    customer_contact: document.getElementById('soCustContact').value.trim(),
    payment_method:   document.getElementById('soPayMethod').value,
    terms:            document.getElementById('soTerms').value,
    due_date:         (payStatus !== 'Paid' && dueDateVal) ? dueDateVal : null,
    notes:            document.getElementById('soNotes').value.trim(),
    items,
  };

  const btn = document.querySelector('[onclick="submitNewSOForm()"]');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Saving…';

  const result = await api.CreateSalesOrder(payload);
  if (result) {
    toast('Sales order created.', 'success');
    navigate('#/sales/orders/' + result.id);
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-floppy me-2"></i>Save Order';
  }
}

// ── Edit Sales Order Form (Draft only) ────────────────────────────────────────

async function loadEditSOForm(id) {
  showLoading();
  const [so, customers, items, cats] = await Promise.all([
    api.GetSalesOrder(id), api.ListCustomers(false), api.ListSalesItems(), api.ListItemCategoriesForModule('sales'),
  ]);
  if (!so) { showView(`<div class="alert alert-warning m-4">Sales order not found.</div>`); return; }
  if (so.doc_status !== 'Draft') {
    toast('Only Draft sales orders can be edited.', 'warning');
    navigate('#/sales/orders/' + id);
    return;
  }

  _soCustomers      = customers || [];
  _soItems          = items     || [];
  _soCategories     = cats      || [];
  _soCustomerPrices = {};
  _soLineCounter    = 0;

  const custOptions = _soCustomers.map(c => {
    const sel = c.id === so.customer_id ? ' selected' : '';
    return `<option value="${c.id}"${sel} data-name="${c.name}" data-address="${c.delivery_address || c.address || ''}" data-contact="${c.contact_number || ''}" data-pgid="${c.price_group_id || ''}">${c.name}</option>`;
  }).join('');

  const fmtDueDate  = so.due_date ? so.due_date.slice(0,10) : '';
  const fmtDate     = so.date     ? so.date.slice(0,10)     : '';

  showView(`
    <div class="container py-4" style="max-width:900px;">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/orders/${id}')"><i class="bi bi-arrow-left"></i></button>
        <div>
          <h4 class="fw-bold mb-0"><i class="bi bi-pencil me-2 text-warning"></i>Edit Sales Order</h4>
          <small class="text-muted">${so.sales_order_number} · Draft</small>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Order Date</label>
              <input type="date" id="soDate" class="form-control" value="${fmtDate}" required>
            </div>
            <div class="col-md-8">
              <label class="form-label small fw-bold">SO # <span class="text-muted fw-normal">(auto)</span></label>
              <input type="text" class="form-control bg-light" disabled value="${so.sales_order_number}">
            </div>
          </div>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-light border-0 py-3"><h6 class="fw-bold mb-0"><i class="bi bi-person me-2"></i>Customer</h6></div>
        <div class="card-body">
          <div class="mb-3">
            <label class="form-label small fw-bold">Existing Account <span class="text-muted fw-normal">(optional)</span></label>
            <select id="soCustomerSelect" class="form-select" onchange="soFillCustomer(this)">
              <option value="">— Walk-in / New Customer —</option>
              ${custOptions}
            </select>
          </div>
          <div class="row g-3">
            <div class="col-md-5">
              <label class="form-label small fw-bold">Customer Name <span class="text-danger">*</span></label>
              <input type="text" id="soCustName" class="form-control" value="${so.customer_name_snapshot || ''}" required>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Contact Number</label>
              <input type="text" id="soCustContact" class="form-control" value="${so.customer_contact_snapshot || ''}">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Terms</label>
              <select id="soTerms" class="form-select">
                ${['COD','7 days','15 days','30 days'].map(t => `<option value="${t}"${so.terms===t?' selected':''}>${t}</option>`).join('')}
              </select>
            </div>
            <div class="col-12">
              <label class="form-label small fw-bold">Delivery Address</label>
              <input type="text" id="soCustAddress" class="form-control" value="${so.customer_address_snapshot || ''}">
            </div>
          </div>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-primary bg-opacity-10 border-0 py-3 d-flex justify-content-between align-items-center">
          <h6 class="fw-bold mb-0"><i class="bi bi-box-seam me-2 text-primary"></i>Order Items</h6>
          <button type="button" class="btn btn-sm btn-outline-primary" onclick="soAddRow()"><i class="bi bi-plus-lg me-1"></i>Add Item</button>
        </div>
        <div class="card-body p-0">
          <div class="table-responsive">
            <table class="table mb-0">
              <thead class="table-light">
                <tr>
                  <th class="ps-3" style="width:18%;">Category</th>
                  <th style="width:24%;">Item</th>
                  <th style="width:10%;" class="text-center">Unit</th>
                  <th style="width:12%;" class="text-end">Qty</th>
                  <th style="width:16%;" class="text-end">Price / Unit (₱)</th>
                  <th style="width:15%;" class="text-end">Line Total</th>
                  <th style="width:5%;"></th>
                </tr>
              </thead>
              <tbody id="soItemsBody"></tbody>
              <tfoot>
                <tr class="table-light">
                  <td colspan="5" class="text-end fw-bold pe-3">Grand Total</td>
                  <td class="fw-bold text-primary fs-5" id="soGrandTotal">₱0.00</td>
                  <td></td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-light border-0 py-3"><h6 class="fw-bold mb-0"><i class="bi bi-cash-coin me-2"></i>Payment</h6></div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Method</label>
              <select id="soPayMethod" class="form-select">
                ${['Cash','GCash','Bank Transfer','Cheque'].map(m => `<option value="${m}"${so.payment_method===m?' selected':''}>${m}</option>`).join('')}
              </select>
            </div>
            <div class="col-md-4" id="soDueDateGroup">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="soDueDate" class="form-control" value="${fmtDueDate}">
            </div>
          </div>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-4">
        <div class="card-body">
          <label class="form-label small fw-bold">Remarks / Notes</label>
          <textarea id="soNotes" class="form-control" rows="2">${so.notes || ''}</textarea>
        </div>
      </div>

      <div class="d-flex gap-2">
        <button class="btn btn-warning btn-lg fw-bold flex-grow-1" onclick="submitEditSOForm(${id})">
          <i class="bi bi-floppy me-2"></i>Update Order
        </button>
        <button class="btn btn-outline-secondary btn-lg" onclick="navigate('#/sales/orders/${id}')">Cancel</button>
      </div>
    </div>
  `);

  // Pre-populate existing items using soAddRow with pre-filled values
  (so.items || []).forEach(item => {
    const masterItem = _soItems.find(i => i.item_name === item.sku);
    soAddRow(
      String(masterItem?.itms_grp_cod || ''),
      item.sku             || '',
      item.unit            || '',
      item.quantity        != null ? item.quantity        : '',
      item.price_per_unit  != null ? item.price_per_unit  : '',
    );
  });
  soUpdateGrandTotal();
}

async function submitEditSOForm(id) {
  const customerIdRaw = document.getElementById('soCustomerSelect').value;
  const custName      = document.getElementById('soCustName').value.trim();
  if (!custName) { toast('Customer name is required.', 'warning'); return; }

  const items = [];
  let valid = true;
  document.querySelectorAll('#soItemsBody tr').forEach(row => {
    const sku   = row.querySelector('.so-item')?.value;
    const unit  = row.querySelector('.so-unit')?.value;
    const qty   = parseFloat(row.querySelector('.so-qty')?.value)   || 0;
    const price = parseFloat(row.querySelector('.so-price')?.value) || 0;
    if (!sku) { valid = false; return; }
    const unitSel  = row.querySelector('.so-unit');
    const uomEntry = parseInt(unitSel?.selectedOptions[0]?.dataset.uomEntry || 0);
    items.push({ sku, unit, uom_entry: uomEntry, quantity: qty, price_per_unit: price, line_total: qty * price });
  });
  if (!valid) { toast('Please select an item for all rows.', 'warning'); return; }
  if (items.length === 0) { toast('Add at least one item.', 'warning'); return; }

  const dueDateVal = document.getElementById('soDueDate').value;

  const payload = {
    date:             document.getElementById('soDate').value,
    customer_id:      customerIdRaw ? parseInt(customerIdRaw, 10) : null,
    customer_name:    custName,
    customer_addr:    document.getElementById('soCustAddress').value.trim(),
    customer_contact: document.getElementById('soCustContact').value.trim(),
    payment_method:   document.getElementById('soPayMethod').value,
    terms:            document.getElementById('soTerms').value,
    due_date:         dueDateVal || null,
    notes:            document.getElementById('soNotes').value.trim(),
    items,
  };

  const btn = document.querySelector('[onclick="submitEditSOForm(' + id + ')"]');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Saving…';

  const result = await api.UpdateSalesOrder(id, payload);
  if (result) {
    toast('Sales order updated.', 'success');
    navigate('#/sales/orders/' + id);
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-floppy me-2"></i>Update Order';
  }
}

function printSO() {
  const so = _soDetail;
  if (!so) return;
  const f = window._pfmt;
  const custName = so.customer_name_snapshot || (so.customer && so.customer.name) || '—';
  const items = so.items || [];
  const itemRows = items.map(i => `
    <tr>
      <td>${i.sku || '—'}</td>
      <td>${i.unit || '—'}</td>
      <td class="text-end">${f.num(i.quantity)}</td>
      <td class="text-end">${f.num(i.open_qty ?? i.quantity)}</td>
      <td class="text-end">${f.currency(i.price_per_unit)}</td>
      <td class="text-end">${f.currency(i.line_total)}</td>
    </tr>`).join('');
  const grandTotal = items.reduce((s, i) => s + (i.line_total || 0), 0);
  const html = `
    <h2>SALES ORDER</h2>
    <div class="sub">${so.sales_order_number || '—'} &nbsp;·&nbsp; <span class="badge">${so.doc_status || '—'}</span> &nbsp;·&nbsp; ${f.date(so.date)}</div>
    <div class="info-grid">
      <div><span>Customer</span><br>${custName}</div>
      <div><span>Date</span><br>${f.date(so.date)}</div>
      <div><span>Payment Method</span><br>${so.payment_method || '—'}</div>
      <div><span>Terms</span><br>${so.terms || '—'}</div>
      ${so.notes ? `<div style="grid-column:1/-1"><span>Notes</span><br>${so.notes}</div>` : ''}
    </div>
    <table>
      <thead><tr><th>Item/SKU</th><th>Unit</th><th class="text-end">Qty</th><th class="text-end">Open Qty</th><th class="text-end">Unit Price</th><th class="text-end">Total</th></tr></thead>
      <tbody>${itemRows}</tbody>
    </table>
    <div class="totals">Grand Total: <strong>${f.currency(grandTotal)}</strong></div>`;
  _printDoc('SO ' + (so.sales_order_number || ''), html);
}

function _resetSalesOrders() {
  _soList = []; _soDetail = null;
  _soItems = []; _soCategories = []; _soCustomerPrices = {};
  _soCustomers = []; _soLineCounter = 0; _soUoMGroups = [];
  window._newDOPrefillSOId = null;
  window._newARInvoicePrefillSOId = null;
  window._newCollectionPrefillInvId = null;
}
