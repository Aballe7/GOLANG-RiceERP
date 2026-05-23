// purchasing-dr.js

let _drList        = [];
let _drOpenHeaders = []; // OpenPOHeaderDTO[] from ListOpenPurchaseHeaders
let _drSuppliers   = [];
let _drDetail      = null;


async function loadDeliveryReceiptsList(tabBar) {
  sessionStorage.removeItem('docReturnRoute');
  const raw = await api.ListDeliveryReceipts(0);
  if (!raw) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load delivery receipts.</div>`));
    return;
  }
  _drList = raw;

  const statusOpts = [
    { v: '',          l: 'All' },
    { v: 'Pending',   l: 'Pending' },
    { v: 'Received',  l: 'Received' },
    { v: 'Cancelled', l: 'Cancelled' },
  ];

  showView(wrapPurch(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small" id="dr-count"></span>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/purchasing/delivery-receipts/new')">
        <i class="bi bi-plus-lg me-1"></i>New Delivery Receipt
      </button>
    </div>
    ${_purchFilterBar(statusOpts, '', 'drFilter()', {
      doc:  _drList.map(r => r.dr_number),
      supp: _drList.map(r => r.supplier_name || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>DR #</th><th>Delivery Date</th><th>Supplier</th><th>PO #</th><th>Amount</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="dr-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  drFilter();
}

window.drFilter = function() {
  const filtered = _applyFilters(_drList, {
    docField: 'dr_number', customerField: 'supplier_name',
    amountField: 'doc_total', dateField: 'posting_date',
  });
  const countEl = document.getElementById('dr-count');
  if (countEl) countEl.textContent = `${filtered.length} / ${_drList.length} record(s)`;

  const rows = filtered.length === 0
    ? `<tr><td colspan="7" class="text-center text-muted py-4">No delivery receipts match the filter.</td></tr>`
    : filtered.map(r => `
        <tr>
          <td class="fw-semibold">${r.dr_number || '—'}</td>
          <td>${formatDate(r.posting_date)}</td>
          <td>${r.supplier_name || '—'}</td>
          <td>${r.purchase_header?.po_number || '—'}</td>
          <td>${formatCurrency(r.doc_total ?? 0)}</td>
          <td>${purchStatusBadge(r.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/delivery-receipts/${r.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  const el = document.getElementById('dr-tbody');
  if (el) el.innerHTML = rows;
};


async function loadDRData(forceRefresh = false) {
  if (forceRefresh || _drOpenHeaders.length === 0 || _drSuppliers.length === 0) {
    const [suppliers, headers] = await Promise.all([
      api.ListSuppliers(true),
      api.ListOpenPurchaseHeaders(0),
    ]);
    _drSuppliers   = suppliers || [];
    _drOpenHeaders = headers  || [];
  }
}



function drCheckChanged() {
  const anyChecked = !!document.querySelector('#drPoTable tbody input[type=checkbox]:checked');
  const btn = document.getElementById('drGenLinesBtn');
  if (btn) btn.disabled = !anyChecked;
}


function drOnSupplierChange() {
  const supId     = parseInt(document.getElementById('drSupplier').value) || 0;
  const poCard    = document.getElementById('drPoCard');
  const tbody     = document.querySelector('#drPoTable tbody');
  const linesCard = document.getElementById('drLinesCard');
  const linesBody = document.getElementById('drLinesBody');

  // Reset lines whenever supplier changes.
  if (linesBody) linesBody.innerHTML = '';
  if (linesCard) linesCard.style.display = 'none';
  drRecalcGrandTotal();

  if (!supId) {
    if (poCard) poCard.style.display = 'none';
    return;
  }

  // Filter open PO headers by supplier
  const openHeaders = _drOpenHeaders.filter(h => parseInt(h.supplier_id) === supId);

  const rows = openHeaders.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-3">No open purchase orders for this supplier.</td></tr>`
    : openHeaders.map(h => {
        const openLines  = h.lines || [];
        const totalOpen  = openLines.reduce((s, l) => s + (l.open_qty || 0), 0);
        const totalValue = openLines.reduce((s, l) => s + (l.open_qty || 0) * (l.price || 0), 0);
        return `
          <tr data-header-id="${h.id}">
            <td class="text-center ps-3">
              <input type="checkbox" class="form-check-input" value="${h.id}"
                onchange="drCheckChanged()">
            </td>
            <td class="fw-semibold">${_esc(h.po_number)}</td>
            <td>${formatDate(h.posting_date)}</td>
            <td class="text-center">
              <span class="badge bg-primary-subtle text-primary border">${openLines.length} line${openLines.length !== 1 ? 's' : ''}</span>
            </td>
            <td class="text-end fw-semibold text-primary">${formatCurrency(totalValue)}</td>
          </tr>`;
      }).join('');

  if (tbody) tbody.innerHTML = rows;
  if (poCard) poCard.style.display = '';
  drCheckChanged();
}


function drGenerateLines() {
  const tbody = document.getElementById('drLinesBody');
  const card  = document.getElementById('drLinesCard');
  if (!tbody || !card) return;

  // Track already-added PurchaseLine IDs to avoid duplicates
  const existingLineIds = new Set(
    Array.from(tbody.querySelectorAll('tr')).map(r => parseInt(r.dataset.lineId))
  );

  document.querySelectorAll('#drPoTable tbody input[type=checkbox]:checked').forEach(cb => {
    const headerId = parseInt(cb.value);
    const header   = _drOpenHeaders.find(h => h.id === headerId);
    if (!header) return;

    (header.lines || []).forEach(line => {
      if (existingLineIds.has(line.id)) return;
      existingLineIds.add(line.id);

      const qtyOpen   = parseFloat(line.open_qty) || 0;
      const price     = parseFloat(line.price)    || 0;
      const lineTotal = qtyOpen * price;

      const row = document.createElement('tr');
      row.dataset.lineId = line.id;
      row.innerHTML = `
        <td class="text-center">
          <button type="button" class="btn btn-sm btn-outline-danger py-0 px-1" onclick="drRemoveLine(this)">
            <i class="bi bi-x"></i>
          </button>
        </td>
        <td class="text-center"><span class="badge bg-light text-dark border">${_esc(header.po_number)}</span></td>
        <td>${_esc(line.description || '')}</td>
        <td class="text-center">${_esc(line.unit || '')}</td>
        <td class="text-end text-muted">${qtyOpen.toLocaleString()}</td>
        <td class="text-end" style="width:120px">
          <input type="number" class="form-control form-control-sm text-end dr-qty-recv"
            value="${qtyOpen}" min="0" max="${qtyOpen}" step="0.001"
            data-purchase-header-id="${headerId}"
            data-purchase-line-id="${line.id}"
            data-item-code="${_esc(line.item_code || '')}"
            data-description="${_esc(line.description || '')}"
            data-unit="${_esc(line.unit || '')}"
            data-qty-ordered="${qtyOpen}"
            data-price="${price}"
            data-uom-entry="${line.i_uom_entry || 0}"
            oninput="drRecalcLine(this)">
        </td>
        <td class="text-end text-muted">${formatCurrency(price)}</td>
        <td class="text-end fw-bold text-primary dr-line-total">${formatCurrency(lineTotal)}</td>
      `;
      tbody.appendChild(row);
    });
  });

  drRecalcGrandTotal();
  card.style.display = tbody.children.length > 0 ? '' : 'none';
}


function drRemoveLine(btn) {
  btn.closest('tr')?.remove();
  drRecalcGrandTotal();
  const tbody = document.getElementById('drLinesBody');
  const card = document.getElementById('drLinesCard');
  if (card && tbody) card.style.display = tbody.children.length > 0 ? '' : 'none';
}


function drRecalcLine(input) {
  const row = input.closest('tr');
  const qty = parseFloat(input.value) || 0;
  const price = parseFloat(input.dataset.price) || 0;
  row.querySelector('.dr-line-total').textContent = formatCurrency(qty * price);
  drRecalcGrandTotal();
}


function drRecalcGrandTotal() {
  let total = 0;
  document.querySelectorAll('#drLinesBody .dr-qty-recv').forEach(inp => {
    total += (parseFloat(inp.value) || 0) * (parseFloat(inp.dataset.price) || 0);
  });
  const el = document.getElementById('drGrandTotal');
  if (el) el.textContent = formatCurrency(total);
}


async function loadNewDRForm() {
  showLoading();
  await loadDRData(true); // always fresh — qty_received changes after each confirmed DR

  // Capture and clear the "Copy To" prefill token immediately after data loads
  // so that navigating away and back doesn't re-trigger it.
  const prefillPOId = window._drPrefillPOId ? parseInt(window._drPrefillPOId) : null;
  window._drPrefillPOId = null;

  // Guard: if the prefill PO already has a Draft DR, redirect to it instead of opening the form
  if (prefillPOId) {
    const poForCheck = await api.GetPurchaseHeader(prefillPOId);
    if (poForCheck) {
      const existingDraft = (poForCheck.delivery_receipts || []).find(dr => dr.status === 'Draft');
      if (existingDraft) {
        toast(`Draft receipt ${existingDraft.dr_number} already exists — confirm or cancel it first.`, 'warning', 8000);
        navigate(`#/purchasing/delivery-receipts/${existingDraft.id}`);
        return;
      }
    }
    window._drPrefillPOId = prefillPOId; // restore for the prefill logic below
  }

  const today = new Date().toISOString().slice(0, 10);

  const supOptions = _drSuppliers.map(s =>
    `<option value="${s.id}">${_esc(s.name)}</option>`
  ).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:1000px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/delivery-receipts')">
          <i class="bi bi-arrow-left"></i> Back
        </button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-truck me-2 text-success"></i>New Delivery Receipt</h4>
          <div class="text-muted small">Select a supplier, then pick purchase orders to receive</div>
        </div>
      </div>

      <!-- Header fields -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">DR Date <span class="text-danger">*</span></label>
              <input type="date" id="drDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Supplier <span class="text-danger">*</span></label>
              <select id="drSupplier" class="form-select" onchange="drOnSupplierChange()" required>
                <option value="">— Select supplier —</option>
                ${supOptions}
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Received By <span class="text-danger">*</span></label>
              <input type="text" id="drReceivedBy" class="form-control" placeholder="Name of receiver" required>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Supplier DR Ref #</label>
              <input type="text" id="drSupplierRef" class="form-control" placeholder="Optional">
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="drNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <!-- PO selection (hidden until supplier chosen) -->
      <div class="card border-0 shadow-sm mb-3" id="drPoCard" style="display:none">
        <div class="card-header bg-transparent border-0 pb-0 pt-3 px-4 d-flex align-items-center justify-content-between">
          <span class="fw-bold">Select Purchase Orders</span>
          <button id="drGenLinesBtn" class="btn btn-sm btn-primary" disabled onclick="drGenerateLines()">
            <i class="bi bi-list-ul me-1"></i>Generate Lines
          </button>
        </div>
        <div class="card-body pt-2 pb-2">
          <div class="table-responsive">
            <table class="table table-sm table-hover align-middle mb-0" id="drPoTable">
              <thead class="table-light">
                <tr>
                  <th class="ps-3" style="width:40px"></th>
                  <th>PO #</th>
                  <th>Date</th>
                  <th class="text-center">Open Lines</th>
                  <th class="text-end text-primary">Open Value</th>
                </tr>
              </thead>
              <tbody></tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Lines (hidden until Generate Lines clicked) -->
      <div class="card border-0 shadow-sm mb-3" id="drLinesCard" style="display:none">
        <div class="card-header bg-transparent border-0 pb-0 pt-3 px-4">
          <span class="fw-bold">Line Items</span>
          <span class="text-muted small ms-2">Edit the quantity received for each item</span>
        </div>
        <div class="card-body pt-2 pb-2">
          <div class="table-responsive">
            <table class="table table-sm align-middle mb-0">
              <thead class="table-light">
                <tr>
                  <th style="width:36px"></th>
                  <th>PO #</th>
                  <th>Description</th>
                  <th class="text-center">Unit</th>
                  <th class="text-end">Open Qty</th>
                  <th class="text-end" style="width:130px">Qty Received <span class="text-danger">*</span></th>
                  <th class="text-end">Unit Price</th>
                  <th class="text-end">Line Total</th>
                </tr>
              </thead>
              <tbody id="drLinesBody"></tbody>
              <tfoot>
                <tr>
                  <td colspan="8" class="text-end fw-bold">Grand Total</td>
                  <td class="text-end fw-bold text-primary" id="drGrandTotal">₱0.00</td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>

      <div class="d-flex justify-content-end gap-2">
        <button class="btn btn-outline-secondary px-4" onclick="navigate('#/purchasing/delivery-receipts')">Cancel</button>
        <button class="btn btn-outline-primary px-4" id="drDraftBtn" onclick="submitNewDR(true)">
          <i class="bi bi-floppy me-1"></i>Save as Draft
        </button>
        <button class="btn btn-success px-4" id="drSubmitBtn" onclick="submitNewDR(false)">
          <i class="bi bi-check-lg me-1"></i>Create &amp; Confirm
        </button>
      </div>
    </div>
  `);

  // ── "Copy To" prefill ─────────────────────────────────────────────────────
  // Triggered when user clicks "Create Delivery Receipt" on a PO detail page.
  // Mimics SAP B1's "Copy To → Goods Receipt PO" behaviour:
  //   1. Auto-select the supplier
  //   2. Render the PO selection table (drOnSupplierChange)
  //   3. Tick the source PO checkbox
  //   4. Generate lines pre-filled with each line's current open_qty
  if (prefillPOId) {
    const header = _drOpenHeaders.find(h => h.id === prefillPOId);
    if (header && header.supplier_id) {
      const supSel = document.getElementById('drSupplier');
      if (supSel) {
        supSel.value = header.supplier_id;
        drOnSupplierChange();                  // renders PO rows filtered by supplier

        // Tick the source PO and generate its lines
        const cb = document.querySelector(`#drPoTable tbody input[value="${prefillPOId}"]`);
        if (cb) {
          cb.checked = true;
          drCheckChanged();                    // enables Generate Lines button
          drGenerateLines();                   // pre-fills lines with open_qty values
        }
      }
    } else {
      // PO has no open lines left — inform the user and leave the form empty
      toast('This purchase order has no open lines to receive.', 'warning');
    }
  }
}


