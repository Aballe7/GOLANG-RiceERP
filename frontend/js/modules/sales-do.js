// sales-do.js

// ── Delivery Orders ───────────────────────────────────────────────────────────

let _doList   = [];
let _doDetail = null;

async function loadDeliveryOrdersList(tabBar) {
  sessionStorage.removeItem('docReturnRoute');
  _doList = await api.ListDeliveryOrders(0) || [];

  const statusOpts = [
    { v: 'Draft',     l: 'Draft' },
    { v: 'Delivered', l: 'Delivered' },
    { v: 'Cancelled', l: 'Cancelled' },
    { v: '',          l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    ${_filterBar(statusOpts, 'Draft', 'doFilter()', {
      doc:  _doList.map(d => d.delivery_number),
      cust: _doList.map(d => (d.sales_order && d.sales_order.customer_name_snapshot) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>DO #</th><th>Delivery Date</th><th>Customer</th><th>SO #</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="do-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  doFilter();
}

window.doFilter = function() {
  const doc    = _sfVal('sf-doc').toLowerCase();
  const cust   = _sfVal('sf-cust').toLowerCase();
  const dateFrom = _sfVal('sf-date-from');
  const dateTo   = _sfVal('sf-date-to');
  const status   = _sfVal('sf-status');

  const filtered = _doList.filter(d => {
    if (doc  && !(d.delivery_number || '').toLowerCase().includes(doc)) return false;
    const custName = ((d.sales_order && d.sales_order.customer_name_snapshot) || '').toLowerCase();
    if (cust && !custName.includes(cust)) return false;
    const rowDate = (d.date || '').slice(0, 10);
    if (dateFrom && rowDate < dateFrom) return false;
    if (dateTo   && rowDate > dateTo)   return false;
    if (status) {
      const allowed = status.split(',');
      if (!allowed.includes(d.status)) return false;
    }
    return true;
  });

  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No delivery orders match the filter.</td></tr>`
    : filtered.map(d => `
        <tr>
          <td class="fw-semibold">${d.delivery_number || '—'}</td>
          <td>${formatDate(d.date)}</td>
          <td>${(d.sales_order && d.sales_order.customer_name_snapshot) || '—'}</td>
          <td>${(d.sales_order && d.sales_order.sales_order_number) || '—'}</td>
          <td>${salesStatusBadge(d.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/delivery-orders/${d.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>`).join('');
  const el = document.getElementById('do-tbody');
  if (el) el.innerHTML = rows;
};

// ── Delivery Order Detail ─────────────────────────────────────────────────────

async function loadDODetail(id) {
  showLoading();
  const doRec = await api.GetDeliveryOrder(id);
  if (!doRec) { showView(`<div class="alert alert-warning m-4">Delivery order not found.</div>`); return; }
  _doDetail = doRec;

  const so      = doRec.sales_order || {};
  const items   = doRec.items || [];
  const arInvs  = doRec.ar_invoices || [];
  const total   = items.reduce((s, i) => s + (i.quantity_delivered * i.price_per_unit), 0);
  const uninvoiced = total - (doRec.amount_invoiced || 0);

  const statusColor = { Delivered: 'success', Cancelled: 'secondary', Draft: 'warning' };
  const badge = `<span class="badge bg-${statusColor[doRec.status] || 'secondary'} fs-6">${doRec.status}</span>`;

  const itemRows = items.map(i => `
    <tr>
      <td class="ps-4 fw-semibold">${i.sku}</td>
      <td class="text-center text-muted">${i.unit}</td>
      <td class="text-end text-muted">${formatNumber(i.quantity_ordered)}</td>
      <td class="text-end fw-bold">${formatNumber(i.quantity_delivered)}</td>
      <td class="text-end">${formatCurrency(i.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity_delivered * i.price_per_unit)}</td>
    </tr>`).join('') || `<tr><td colspan="6" class="text-center text-muted py-3">No items.</td></tr>`;

  const arRows = arInvs.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between">
        <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices</span>
        ${uninvoiced > 0.005 ? `<button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromDO(${doRec.id})">+ Create AR Invoice</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light"><tr><th class="ps-4">Invoice #</th><th>Date</th><th>Status</th><th class="text-end pe-4">Total</th></tr></thead>
          <tbody>
            ${arInvs.map(inv => `<tr>
              <td class="ps-4 fw-semibold"><a href="#" onclick="navigate('#/sales/ar-invoices/${inv.id}');return false;" class="text-decoration-none">${inv.invoice_number}</a></td>
              <td class="text-muted small">${formatDate(inv.date)}</td>
              <td>${salesStatusBadge(inv.status)}</td>
              <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(inv.total_amount || 0)}</td>
            </tr>`).join('')}
          </tbody>
        </table>
      </div>
    </div>`;

  const confirmBtn = doRec.status === 'Draft' ? `
    <button class="btn btn-sm btn-success" onclick="confirmDO(${doRec.id})">
      <i class="bi bi-check-lg me-1"></i>Confirm Delivery
    </button>` : '';
  const createARBtn = doRec.status === 'Delivered' && uninvoiced > 0.005 ? `
    <button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromDO(${doRec.id})">
      <i class="bi bi-file-earmark-plus me-1"></i>Create AR Invoice
    </button>` : '';
  // Cancel: allowed for Draft (no stock to reverse) or Delivered with no AR invoices
  const canCancel = doRec.status === 'Draft' ||
    (doRec.status === 'Delivered' && arInvs.length === 0);
  const cancelBtn = canCancel ? `
    <button class="btn btn-sm btn-outline-danger" onclick="cancelDO(${doRec.id})">
      <i class="bi bi-x-circle me-1"></i>Cancel DO
    </button>` : '';

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate(sessionStorage.getItem('docReturnRoute')||'#/sales/delivery-orders');sessionStorage.removeItem('docReturnRoute')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-truck me-2" style="color:#ec4899"></i>${doRec.delivery_number}</h4>
          <small class="text-muted">Delivery Order · ${formatDate(doRec.date)}</small>
        </div>
        ${badge}
        <button class="btn btn-sm btn-outline-secondary" onclick="printDO()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('sales',${doRec.sales_order_id},'DO',${doRec.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
        ${confirmBtn}
        ${createARBtn}
        ${cancelBtn}
      </div>

      ${doRec.status === 'Delivered' ? `
        <div class="card border-0 shadow-sm mb-3">
          <div class="card-header bg-white border-bottom py-2"><span class="small fw-bold text-muted">INVOICE STATUS</span></div>
          <div class="card-body">
            <div class="row g-2 text-center">
              <div class="col"><div class="small text-muted mb-1">DO Total</div><div class="fw-bold">${formatCurrency(total)}</div></div>
              <div class="col-auto d-flex align-items-center text-muted">→</div>
              <div class="col"><div class="small text-muted mb-1">AR Invoiced</div><div class="fw-bold text-primary">${formatCurrency(doRec.amount_invoiced || 0)}</div></div>
              <div class="col-auto d-flex align-items-center text-muted">→</div>
              <div class="col"><div class="small text-muted mb-1">Uninvoiced</div>
                <div class="fw-bold ${uninvoiced > 0.005 ? 'text-danger' : 'text-success'}">${uninvoiced > 0.005 ? formatCurrency(uninvoiced) : '<i class="bi bi-check-circle-fill me-1"></i>Fully Invoiced'}</div>
              </div>
            </div>
          </div>
        </div>` : ''}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-info-circle me-2"></i>Delivery Details</span></div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted">Sales Order</div>
              <a href="#" onclick="navigate('#/sales/orders/${doRec.sales_order_id}');return false;" class="text-decoration-none fw-semibold">${so.sales_order_number || '—'}</a>
            </div>
            <div class="col-md-4"><div class="small text-muted">Customer</div><div class="fw-semibold">${so.customer_name_snapshot || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted">Delivered By</div><div>${doRec.delivered_by || '—'}</div></div>
            ${doRec.notes ? `<div class="col-12"><div class="small text-muted">Notes</div><div>${doRec.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-egg me-2 text-warning"></i>Items Delivered</span></div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Egg Size</th><th class="text-center">Unit</th><th class="text-end">Ordered</th><th class="text-end">Delivered</th><th class="text-end">Price/Unit</th><th class="text-end pe-4">Line Total</th></tr></thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="5" class="text-end pe-3 ps-4">Total</td><td class="text-end pe-4 text-primary fs-5">${formatCurrency(total)}</td></tr></tfoot>
          </table>
        </div>
      </div>

      ${arRows}
    </div>
  `);
}

async function confirmDO(id) {
  if (!confirm('Confirm delivery and update inventory?')) return;
  const r = await api.ConfirmDeliveryOrder(id);
  if (r) await loadDODetail(id);
}

async function cancelDO(id) {
  if (!confirm('Cancel this delivery order? Stock movements will be reversed.')) return;
  const r = await api.CancelDeliveryOrder(id);
  if (r) await loadDODetail(id);
}

function newARInvoiceFromDO(doId) {
  window._newARInvoicePrefillDOId = doId;
  navigate('#/sales/ar-invoices/new');
}

// ── New Delivery Order Form ───────────────────────────────────────────────────

let _newDOSalesOrders = [];

async function loadNewDOForm() {
  showLoading();
  const soId = window._newDOPrefillSOId || null;
  window._newDOPrefillSOId = null;

  let soData = null;
  if (soId) {
    soData = await api.GetSalesOrder(soId);
  }
  if (!soData) {
    // Show SO selector first
    const soList = await api.ListSalesOrders();
    _newDOSalesOrders = (soList || []).filter(o => o.payment_status !== 'Void');
    const opts = _newDOSalesOrders.map(o =>
      `<option value="${o.id}">${o.sales_order_number} — ${o.customer_name_snapshot || 'Walk-in'} (${formatCurrency(o.grand_total)})</option>`
    ).join('');
    showView(`
      <div class="container p-4" style="max-width:600px">
        <div class="d-flex align-items-center gap-2 mb-4">
          <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/delivery-orders')"><i class="bi bi-arrow-left"></i></button>
          <h4 class="fw-bold mb-0">New Delivery Order</h4>
        </div>
        <div class="card border-0 shadow-sm">
          <div class="card-body">
            <label class="form-label fw-bold">Select Sales Order <span class="text-danger">*</span></label>
            <select class="form-select mb-3" id="doSOPicker">
              <option value="">— Select Sales Order —</option>${opts}
            </select>
            <button class="btn btn-primary" onclick="doPickSO()">Continue</button>
          </div>
        </div>
      </div>`);
    return;
  }
  const existingDraft = (soData.deliveries || []).find(d => d.status === 'Draft');
  if (existingDraft) {
    toast(`Draft delivery ${existingDraft.delivery_number} already exists — confirm or cancel it first.`, 'warning', 8000);
    navigate(`#/sales/delivery-orders/${existingDraft.id}`);
    return;
  }
  await renderNewDOForm(soData);
}

async function doPickSO() {
  const id = parseInt(document.getElementById('doSOPicker').value);
  if (!id) { toast('Please select a sales order.', 'warning'); return; }
  const so = await api.GetSalesOrder(id);
  if (!so) return;
  const existingDraft = (so.deliveries || []).find(d => d.status === 'Draft');
  if (existingDraft) {
    toast(`Draft delivery ${existingDraft.delivery_number} already exists — confirm or cancel it first.`, 'warning', 8000);
    navigate(`#/sales/delivery-orders/${existingDraft.id}`);
    return;
  }
  await renderNewDOForm(so);
}

async function renderNewDOForm(so) {
  const today = new Date().toISOString().slice(0, 10);
  const custName = so.customer_name_snapshot || (so.customer && so.customer.name) || 'Walk-in';
  const items = so.items || [];

  const itemRows = items.map((item, idx) => {
    const openQty = (item.open_qty != null && item.open_qty > 0) ? item.open_qty : item.quantity;
    const lineTotal = openQty * item.price_per_unit;
    return `
    <tr class="do-item-row">
      <td class="ps-4">
        <input type="checkbox" class="form-check-input do-row-check" checked onchange="doToggleRow(this)">
        <input type="hidden" class="do-sku" value="${item.sku}">
        <input type="hidden" class="do-unit" value="${item.unit}">
        <input type="hidden" class="do-price" value="${item.price_per_unit}">
        <input type="hidden" class="do-so-item-id" value="${item.id || ''}">
      </td>
      <td class="fw-semibold">${item.sku}</td>
      <td class="text-center text-muted">${item.unit}</td>
      <td class="text-center text-muted">${formatNumber(openQty)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-center do-qty"
          value="${openQty}" min="0" max="${openQty}" step="0.5"
          oninput="doUpdateTotal()">
      </td>
      <td class="text-center text-muted">${formatCurrency(item.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold do-line-total text-primary">${formatCurrency(lineTotal)}</td>
    </tr>`;
  }).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/orders/${so.id}')"><i class="bi bi-arrow-left"></i> Back to SO</button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-truck me-2" style="color:#ec4899"></i>New Delivery Order</h4>
          <div class="text-muted small">Copied from ${so.sales_order_number} — ${custName}</div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Delivery Date <span class="text-danger">*</span></label>
              <input type="date" id="doDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Delivered By</label>
              <input type="text" id="doDeliveredBy" class="form-control" placeholder="Driver / staff name">
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="doNotes" class="form-control" placeholder="e.g. Partial delivery">
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-egg me-2 text-warning"></i>Items to Deliver</span>
          <span class="text-muted small">Adjust <strong>Qty to Deliver</strong> for partial delivery</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:5%">
                  <input type="checkbox" class="form-check-input" id="doCheckAll" checked onchange="doToggleAll(this)">
                </th>
                <th>Egg Size</th><th class="text-center">Unit</th><th class="text-center">Ordered</th>
                <th class="text-center">Qty to Deliver</th><th class="text-center">Price/Unit</th><th class="text-end pe-4">Line Total</th>
              </tr>
            </thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr><td colspan="6" class="text-end pe-3 fw-bold">Delivery Total</td>
                  <td class="text-end pe-4 fw-bold text-primary fs-5" id="doGrandTotal">${formatCurrency(so.grand_total)}</td></tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 justify-content-end">
        <button class="btn btn-outline-secondary px-4" onclick="navigate('#/sales/orders/${so.id}')">Cancel</button>
        <button class="btn btn-primary px-4" onclick="submitNewDO(${so.id})">
          <i class="bi bi-truck me-1"></i>Create Delivery Order
        </button>
      </div>
    </div>
  `);
}

function doUpdateTotal() {
  let grand = 0;
  document.querySelectorAll('.do-item-row').forEach(row => {
    if (!row.querySelector('.do-row-check').checked) return;
    const qty   = parseFloat(row.querySelector('.do-qty').value) || 0;
    const price = parseFloat(row.querySelector('.do-price').value) || 0;
    const line  = qty * price;
    row.querySelector('.do-line-total').textContent = formatCurrency(line);
    grand += line;
  });
  const el = document.getElementById('doGrandTotal');
  if (el) el.textContent = formatCurrency(grand);
}

function doToggleRow(cb) {
  const row = cb.closest('tr');
  const qty = row.querySelector('.do-qty');
  row.style.opacity = cb.checked ? '1' : '0.4';
  if (!cb.checked) { qty.value = 0; }
  doUpdateTotal();
}

function doToggleAll(master) {
  document.querySelectorAll('.do-row-check').forEach(cb => { cb.checked = master.checked; doToggleRow(cb); });
}

async function submitNewDO(soId) {
  const items = [];
  document.querySelectorAll('.do-item-row').forEach(row => {
    if (!row.querySelector('.do-row-check').checked) return;
    const qty = parseFloat(row.querySelector('.do-qty').value) || 0;
    if (qty <= 0) return;
    const soItemIdRaw = row.querySelector('.do-so-item-id')?.value;
    items.push({
      sku:                  row.querySelector('.do-sku').value,
      unit:                 row.querySelector('.do-unit').value,
      quantity_ordered:     qty,
      quantity_delivered:   qty,
      price_per_unit:       parseFloat(row.querySelector('.do-price').value) || 0,
      sales_order_item_id:  soItemIdRaw ? parseInt(soItemIdRaw, 10) : null,
    });
  });
  if (items.length === 0) { toast('No items selected.', 'warning'); return; }

  const result = await api.CreateDeliveryOrder({
    sales_order_id: soId,
    date:           document.getElementById('doDate').value,
    delivered_by:   document.getElementById('doDeliveredBy').value,
    notes:          document.getElementById('doNotes').value,
    items,
  });
  if (result) navigate('#/sales/delivery-orders');
}

function printDO() {
  const doRec = _doDetail;
  if (!doRec) return;
  const f = window._pfmt;
  const so = doRec.sales_order || {};
  const items = doRec.items || [];
  const total = items.reduce((s, i) => s + (i.quantity_delivered * i.price_per_unit), 0);
  const itemRows = items.map(i => `
    <tr>
      <td>${i.sku || '—'}</td>
      <td>${i.unit || '—'}</td>
      <td class="text-end">${f.num(i.quantity_ordered)}</td>
      <td class="text-end">${f.num(i.quantity_delivered)}</td>
      <td class="text-end">${f.currency(i.price_per_unit)}</td>
      <td class="text-end">${f.currency(i.quantity_delivered * i.price_per_unit)}</td>
    </tr>`).join('');
  const html = `
    <h2>DELIVERY ORDER</h2>
    <div class="sub">${doRec.delivery_number || '—'} &nbsp;·&nbsp; <span class="badge">${doRec.status || '—'}</span> &nbsp;·&nbsp; ${f.date(doRec.date)}</div>
    <div class="info-grid">
      <div><span>Customer</span><br>${so.customer_name_snapshot || '—'}</div>
      <div><span>Sales Order</span><br>${so.sales_order_number || '—'}</div>
      <div><span>Delivered By</span><br>${doRec.delivered_by || '—'}</div>
      <div><span>Date</span><br>${f.date(doRec.date)}</div>
      ${doRec.notes ? `<div style="grid-column:1/-1"><span>Notes</span><br>${doRec.notes}</div>` : ''}
    </div>
    <table>
      <thead><tr><th>Item/SKU</th><th>Unit</th><th class="text-end">Qty Ordered</th><th class="text-end">Qty Delivered</th><th class="text-end">Unit Price</th><th class="text-end">Total</th></tr></thead>
      <tbody>${itemRows}</tbody>
    </table>
    <div class="totals">Total Delivered Value: <strong>${f.currency(total)}</strong></div>`;
  _printDoc('DO ' + (doRec.delivery_number || ''), html);
}

function _resetSalesDO() {
  _doList = []; _doDetail = null; _newDOSalesOrders = [];
  window._newARInvoicePrefillDOId = null;
}
