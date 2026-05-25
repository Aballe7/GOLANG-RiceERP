// purchasing-ap-invoice.js

let _apList          = [];
let _apInvDRs        = [];
let _apInvSuppliers  = [];
let _apInvDetail     = null;
let _apInvTermsList  = [];
window._apOverdueOnly = false; // true when the "Overdue" preset is active


async function loadApInvoiceData(forceRefresh = false) {
  if (forceRefresh || _apInvDRs.length === 0 || _apInvSuppliers.length === 0) {
    const [suppliers, drs, terms] = await Promise.all([
      api.ListSuppliers(true),
      api.ListDeliveryReceipts(0),
      api.ListPurchasingLookup('payment_terms'),
    ]);
    _apInvSuppliers = suppliers || [];
    _apInvDRs       = drs || [];
    _apInvTermsList = terms || [];
  }
}


function _apInvDrLabel(dr) {
  return dr.dr_number || ('DR-' + String(dr.id).padStart(5, '0'));
}


async function loadApInvoicesList(tabBar) {
  const raw = await api.ListAPInvoices(null, '');
  if (!raw) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load AP invoices.</div>`));
    return;
  }
  _apList = raw;

  const statusOpts = [
    { v: 'Open,Partial', l: 'Open / Partial' },
    { v: 'Open',         l: 'Open' },
    { v: 'Partial',      l: 'Partial' },
    { v: 'Paid',         l: 'Paid' },
    { v: 'Cancelled',    l: 'Cancelled' },
    { v: '',             l: 'All' },
  ];

  showView(wrapPurch(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <h5 class="fw-bold mb-0">AP Invoices</h5>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/purchasing/ap-invoices/new')">
        <i class="bi bi-plus-lg me-1"></i>New AP Invoice
      </button>
    </div>

    <!-- Date preset buttons -->
    <div class="d-flex flex-wrap gap-2 mb-3" id="ap-presets">
      <span class="text-muted small align-self-center me-1"><i class="bi bi-calendar3 me-1"></i>Quick range:</span>
      <button class="btn btn-sm btn-outline-secondary" data-preset="this-month" onclick="apInvDatePreset('this-month')">This Month</button>
      <button class="btn btn-sm btn-outline-secondary" data-preset="last-30"    onclick="apInvDatePreset('last-30')">Last 30 Days</button>
      <button class="btn btn-sm btn-outline-secondary" data-preset="last-90"    onclick="apInvDatePreset('last-90')">Last 90 Days</button>
      <button class="btn btn-sm btn-outline-secondary" data-preset="custom"     onclick="apInvDatePreset('custom')"><i class="bi bi-sliders me-1"></i>Custom Range</button>
      <button class="btn btn-sm btn-outline-danger"    data-preset="overdue"    onclick="apInvDatePreset('overdue')"><i class="bi bi-exclamation-circle me-1"></i>Overdue</button>
    </div>

    <!-- KPI summary strip -->
    <div class="row g-3 mb-3">
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-receipt me-1"></i>Total Purchases</div>
            <div class="fw-bold fs-5 text-dark" id="ap-kpi-total">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ap-kpi-total-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-check-circle me-1 text-success"></i>Paid</div>
            <div class="fw-bold fs-5 text-success" id="ap-kpi-paid">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ap-kpi-paid-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-hourglass-split me-1 text-warning"></i>Outstanding</div>
            <div class="fw-bold fs-5 text-warning" id="ap-kpi-outstanding">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ap-kpi-outstanding-sub"></div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body py-2 px-3">
            <div class="text-muted small mb-1"><i class="bi bi-exclamation-circle me-1 text-danger"></i>Open Invoices</div>
            <div class="fw-bold fs-5 text-danger" id="ap-kpi-open">—</div>
            <div class="text-muted" style="font-size:.7rem" id="ap-kpi-open-sub"></div>
          </div>
        </div>
      </div>
    </div>

    ${_purchFilterBar(statusOpts, 'Open,Partial', 'apInvFilter()', {
      doc:  _apList.map(i => i.invoice_number),
      supp: _apList.map(i => i.supplier_name || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Invoice #</th><th>Date</th><th>Due</th><th>Supplier</th><th>Amount</th><th>Balance</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="ap-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  apInvFilter();
}

window.apInvDatePreset = function(preset) {
  const today = new Date();
  const fmt   = d => d.toISOString().slice(0, 10);
  let from = '', to = fmt(today);

  // Reset overdue mode unless this IS the overdue preset
  if (preset !== 'overdue') window._apOverdueOnly = false;

  if (preset === 'this-month') {
    from = fmt(new Date(today.getFullYear(), today.getMonth(), 1));
  } else if (preset === 'last-30') {
    const d = new Date(today); d.setDate(d.getDate() - 30); from = fmt(d);
  } else if (preset === 'last-90') {
    const d = new Date(today); d.setDate(d.getDate() - 90); from = fmt(d);
  } else if (preset === 'custom') {
    const fromEl = document.getElementById('sf-date-from');
    if (fromEl) { fromEl.focus(); fromEl.showPicker && fromEl.showPicker(); }
    _apMarkPreset('custom');
    return;
  } else if (preset === 'overdue') {
    // Clear date range, force status to Open/Partial, enable overdue flag
    const fromEl = document.getElementById('sf-date-from');
    const toEl   = document.getElementById('sf-date-to');
    const statEl = document.getElementById('sf-status');
    if (fromEl) fromEl.value = '';
    if (toEl)   toEl.value   = '';
    if (statEl) statEl.value = 'Open,Partial';
    window._apOverdueOnly = true;
    _apMarkPreset('overdue');
    apInvFilter();
    return;
  }

  const fromEl = document.getElementById('sf-date-from');
  const toEl   = document.getElementById('sf-date-to');
  if (fromEl) fromEl.value = from;
  if (toEl)   toEl.value   = to;
  _apMarkPreset(preset);
  apInvFilter();
};

function _apMarkPreset(active) {
  document.querySelectorAll('#ap-presets [data-preset]').forEach(btn => {
    const isActive = btn.dataset.preset === active;
    btn.classList.toggle('btn-secondary', isActive);
    btn.classList.toggle('btn-outline-secondary', !isActive);
  });
}

window.apInvFilter = function() {
  // If user changed the status away from Open,Partial while overdue mode was active, exit overdue mode
  if (window._apOverdueOnly && _sfVal('sf-status') !== 'Open,Partial') {
    window._apOverdueOnly = false;
    _apMarkPreset('');
  }

  const filtered = _applyFilters(_apList, {
    docField: 'invoice_number', customerField: 'supplier_name',
    amountField: 'doc_total', dateField: 'posting_date',
    defaultStatuses: ['Open', 'Partial'],
  });

  // When overdue mode is on, further restrict to past-due rows only
  const today = new Date().toISOString().slice(0, 10);
  const displayRows = window._apOverdueOnly
    ? filtered.filter(i => {
        const due = (i.due_date || '').slice(0, 10);
        return due && due < today && (i.status === 'Open' || i.status === 'Partial');
      })
    : filtered;

  // ── KPI computation (all non-Cancelled, date-filtered) ────────────────────
  const dateFrom = _sfVal('sf-date-from');
  const dateTo   = _sfVal('sf-date-to');
  const activeAP = _apList.filter(inv => {
    if (inv.status === 'Cancelled') return false;
    const d = (inv.posting_date || '').slice(0, 10);
    if (dateFrom && d < dateFrom) return false;
    if (dateTo   && d > dateTo)   return false;
    return true;
  });

  const total       = activeAP.reduce((s, i) => s + (i.doc_total ?? 0), 0);
  const paid        = activeAP.reduce((s, i) => s + (i.amount_paid_stored ?? 0), 0);
  const outstanding = total - paid;
  const openInvoices    = activeAP.filter(i => i.status === 'Open' || i.status === 'Partial');
  const overdueInvoices = openInvoices.filter(i => {
    const due = (i.due_date || '').slice(0, 10);
    return due && due < today;
  });

  const _kpi = id => document.getElementById(id);
  if (_kpi('ap-kpi-total'))          _kpi('ap-kpi-total').textContent          = formatCurrency(total);
  if (_kpi('ap-kpi-total-sub'))      _kpi('ap-kpi-total-sub').textContent      = `${activeAP.length} invoice${activeAP.length !== 1 ? 's' : ''}`;
  if (_kpi('ap-kpi-paid'))           _kpi('ap-kpi-paid').textContent           = formatCurrency(paid);
  if (_kpi('ap-kpi-paid-sub'))       _kpi('ap-kpi-paid-sub').textContent       = total > 0 ? `${Math.round(paid / total * 100)}% of purchases` : '';
  if (_kpi('ap-kpi-outstanding'))    _kpi('ap-kpi-outstanding').textContent    = formatCurrency(outstanding);
  if (_kpi('ap-kpi-outstanding-sub')) _kpi('ap-kpi-outstanding-sub').textContent = total > 0 ? `${Math.round(outstanding / total * 100)}% of purchases` : '';
  if (_kpi('ap-kpi-open'))           _kpi('ap-kpi-open').textContent           = openInvoices.length;
  if (_kpi('ap-kpi-open-sub')) {
    const openSub = _kpi('ap-kpi-open-sub');
    if (overdueInvoices.length > 0) {
      openSub.innerHTML = `<span class="text-danger fw-semibold">${overdueInvoices.length} overdue</span>`;
    } else if (openInvoices.length > 0) {
      openSub.textContent = formatCurrency(openInvoices.reduce((s, i) => s + Math.max(0, (i.doc_total ?? 0) - (i.amount_paid_stored ?? 0)), 0)) + ' due';
    } else {
      openSub.textContent = 'All clear';
    }
  }

  const rows = displayRows.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No AP invoices match the filter.</td></tr>`
    : displayRows.map(i => {
        const balance = Math.max(0, (i.doc_total ?? 0) - (i.amount_paid_stored ?? 0));
        const due = (i.due_date || '').slice(0, 10);
        const isOverdue = due && due < today && (i.status === 'Open' || i.status === 'Partial');
        const rowCls      = isOverdue ? ' class="table-warning"' : '';
        const overdueBadge = isOverdue ? `<span class="badge bg-danger ms-1">Overdue</span>` : '';
        return `
        <tr${rowCls}>
          <td class="fw-semibold">${i.invoice_number || '—'}</td>
          <td>${formatDate(i.posting_date)}</td>
          <td>${isOverdue ? `<span class="text-danger fw-semibold">${formatDate(i.due_date)}</span>` : formatDate(i.due_date)}</td>
          <td>${i.supplier_name || '—'}</td>
          <td>${formatCurrency(i.doc_total ?? 0)}</td>
          <td class="${balance > 0 ? 'text-danger fw-semibold' : ''}">${formatCurrency(balance)}</td>
          <td>${purchStatusBadge(i.status)}${overdueBadge}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-invoices/${i.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>`;
      }).join('');

  const el = document.getElementById('ap-tbody');
  if (el) el.innerHTML = rows;
};


