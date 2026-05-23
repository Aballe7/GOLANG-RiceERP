// purchasing-ap-payment.js

let _apPayList         = [];
let _apPayAllInvoices  = [];
let _apPayDetail       = null;
let _apPayMethodsList  = [];


async function loadApPaymentsList(tabBar) {
  const raw = await api.ListAPPayments();
  if (!raw) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load AP payments.</div>`));
    return;
  }
  _apPayList = raw;

  showView(wrapPurch(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small" id="appay-count"></span>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/purchasing/ap-payments/new')">
        <i class="bi bi-plus-lg me-1"></i>Record Payment
      </button>
    </div>
    ${_purchFilterBar([], '', 'apPayFilter()', {
      doc:  _apPayList.map(p => p.payment_number),
      supp: _apPayList.map(p => p.supplier_name_snapshot || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Payment #</th><th>Date</th><th>Supplier</th><th>Method</th><th>Amount</th><th></th></tr>
          </thead>
          <tbody id="appay-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  apPayFilter();
}

window.apPayFilter = function() {
  const filtered = _applyFilters(_apPayList, {
    docField: 'payment_number', customerField: 'supplier_name_snapshot',
    amountField: 'total_amount', dateField: 'date',
  });
  const countEl = document.getElementById('appay-count');
  if (countEl) countEl.textContent = `${filtered.length} / ${_apPayList.length} record(s)`;

  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No AP payments match the filter.</td></tr>`
    : filtered.map(p => `
        <tr>
          <td class="fw-semibold">${p.payment_number || '—'}</td>
          <td>${formatDate(p.date)}</td>
          <td>${p.supplier_name_snapshot || '—'}</td>
          <td>${p.payment_method || '—'}</td>
          <td>${formatCurrency(p.total_amount ?? 0)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-payments/${p.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  const el = document.getElementById('appay-tbody');
  if (el) el.innerHTML = rows;
};


async function loadAPPaymentDetail(id) {
  showLoading();
  const pay = await api.GetAPPayment(id);
  if (!pay) {
    showView(`<div class="alert alert-warning m-4">AP Payment not found.</div>`);
    return;
  }
  _apPayDetail = pay;

  const lines = pay.lines || [];
  const relPoId = lines[0]?.ap_invoice?.purchase_header_id || 0;
  const lineRows = lines.length === 0
    ? `<tr><td colspan="4" class="text-center text-muted py-3">No invoice lines.</td></tr>`
    : lines.map(l => {
        const inv = l.ap_invoice || {};
        const bal = (inv.doc_total || 0) - (inv.amount_paid_stored || 0);
        return `<tr>
          <td class="ps-4 fw-semibold">
            <a href="#" onclick="navigate('#/purchasing/ap-invoices/${l.ap_invoice_id}');return false;" class="text-decoration-none">
              ${inv.invoice_number || '—'}
            </a>
          </td>
          <td class="text-muted small">${formatDate(inv.posting_date)}</td>
          <td>${purchStatusBadge(inv.status)}</td>
          <td class="text-end pe-4 fw-bold text-warning">${formatCurrency(l.amount_applied)}</td>
        </tr>`;
      }).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:800px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/ap-payments')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-cash-stack me-2 text-warning"></i>${pay.payment_number}</h4>
          <small class="text-muted">AP Payment · ${formatDate(pay.date)}</small>
        </div>
        <span class="badge bg-${pay.status === 'Cancelled' ? 'secondary' : 'success'} fs-6">${pay.status || 'Posted'}</span>
        <button class="btn btn-sm btn-outline-secondary" onclick="printAPPayment()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('purchasing',${relPoId},'AP_PAY',${pay.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
        ${pay.status !== 'Cancelled' ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelAPPay(${pay.id})">
          <i class="bi bi-x-circle me-1"></i>Cancel</button>` : ''}
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <div class="small text-muted fw-bold">SUPPLIER</div>
              <div class="fw-semibold">${pay.supplier_name_snapshot || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">DATE</div>
              <div>${formatDate(pay.date)}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">PAYMENT METHOD</div>
              <div>${pay.payment_method || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">REFERENCE #</div>
              <div>${pay.reference_number || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">TOTAL AMOUNT</div>
              <div class="fw-bold fs-5 text-warning">${formatCurrency(pay.total_amount)}</div>
            </div>
            ${pay.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${pay.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-primary"></i>AP Invoices Settled</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr><th class="ps-4">Invoice #</th><th>Date</th><th>Status</th><th class="text-end pe-4">Amount Applied</th></tr>
            </thead>
            <tbody>${lineRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="3" class="text-end pe-3 ps-4">Total Payment</td>
                <td class="text-end pe-4 text-warning fs-5">${formatCurrency(pay.total_amount)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>
  `);
}


async function loadNewAPPaymentForm() {
  showLoading();
  const [suppliers, allInvoices, payMethods] = await Promise.all([
    api.ListSuppliers(true),
    api.ListAPInvoices(null, ''),
    api.ListPurchasingLookup('payment_method'),
  ]);
  const suppList = suppliers || [];
  _apPayAllInvoices = (allInvoices || []).filter(i => i.status === 'Open' || i.status === 'Partial');
  _apPayMethodsList = Array.isArray(payMethods) ? payMethods : [];

  const prefillInvId = window._apPaymentPrefillInvId || null;
  window._apPaymentPrefillInvId = null;

  const today = new Date().toISOString().slice(0, 10);

  const suppOptions = suppList.map(s =>
    `<option value="${s.id}" data-name="${s.name}">${s.name}</option>`
  ).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-payments')">
          <i class="bi bi-arrow-left"></i> Back
        </button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-cash-stack me-2 text-warning"></i>New AP Payment</h4>
          <div class="text-muted small">One payment — applies to one or more AP Invoices</div>
        </div>
      </div>

      <!-- Step 1: Supplier -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-truck me-2 text-warning"></i>Supplier</span>
        </div>
        <div class="card-body">
          <div class="row g-3 align-items-end">
            <div class="col-md-6">
              <label class="form-label small fw-bold">Select Supplier <span class="text-danger">*</span></label>
              <select id="apPaySupplierSel" class="form-select" onchange="apPayOnSupplierChange()">
                <option value="">— Select a supplier —</option>
                ${suppOptions}
              </select>
            </div>
            <div class="col-md-6" id="apPaySuppSummary" style="display:none">
              <div class="p-3 rounded d-flex gap-4" style="background:#fffbeb;border:1px solid #fde68a">
                <div>
                  <div class="small text-muted">Open Invoices</div>
                  <div class="fw-bold fs-5 text-warning" id="apPaySuppCount">—</div>
                </div>
                <div>
                  <div class="small text-muted">Total Outstanding</div>
                  <div class="fw-bold fs-5 text-danger" id="apPaySuppTotal">—</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Step 2: Invoices -->
      <div class="card border-0 shadow-sm mb-3" id="apPayInvCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center flex-wrap gap-2">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-primary"></i>AP Invoices to Pay</span>
          <div class="d-flex gap-2">
            <button type="button" class="btn btn-sm btn-outline-success" onclick="apPaySelectAll(true)">
              <i class="bi bi-check-all me-1"></i>Select All
            </button>
            <button type="button" class="btn btn-sm btn-outline-secondary" onclick="apPaySelectAll(false)">Deselect All</button>
            <button type="button" class="btn btn-sm btn-outline-warning" onclick="apPayFillBalances()">
              <i class="bi bi-arrow-down-circle me-1"></i>Fill Balances
            </button>
          </div>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:40px">
                  <input type="checkbox" class="form-check-input" id="apPayCheckAll" checked onchange="apPayToggleAll(this)">
                </th>
                <th>Invoice #</th><th>Date</th>
                <th class="text-end">Invoice Total</th><th class="text-end">Paid</th>
                <th class="text-end">Balance Due</th><th class="text-center" style="width:160px">Amount to Pay ₱</th>
              </tr>
            </thead>
            <tbody id="apPayInvBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4 text-muted">TOTAL TO PAY</td>
                <td class="text-end text-danger" id="apPayTotalBal">—</td>
                <td class="text-center fw-bold text-warning fs-6" id="apPayTotalApply">—</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <!-- No invoices -->
      <div id="apPayNoInvMsg" style="display:none" class="card border-0 shadow-sm mb-3">
        <div class="card-body text-center py-4 text-muted">
          <i class="bi bi-check-circle-fill text-success" style="font-size:2rem"></i>
          <div class="mt-2 fw-semibold">No open AP invoices for this supplier</div>
        </div>
      </div>

      <!-- Step 3: Payment Details -->
      <div class="card border-0 shadow-sm mb-3" id="apPayDetailsCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-credit-card me-2"></i>Payment Details</span>
        </div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">Date <span class="text-danger">*</span></label>
              <input type="date" id="apPayDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Payment Method</label>
              <select id="apPayMethod" class="form-select">
                ${buildLookupOptions(_apPayMethodsList, 'Cash')}
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Reference #</label>
              <input type="text" id="apPayRef" class="form-control" placeholder="Optional">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="apPayNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <!-- Submit Bar -->
      <div class="card border-0 shadow-sm" id="apPaySubmitCard" style="display:none">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col">
              <div class="small text-muted mb-1">Selected</div>
              <div class="fw-semibold" id="apPaySelCount">0 invoice(s)</div>
            </div>
            <div class="col text-center">
              <div class="small text-muted mb-1">Total Payment Amount</div>
              <div class="fw-bold fs-4 text-warning" id="apPayGrandTotal">₱0.00</div>
            </div>
            <div class="col text-end d-flex gap-2 justify-content-end">
              <button class="btn btn-outline-secondary px-4" onclick="navigate('#/purchasing/ap-payments')">Cancel</button>
              <button class="btn btn-warning px-5 text-white fw-bold" id="apPaySubmitBtn" disabled onclick="submitAPPayment()">
                <i class="bi bi-cash-stack me-2"></i>Post Payment
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `);

  // If prefill: auto-select supplier from invoice
  if (prefillInvId) {
    const inv = _apPayAllInvoices.find(i => i.id === prefillInvId);
    if (inv) {
      const suppSel = document.getElementById('apPaySupplierSel');
      // find by name match
      const opt = Array.from(suppSel.options).find(o => o.dataset.name === inv.supplier_name);
      if (opt) {
        suppSel.value = opt.value;
        apPayOnSupplierChange(prefillInvId);
      }
    }
  }
}


function apPayOnSupplierChange(prefillInvId) {
  const sel  = document.getElementById('apPaySupplierSel');
  const suppName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  if (!sel.value) {
    ['apPayInvCard','apPayDetailsCard','apPaySubmitCard','apPaySuppSummary','apPayNoInvMsg']
      .forEach(id => { const el = document.getElementById(id); if (el) el.style.display = 'none'; });
    return;
  }
  const openInvs = _apPayAllInvoices.filter(i => i.supplier_name === suppName);
  apPayRenderInvoices(openInvs, prefillInvId);
}


function apPayRenderInvoices(invoices, prefillInvId) {
  const tbody = document.getElementById('apPayInvBody');
  if (!tbody) return;
  tbody.innerHTML = '';

  if (invoices.length === 0) {
    ['apPayInvCard','apPayDetailsCard','apPaySubmitCard'].forEach(id => {
      const el = document.getElementById(id); if (el) el.style.display = 'none';
    });
    document.getElementById('apPayNoInvMsg').style.display = '';
    document.getElementById('apPaySuppSummary').style.display = 'none';
    return;
  }

  let totalBal = 0;
  invoices.forEach(inv => {
    const balance = Math.max(0, (inv.doc_total || 0) - (inv.amount_paid_stored || 0));
    totalBal += balance;
    const isPrefill = prefillInvId && inv.id !== prefillInvId;
    const row = document.createElement('tr');
    row.className = 'apPay-inv-row';
    row.innerHTML = `
      <td class="ps-4">
        <input type="checkbox" class="form-check-input apPay-inv-check" data-inv-id="${inv.id}"
          ${isPrefill ? '' : 'checked'} onchange="apPayToggleRow(this)">
      </td>
      <td class="fw-semibold">${inv.invoice_number}${inv.status === 'Partial' ? ' <span class="badge bg-warning text-dark ms-1" style="font-size:10px">Partial</span>' : ''}</td>
      <td class="text-muted small">${formatDate(inv.posting_date)}</td>
      <td class="text-end text-muted">${formatCurrency(inv.doc_total || 0)}</td>
      <td class="text-end text-success small">${formatCurrency(inv.amount_paid_stored || 0)}</td>
      <td class="text-end fw-bold text-danger">${formatCurrency(balance)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-end apPay-amt"
          value="${isPrefill ? '0.00' : balance.toFixed(2)}"
          min="0" max="${balance}" step="0.01" data-max="${balance}"
          ${isPrefill ? 'disabled' : ''}
          oninput="apPayUpdateTotals()">
      </td>`;
    tbody.appendChild(row);
  });

  document.getElementById('apPayTotalBal').textContent  = formatCurrency(totalBal);
  document.getElementById('apPaySuppCount').textContent = invoices.length;
  document.getElementById('apPaySuppTotal').textContent = formatCurrency(totalBal);
  document.getElementById('apPaySuppSummary').style.display  = '';
  document.getElementById('apPayInvCard').style.display      = '';
  document.getElementById('apPayNoInvMsg').style.display     = 'none';
  document.getElementById('apPayDetailsCard').style.display  = '';
  document.getElementById('apPaySubmitCard').style.display   = '';
  document.getElementById('apPayCheckAll').checked = true;
  apPayUpdateTotals();
}


function apPayToggleAll(master) {
  document.querySelectorAll('.apPay-inv-check').forEach(cb => { cb.checked = master.checked; apPayToggleRow(cb); });
}


function apPayToggleRow(cb) {
  const row = cb.closest('tr');
  const amt = row.querySelector('.apPay-amt');
  if (!cb.checked) { amt.disabled = true; amt.value = '0'; row.style.opacity = '0.4'; }
  else             { amt.disabled = false; amt.value = amt.dataset.max; row.style.opacity = '1'; }
  apPayUpdateTotals();
}


function apPaySelectAll(state) {
  document.querySelectorAll('.apPay-inv-check').forEach(cb => { cb.checked = state; apPayToggleRow(cb); });
  const master = document.getElementById('apPayCheckAll');
  if (master) master.checked = state;
}


function apPayFillBalances() {
  document.querySelectorAll('.apPay-inv-row').forEach(row => {
    const cb = row.querySelector('.apPay-inv-check');
    const amt = row.querySelector('.apPay-amt');
    if (cb && cb.checked && amt) amt.value = amt.dataset.max;
  });
  apPayUpdateTotals();
}


function apPayUpdateTotals() {
  let total = 0, count = 0;
  document.querySelectorAll('.apPay-inv-row').forEach(row => {
    const cb = row.querySelector('.apPay-inv-check');
    const amt = row.querySelector('.apPay-amt');
    if (cb && cb.checked && amt) { const v = parseFloat(amt.value) || 0; total += v; if (v > 0) count++; }
  });
  const applyEl = document.getElementById('apPayTotalApply');
  const grandEl = document.getElementById('apPayGrandTotal');
  const countEl = document.getElementById('apPaySelCount');
  const btn     = document.getElementById('apPaySubmitBtn');
  if (applyEl) applyEl.textContent = formatCurrency(total);
  if (grandEl) grandEl.textContent = formatCurrency(total);
  if (countEl) countEl.textContent = count + ' invoice(s)';
  if (btn)     btn.disabled = total <= 0;
}


async function submitAPPayment() {
  const sel      = document.getElementById('apPaySupplierSel');
  const suppId   = parseInt(sel.value) || null;
  const suppName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  const lines    = [];

  document.querySelectorAll('.apPay-inv-row').forEach(row => {
    const cb  = row.querySelector('.apPay-inv-check');
    const amt = row.querySelector('.apPay-amt');
    const id  = parseInt(cb.dataset.invId);
    const val = parseFloat(amt.value) || 0;
    if (cb.checked && val > 0) lines.push({ ap_invoice_id: id, amount_applied: val });
  });

  if (lines.length === 0) { toast('No invoices selected.', 'warning'); return; }

  // Validate: no line may exceed its invoice balance
  const overPaid = [];
  document.querySelectorAll('.apPay-inv-row').forEach(row => {
    const cb  = row.querySelector('.apPay-inv-check');
    const amt = row.querySelector('.apPay-amt');
    if (cb && cb.checked && amt) {
      const val = parseFloat(amt.value) || 0;
      const max = parseFloat(amt.dataset.max) || 0;
      if (val > max + 0.005) overPaid.push(row.querySelector('.fw-semibold')?.textContent?.trim() || '?');
    }
  });
  if (overPaid.length > 0) {
    toast(`Amount exceeds balance for: ${overPaid.join(', ')}`, 'danger');
    return;
  }

  const btn = document.getElementById('apPaySubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Posting…';

  const result = await api.CreateAPPayment({
    date:           document.getElementById('apPayDate').value,
    supplier_id:    suppId,
    supplier_name:  suppName,
    payment_method: document.getElementById('apPayMethod').value,
    ref_num:        document.getElementById('apPayRef').value,
    notes:          document.getElementById('apPayNotes').value,
    lines,
  });

  if (result) {
    navigate('#/purchasing/ap-payments');
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-cash-stack me-2"></i>Post Payment';
  }
}


async function cancelAPPay(id) {
  if (!confirm('Cancel this AP Payment? This will reverse the accounting entry and reopen the linked AP Invoice(s).')) return;
  const r = await api.CancelAPPayment(id);
  if (r) await loadAPPaymentDetail(id);
}


function printAPPayment() {
  const pay = _apPayDetail;
  if (!pay) return;
  const f = window._pfmt;
  const lines = pay.lines || [];
  const lineRows = lines.map(l => {
    const inv = l.ap_invoice || {};
    return `<tr>
      <td>${inv.invoice_number || '—'}</td>
      <td>${f.date(inv.posting_date)}</td>
      <td>${inv.status || '—'}</td>
      <td class="text-end">${f.currency(l.amount_applied)}</td>
    </tr>`;
  }).join('');
  const html = `
    <h2>AP PAYMENT</h2>
    <div class="sub">${pay.payment_number || '—'} &nbsp;·&nbsp; ${f.date(pay.date)}</div>
    <div class="info-grid">
      <div><span>Supplier</span><br>${pay.supplier_name_snapshot || '—'}</div>
      <div><span>Date</span><br>${f.date(pay.date)}</div>
      <div><span>Payment Method</span><br>${pay.payment_method || '—'}</div>
      <div><span>Reference #</span><br>${pay.reference_number || '—'}</div>
      ${pay.notes ? `<div style="grid-column:1/-1"><span>Notes</span><br>${pay.notes}</div>` : ''}
    </div>
    <div class="section-title">Invoices Settled</div>
    <table>
      <thead><tr><th>Invoice #</th><th>Invoice Date</th><th>Status</th><th class="text-end">Amount Applied</th></tr></thead>
      <tbody>${lineRows}</tbody>
    </table>
    <div class="totals">Total Payment: <strong>${f.currency(pay.total_amount)}</strong></div>`;
  _printDoc('AP Payment ' + (pay.payment_number || ''), html);
}

function _resetPurchasingAPPayment() {
  _apPayAllInvoices = []; _apPayDetail = null; _apPayMethodsList = [];
}
