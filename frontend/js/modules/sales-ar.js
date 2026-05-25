// sales-ar.js

// ── AR Invoices ───────────────────────────────────────────────────────────────

let _arList   = [];
let _arDetail = null;
window._arOverdueOnly = false; // true when the "Overdue" preset is active

async function loadArInvoicesList(tabBar) {
  _arList = await api.ListARInvoices(null, '') || [];

  const statusOpts = [
    { v: 'Open,Partial', l: 'Open / Partial' },
    { v: 'Open',    l: 'Open' },
    { v: 'Partial', l: 'Partial' },
    { v: 'Paid',    l: 'Paid' },
    { v: 'Cancelled', l: 'Cancelled' },
    { v: '',        l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <h5 class="fw-bold mb-0">AR Invoices</h5>
      <div class="d-flex gap-2">
        <button class="btn btn-outline-primary btn-sm" onclick="navigate('#/sales/ar-invoices/new')">
          <i class="bi bi-plus-lg me-1"></i>New AR Invoice
        </button>
        <button class="btn btn-primary btn-sm" onclick="navigate('#/sales/ar-invoices/standalone')">
          <i class="bi bi-file-earmark-plus me-1"></i>New Standalone
        </button>
      </div>
    </div>

    <!-- Date preset buttons -->
    <div class="d-flex flex-wrap gap-2 mb-3" id="ar-presets">
      <span class="text-muted small align-self-center me-1"><i class="bi bi-calendar3 me-1"></i>Quick range:</span>
      <button class="btn btn-sm btn-outline-secondary" data-preset="this-month" onclick="arDatePreset('this-month')">This Month</button>
      <button class="btn btn-sm btn-outline-secondary" data-preset="last-30"    onclick="arDatePreset('last-30')">Last 30 Days</button>
      <button class="btn btn-sm btn-outline-secondary" data-preset="last-90"    onclick="arDatePreset('last-90')">Last 90 Days</button>
      <button class="btn btn-sm btn-outline-secondary" data-preset="custom"     onclick="arDatePreset('custom')"><i class="bi bi-sliders me-1"></i>Custom Range</button>
      <button class="btn btn-sm btn-outline-danger"    data-preset="overdue"    onclick="arDatePreset('overdue')"><i class="bi bi-exclamation-circle me-1"></i>Overdue</button>
    </div>

    <!-- KPI summary strip (updated by arFilter) -->
    <div class="row g-3 mb-3">
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-receipt me-1"></i>Total Revenue</div>
            <div class="fw-bold fs-5 text-dark" id="ar-kpi-revenue">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ar-kpi-revenue-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-check-circle me-1 text-success"></i>Collected</div>
            <div class="fw-bold fs-5 text-success" id="ar-kpi-collected">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ar-kpi-collected-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-hourglass-split me-1 text-warning"></i>Outstanding</div>
            <div class="fw-bold fs-5 text-warning" id="ar-kpi-outstanding">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ar-kpi-outstanding-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-exclamation-circle me-1 text-danger"></i>Open Invoices</div>
            <div class="fw-bold fs-5 text-danger" id="ar-kpi-open">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ar-kpi-open-sub"></div>
          </div>
        </div>
      </div>
    </div>

    ${_filterBar(statusOpts, 'Open,Partial', 'arFilter()', {
      doc:  _arList.map(i => i.invoice_number),
      cust: _arList.map(i => i.customer_name_snapshot || (i.customer && i.customer.name) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>AR Invoice #</th><th>Date</th><th>Due</th><th>Customer</th><th>Amount</th><th>Balance</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="ar-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  arFilter();
}

window.arDatePreset = function(preset) {
  const today = new Date();
  const fmt   = d => d.toISOString().slice(0, 10);
  let from = '', to = fmt(today);

  // Reset overdue mode unless this IS the overdue preset
  if (preset !== 'overdue') window._arOverdueOnly = false;

  if (preset === 'this-month') {
    from = fmt(new Date(today.getFullYear(), today.getMonth(), 1));
  } else if (preset === 'last-30') {
    const d = new Date(today); d.setDate(d.getDate() - 30); from = fmt(d);
  } else if (preset === 'last-90') {
    const d = new Date(today); d.setDate(d.getDate() - 90); from = fmt(d);
  } else if (preset === 'custom') {
    const fromEl = document.getElementById('sf-date-from');
    if (fromEl) { fromEl.focus(); fromEl.showPicker && fromEl.showPicker(); }
    _arMarkPreset('custom');
    return;
  } else if (preset === 'overdue') {
    // Clear date range, force status to Open/Partial, enable overdue flag
    const fromEl = document.getElementById('sf-date-from');
    const toEl   = document.getElementById('sf-date-to');
    const statEl = document.getElementById('sf-status');
    if (fromEl) fromEl.value = '';
    if (toEl)   toEl.value   = '';
    if (statEl) statEl.value = 'Open,Partial';
    window._arOverdueOnly = true;
    _arMarkPreset('overdue');
    arFilter();
    return;
  }

  const fromEl = document.getElementById('sf-date-from');
  const toEl   = document.getElementById('sf-date-to');
  if (fromEl) fromEl.value = from;
  if (toEl)   toEl.value   = to;
  _arMarkPreset(preset);
  arFilter();
};

function _arMarkPreset(active) {
  document.querySelectorAll('#ar-presets [data-preset]').forEach(btn => {
    const isActive = btn.dataset.preset === active;
    btn.classList.toggle('btn-secondary', isActive);
    btn.classList.toggle('btn-outline-secondary', !isActive);
  });
}

window.arFilter = function() {
  // If user changed the status away from Open,Partial while overdue mode was active, exit overdue mode
  if (window._arOverdueOnly && _sfVal('sf-status') !== 'Open,Partial') {
    window._arOverdueOnly = false;
    _arMarkPreset('');
  }

  const filtered = _applyFilters(_arList, {
    docField: 'invoice_number', amountField: 'total_amount',
    defaultStatuses: ['Open','Partial'],
  });

  // When overdue mode is on, further restrict to past-due rows only
  const today = new Date().toISOString().slice(0, 10);
  const displayRows = window._arOverdueOnly
    ? filtered.filter(i => {
        const due = (i.due_date || '').slice(0, 10);
        return due && due < today && (i.status === 'Open' || i.status === 'Partial');
      })
    : filtered;

  // ── KPI computation (all non-Cancelled, date-filtered) ────────────────────
  const dateFrom = _sfVal('sf-date-from');
  const dateTo   = _sfVal('sf-date-to');
  const activeAR = _arList.filter(inv => {
    if (inv.status === 'Cancelled') return false;
    const d = (inv.date || '').slice(0, 10);
    if (dateFrom && d < dateFrom) return false;
    if (dateTo   && d > dateTo)   return false;
    return true;
  });
  const revenue     = activeAR.reduce((s, i) => s + (i.total_amount    ?? 0), 0);
  const collected   = activeAR.reduce((s, i) => s + (i.amount_collected ?? 0), 0);
  const outstanding = revenue - collected;
  const openInvoices   = activeAR.filter(i => i.status === 'Open' || i.status === 'Partial');
  const overdueInvoices = openInvoices.filter(i => {
    const due = (i.due_date || '').slice(0, 10);
    return due && due < today;
  });
  const _kpi = id => document.getElementById(id);
  if (_kpi('ar-kpi-revenue'))       _kpi('ar-kpi-revenue').textContent       = formatCurrency(revenue);
  if (_kpi('ar-kpi-revenue-sub'))   _kpi('ar-kpi-revenue-sub').textContent   = `${activeAR.length} invoice${activeAR.length !== 1 ? 's' : ''}`;
  if (_kpi('ar-kpi-collected'))     _kpi('ar-kpi-collected').textContent     = formatCurrency(collected);
  if (_kpi('ar-kpi-collected-sub')) _kpi('ar-kpi-collected-sub').textContent = revenue > 0 ? `${Math.round(collected / revenue * 100)}% of revenue` : '';
  if (_kpi('ar-kpi-outstanding'))   _kpi('ar-kpi-outstanding').textContent   = formatCurrency(outstanding);
  if (_kpi('ar-kpi-outstanding-sub')) _kpi('ar-kpi-outstanding-sub').textContent = revenue > 0 ? `${Math.round(outstanding / revenue * 100)}% of revenue` : '';
  if (_kpi('ar-kpi-open'))          _kpi('ar-kpi-open').textContent          = openInvoices.length;
  if (_kpi('ar-kpi-open-sub')) {
    const openSub = _kpi('ar-kpi-open-sub');
    if (overdueInvoices.length > 0) {
      openSub.innerHTML = `<span class="text-danger fw-semibold">${overdueInvoices.length} overdue</span>`;
    } else if (openInvoices.length > 0) {
      openSub.textContent = formatCurrency(openInvoices.reduce((s, i) => s + Math.max(0, (i.total_amount ?? 0) - (i.amount_collected ?? 0)), 0)) + ' due';
    } else {
      openSub.textContent = 'All clear';
    }
  }

  const rows = displayRows.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No AR invoices match the filter.</td></tr>`
    : displayRows.map(i => {
        const balance = (i.total_amount ?? 0) - (i.amount_collected ?? 0);
        const due = (i.due_date || '').slice(0, 10);
        const isOverdue = due && due < today && (i.status === 'Open' || i.status === 'Partial');
        const rowCls     = isOverdue ? ' class="table-warning"' : '';
        const overdueBadge = isOverdue ? `<span class="badge bg-danger ms-1">Overdue</span>` : '';
        return `
        <tr${rowCls}>
          <td class="fw-semibold">${i.invoice_number || '—'}</td>
          <td>${formatDate(i.date)}</td>
          <td>${isOverdue ? `<span class="text-danger fw-semibold">${formatDate(i.due_date)}</span>` : formatDate(i.due_date)}</td>
          <td>${i.customer_name_snapshot || (i.customer && i.customer.name) || '—'}</td>
          <td>${formatCurrency(i.total_amount ?? 0)}</td>
          <td class="${balance > 0 ? 'text-danger fw-semibold' : ''}">${formatCurrency(balance)}</td>
          <td>${salesStatusBadge(i.status)}${overdueBadge}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary me-1" onclick="navigate('#/sales/ar-invoices/${i.id}')">
              <i class="bi bi-eye"></i>
            </button>
            ${(i.status === 'Open' || i.status === 'Partial') ? `
              <button class="btn btn-sm btn-outline-success" title="Record Collection" onclick="newCollectionFromARInv(${i.id})">
                <i class="bi bi-cash"></i>
              </button>` : ''}
          </td>
        </tr>`;
      }).join('');
  const el = document.getElementById('ar-tbody');
  if (el) el.innerHTML = rows;
};

// ── AR Invoice Detail ─────────────────────────────────────────────────────────

async function loadARInvoiceDetail(id) {
  showLoading();
  const inv = await api.GetARInvoice(id);
  if (!inv) { showView(`<div class="alert alert-warning m-4">AR Invoice not found.</div>`); return; }
  _arDetail = inv;

  const items   = inv.items || [];
  const colLines = inv.collection_lines || [];
  const collected = inv.amount_collected || 0;
  const balance   = Math.max(0, (inv.total_amount || 0) - collected);
  const so      = inv.sales_order || null;
  const doRecs  = inv.delivery_orders || [];
  const custName = inv.customer_name_snapshot || (inv.customer && inv.customer.name) || '—';

  const statusColor = { Paid: 'success', Partial: 'warning', Open: 'primary', Cancelled: 'secondary' };
  const badge = `<span class="badge bg-${statusColor[inv.status] || 'secondary'} fs-6">${inv.status}</span>`;

  const canCollect = inv.status === 'Open' || inv.status === 'Partial';
  const canCancel  = (inv.status === 'Open' || inv.status === 'Partial') && colLines.length === 0;

  const statusBanner = inv.status === 'Paid'
    ? `<div class="alert alert-success border-0 mb-3 d-flex align-items-center gap-2"><i class="bi bi-check-circle-fill fs-5"></i><strong>Fully Collected</strong> — ${formatCurrency(inv.total_amount)} received</div>`
    : inv.status === 'Partial'
    ? `<div class="alert alert-warning border-0 mb-3 d-flex align-items-center gap-2"><i class="bi bi-clock-fill fs-5"></i>Balance: <strong>${formatCurrency(balance)}</strong> remaining</div>`
    : inv.status === 'Open'
    ? `<div class="alert alert-primary border-0 mb-3">${formatCurrency(inv.total_amount)} outstanding</div>`
    : '';

  const itemRows = items.map(i => `
    <tr>
      <td class="ps-4 fw-semibold">${i.sku}</td>
      <td class="text-center text-muted">${i.unit}</td>
      <td class="text-center">${formatNumber(i.quantity)}</td>
      <td class="text-center">${formatCurrency(i.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity * i.price_per_unit)}</td>
    </tr>`).join('') || `<tr><td colspan="5" class="text-center text-muted py-3">No items.</td></tr>`;

  const colRowsHtml = colLines.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
        <span class="fw-bold"><i class="bi bi-cash-coin me-2 text-success"></i>Collections Applied</span>
        ${canCollect ? `<button class="btn btn-sm btn-outline-success" onclick="newCollectionFromARInv(${inv.id})">+ Add Collection</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th class="ps-4">Collection #</th><th>Date</th><th>Method</th><th>Reference</th><th class="text-end pe-4">Amount Applied</th></tr>
          </thead>
          <tbody>
            ${colLines.map(cl => {
              const col = cl.collection || {};
              return `<tr>
                <td class="ps-4 fw-bold">${col.collection_number || '—'}</td>
                <td class="text-muted small">${formatDate(col.date)}</td>
                <td>${col.payment_method || '—'}</td>
                <td class="text-muted">${col.reference_number || '—'}</td>
                <td class="text-end pe-4 fw-bold text-success">${formatCurrency(cl.amount_applied)}</td>
              </tr>`;
            }).join('')}
          </tbody>
          <tfoot class="table-light fw-bold small">
            <tr><td colspan="4" class="ps-4 text-muted">Invoice Total / Collected / Balance</td><td class="text-end pe-4 text-primary">${formatCurrency(inv.total_amount)}</td></tr>
            <tr><td colspan="4"></td><td class="text-end pe-4 text-success">${formatCurrency(collected)} collected</td></tr>
            <tr><td colspan="4"></td>
              <td class="text-end pe-4 ${balance <= 0.005 ? 'text-success' : 'text-danger fw-bold'}">
                ${balance <= 0.005 ? '<i class="bi bi-check-circle-fill me-1"></i>Fully Collected' : formatCurrency(balance) + ' remaining'}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/ar-invoices')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-file-earmark-text me-2 text-info"></i>${inv.invoice_number}</h4>
          <small class="text-muted">AR Invoice · ${formatDate(inv.date)}</small>
        </div>
        ${badge}
        <button class="btn btn-sm btn-outline-secondary" onclick="printARInvoice()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('sales',${inv.sales_order_id},'AR_INV',${inv.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
        ${canCollect ? `<button class="btn btn-sm btn-success text-white fw-bold" onclick="newCollectionFromARInv(${inv.id})">
          <i class="bi bi-cash-coin me-1"></i>Record Collection</button>` : ''}
        ${canCancel ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelARInv(${inv.id})">Cancel</button>` : ''}
      </div>

      ${statusBanner}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted fw-bold">CUSTOMER</div><div class="fw-semibold">${custName}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">INVOICE DATE</div><div>${formatDate(inv.date)}</div></div>
            ${inv.due_date ? `<div class="col-md-4"><div class="small text-muted fw-bold">DUE DATE</div><div>${formatDate(inv.due_date)}</div></div>` : ''}
            <div class="col-md-4"><div class="small text-muted fw-bold">TERMS</div><div>${inv.terms || '—'}</div></div>
            ${so ? `<div class="col-md-4"><div class="small text-muted fw-bold">SALES ORDER</div>
              <a href="#" onclick="navigate('#/sales/orders/${inv.sales_order_id}');return false;" class="text-decoration-none fw-semibold">${so.sales_order_number}</a></div>` : ''}
            ${doRecs.length > 0 ? `<div class="col-md-${doRecs.length === 1 ? '4' : '8'}"><div class="small text-muted fw-bold">DELIVERY ORDER${doRecs.length > 1 ? 'S' : ''}</div>
              <div class="d-flex flex-wrap gap-2">${doRecs.map(d => `<a href="#" onclick="navigate('#/sales/delivery-orders/${d.id}');return false;" class="text-decoration-none fw-semibold">${d.delivery_number}</a>`).join('<span class="text-muted">·</span>')}</div></div>` : ''}
            ${inv.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${inv.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-box me-2 text-warning"></i>Line Items</span></div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Item / SKU</th><th class="text-center">Unit</th><th class="text-center">Qty</th><th class="text-center">Price/Unit</th><th class="text-end pe-4">Total</th></tr></thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="4" class="text-end pe-3 ps-4">Total Amount</td><td class="text-end pe-4 text-primary fs-5">${formatCurrency(inv.total_amount)}</td></tr></tfoot>
          </table>
        </div>
      </div>

      ${colRowsHtml}
    </div>
  `);
}

async function cancelARInv(id) {
  if (!confirm('Cancel this AR Invoice? This will reverse the accounting entry.')) return;
  const r = await api.CancelARInvoice(id);
  if (r) await loadARInvoiceDetail(id);
}

// ── New AR Invoice Form ───────────────────────────────────────────────────────

let _arInvDOList    = [];
let _arInvCustomers = [];

async function loadNewARInvoiceForm() {
  showLoading();
  const [customers, doList] = await Promise.all([
    api.ListCustomers(false),
    api.ListDeliveryOrders(0),
  ]);
  _arInvCustomers = customers || [];
  _arInvDOList    = (doList || []).filter(d => d.status === 'Delivered');

  const prefillDOId = window._newARInvoicePrefillDOId || null;
  window._newARInvoicePrefillDOId = null;

  const today = new Date().toISOString().slice(0, 10);

  const custOptions = _arInvCustomers.map(c =>
    `<option value="${c.id}">${_esc(c.name)}</option>`
  ).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:1000px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/ar-invoices')">
          <i class="bi bi-arrow-left"></i> Back
        </button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>New AR Invoice</h4>
          <div class="text-muted small">Select a customer, then pick delivery orders to invoice</div>
        </div>
      </div>

      <!-- Header fields -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">Invoice Date <span class="text-danger">*</span></label>
              <input type="date" id="arInvDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Customer <span class="text-danger">*</span></label>
              <select id="arInvCustomer" class="form-select" onchange="arInvOnCustomerChange()" required>
                <option value="">— Select customer —</option>
                ${custOptions}
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="arInvDueDate" class="form-control">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Terms</label>
              <select id="arInvTerms" class="form-select">
                <option>COD</option><option>7 days</option><option>15 days</option><option>30 days</option>
              </select>
            </div>
            <div class="col-12">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="arInvNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <!-- DO selection (hidden until customer chosen) -->
      <div class="card border-0 shadow-sm mb-3" id="arInvDOCard" style="display:none">
        <div class="card-header bg-transparent border-0 pb-0 pt-3 px-4 d-flex align-items-center justify-content-between">
          <span class="fw-bold">Select Delivery Orders</span>
          <button id="arInvGenLinesBtn" class="btn btn-sm btn-primary" disabled onclick="arInvGenerateLines()">
            <i class="bi bi-list-ul me-1"></i>Generate Lines
          </button>
        </div>
        <div class="card-body pt-2 pb-2">
          <div class="table-responsive">
            <table class="table table-sm table-hover align-middle mb-0" id="arInvDOTable">
              <thead class="table-light">
                <tr>
                  <th class="ps-3" style="width:40px"></th>
                  <th>DO #</th>
                  <th>Date</th>
                  <th class="text-end">DO Total</th>
                </tr>
              </thead>
              <tbody></tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Lines (hidden until Generate Lines clicked) -->
      <div class="card border-0 shadow-sm mb-3" id="arInvLinesCard" style="display:none">
        <div class="card-header bg-transparent border-0 pb-0 pt-3 px-4 d-flex align-items-center justify-content-between">
          <span class="fw-bold"><i class="bi bi-box me-2 text-warning"></i>Invoice Items</span>
          <span class="text-muted small">Adjust quantity for partial invoice</span>
        </div>
        <div class="card-body pt-2 pb-2">
          <div class="table-responsive">
            <table class="table table-sm align-middle mb-0">
              <thead class="table-light">
                <tr>
                  <th style="width:36px"></th>
                  <th>Base DO #</th>
                  <th>Item / SKU</th>
                  <th class="text-center">Unit</th>
                  <th class="text-end">Available</th>
                  <th class="text-end" style="width:120px">Qty to Invoice <span class="text-danger">*</span></th>
                  <th class="text-end">Unit Price</th>
                  <th class="text-end">Line Total</th>
                </tr>
              </thead>
              <tbody id="arInvLinesBody"></tbody>
              <tfoot>
                <tr>
                  <td colspan="7" class="text-end fw-bold pe-3">Invoice Total</td>
                  <td class="text-end fw-bold text-primary fs-5 pe-2" id="arInvGrandTotal">₱0.00</td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>

      <div class="d-flex justify-content-end gap-2">
        <button class="btn btn-outline-secondary px-4" onclick="navigate('#/sales/ar-invoices')">Cancel</button>
        <button class="btn btn-primary px-4" id="arInvSubmitBtn" onclick="submitNewARInvoice()">
          <i class="bi bi-file-earmark-text me-1"></i>Create AR Invoice
        </button>
      </div>
    </div>
  `);

  if (prefillDOId) {
    const preDO = _arInvDOList.find(d => d.id === parseInt(prefillDOId));
    if (preDO) {
      const custSel = document.getElementById('arInvCustomer');
      const custId  = preDO.sales_order?.customer_id;
      if (custSel && custId) {
        custSel.value = custId;
        arInvOnCustomerChange();
      }
      const cb = document.querySelector(`#arInvDOTable tbody input[value="${prefillDOId}"]`);
      if (cb) { cb.checked = true; arInvCheckChanged(); arInvGenerateLines(); }
    }
  }
}


function arInvOnCustomerChange() {
  const custId    = parseInt(document.getElementById('arInvCustomer').value) || 0;
  const doCard    = document.getElementById('arInvDOCard');
  const tbody     = document.querySelector('#arInvDOTable tbody');
  const linesCard = document.getElementById('arInvLinesCard');
  const linesBody = document.getElementById('arInvLinesBody');

  if (linesBody) linesBody.innerHTML = '';
  if (linesCard) linesCard.style.display = 'none';
  arInvRecalcGrandTotal();

  if (!custId) {
    if (doCard) doCard.style.display = 'none';
    return;
  }

  const filteredDOs = _arInvDOList.filter(d =>
    (parseInt(d.sales_order?.customer_id) || 0) === custId
  );

  const rows = filteredDOs.length === 0
    ? `<tr><td colspan="4" class="text-center text-muted py-3">No delivered orders for this customer.</td></tr>`
    : filteredDOs.map(d => {
        const doTotal = (d.items || []).reduce((s, i) => s + ((i.quantity_delivered || 0) * (i.price_per_unit || 0)), 0);
        return `
          <tr>
            <td class="text-center ps-3">
              <input type="checkbox" class="form-check-input" value="${d.id}"
                onchange="arInvCheckChanged()">
            </td>
            <td class="fw-semibold">${_esc(d.delivery_number)}</td>
            <td>${formatDate(d.date)}</td>
            <td class="text-end">${formatCurrency(doTotal)}</td>
          </tr>`;
      }).join('');

  if (tbody) tbody.innerHTML = rows;
  if (doCard) doCard.style.display = '';
  arInvCheckChanged();
}


function arInvCheckChanged() {
  const anyChecked = !!document.querySelector('#arInvDOTable tbody input[type=checkbox]:checked');
  const btn = document.getElementById('arInvGenLinesBtn');
  if (btn) btn.disabled = !anyChecked;
}


function arInvGenerateLines() {
  const tbody = document.getElementById('arInvLinesBody');
  const card  = document.getElementById('arInvLinesCard');
  if (!tbody || !card) return;

  const existing = new Set(
    Array.from(tbody.querySelectorAll('tr')).map(r => r.dataset.lineKey)
  );

  document.querySelectorAll('#arInvDOTable tbody input[type=checkbox]:checked').forEach(cb => {
    const doId  = parseInt(cb.value);
    const doRec = _arInvDOList.find(x => x.id === doId);
    if (!doRec) return;
    const doNumber = doRec.delivery_number;

    (doRec.items || []).forEach(item => {
      const qty = parseFloat(item.quantity_delivered) || 0;
      if (qty <= 0) return;
      const lineKey = `${doId}-${item.sku}`;
      if (existing.has(lineKey)) return;
      existing.add(lineKey);

      const price     = parseFloat(item.price_per_unit) || 0;
      const lineTotal = qty * price;
      const row = document.createElement('tr');
      row.dataset.lineKey = lineKey;
      row.innerHTML = `
        <td class="text-center">
          <button type="button" class="btn btn-sm btn-outline-danger py-0 px-1" onclick="arInvRemoveLine(this)">
            <i class="bi bi-x"></i>
          </button>
        </td>
        <td class="text-center"><span class="badge bg-light text-dark border">${_esc(doNumber)}</span></td>
        <td class="fw-semibold">${_esc(item.sku || '')}</td>
        <td class="text-center">${_esc(item.unit || '')}</td>
        <td class="text-end text-muted">${qty.toLocaleString()}</td>
        <td class="text-end" style="width:120px">
          <input type="number" class="form-control form-control-sm text-end ar-inv-qty"
            value="${qty}" min="0.001" max="${qty}" step="0.001"
            data-do-id="${doId}"
            data-sku="${_esc(item.sku || '')}"
            data-unit="${_esc(item.unit || '')}"
            data-price="${price}"
            oninput="arInvRecalcLine(this)">
        </td>
        <td class="text-end text-muted">${formatCurrency(price)}</td>
        <td class="text-end fw-bold text-primary ar-inv-line-total">${formatCurrency(lineTotal)}</td>
      `;
      tbody.appendChild(row);
    });
  });

  arInvRecalcGrandTotal();
  card.style.display = tbody.children.length > 0 ? '' : 'none';
}


function arInvRemoveLine(btn) {
  btn.closest('tr')?.remove();
  arInvRecalcGrandTotal();
  const tbody = document.getElementById('arInvLinesBody');
  const card  = document.getElementById('arInvLinesCard');
  if (card && tbody) card.style.display = tbody.children.length > 0 ? '' : 'none';
}


function arInvRecalcLine(input) {
  const row   = input.closest('tr');
  const qty   = parseFloat(input.value) || 0;
  const price = parseFloat(input.dataset.price) || 0;
  row.querySelector('.ar-inv-line-total').textContent = formatCurrency(qty * price);
  arInvRecalcGrandTotal();
}


function arInvRecalcGrandTotal() {
  let total = 0;
  document.querySelectorAll('#arInvLinesBody .ar-inv-qty').forEach(inp => {
    total += (parseFloat(inp.value) || 0) * (parseFloat(inp.dataset.price) || 0);
  });
  const el = document.getElementById('arInvGrandTotal');
  if (el) el.textContent = formatCurrency(total);
}


async function submitNewARInvoice() {
  const custId  = document.getElementById('arInvCustomer').value;
  const dateVal = document.getElementById('arInvDate').value;

  if (!custId)  { toast('Please select a customer.', 'warning'); return; }
  if (!dateVal) { toast('Invoice date is required.', 'warning'); return; }

  const lineInputs = document.querySelectorAll('#arInvLinesBody .ar-inv-qty');
  if (lineInputs.length === 0) {
    toast('Generate lines from at least one delivery order first.', 'warning');
    return;
  }

  const items   = [];
  const doIdSet = new Set();

  lineInputs.forEach(inp => {
    const qty = parseFloat(inp.value) || 0;
    if (qty <= 0) return;
    const doId = parseInt(inp.dataset.doId) || 0;
    if (doId) doIdSet.add(doId);
    items.push({
      sku:               inp.dataset.sku,
      unit:              inp.dataset.unit,
      quantity:          qty,
      price_per_unit:    parseFloat(inp.dataset.price) || 0,
      delivery_order_id: doId || null,
    });
  });

  if (items.length === 0) { toast('No invoice lines with quantity > 0.', 'warning'); return; }

  const cust       = _arInvCustomers.find(c => c.id === parseInt(custId));
  const dueDateVal = document.getElementById('arInvDueDate').value;

  const btn = document.getElementById('arInvSubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Creating…';

  const result = await api.CreateARInvoice({
    delivery_order_ids: [...doIdSet],
    customer_id:        cust ? cust.id : null,
    customer_name:      cust?.name || '',
    customer_addr:      cust?.delivery_address || cust?.address || '',
    customer_contact:   cust?.contact_number || '',
    date:               dateVal,
    terms:              document.getElementById('arInvTerms').value,
    notes:              document.getElementById('arInvNotes').value,
    due_date:           dueDateVal || null,
    items,
  });

  if (result) {
    navigate('#/sales/ar-invoices/' + result.id);
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-file-earmark-text me-1"></i>Create AR Invoice';
  }
}

// ── Standalone AR Invoice (no SO/DO required) ─────────────────────────────────

let _sarItems      = [];
let _sarCategories = [];
let _sarUoMGroups  = [];

async function loadNewStandaloneARForm() {
  showLoading();
  const [customers, items, cats, groups] = await Promise.all([
    api.ListCustomers(false), api.ListSalesItems(), api.ListItemCategoriesForModule('sales'), api.ListUoMGroups(),
  ]);
  _sarItems      = items  || [];
  _sarCategories = cats   || [];
  _sarUoMGroups  = groups || [];
  const today = new Date().toISOString().slice(0, 10);

  const custOptions = customers.map(c =>
    `<option value="${c.id}" data-name="${c.name}" data-contact="${c.contact_number || ''}" data-address="${c.delivery_address || c.address || ''}">${c.name}</option>`
  ).join('');
  showView(`
    <div class="container py-4" style="max-width:900px;">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/ar-invoices')"><i class="bi bi-arrow-left"></i></button>
        <div>
          <h4 class="fw-bold mb-0"><i class="bi bi-file-earmark-plus me-2 text-info"></i>New Standalone AR Invoice</h4>
          <small class="text-muted">Direct billing — no Sales Order or Delivery required</small>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Invoice Date <span class="text-danger">*</span></label>
              <input type="date" id="sarDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="sarDueDate" class="form-control">
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Terms</label>
              <select id="sarTerms" class="form-select">
                <option value="COD">COD</option>
                <option value="7 days">7 days</option>
                <option value="15 days">15 days</option>
                <option value="30 days">30 days</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-light border-0 py-3"><h6 class="fw-bold mb-0"><i class="bi bi-person me-2"></i>Customer</h6></div>
        <div class="card-body">
          <div class="mb-3">
            <label class="form-label small fw-bold">Existing Account <span class="text-muted fw-normal">(optional)</span></label>
            <select id="sarCustSelect" class="form-select" onchange="sarFillCustomer(this)">
              <option value="">— Walk-in / New Customer —</option>
              ${custOptions}
            </select>
          </div>
          <div class="row g-3">
            <div class="col-md-6">
              <label class="form-label small fw-bold">Customer Name <span class="text-danger">*</span></label>
              <input type="text" id="sarCustName" class="form-control" placeholder="Full name" required>
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Contact</label>
              <input type="text" id="sarCustContact" class="form-control" placeholder="09XX XXX XXXX">
            </div>
            <div class="col-12">
              <label class="form-label small fw-bold">Address</label>
              <input type="text" id="sarCustAddress" class="form-control" placeholder="Delivery address">
            </div>
          </div>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-info bg-opacity-10 border-0 py-3 d-flex justify-content-between align-items-center">
          <h6 class="fw-bold mb-0"><i class="bi bi-box me-2 text-warning"></i>Invoice Items</h6>
          <button type="button" class="btn btn-sm btn-outline-info" onclick="sarAddRow()"><i class="bi bi-plus-lg me-1"></i>Add Item</button>
        </div>
        <div class="card-body p-0">
          <div class="table-responsive">
            <table class="table mb-0">
              <thead class="table-light">
                <tr>
                  <th class="ps-3" style="width:16%;">Category</th>
                  <th style="width:26%;">Item</th>
                  <th style="width:11%;">Unit</th>
                  <th style="width:11%;">Qty</th>
                  <th style="width:16%;">Price / Unit (₱)</th>
                  <th style="width:15%;">Line Total</th>
                  <th style="width:5%;"></th>
                </tr>
              </thead>
              <tbody id="sarItemsBody"></tbody>
              <tfoot>
                <tr class="table-light">
                  <td colspan="5" class="text-end fw-bold pe-3">Invoice Total</td>
                  <td class="fw-bold text-primary fs-5" id="sarTotal">₱0.00</td>
                  <td></td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>

      <div class="card shadow-sm border-0 mb-4">
        <div class="card-body">
          <label class="form-label small fw-bold">Notes</label>
          <textarea id="sarNotes" class="form-control" rows="2" placeholder="Payment instructions, remarks…"></textarea>
        </div>
      </div>

      <div class="d-flex gap-2">
        <button class="btn btn-primary btn-lg fw-bold flex-grow-1" onclick="submitStandaloneARForm()">
          <i class="bi bi-file-earmark-text me-2"></i>Create AR Invoice
        </button>
        <button class="btn btn-outline-secondary btn-lg" onclick="navigate('#/sales/ar-invoices')">Cancel</button>
      </div>
    </div>
  `);

  sarAddRow();
}

window.sarFillCustomer = function(sel) {
  const opt = sel.options[sel.selectedIndex];
  document.getElementById('sarCustName').value    = opt.dataset.name    || '';
  document.getElementById('sarCustContact').value = opt.dataset.contact || '';
  document.getElementById('sarCustAddress').value = opt.dataset.address || '';
};

window.sarAddRow = function() {
  const tbody = document.getElementById('sarItemsBody');
  if (!tbody) return;

  const catOptions = _sarCategories.map(c =>
    `<option value="${c.itms_grp_cod}">${c.itms_grp_nam}</option>`
  ).join('');

  const tr = document.createElement('tr');
  tr.innerHTML = `
    <td class="ps-3">
      <select class="form-select form-select-sm sar-cat" onchange="sarOnCategoryChange(this)">
        <option value="">— All —</option>
        ${catOptions}
      </select>
    </td>
    <td>
      <div class="position-relative">
        <input type="text" class="form-control form-control-sm sar-item"
               placeholder="Type to search…" autocomplete="off"
               oninput="sarSuggestItem(this)" onfocus="sarSuggestItem(this)"
               onblur="sarHideSuggestions(this)" required>
        <div class="sar-suggestions list-group"
             style="position:fixed;z-index:9999;display:none;max-height:200px;overflow-y:auto;"></div>
      </div>
    </td>
    <td>
      <select class="form-select form-select-sm text-center sar-unit" onchange="sarOnUnitChange(this)"></select>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end sar-qty"
             placeholder="0" min="0.001" step="0.001" required oninput="sarUpdateRow(this)">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end sar-price"
             placeholder="0.00" min="0" step="0.01" required oninput="sarUpdateRow(this)">
    </td>
    <td class="fw-bold sar-line-total text-primary">₱0.00</td>
    <td>
      <button type="button" class="btn btn-sm btn-outline-danger py-0"
              onclick="this.closest('tr').remove(); sarUpdateTotal();">
        <i class="bi bi-x"></i>
      </button>
    </td>`;
  tbody.appendChild(tr);
};

window.sarOnCategoryChange = function(sel) {
  const row       = sel.closest('tr');
  const itemInput = row.querySelector('.sar-item');
  if (itemInput) {
    itemInput.value = '';
    const unitSel = row.querySelector('.sar-unit');
    if (unitSel) unitSel.innerHTML = '';
    sarSuggestItem(itemInput);
  }
};

window.sarSuggestItem = function(input) {
  const row  = input.closest('tr');
  const cat  = row.querySelector('.sar-cat')?.value || '';
  const term = input.value.toLowerCase();
  const pool = cat ? _sarItems.filter(i => String(i.itms_grp_cod) === cat) : _sarItems;
  const matches = term
    ? pool.filter(i =>
        (i.item_name || '').toLowerCase().includes(term) ||
        (i.item_code || '').toLowerCase().includes(term)
      ).slice(0, 15)
    : pool.slice(0, 15);

  const sugDiv = input.parentElement?.querySelector('.sar-suggestions');
  if (!sugDiv) return;
  if (matches.length === 0) { sugDiv.style.display = 'none'; return; }

  sugDiv.innerHTML = matches.map(i =>
    `<button type="button" class="list-group-item list-group-item-action py-1 px-2 small"
             data-item-name="${i.item_name}"
             onmousedown="sarSelectSuggestion(this)">
       <span class="fw-semibold">${i.item_name}</span>
       <small class="text-muted ms-2">${i.item_code}</small>
     </button>`
  ).join('');
  const rect = input.getBoundingClientRect();
  sugDiv.style.top   = rect.bottom + 'px';
  sugDiv.style.left  = rect.left   + 'px';
  sugDiv.style.width = rect.width  + 'px';
  sugDiv.style.display = 'block';
};

window.sarSelectSuggestion = function(btn) {
  const wrapper = btn.closest('.position-relative');
  const input   = wrapper?.querySelector('.sar-item');
  const sugDiv  = wrapper?.querySelector('.sar-suggestions');
  if (!input) return;
  input.value = btn.dataset.itemName;
  if (sugDiv) sugDiv.style.display = 'none';
  sarOnItemSelect(input);
};

window.sarHideSuggestions = function(input) {
  setTimeout(() => {
    const sugDiv = input.parentElement?.querySelector('.sar-suggestions');
    if (sugDiv) sugDiv.style.display = 'none';
  }, 200);
};

window.sarOnItemSelect = function(input) {
  const row  = input.closest('tr');
  const item = _sarItems.find(i => i.item_name === input.value);
  if (item) sarPopulateUnitDropdown(row, item, '');
  sarUpdateRow(input);
  sarApplyPriceToRow(row);
};

function sarPopulateUnitDropdown(row, item, preUnit) {
  const sel = row.querySelector('.sar-unit');
  if (!sel) return;
  const units = item?.ugp_entry
    ? getUoMGroupUnits(item.ugp_entry, _sarUoMGroups)
    : [];
  const defaultCode = preUnit || item?.invntry_uom || '';
  if (units.length === 0 && defaultCode) {
    units.push({ code: defaultCode, name: defaultCode });
  }
  sel.innerHTML = units.map(u =>
    `<option value="${u.code}"${u.code === defaultCode ? ' selected' : ''}>${u.name || u.code}</option>`
  ).join('');
}

window.sarOnUnitChange = function(sel) {
  const row      = sel.closest('tr');
  const itemName = row.querySelector('.sar-item')?.value;
  const item     = _sarItems.find(i => i.item_name === itemName);
  const price    = resolveUoMPrice(item, sel.value, _sarUoMGroups);
  if (price !== null) {
    const priceEl = row.querySelector('.sar-price');
    if (priceEl) { priceEl.value = price.toFixed(2); sarUpdateRow(priceEl); }
  }
};

function sarApplyPriceToRow(row) {
  const itemName = row.querySelector('.sar-item')?.value;
  const priceEl  = row.querySelector('.sar-price');
  if (!itemName || !priceEl) return;
  const item = _sarItems.find(i => i.item_name === itemName);
  if (item?.avg_price) {
    priceEl.value = Number(item.avg_price).toFixed(2);
  }
  sarUpdateRow(priceEl);
}

window.sarUpdateRow = function(el) {
  const row   = el.closest('tr');
  const qty   = parseFloat(row.querySelector('.sar-qty')?.value)   || 0;
  const price = parseFloat(row.querySelector('.sar-price')?.value) || 0;
  row.querySelector('.sar-line-total').textContent = formatCurrency(qty * price);
  sarUpdateTotal();
};

window.sarUpdateTotal = function() {
  let total = 0;
  document.querySelectorAll('#sarItemsBody tr').forEach(row => {
    total += (parseFloat(row.querySelector('.sar-qty')?.value)   || 0) *
             (parseFloat(row.querySelector('.sar-price')?.value) || 0);
  });
  const el = document.getElementById('sarTotal');
  if (el) el.textContent = formatCurrency(total);
};

async function submitStandaloneARForm() {
  const custIdRaw = document.getElementById('sarCustSelect').value;
  const custName  = document.getElementById('sarCustName').value.trim();
  if (!custName) { toast('Customer name is required.', 'warning'); return; }

  const items = [];
  let valid = true;
  document.querySelectorAll('#sarItemsBody tr').forEach(row => {
    const sku   = row.querySelector('.sar-item')?.value;
    const unit  = row.querySelector('.sar-unit')?.value;
    const qty   = parseFloat(row.querySelector('.sar-qty')?.value)   || 0;
    const price = parseFloat(row.querySelector('.sar-price')?.value) || 0;
    if (!sku) { valid = false; return; }
    items.push({ sku, unit, quantity: qty, price_per_unit: price });
  });
  if (!valid) { toast('Please select an item for all rows.', 'warning'); return; }
  if (items.length === 0) { toast('Add at least one item.', 'warning'); return; }

  const dueDateVal = document.getElementById('sarDueDate').value;
  const payload = {
    delivery_order_ids: [],
    customer_id:       custIdRaw ? parseInt(custIdRaw, 10) : null,
    customer_name:    custName,
    customer_addr:    document.getElementById('sarCustAddress').value.trim(),
    customer_contact: document.getElementById('sarCustContact').value.trim(),
    date:             document.getElementById('sarDate').value,
    terms:            document.getElementById('sarTerms').value,
    notes:            document.getElementById('sarNotes').value.trim(),
    due_date:         dueDateVal || null,
    items,
  };

  const btn = document.querySelector('[onclick="submitStandaloneARForm()"]');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Saving…';

  const result = await api.CreateARInvoice(payload);
  if (result) {
    toast('AR invoice created.', 'success');
    navigate('#/sales/ar-invoices/' + result.id);
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-file-earmark-text me-2"></i>Create AR Invoice';
  }
}

function printARInvoice() {
  const inv = _arDetail;
  if (!inv) return;
  const f = window._pfmt;
  const custName = inv.customer_name_snapshot || (inv.customer && inv.customer.name) || '—';
  const so     = inv.sales_order || null;
  const doRecs = inv.delivery_orders || [];
  const items  = inv.items || [];
  const colLines = inv.collection_lines || [];
  const collected = inv.amount_collected || 0;
  const balance = Math.max(0, (inv.total_amount || 0) - collected);
  const itemRows = items.map(i => `
    <tr>
      <td>${i.sku || '—'}</td>
      <td>${i.unit || '—'}</td>
      <td class="text-end">${f.num(i.quantity)}</td>
      <td class="text-end">${f.currency(i.price_per_unit)}</td>
      <td class="text-end">${f.currency(i.quantity * i.price_per_unit)}</td>
    </tr>`).join('');
  const colRows = colLines.length === 0 ? '' : `
    <div class="section-title">Collections Applied</div>
    <table>
      <thead><tr><th>Collection #</th><th>Date</th><th>Method</th><th class="text-end">Amount Applied</th></tr></thead>
      <tbody>${colLines.map(cl => {
        const col = cl.collection || {};
        return `<tr>
          <td>${col.collection_number || '—'}</td>
          <td>${f.date(col.date)}</td>
          <td>${col.payment_method || '—'}</td>
          <td class="text-end">${f.currency(cl.amount_applied)}</td>
        </tr>`;
      }).join('')}</tbody>
    </table>`;
  const html = `
    <h2>AR INVOICE</h2>
    <div class="sub">${inv.invoice_number || '—'} &nbsp;·&nbsp; <span class="badge">${inv.status || '—'}</span> &nbsp;·&nbsp; ${f.date(inv.date)}</div>
    <div class="info-grid">
      <div><span>Customer</span><br>${custName}</div>
      <div><span>Invoice Date</span><br>${f.date(inv.date)}</div>
      ${inv.due_date ? `<div><span>Due Date</span><br>${f.date(inv.due_date)}</div>` : ''}
      <div><span>Terms</span><br>${inv.terms || '—'}</div>
      ${so ? `<div><span>Sales Order</span><br>${so.sales_order_number}</div>` : ''}
      ${doRecs.length > 0 ? `<div><span>Delivery Order${doRecs.length > 1 ? 's' : ''}</span><br>${doRecs.map(d => d.delivery_number).join(', ')}</div>` : ''}
      ${inv.notes ? `<div style="grid-column:1/-1"><span>Notes</span><br>${inv.notes}</div>` : ''}
    </div>
    <table>
      <thead><tr><th>Item/SKU</th><th>Unit</th><th class="text-end">Qty</th><th class="text-end">Unit Price</th><th class="text-end">Total</th></tr></thead>
      <tbody>${itemRows}</tbody>
    </table>
    <div class="totals">
      Total Amount: <strong>${f.currency(inv.total_amount)}</strong><br>
      Amount Collected: ${f.currency(collected)}<br>
      Balance: ${f.currency(balance)}
    </div>
    ${colRows}`;
  _printDoc('AR Invoice ' + (inv.invoice_number || ''), html);
}

function _resetSalesAR() {
  _arList = []; _arDetail = null;
  _arInvDOList = []; _arInvCustomers = [];
  _sarItems = []; _sarCategories = []; _sarUoMGroups = [];
}