function apInvCheckChanged() {
  const anyChecked = !!document.querySelector('#apInvDrTable tbody input[type=checkbox]:checked');
  const btn = document.getElementById('apInvGenLinesBtn');
  if (btn) btn.disabled = !anyChecked;
}


function apInvOnSupplierChange() {
  const supId     = parseInt(document.getElementById('apInvSupplier').value) || 0;
  const drCard    = document.getElementById('apInvDrCard');
  const tbody     = document.querySelector('#apInvDrTable tbody');
  const linesCard = document.getElementById('apInvLinesCard');
  const linesBody = document.getElementById('apInvLinesBody');

  // Reset lines whenever supplier changes
  if (linesBody) linesBody.innerHTML = '';
  if (linesCard) linesCard.style.display = 'none';
  _apInvWHTResult = null;
  const whtPanel = document.getElementById('apInvWHTPanel');
  if (whtPanel) whtPanel.classList.add('d-none');
  apInvRecalcGrandTotal();

  if (!supId) {
    if (drCard) drCard.style.display = 'none';
    return;
  }

  // Filter: only Received DRs for this supplier that have at least one open (uninvoiced) line
  const openDRs = _apInvDRs.filter(dr => {
    if (dr.status !== 'Received') return false;
    if (parseInt(dr.supplier_id) !== supId) return false;
    return (dr.lines || []).some(l => (parseFloat(l.open_qty) || 0) > 0);
  });

  const rows = openDRs.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-3">No open delivery receipts for this supplier.</td></tr>`
    : openDRs.map(dr => {
        const drLabel  = _apInvDrLabel(dr);
        const drTotal  = (dr.lines || []).reduce((s, l) => s + ((l.quantity || 0) * (l.price || 0)), 0);
        const invoiced = dr.doc_total || 0;
        const openAmt  = drTotal - invoiced;
        return `
          <tr data-dr-id="${dr.id}" data-purchase-header-id="${dr.purchase_header_id}">
            <td class="text-center ps-3">
              <input type="checkbox" class="form-check-input" value="${dr.id}"
                onchange="apInvCheckChanged()">
            </td>
            <td class="fw-semibold">${_esc(drLabel)}</td>
            <td>${formatDate(dr.posting_date)}</td>
            <td class="text-end">${formatCurrency(drTotal)}</td>
            <td class="text-end text-primary">${invoiced > 0.005 ? formatCurrency(invoiced) : '—'}</td>
            <td class="text-end fw-semibold text-success">${formatCurrency(openAmt)}</td>
          </tr>`;
      }).join('');

  if (tbody) tbody.innerHTML = rows;
  if (drCard) drCard.style.display = '';
  apInvCheckChanged();
}