async function submitNewDR(draft = false) {
  const dateVal    = document.getElementById('drDate').value;
  const receivedBy = document.getElementById('drReceivedBy').value.trim();
  const supId      = document.getElementById('drSupplier')?.value;
  if (!dateVal)    { toast('DR date is required.', 'warning'); return; }
  if (!supId)      { toast('Please select a supplier.', 'warning'); return; }
  if (!receivedBy) { toast('Received by is required.', 'warning'); return; }

  const lineInputs = document.querySelectorAll('#drLinesBody .dr-qty-recv');
  if (lineInputs.length === 0) { toast('Generate lines from at least one purchase order first.', 'warning'); return; }

  const lines = Array.from(lineInputs).map(inp => ({
    purchase_header_id: parseInt(inp.dataset.purchaseHeaderId) || 0,
    purchase_line_id:   parseInt(inp.dataset.purchaseLineId)   || 0,
    item_code:        inp.dataset.itemCode    || '',
    description:      inp.dataset.description || '',
    category:         inp.dataset.category    || '',
    unit:             inp.dataset.unit        || '',
    uom_entry:        parseInt(inp.dataset.uomEntry || '0') || 0,
    quantity_ordered: parseFloat(inp.dataset.qtyOrdered) || 0,
    quantity:         parseFloat(inp.value) || 0,
    price:            parseFloat(inp.dataset.price) || 0,
  }));

  const payload = {
    date:        dateVal,
    received_by: receivedBy,
    ref_number:  document.getElementById('drSupplierRef').value,
    comments:    document.getElementById('drNotes').value,
    lines,
  };

  const btn      = draft ? document.getElementById('drDraftBtn')  : document.getElementById('drSubmitBtn');
  const otherBtn = draft ? document.getElementById('drSubmitBtn') : document.getElementById('drDraftBtn');
  btn.disabled = true;
  otherBtn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';

  const res = await api.CreateDeliveryReceipt(payload);
  if (!res) {
    btn.disabled = false;
    otherBtn.disabled = false;
    btn.innerHTML = draft
      ? '<i class="bi bi-floppy me-1"></i>Save as Draft'
      : '<i class="bi bi-check-lg me-1"></i>Create &amp; Confirm';
    return;
  }

  if (draft) {
    navigate(`#/purchasing/delivery-receipts/${res.id}`);
    return;
  }

  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Confirming…';
  await api.ConfirmDeliveryReceipt(res.id);
  _drOpenHeaders = []; // invalidate so next DR form fetches fresh open_qty values
  navigate(`#/purchasing/delivery-receipts/${res.id}`);
}


