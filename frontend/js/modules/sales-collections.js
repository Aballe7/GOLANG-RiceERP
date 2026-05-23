// sales-collections.js

// ── Collections ───────────────────────────────────────────────────────────────

let _crList   = [];
let _colDetail = null;

async function loadCollectionsList(tabBar) {
  _crList = await api.ListCollections() || [];

  const statusOpts = [
    { v: '', l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="navigate('#/sales/collections/new')">
        <i class="bi bi-plus-lg me-1"></i>Record Collection
      </button>
    </div>
    ${_filterBar(statusOpts, '', 'crFilter()', {
      doc:  _crList.map(c => c.collection_number),
      cust: _crList.map(c => c.customer_name_snapshot || (c.customer && c.customer.name) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Collection #</th><th>Date</th><th>Customer</th><th>Method</th><th>Amount</th><th></th></tr>
          </thead>
          <tbody id="cr-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  crFilter();
}

window.crFilter = function() {
  const filtered = _applyFilters(_crList, {
    docField:    'collection_number',
    amountField: 'total_amount',
  });

  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No collections match the filter.</td></tr>`
    : filtered.map(c => `
        <tr>
          <td class="fw-semibold">${c.collection_number || '—'}</td>
          <td>${formatDate(c.date)}</td>
          <td>${c.customer_name_snapshot || (c.customer && c.customer.name) || '—'}</td>
          <td>${c.payment_method || '—'}</td>
          <td>${formatCurrency(c.total_amount ?? 0)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/collections/${c.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>`).join('');
  const el = document.getElementById('cr-tbody');
  if (el) el.innerHTML = rows;
};

// ── Collection Detail ─────────────────────────────────────────────────────────

async function loadCollectionDetail(id) {
  showLoading();
  const col = await api.GetCollection(id);
  if (!col) { showView(`<div class="alert alert-warning m-4">Collection not found.</div>`); return; }
  _colDetail = col;

  const lines = col.lines || [];
  const relSoId = lines[0]?.ar_invoice?.sales_order_id || 0;
  const custName = col.customer_name_snapshot || (col.customer && col.customer.name) || '—';

  const lineRows = lines.map(l => {
    const inv = l.ar_invoice || {};
    const bal = Math.max(0, (inv.total_amount || 0) - (inv.amount_collected || 0));
    return `<tr>
      <td class="ps-4 fw-semibold">
        <a href="#" onclick="navigate('#/sales/ar-invoices/${l.ar_invoice_id}');return false;" class="text-decoration-none">
          ${inv.invoice_number || '—'}
        </a>
      </td>
      <td class="text-muted small">${formatDate(inv.date)}</td>
      <td>${salesStatusBadge(inv.status)}</td>
      <td class="text-end pe-4 fw-bold text-success">${formatCurrency(l.amount_applied)}</td>
    </tr>`;
  }).join('') || `<tr><td colspan="4" class="text-center text-muted py-3">No invoice lines.</td></tr>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:800px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/collections')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-cash-coin me-2 text-success"></i>${col.collection_number}</h4>
          <small class="text-muted">Collection · ${formatDate(col.date)}</small>
        </div>
        <span class="badge bg-success fs-6">Posted</span>
        <button class="btn btn-sm btn-outline-secondary" onclick="printCollection()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('sales',${relSoId},'COL',${col.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted fw-bold">CUSTOMER</div><div class="fw-semibold">${custName}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">DATE</div><div>${formatDate(col.date)}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">PAYMENT METHOD</div><div>${col.payment_method || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">REFERENCE #</div><div>${col.reference_number || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">TOTAL AMOUNT</div><div class="fw-bold fs-5 text-success">${formatCurrency(col.total_amount)}</div></div>
            ${col.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${col.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices Settled</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Invoice #</th><th>Date</th><th>Status</th><th class="text-end pe-4">Amount Applied</th></tr></thead>
            <tbody>${lineRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="3" class="text-end pe-3 ps-4">Total Collection</td><td class="text-end pe-4 text-success fs-5">${formatCurrency(col.total_amount)}</td></tr></tfoot>
          </table>
        </div>
      </div>
    </div>
  `);
}

function printCollection() {
  const col = _colDetail;
  if (!col) return;
  const custName = col.customer_name_snapshot || (col.customer && col.customer.name) || '—';
  const lines = col.lines || [];
  const lineRows = lines.map(l => {
    const inv = l.ar_invoice || {};
    return `<tr>
      <td>${inv.invoice_number || '—'}</td>
      <td>${_pfmt.date(inv.date)}</td>
      <td>${inv.status || '—'}</td>
      <td class="text-end">${_pfmt.currency(l.amount_applied)}</td>
    </tr>`;
  }).join('') || `<tr><td colspan="4" class="text-center text-muted">No lines.</td></tr>`;

  const html = `
    <h2>Collection Receipt</h2>
    <div class="sub">${col.collection_number} &nbsp;·&nbsp; ${_pfmt.date(col.date)}</div>
    <div class="info-grid">
      <div><span>Customer</span><br>${custName}</div>
      <div><span>Payment Method</span><br>${col.payment_method || '—'}</div>
      <div><span>Reference #</span><br>${col.reference_number || '—'}</div>
      <div><span>Total Amount</span><br><strong>${_pfmt.currency(col.total_amount)}</strong></div>
      ${col.notes ? `<div class="col-span-2"><span>Notes</span><br>${col.notes}</div>` : ''}
    </div>
    <div class="section-title">AR Invoices Settled</div>
    <table>
      <thead><tr><th>Invoice #</th><th>Date</th><th>Status</th><th class="text-end">Amount Applied</th></tr></thead>
      <tbody>${lineRows}</tbody>
    </table>
    <div class="totals"><strong>Total Collection: ${_pfmt.currency(col.total_amount)}</strong></div>`;

  _printDoc('Collection ' + col.collection_number, html);
}

// ── New Collection Form ───────────────────────────────────────────────────────

let _colAllInvoices = [];

async function loadNewCollectionForm() {
  showLoading();
  const [customers, allInvoices] = await Promise.all([
    api.ListCustomers(true),
    api.ListARInvoices(null, ''),
  ]);
  const custList = customers || [];
  _colAllInvoices = (allInvoices || []).filter(i => i.status === 'Open' || i.status === 'Partial');

  const prefillInvId = window._newCollectionPrefillInvId || null;
  window._newCollectionPrefillInvId = null;

  const today = new Date().toISOString().slice(0, 10);
  const custOpts = custList.map(c => `<option value="${c.id}" data-name="${c.name}">${c.name}</option>`).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/collections')"><i class="bi bi-arrow-left"></i> Back</button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-cash-coin me-2 text-success"></i>Record Collection</h4>
          <div class="text-muted small">One receipt — applies to one or more AR Invoices</div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-person me-2 text-success"></i>Customer</span></div>
        <div class="card-body">
          <div class="row g-3 align-items-end">
            <div class="col-md-6">
              <label class="form-label small fw-bold">Select Customer <span class="text-danger">*</span></label>
              <select id="colCustSel" class="form-select" onchange="colOnCustChange()">
                <option value="">— Select a customer —</option>${custOpts}
              </select>
            </div>
            <div class="col-md-6" id="colCustSummary" style="display:none">
              <div class="p-3 rounded d-flex gap-4" style="background:#f0fdf4;border:1px solid #bbf7d0">
                <div><div class="small text-muted">Open Invoices</div><div class="fw-bold fs-5 text-success" id="colSummCount">—</div></div>
                <div><div class="small text-muted">Total Outstanding</div><div class="fw-bold fs-5 text-danger" id="colSummTotal">—</div></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3" id="colInvCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center flex-wrap gap-2">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices to Collect</span>
          <div class="d-flex gap-2">
            <button type="button" class="btn btn-sm btn-outline-success" onclick="colSelectAll(true)"><i class="bi bi-check-all me-1"></i>Select All</button>
            <button type="button" class="btn btn-sm btn-outline-secondary" onclick="colSelectAll(false)">Deselect All</button>
            <button type="button" class="btn btn-sm btn-outline-warning" onclick="colFillBalances()"><i class="bi bi-arrow-down-circle me-1"></i>Fill Balances</button>
          </div>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:40px"><input type="checkbox" class="form-check-input" id="colCheckAll" checked onchange="colToggleAll(this)"></th>
                <th>Invoice #</th><th>Date</th>
                <th class="text-end">Invoice Total</th><th class="text-end">Collected</th>
                <th class="text-end">Balance Due</th><th class="text-center" style="width:160px">Amount to Collect ₱</th>
              </tr>
            </thead>
            <tbody id="colInvBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4 text-muted">TOTAL TO COLLECT</td>
                <td class="text-end text-danger" id="colTotalBal">—</td>
                <td class="text-center fw-bold text-success fs-6" id="colTotalApply">—</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div id="colNoInvMsg" style="display:none" class="card border-0 shadow-sm mb-3">
        <div class="card-body text-center py-4 text-muted">
          <i class="bi bi-check-circle-fill text-success" style="font-size:2rem"></i>
          <div class="mt-2 fw-semibold">No open AR invoices for this customer</div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3" id="colDetailsCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-credit-card me-2"></i>Payment Details</span></div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3"><label class="form-label small fw-bold">Date <span class="text-danger">*</span></label><input type="date" id="colDate" class="form-control" value="${today}" required></div>
            <div class="col-md-3"><label class="form-label small fw-bold">Payment Method</label>
              <select id="colMethod" class="form-select"><option>Cash</option><option>Bank Transfer</option><option>GCash</option><option>Check</option></select>
            </div>
            <div class="col-md-3"><label class="form-label small fw-bold">Reference #</label><input type="text" id="colRef" class="form-control" placeholder="Optional"></div>
            <div class="col-md-3"><label class="form-label small fw-bold">Notes</label><input type="text" id="colNotes" class="form-control" placeholder="Optional"></div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm" id="colSubmitCard" style="display:none">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col"><div class="small text-muted mb-1">Selected</div><div class="fw-semibold" id="colSelCount">0 invoice(s)</div></div>
            <div class="col text-center">
              <div class="small text-muted mb-1">Total Collection Amount</div>
              <div class="fw-bold fs-4 text-success" id="colGrandTotal">₱0.00</div>
            </div>
            <div class="col text-end d-flex gap-2 justify-content-end">
              <button class="btn btn-outline-secondary px-4" onclick="navigate('#/sales/collections')">Cancel</button>
              <button class="btn btn-success px-5 text-white fw-bold" id="colSubmitBtn" disabled onclick="submitCollection()">
                <i class="bi bi-cash-coin me-2"></i>Post Collection
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `);

  if (prefillInvId) {
    const inv = _colAllInvoices.find(i => i.id === prefillInvId);
    if (inv) {
      const custSel = document.getElementById('colCustSel');
      const opt = Array.from(custSel.options).find(o => {
        const c = customers.find(c => c.id === parseInt(o.value));
        return c && c.name === inv.customer_name_snapshot;
      });
      if (opt) { custSel.value = opt.value; colOnCustChange(prefillInvId); }
    }
  }
}

function colOnCustChange(prefillInvId) {
  const sel = document.getElementById('colCustSel');
  const custName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  if (!sel.value) {
    ['colInvCard','colDetailsCard','colSubmitCard','colCustSummary','colNoInvMsg']
      .forEach(id => { const el = document.getElementById(id); if (el) el.style.display = 'none'; });
    return;
  }
  const openInvs = _colAllInvoices.filter(i =>
    i.customer_name_snapshot === custName || (i.customer && i.customer.name === custName)
  );
  colRenderInvoices(openInvs, prefillInvId);
}

function colRenderInvoices(invoices, prefillInvId) {
  const tbody = document.getElementById('colInvBody');
  if (!tbody) return;
  tbody.innerHTML = '';

  if (invoices.length === 0) {
    ['colInvCard','colDetailsCard','colSubmitCard'].forEach(id => { const el = document.getElementById(id); if (el) el.style.display = 'none'; });
    document.getElementById('colNoInvMsg').style.display = '';
    document.getElementById('colCustSummary').style.display = 'none';
    return;
  }

  let totalBal = 0;
  invoices.forEach(inv => {
    const balance = Math.max(0, (inv.total_amount || 0) - (inv.amount_collected || 0));
    totalBal += balance;
    const isPrefill = prefillInvId && inv.id !== prefillInvId;
    const row = document.createElement('tr');
    row.className = 'col-inv-row';
    row.innerHTML = `
      <td class="ps-4"><input type="checkbox" class="form-check-input col-inv-check" data-inv-id="${inv.id}"
        ${isPrefill ? '' : 'checked'} onchange="colToggleRow(this)"></td>
      <td class="fw-semibold">${inv.invoice_number}${inv.status === 'Partial' ? ' <span class="badge bg-warning text-dark ms-1" style="font-size:10px">Partial</span>' : ''}</td>
      <td class="text-muted small">${formatDate(inv.date)}</td>
      <td class="text-end text-muted">${formatCurrency(inv.total_amount || 0)}</td>
      <td class="text-end text-success small">${formatCurrency(inv.amount_collected || 0)}</td>
      <td class="text-end fw-bold text-danger">${formatCurrency(balance)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-end col-amt"
          value="${isPrefill ? '0.00' : balance.toFixed(2)}"
          min="0" max="${balance}" step="0.01" data-max="${balance}"
          ${isPrefill ? 'disabled' : ''}
          oninput="colUpdateTotals()">
      </td>`;
    tbody.appendChild(row);
  });

  document.getElementById('colTotalBal').textContent   = formatCurrency(totalBal);
  document.getElementById('colSummCount').textContent  = invoices.length;
  document.getElementById('colSummTotal').textContent  = formatCurrency(totalBal);
  document.getElementById('colCustSummary').style.display = '';
  document.getElementById('colInvCard').style.display     = '';
  document.getElementById('colNoInvMsg').style.display    = 'none';
  document.getElementById('colDetailsCard').style.display = '';
  document.getElementById('colSubmitCard').style.display  = '';
  document.getElementById('colCheckAll').checked = true;
  colUpdateTotals();
}

function colToggleAll(master) {
  document.querySelectorAll('.col-inv-check').forEach(cb => { cb.checked = master.checked; colToggleRow(cb); });
}
function colToggleRow(cb) {
  const row = cb.closest('tr'), amt = row.querySelector('.col-amt');
  if (!cb.checked) { amt.disabled = true; amt.value = '0'; row.style.opacity = '0.4'; }
  else             { amt.disabled = false; amt.value = amt.dataset.max; row.style.opacity = '1'; }
  colUpdateTotals();
}
function colSelectAll(state) {
  document.querySelectorAll('.col-inv-check').forEach(cb => { cb.checked = state; colToggleRow(cb); });
  const master = document.getElementById('colCheckAll');
  if (master) master.checked = state;
}
function colFillBalances() {
  document.querySelectorAll('.col-inv-row').forEach(row => {
    const cb = row.querySelector('.col-inv-check'), amt = row.querySelector('.col-amt');
    if (cb && cb.checked && amt) amt.value = amt.dataset.max;
  });
  colUpdateTotals();
}
function colUpdateTotals() {
  let total = 0, count = 0;
  document.querySelectorAll('.col-inv-row').forEach(row => {
    const cb = row.querySelector('.col-inv-check'), amt = row.querySelector('.col-amt');
    if (cb && cb.checked && amt) { const v = parseFloat(amt.value) || 0; total += v; if (v > 0) count++; }
  });
  const applyEl = document.getElementById('colTotalApply');
  const grandEl = document.getElementById('colGrandTotal');
  const countEl = document.getElementById('colSelCount');
  const btn     = document.getElementById('colSubmitBtn');
  if (applyEl) applyEl.textContent = formatCurrency(total);
  if (grandEl) grandEl.textContent = formatCurrency(total);
  if (countEl) countEl.textContent = count + ' invoice(s)';
  if (btn)     btn.disabled = total <= 0;
}

async function submitCollection() {
  const sel      = document.getElementById('colCustSel');
  const custId   = parseInt(sel.value) || null;
  const custName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  const lines    = [];

  document.querySelectorAll('.col-inv-row').forEach(row => {
    const cb  = row.querySelector('.col-inv-check');
    const amt = row.querySelector('.col-amt');
    const id  = parseInt(cb.dataset.invId);
    const val = parseFloat(amt.value) || 0;
    if (cb.checked && val > 0) lines.push({ ar_invoice_id: id, amount_applied: val });
  });

  if (lines.length === 0) { toast('No invoices selected.', 'warning'); return; }

  const btn = document.getElementById('colSubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Posting…';

  const result = await api.CreateCollection({
    date:           document.getElementById('colDate').value,
    customer_id:    custId,
    customer_name:  custName,
    payment_method: document.getElementById('colMethod').value,
    ref_num:        document.getElementById('colRef').value,
    notes:          document.getElementById('colNotes').value,
    lines,
  });

  if (result) {
    navigate('#/sales/collections');
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-cash-coin me-2"></i>Post Collection';
  }
}

function _resetSalesCollections() {
  _crList = []; _colDetail = null; _colAllInvoices = [];
}