function apInvGenerateLines() {
  const tbody = document.getElementById('apInvLinesBody');
  const card  = document.getElementById('apInvLinesCard');
  if (!tbody || !card) return;

  // Track already-added dr_line IDs to avoid duplicates
  const existing = new Set(
    Array.from(tbody.querySelectorAll('tr')).map(r => parseInt(r.dataset.drLineId))
  );

  document.querySelectorAll('#apInvDrTable tbody input[type=checkbox]:checked').forEach(cb => {
    const drId = parseInt(cb.value);
    const dr   = _apInvDRs.find(x => x.id === drId);
    if (!dr) return;
    const drLabel = _apInvDrLabel(dr);

    (dr.lines || []).forEach(line => {
      const openQty = parseFloat(line.open_qty) || 0;
      if (openQty <= 0) return;       // skip fully invoiced lines
      if (existing.has(line.id)) return; // skip duplicates
      existing.add(line.id);

      const price     = parseFloat(line.price) || 0;
      const lineTotal = openQty * price;
      const row = document.createElement('tr');
      row.dataset.drLineId = line.id;
      row.dataset.drId     = drId;
      row.innerHTML = `
        <td class="text-center">
          <button type="button" class="btn btn-sm btn-outline-danger py-0 px-1" onclick="apInvRemoveLine(this)">
            <i class="bi bi-x"></i>
          </button>
        </td>
        <td class="text-center"><span class="badge bg-light text-dark border">${_esc(drLabel)}</span></td>
        <td>${_esc(line.description || '')}</td>
        <td class="text-center"><span class="badge bg-secondary-subtle text-dark border">${_esc(line.category || '—')}</span></td>
        <td class="text-center">${_esc(line.unit || '')}</td>
        <td class="text-end text-muted">${openQty.toLocaleString()}</td>
        <td class="text-end" style="width:120px">
          <input type="number" class="form-control form-control-sm text-end ap-inv-qty"
            value="${openQty}" min="0.001" max="${openQty}" step="0.001"
            data-dr-line-id="${line.id}"
            data-dr-id="${drId}"
            data-purchase-header-id="${dr.purchase_header_id}"
            data-base-doc-entry="${drId}"
            data-base-line-num="${line.line_num || 0}"
            data-description="${_esc(line.description || '')}"
            data-category="${_esc(line.category || '')}"
            data-unit="${_esc(line.unit || '')}"
            data-price="${price}"
            oninput="apInvRecalcLine(this)">
        </td>
        <td class="text-end text-muted">${formatCurrency(price)}</td>
        <td class="text-end fw-bold text-primary ap-inv-line-total">${formatCurrency(lineTotal)}</td>
      `;
      tbody.appendChild(row);
    });
  });

  apInvRecalcGrandTotal();
  card.style.display = tbody.children.length > 0 ? '' : 'none';
}