async function loadDRDetail(id) {
  showLoading();
  const dr = await api.GetDeliveryReceipt(id);
  if (!dr) {
    showView(`<div class="alert alert-warning m-4">Delivery receipt not found.</div>`);
    return;
  }
  _drDetail = dr;

  const po    = dr.purchase_header || {};
  const items  = dr.lines || [];
  const apInvs = dr.ap_invoices || [];

  const total = items.reduce((s, i) => s + (i.quantity * i.price), 0);
  const uninvoiced = total - (dr.doc_total || 0);

  const statusColor = { Received: 'success', Cancelled: 'secondary', Draft: 'warning' };
  const badge = `<span class="badge bg-${statusColor[dr.status] || 'secondary'} fs-6">${dr.status}</span>`;

  const hasAPInvoices = (dr.ap_invoices || []).length > 0;
  const canCancelDR = (dr.status === 'Draft') ||
    (dr.status === 'Received' && !hasAPInvoices);

  const actionBtns = dr.status === 'Draft' ? `
    <button class="btn btn-sm btn-success" onclick="confirmDR(${dr.id})">
      <i class="bi bi-check-lg me-1"></i>Confirm Receipt
    </button>
    <button class="btn btn-sm btn-outline-danger" onclick="cancelDR(${dr.id})">Cancel</button>
  ` : (dr.status === 'Received' ? `
    ${uninvoiced > 0.005 ? `<button class="btn btn-sm btn-primary" onclick="newAPInvoiceFromDR(${dr.id})">
      <i class="bi bi-file-earmark-plus me-1"></i>Create AP Invoice
    </button>` : ''}
    ${canCancelDR ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelDR(${dr.id})">
      <i class="bi bi-x-circle me-1"></i>Cancel DR
    </button>` : ''}
  ` : '');

  const invoiceStatusCard = dr.status === 'Received' ? `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-2">
        <span class="small fw-bold text-muted">INVOICE STATUS</span>
      </div>
      <div class="card-body">
        <div class="row g-2 text-center">
          <div class="col">
            <div class="small text-muted mb-1">DR Total</div>
            <div class="fw-bold">${formatCurrency(total)}</div>
          </div>
          <div class="col-auto d-flex align-items-center text-muted">→</div>
          <div class="col">
            <div class="small text-muted mb-1">AP Invoiced</div>
            <div class="fw-bold text-primary">${formatCurrency(dr.doc_total || 0)}</div>
          </div>
          <div class="col-auto d-flex align-items-center text-muted">→</div>
          <div class="col">
            <div class="small text-muted mb-1">Uninvoiced</div>
            <div class="fw-bold ${uninvoiced > 0.005 ? 'text-danger' : 'text-success'}">
              ${uninvoiced > 0.005 ? formatCurrency(uninvoiced) : '<i class="bi bi-check-circle-fill me-1"></i>Fully Invoiced'}
            </div>
          </div>
        </div>
      </div>
    </div>
  ` : '';

  const itemRows = items.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-3">No items.</td></tr>`
    : items.map(i => `
      <tr>
        <td class="ps-4"><span class="badge bg-light text-dark border">${i.category || '—'}</span></td>
        <td class="fw-semibold">${i.description}</td>
        <td class="text-end text-muted">${formatNumber(i.quantity_ordered)} ${i.unit}</td>
        <td class="text-end fw-bold">${formatNumber(i.quantity)} ${i.unit}</td>
        <td class="text-end">${formatCurrency(i.price)}</td>
        <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity * i.price)}</td>
      </tr>`).join('');

  const apRows = apInvs.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between">
        <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-primary"></i>AP Invoices</span>
        ${uninvoiced > 0.005 ? `<button class="btn btn-sm btn-outline-primary" onclick="newAPInvoiceFromDR(${dr.id})">+ Add AP Invoice</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th class="ps-4">Invoice #</th><th>Date</th><th class="text-end">Total</th>
                <th class="text-end">Paid</th><th class="text-end">Balance</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            ${apInvs.map(inv => {
              const bal = (inv.doc_total || 0) - (inv.amount_paid_stored || 0);
              return `<tr>
                <td class="ps-4 fw-semibold">${inv.invoice_number}</td>
                <td class="text-muted small">${formatDate(inv.posting_date)}</td>
                <td class="text-end">${formatCurrency(inv.doc_total || 0)}</td>
                <td class="text-end text-success small">${formatCurrency(inv.amount_paid_stored || 0)}</td>
                <td class="text-end fw-bold ${bal <= 0.005 ? 'text-success' : 'text-danger'}">${bal <= 0.005 ? 'Paid' : formatCurrency(bal)}</td>
                <td>${purchStatusBadge(inv.status)}</td>
                <td class="text-end pe-3">
                  <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-invoices/${inv.id}')">View</button>
                </td>
              </tr>`;
            }).join('')}
          </tbody>
        </table>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate(sessionStorage.getItem('docReturnRoute')||'#/purchasing/delivery-receipts');sessionStorage.removeItem('docReturnRoute')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-box-seam me-2 text-success"></i>${dr.dr_number}</h4>
          <small class="text-muted">Delivery Receipt · ${formatDate(dr.posting_date)}</small>
        </div>
        ${badge}
        <button class="btn btn-sm btn-outline-secondary" onclick="printDR()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('purchasing',${dr.purchase_header_id},'DR',${dr.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
        ${actionBtns}
      </div>

      ${invoiceStatusCard}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-info-circle me-2"></i>Receipt Details</span>
        </div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <div class="small text-muted">Purchase Order</div>
              <div class="fw-semibold">
                <a href="#" onclick="navigate('#/purchasing/purchases/${dr.purchase_header_id}');return false;" class="text-decoration-none">
                  ${po.po_number || '—'}
                </a>
              </div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted">Supplier</div>
              <div class="fw-semibold">${dr.supplier_name || po.supplier_name || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted">Supplier DR Ref #</div>
              <div>${dr.ref_number || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted">Received By</div>
              <div>${dr.received_by || '—'}</div>
            </div>
            ${dr.comments ? `<div class="col-12"><div class="small text-muted">Notes</div><div>${dr.comments}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-box-seam me-2"></i>Items Received</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr><th class="ps-4">Category</th><th>Item</th><th class="text-end">Ordered</th>
                  <th class="text-end">Received</th><th class="text-end">Unit Price</th><th class="text-end pe-4">Line Total</th></tr>
            </thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4">Total</td>
                <td class="text-end pe-4 text-primary fs-5">${formatCurrency(total)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      ${apRows}
    </div>
  `);
}


async function confirmDR(id) {
  if (!confirm('Confirm delivery and update inventory?')) return;
  const r = await api.ConfirmDeliveryReceipt(id);
  if (r) {
    _drOpenHeaders = []; // invalidate cache so next New DR form shows updated open_qty
    await loadDRDetail(id);
  }
}


async function cancelDR(id) {
  const dr = _drDetail || {};
  const msg = dr.status === 'Received'
    ? 'Cancel this confirmed delivery receipt? This will reverse the goods-receipt accounting entry and reopen the source PO lines.'
    : 'Cancel this delivery receipt?';
  if (!confirm(msg)) return;
  const r = await api.CancelDeliveryReceipt(id);
  if (r) await loadDRDetail(id);
}


function newAPInvoiceFromDR(drId) {
  window._apInvoicePrefillDRId = drId;
  navigate('#/purchasing/ap-invoices/new');
}


function printDR() {
  const dr = _drDetail;
  if (!dr) return;
  const f = window._pfmt;
  const po = dr.purchase_header || {};
  const items = dr.lines || [];
  const total = items.reduce((s, i) => s + (i.quantity * i.price), 0);
  const itemRows = items.map(i => `
    <tr>
      <td>${i.description || '—'}</td>
      <td>${i.category || '—'}</td>
      <td>${i.unit || '—'}</td>
      <td class="text-end">${f.num(i.quantity_ordered)}</td>
      <td class="text-end">${f.num(i.quantity)}</td>
      <td class="text-end">${f.currency(i.price)}</td>
      <td class="text-end">${f.currency(i.quantity * i.price)}</td>
    </tr>`).join('');
  const html = `
    <h2>DELIVERY RECEIPT</h2>
    <div class="sub">${dr.dr_number || '—'} &nbsp;·&nbsp; <span class="badge">${dr.status || '—'}</span> &nbsp;·&nbsp; ${f.date(dr.posting_date)}</div>
    <div class="info-grid">
      <div><span>Purchase Order</span><br>${po.po_number || '—'}</div>
      <div><span>Supplier</span><br>${dr.supplier_name || po.supplier_name || '—'}</div>
      <div><span>Supplier DR Ref #</span><br>${dr.ref_number || '—'}</div>
      <div><span>Received By</span><br>${dr.received_by || '—'}</div>
      ${dr.comments ? `<div style="grid-column:1/-1"><span>Notes</span><br>${dr.comments}</div>` : ''}
    </div>
    <table>
      <thead><tr><th>Item</th><th>Category</th><th>Unit</th><th class="text-end">Ordered</th><th class="text-end">Received</th><th class="text-end">Unit Price</th><th class="text-end">Line Total</th></tr></thead>
      <tbody>${itemRows}</tbody>
    </table>
    <div class="totals">Total Received Value: <strong>${f.currency(total)}</strong></div>`;
  _printDoc('DR ' + (dr.dr_number || ''), html);
}

function _resetPurchasingDR() {
  _drOpenHeaders = []; _drSuppliers = []; _drDetail = null;
  window._apInvoicePrefillDRId = null;
}