function apInvRemoveLine(btn) {
  btn.closest('tr')?.remove();
  apInvRecalcGrandTotal();
  const tbody = document.getElementById('apInvLinesBody');
  const card  = document.getElementById('apInvLinesCard');
  if (card && tbody) card.style.display = tbody.children.length > 0 ? '' : 'none';
}


function apInvRecalcLine(input) {
  const row   = input.closest('tr');
  const qty   = parseFloat(input.value) || 0;
  const price = parseFloat(input.dataset.price) || 0;
  row.querySelector('.ap-inv-line-total').textContent = formatCurrency(qty * price);
  apInvRecalcGrandTotal();
}


function apInvRecalcGrandTotal() {
  let total = 0;
  document.querySelectorAll('#apInvLinesBody .ap-inv-qty').forEach(inp => {
    total += (parseFloat(inp.value) || 0) * (parseFloat(inp.dataset.price) || 0);
  });
  const el = document.getElementById('apInvGrandTotal');
  if (el) el.textContent = formatCurrency(total);
  apInvRecomputeWHT(total);
}

// ── WHT auto-compute ──────────────────────────────────────────────────────────
// Stores last WHT result so submitNewAPInvoice() can read it.
let _apInvWHTResult = null;

async function apInvRecomputeWHT(grossTotal) {
  const supId  = document.getElementById('apInvSupplier')?.value;
  const panel  = document.getElementById('apInvWHTPanel');
  if (!panel || !supId || !grossTotal) {
    if (panel) panel.classList.add('d-none');
    _apInvWHTResult = null;
    return;
  }

  const sup = _apInvSuppliers.find(s => s.id === parseInt(supId));
  if (!sup || !sup.wht_category || sup.wht_category === 'NONE') {
    panel.classList.add('d-none');
    _apInvWHTResult = null;
    return;
  }

  // VAT-exclusive base: if supplier is VAT-registered, base = gross / 1.12
  // If not VAT-registered, the gross IS the VAT-exclusive amount (no VAT on invoice)
  const vatExclusive = sup.is_vat_registered
    ? Math.round((grossTotal / 1.12) * 100) / 100
    : grossTotal;

  const dateVal = document.getElementById('apInvDate')?.value || new Date().toISOString().slice(0, 10);

  const res = await api.ComputeWHT({
    supplier_id:          sup.id,
    wht_category:         sup.wht_category,
    vat_exclusive_amount: vatExclusive,
    gross_amount:         grossTotal,
    invoice_date:         dateVal,
  });

  if (!res) { panel.classList.add('d-none'); _apInvWHTResult = null; return; }
  const w = res;
  _apInvWHTResult = w;

  panel.classList.remove('d-none');
  panel.innerHTML = w.applicable ? `
    <div class="card border-warning border-0 shadow-sm">
      <div class="card-header py-2 px-3" style="background:#fff9e6">
        <span class="fw-bold text-warning-emphasis">
          <i class="bi bi-receipt me-1"></i>BIR Withholding Tax (EWT) Applied
        </span>
        <span class="badge bg-warning text-dark ms-2">${w.atc_code} — ${w.rate_pct}</span>
      </div>
      <div class="card-body py-2 px-3">
        <div class="row g-2 small">
          <div class="col-md-3">
            <div class="text-muted">Gross Invoice</div>
            <div class="fw-semibold">${formatCurrency(w.gross_amount)}</div>
          </div>
          <div class="col-md-3">
            <div class="text-muted">VAT-Exclusive Base</div>
            <div class="fw-semibold">${formatCurrency(w.vat_exclusive_base)}</div>
          </div>
          <div class="col-md-3">
            <div class="text-muted">WHT Amount (${w.rate_pct})</div>
            <div class="fw-semibold text-danger">− ${formatCurrency(w.amount)}</div>
          </div>
          <div class="col-md-3">
            <div class="text-muted">Net Payable to Supplier</div>
            <div class="fw-bold text-primary fs-6">${formatCurrency(w.net_payable)}</div>
          </div>
        </div>
        <div class="text-muted mt-1" style="font-size:11px">
          <i class="bi bi-info-circle me-1"></i>${w.reason}
          ${w.ytd_purchases > 0 ? `&ensp;|&ensp; YTD purchases: ${formatCurrency(w.ytd_purchases)}` : ''}
        </div>
        <div class="mt-2 d-flex gap-2 align-items-center" style="font-size:12px">
          <span class="text-muted">Journal entry:</span>
          <span>Dr GRNI ${formatCurrency(w.gross_amount)}</span>
          <span class="text-muted">→</span>
          <span>Cr AP Payable ${formatCurrency(w.net_payable)}</span>
          <span class="text-muted">+</span>
          <span>Cr WHT Payable ${formatCurrency(w.amount)}</span>
        </div>
      </div>
    </div>` : `
    <div class="alert alert-light border py-2 px-3 small mb-0">
      <i class="bi bi-info-circle me-1 text-muted"></i>
      <strong>${sup.name}</strong> — ${w.reason}
      ${w.ytd_purchases > 0 ? `<br><span class="text-muted ms-3">YTD purchases this year: ${formatCurrency(w.ytd_purchases)}</span>` : ''}
    </div>`;
}


async function loadNewAPInvoiceForm() {
  showLoading();
  await loadApInvoiceData(true);
  const today = new Date().toISOString().slice(0, 10);

  const prefillDRId = window._apInvoicePrefillDRId || null;
  window._apInvoicePrefillDRId = null;

  const supOptions = _apInvSuppliers.map(s =>
    `<option value="${s.id}">${_esc(s.name)}</option>`
  ).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:1000px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-invoices')">
          <i class="bi bi-arrow-left"></i> Back
        </button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-file-earmark-text me-2" style="color:#8b5cf6"></i>New AP Invoice</h4>
          <div class="text-muted small">Select a supplier, then pick delivery receipts to invoice</div>
        </div>
      </div>

      <!-- Header fields -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">Invoice Date <span class="text-danger">*</span></label>
              <input type="date" id="apInvDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Supplier <span class="text-danger">*</span></label>
              <select id="apInvSupplier" class="form-select" onchange="apInvOnSupplierChange()" required>
                <option value="">— Select supplier —</option>
                ${supOptions}
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="apInvDueDate" class="form-control">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Terms</label>
              <select id="apInvTerms" class="form-select">
                ${buildLookupOptions(_apInvTermsList, 'COD')}
              </select>
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Supplier Invoice Ref #</label>
              <input type="text" id="apInvRef" class="form-control" placeholder="Supplier's invoice number">
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="apInvNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <!-- DR selection (hidden until supplier chosen) -->
      <div class="card border-0 shadow-sm mb-3" id="apInvDrCard" style="display:none">
        <div class="card-header bg-transparent border-0 pb-0 pt-3 px-4 d-flex align-items-center justify-content-between">
          <span class="fw-bold">Select Delivery Receipts</span>
          <button id="apInvGenLinesBtn" class="btn btn-sm btn-primary" disabled onclick="apInvGenerateLines()">
            <i class="bi bi-list-ul me-1"></i>Generate Lines
          </button>
        </div>
        <div class="card-body pt-2 pb-2">
          <div class="table-responsive">
            <table class="table table-sm table-hover align-middle mb-0" id="apInvDrTable">
              <thead class="table-light">
                <tr>
                  <th class="ps-3" style="width:40px"></th>
                  <th>DR #</th>
                  <th>Date</th>
                  <th class="text-end">DR Total</th>
                  <th class="text-end text-primary">Invoiced</th>
                  <th class="text-end text-success">Open</th>
                </tr>
              </thead>
              <tbody></tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Lines (hidden until Generate Lines clicked) -->
      <div class="card border-0 shadow-sm mb-3" id="apInvLinesCard" style="display:none">
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
                  <th>Base DR #</th>
                  <th>Item</th>
                  <th class="text-center">Category</th>
                  <th class="text-center">Unit</th>
                  <th class="text-end">Available</th>
                  <th class="text-end" style="width:120px">Qty to Invoice <span class="text-danger">*</span></th>
                  <th class="text-end">Unit Price</th>
                  <th class="text-end">Line Total</th>
                </tr>
              </thead>
              <tbody id="apInvLinesBody"></tbody>
              <tfoot>
                <tr>
                  <td colspan="8" class="text-end fw-bold pe-3">Invoice Total</td>
                  <td class="text-end fw-bold text-primary fs-5 pe-2" id="apInvGrandTotal">₱0.00</td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>

      <!-- WHT panel (shown/hidden automatically when supplier has WHT category) -->
      <div id="apInvWHTPanel" class="mb-3 d-none"></div>

      <div class="d-flex justify-content-end gap-2">
        <button class="btn btn-outline-secondary px-4" onclick="navigate('#/purchasing/ap-invoices')">Cancel</button>
        <button class="btn btn-primary px-4" id="apInvSubmitBtn" onclick="submitNewAPInvoice()">
          <i class="bi bi-file-earmark-text me-1"></i>Create AP Invoice
        </button>
      </div>
    </div>
  `);

  if (prefillDRId) {
    const dr = _apInvDRs.find(d => d.id === parseInt(prefillDRId));
    if (dr) {
      const supSel     = document.getElementById('apInvSupplier');
      const supplierId = parseInt(dr.supplier_id) || 0;
      if (supSel && supplierId) {
        supSel.value = supplierId;
        apInvOnSupplierChange();
      }
      // Auto-check the prefill DR and generate its lines
      const cb = document.querySelector(`#apInvDrTable tbody input[value="${prefillDRId}"]`);
      if (cb) {
        cb.checked = true;
        apInvCheckChanged();
        apInvGenerateLines();
      }
    }
  }
}


async function submitNewAPInvoice() {
  const supId   = document.getElementById('apInvSupplier').value;
  const dateVal = document.getElementById('apInvDate').value;

  if (!supId)   { toast('Please select a supplier.', 'warning'); return; }
  if (!dateVal) { toast('Invoice date is required.', 'warning'); return; }

  const lineInputs = document.querySelectorAll('#apInvLinesBody .ap-inv-qty');
  if (lineInputs.length === 0) {
    toast('Generate lines from at least one delivery receipt first.', 'warning');
    return;
  }

  const lines = [];
  let firstPurchaseHeaderId = 0;
  let firstDrId = null;

  lineInputs.forEach(inp => {
    const qty = parseFloat(inp.value) || 0;
    if (qty <= 0) return;
    if (!firstPurchaseHeaderId) firstPurchaseHeaderId = parseInt(inp.dataset.purchaseHeaderId) || 0;
    if (firstDrId === null && inp.dataset.drId) firstDrId = parseInt(inp.dataset.drId);
    lines.push({
      description:    inp.dataset.description,
      category:       inp.dataset.category,
      unit:           inp.dataset.unit,
      quantity:       qty,
      price:          parseFloat(inp.dataset.price) || 0,
      dr_line_id:     parseInt(inp.dataset.drLineId) || null,
      base_doc_entry: parseInt(inp.dataset.baseDocEntry) || null,
      base_line_num:  parseInt(inp.dataset.baseLineNum)  || null,
    });
  });

  if (lines.length === 0) { toast('No invoice lines with quantity > 0.', 'warning'); return; }
  if (!firstPurchaseHeaderId) { toast('Could not determine source Purchase Order.', 'warning'); return; }

  const sup        = _apInvSuppliers.find(s => s.id === parseInt(supId));
  const dueDateVal = document.getElementById('apInvDueDate').value;

  const btn = document.getElementById('apInvSubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Creating…';

  const result = await api.CreateAPInvoice({
    purchase_header_id:   firstPurchaseHeaderId,
    dr_id:                firstDrId,
    date:                 dateVal,
    terms:                document.getElementById('apInvTerms').value,
    supplier_name:        sup?.name || '',
    ref_number:           document.getElementById('apInvRef').value,
    comments:             document.getElementById('apInvNotes').value,
    due_date:             dueDateVal || null,
    lines,
    // WHT fields — zeros when no WHT applies
    vat_exclusive_amount: _apInvWHTResult?.vat_exclusive_base ?? 0,
    wht_rate:             _apInvWHTResult?.applicable ? (_apInvWHTResult.rate ?? 0) : 0,
    wht_amount:           _apInvWHTResult?.applicable ? (_apInvWHTResult.amount ?? 0) : 0,
    wht_atc_code:         _apInvWHTResult?.applicable ? (_apInvWHTResult.atc_code ?? '') : '',
  });

  if (result) {
    _apInvDRs = []; // invalidate cache — open_qty changed
    navigate(`#/purchasing/ap-invoices/${result.id}`);
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-file-earmark-text me-1"></i>Create AP Invoice';
  }
}


async function loadAPInvoiceDetail(id) {
  showLoading();
  const inv = await api.GetAPInvoice(id);
  if (!inv) {
    showView(`<div class="alert alert-warning m-4">AP Invoice not found.</div>`);
    return;
  }
  _apInvDetail = inv;

  const items        = inv.lines || [];
  const paymentLines = inv.payment_lines || [];
  const totalPaid    = inv.amount_paid_stored || 0;
  const balance      = Math.max(0, (inv.doc_total || 0) - totalPaid);
  const po           = inv.purchase_header || {};
  const dr           = inv.delivery_receipt || null;

  const statusColor  = { Paid: 'success', Partial: 'warning', Open: 'primary', Cancelled: 'secondary' };
  const badge        = `<span class="badge bg-${statusColor[inv.status] || 'secondary'} fs-6">${inv.status}</span>`;

  const canPay    = inv.status === 'Open' || inv.status === 'Partial';
  const canCancel = (inv.status === 'Open' || inv.status === 'Partial') && paymentLines.length === 0;

  const statusBanner = inv.status === 'Paid'
    ? `<div class="alert alert-success border-0 mb-3 d-flex align-items-center gap-2">
        <i class="bi bi-check-circle-fill fs-5"></i><strong>Fully Paid</strong> — ${formatCurrency(inv.doc_total)} settled</div>`
    : inv.status === 'Partial'
    ? `<div class="alert alert-warning border-0 mb-3 d-flex align-items-center gap-2">
        <i class="bi bi-clock-fill fs-5"></i>Balance: <strong>${formatCurrency(balance)}</strong> remaining</div>`
    : inv.status === 'Open'
    ? `<div class="alert alert-primary border-0 mb-3">${formatCurrency(inv.doc_total)} outstanding</div>`
    : '';

  const itemRows = items.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-3">No items.</td></tr>`
    : items.map(i => `
      <tr>
        <td class="ps-4"><span class="badge bg-light text-dark border">${i.category || '—'}</span></td>
        <td class="fw-semibold">${i.description}</td>
        <td class="text-center">${formatNumber(i.quantity)}</td>
        <td class="text-center text-muted">${i.unit}</td>
        <td class="text-center">${formatCurrency(i.price)}</td>
        <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity * i.price)}</td>
      </tr>`).join('');

  const payRows = paymentLines.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
        <span class="fw-bold"><i class="bi bi-cash-stack me-2 text-warning"></i>Payments Applied</span>
        ${canPay ? `<button class="btn btn-sm btn-outline-warning" onclick="newAPPaymentFromInvoice(${inv.id})">+ Add Payment</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th class="ps-4">Payment #</th><th>Date</th><th>Method</th><th>Reference</th>
                <th class="text-end pe-4">Amount Applied</th></tr>
          </thead>
          <tbody>
            ${paymentLines.map(pl => {
              const pay = pl.payment || {};
              return `<tr>
                <td class="ps-4 fw-bold">${pay.payment_number || '—'}</td>
                <td class="text-muted small">${formatDate(pay.date)}</td>
                <td>${pay.payment_method || '—'}</td>
                <td class="text-muted">${pay.reference_number || '—'}</td>
                <td class="text-end pe-4 fw-bold text-warning">${formatCurrency(pl.amount_applied)}</td>
              </tr>`;
            }).join('')}
          </tbody>
          <tfoot class="table-light fw-bold small">
            <tr>
              <td colspan="4" class="ps-4 text-muted">Invoice Total / Paid / Balance</td>
              <td class="text-end pe-4 text-primary">${formatCurrency(inv.doc_total)}</td>
            </tr>
            <tr>
              <td colspan="4"></td>
              <td class="text-end pe-4 text-success">${formatCurrency(totalPaid)} paid</td>
            </tr>
            <tr>
              <td colspan="4"></td>
              <td class="text-end pe-4 ${balance <= 0.005 ? 'text-success' : 'text-danger fw-bold'}">
                ${balance <= 0.005 ? '<i class="bi bi-check-circle-fill me-1"></i>Fully Paid' : formatCurrency(balance) + ' remaining'}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/ap-invoices')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-file-earmark-text me-2 text-primary"></i>${inv.invoice_number}</h4>
          <small class="text-muted">AP Invoice · ${formatDate(inv.posting_date)}</small>
        </div>
        ${badge}
        <button class="btn btn-sm btn-outline-secondary" onclick="printAPInvoice()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('purchasing',${inv.purchase_header_id},'AP_INV',${inv.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
        ${canPay ? `<button class="btn btn-sm btn-warning text-white fw-bold" onclick="newAPPaymentFromInvoice(${inv.id})">
          <i class="bi bi-cash-stack me-1"></i>Record Payment</button>` : ''}
        ${canCancel ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelAPInv(${inv.id})">Cancel</button>` : ''}
      </div>

      ${statusBanner}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <div class="small text-muted fw-bold">SUPPLIER</div>
              <div class="fw-semibold">${inv.supplier_name || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">INVOICE DATE</div>
              <div>${formatDate(inv.posting_date)}</div>
            </div>
            ${inv.due_date ? `<div class="col-md-4"><div class="small text-muted fw-bold">DUE DATE</div><div>${formatDate(inv.due_date)}</div></div>` : ''}
            <div class="col-md-4">
              <div class="small text-muted fw-bold">TERMS</div>
              <div>${inv.terms || '—'}</div>
            </div>
            ${inv.ref_number ? `<div class="col-md-4"><div class="small text-muted fw-bold">SUPPLIER INV REF #</div><div>${inv.ref_number}</div></div>` : ''}
            <div class="col-md-4">
              <div class="small text-muted fw-bold">PURCHASE ORDER</div>
              <a href="#" onclick="navigate('#/purchasing/purchases/${inv.purchase_header_id}');return false;" class="text-decoration-none fw-semibold">
                ${po.po_number || '—'}
              </a>
            </div>
            ${dr ? `<div class="col-md-4"><div class="small text-muted fw-bold">DELIVERY RECEIPT</div>
              <a href="#" onclick="navigate('#/purchasing/delivery-receipts/${inv.delivery_receipt_id}');return false;" class="text-decoration-none fw-semibold">${dr.dr_number}</a></div>` : ''}
            ${inv.comments ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${inv.comments}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-box-seam me-2"></i>Line Items</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr><th class="ps-4">Category</th><th>Item</th><th class="text-center">Qty</th>
                  <th class="text-center">Unit</th><th class="text-center">Unit Price</th><th class="text-end pe-4">Total</th></tr>
            </thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4">Invoice Total (Gross)</td>
                <td class="text-end pe-4 text-primary fs-5">${formatCurrency(inv.doc_total)}</td>
              </tr>
              ${inv.wht_amount > 0 ? `
              <tr class="text-warning-emphasis">
                <td colspan="5" class="text-end pe-3 ps-4 small">
                  <i class="bi bi-receipt me-1"></i>
                  Withholding Tax (${inv.wht_atc_code} ${(inv.wht_rate * 100).toFixed(0)}%) — BIR EWT
                </td>
                <td class="text-end pe-4 text-danger">− ${formatCurrency(inv.wht_amount)}</td>
              </tr>
              <tr class="fw-bold">
                <td colspan="5" class="text-end pe-3 ps-4">Net Payable to Supplier</td>
                <td class="text-end pe-4 text-success fs-5">${formatCurrency(inv.net_payable)}</td>
              </tr>` : ''}
            </tfoot>
          </table>
        </div>
      </div>

      ${payRows}
    </div>
  `);
}


async function cancelAPInv(id) {
  if (!confirm('Cancel this AP Invoice? This will reverse the accounting entry.')) return;
  const r = await api.CancelAPInvoice(id);
  if (r) await loadAPInvoiceDetail(id);
}


function newAPPaymentFromInvoice(invId) {
  window._apPaymentPrefillInvId = invId;
  navigate('#/purchasing/ap-payments/new');
}

function printAPInvoice() {
  const inv = _apInvDetail;
  if (!inv) return;
  const f = window._pfmt;
  const po = inv.purchase_header || {};
  const dr = inv.delivery_receipt || null;
  const items = inv.lines || [];
  const paymentLines = inv.payment_lines || [];
  const totalPaid = inv.amount_paid_stored || 0;
  const balance = Math.max(0, (inv.doc_total || 0) - totalPaid);
  const itemRows = items.map(i => `
    <tr>
      <td>${i.description || '—'}</td>
      <td>${i.category || '—'}</td>
      <td class="text-end">${f.num(i.quantity)}</td>
      <td>${i.unit || '—'}</td>
      <td class="text-end">${f.currency(i.price)}</td>
      <td class="text-end">${f.currency(i.quantity * i.price)}</td>
    </tr>`).join('');
  const payRows = paymentLines.length === 0 ? '' : `
    <div class="section-title">Payments Applied</div>
    <table>
      <thead><tr><th>Payment #</th><th>Date</th><th>Method</th><th class="text-end">Amount Applied</th></tr></thead>
      <tbody>${paymentLines.map(pl => {
        const pay = pl.payment || {};
        return `<tr>
          <td>${pay.payment_number || '—'}</td>
          <td>${f.date(pay.date)}</td>
          <td>${pay.payment_method || '—'}</td>
          <td class="text-end">${f.currency(pl.amount_applied)}</td>
        </tr>`;
      }).join('')}</tbody>
    </table>`;
  const html = `
    <h2>AP INVOICE</h2>
    <div class="sub">${inv.invoice_number || '—'} &nbsp;·&nbsp; <span class="badge">${inv.status || '—'}</span> &nbsp;·&nbsp; ${f.date(inv.posting_date)}</div>
    <div class="info-grid">
      <div><span>Supplier</span><br>${inv.supplier_name || '—'}</div>
      <div><span>Invoice Date</span><br>${f.date(inv.posting_date)}</div>
      ${inv.due_date ? `<div><span>Due Date</span><br>${f.date(inv.due_date)}</div>` : ''}
      <div><span>Terms</span><br>${inv.terms || '—'}</div>
      ${inv.ref_number ? `<div><span>Supplier Inv Ref #</span><br>${inv.ref_number}</div>` : ''}
      <div><span>Purchase Order</span><br>${po.po_number || '—'}</div>
      ${dr ? `<div><span>Delivery Receipt</span><br>${dr.dr_number}</div>` : ''}
      ${inv.comments ? `<div style="grid-column:1/-1"><span>Notes</span><br>${inv.comments}</div>` : ''}
    </div>
    <table>
      <thead><tr><th>Item</th><th>Category</th><th class="text-end">Qty</th><th>Unit</th><th class="text-end">Unit Price</th><th class="text-end">Total</th></tr></thead>
      <tbody>${itemRows}</tbody>
    </table>
    <div class="totals">
      Doc Total: <strong>${f.currency(inv.doc_total)}</strong><br>
      Amount Paid: ${f.currency(totalPaid)}<br>
      Balance: ${f.currency(balance)}
    </div>
    ${payRows}`;
  _printDoc('AP Invoice ' + (inv.invoice_number || ''), html);
}

function _resetPurchasingAPInvoice() {
  _apList = []; _apInvDRs = []; _apInvSuppliers = []; _apInvDetail = null;
  _apInvTermsList = []; _apInvWHTResult = null;
  window._apPaymentPrefillInvId = null;
}
