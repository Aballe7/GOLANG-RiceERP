// purchasing-po.js — SAP B1-aligned Purchase Order (OPOR/POR1)

let _poList   = [];
let _poDetail = null;
let _poItemMaster = null, _poCategories = [], _poUoMGroups = [], _poLineCounter = 0;

async function loadPurchasesList(tabBar) {
  const raw = await api.ListPurchaseHeaders();
  if (!raw) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load purchase orders.</div>`));
    return;
  }
  _poList = raw;

  const statusOpts = [
    { v: '',          l: 'All' },
    { v: 'Pending',   l: 'Pending' },
    { v: 'Partial',   l: 'Partial' },
    { v: 'Delivered', l: 'Delivered' },
    { v: 'Cancelled', l: 'Cancelled' },
  ];

  showView(wrapPurch(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small" id="po-count"></span>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/purchasing/purchases/new')">
        <i class="bi bi-plus-lg me-1"></i>New Purchase Order
      </button>
    </div>
    ${_purchFilterBar(statusOpts, '', 'poFilter()', {
      doc:  _poList.map(p => p.po_number),
      supp: _poList.map(p => p.supplier_name || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>PO #</th><th>Date</th><th>Supplier</th><th>Amount</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="po-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  poFilter();
}

window.poFilter = function() {
  const filtered = _applyFilters(_poList, {
    docField: 'po_number', customerField: 'supplier_name',
    amountField: 'doc_total', dateField: 'posting_date',
  });
  const countEl = document.getElementById('po-count');
  if (countEl) countEl.textContent = `${filtered.length} / ${_poList.length} record(s)`;

  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No purchase orders match the filter.</td></tr>`
    : filtered.map(p => `
        <tr>
          <td class="fw-semibold">${p.po_number || '—'}</td>
          <td>${formatDate(p.posting_date)}</td>
          <td>${p.supplier_name || '—'}</td>
          <td>${formatCurrency(p.doc_total ?? 0)}</td>
          <td>${purchStatusBadge(p.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="viewPurchase(${p.id})">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  const el = document.getElementById('po-tbody');
  if (el) el.innerHTML = rows;
};


function viewPurchase(id) { navigate(`#/purchasing/purchases/${id}`); }


async function cancelPurchaseOrder(id) {
  if (!confirm('Cancel this Purchase Order? This cannot be undone.')) return;
  const r = await api.CancelPurchaseHeader(id);
  if (r) await loadPurchaseDetail(id);
}


async function loadNewPurchaseForm() {
  const [suppliers, items, cats, groups, payMethods] = await Promise.all([
    api.ListSuppliers(true),
    api.ListPurchasingItems(),
    api.ListItemCategoriesForModule('purchasing'),
    api.ListUoMGroups(),
    api.ListPurchasingLookup('payment_method'),
  ]);
  const _paymentMethods = payMethods || [];
  _poItemMaster = items  || [];
  _poCategories = cats   || [];
  _poUoMGroups  = groups || [];
  const supplierOptions = suppliers.map(s =>
    `<option value="${s.id}" data-name="${s.name}">${s.name}</option>`
  ).join('');

  const today = new Date().toISOString().slice(0, 10);

  showView(`
    <div class="container p-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/purchases')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">New Purchase Order</h4>
      </div>

      <!-- Header -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">PO Number</label>
              <input type="text" class="form-control" id="poPONumber" readonly placeholder="Auto">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Date <span class="text-danger">*</span></label>
              <input type="date" class="form-control" id="poDate" required value="${today}">
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Supplier</label>
              <select class="form-select" id="poSupplierID" onchange="poFillSupplierName()">
                <option value="">— Select supplier —</option>
                ${supplierOptions}
              </select>
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Supplier Name (snapshot)</label>
              <input type="text" class="form-control" id="poSupplierName" placeholder="Auto-filled or type manually">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Payment Method</label>
              <select class="form-select" id="poPaymentMethod">
                ${buildLookupOptions(_paymentMethods, 'Cash')}
              </select>
            </div>
            <div class="col-md-12">
              <label class="form-label small fw-bold">Remarks</label>
              <textarea class="form-control" id="poRemarks" rows="2" placeholder="Optional"></textarea>
            </div>
          </div>
        </div>
      </div>

      <!-- Line Items -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-box-seam me-2"></i>Line Items</span>
          <button class="btn btn-sm btn-outline-primary" type="button" onclick="poAddLine()">
            <i class="bi bi-plus-lg me-1"></i>Add Line
          </button>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0" id="poLinesTable">
            <thead class="table-light">
              <tr>
                <th style="width:5%"></th>
                <th style="width:20%">Category</th>
                <th>Item</th>
                <th style="width:10%" class="text-center">Unit</th>
                <th style="width:12%" class="text-end">Qty</th>
                <th style="width:15%" class="text-end">Unit Price</th>
                <th style="width:15%" class="text-end">Line Total</th>
              </tr>
            </thead>
            <tbody id="poLinesBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="6" class="text-end pe-3">PO Total</td>
                <td class="text-end pe-3 text-primary fs-5" id="poGrandTotal">₱0.00</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 mt-3 justify-content-end">
        <button type="button" class="btn btn-outline-secondary"
          onclick="navigate('#/purchasing/purchases')">Cancel</button>
        <button type="button" class="btn btn-primary px-4" id="poSaveBtn" onclick="submitNewPO()">
          <i class="bi bi-check-lg me-1"></i>Save Purchase Order
        </button>
      </div>
    </div>
  `);

  _poLineCounter = 0;

  // Autopopulate PO number
  const poField = document.getElementById('poPONumber');
  if (poField) {
    const next = await api.GetNextPONumber();
    if (next) poField.value = next;
  }

  // Add one empty line by default
  poAddLine();
}


function poFillSupplierName() {
  const sel = document.getElementById('poSupplierID');
  const opt = sel.options[sel.selectedIndex];
  if (opt && opt.dataset.name) {
    document.getElementById('poSupplierName').value = opt.dataset.name;
  }
}


function poAddLine() {
  const tbody = document.getElementById('poLinesBody');
  const dlId = 'poItemSuggestions_' + (_poLineCounter++);
  const dl = document.createElement('datalist');
  dl.id = dlId;
  document.body.appendChild(dl);

  const row = document.createElement('tr');
  row.className = 'po-line-row';
  row.dataset.dlId = dlId;
  row.innerHTML = `
    <td class="ps-3">
      <button type="button" class="btn btn-sm btn-outline-danger" onclick="poRemoveLine(this)">
        <i class="bi bi-x"></i>
      </button>
    </td>
    <td>
      <select class="form-select form-select-sm po-cat" onchange="poOnCategoryChange(this)">
        <option value="">— All —</option>
        ${(_poCategories||[]).map(c=>`<option value="${c.itms_grp_cod}">${c.itms_grp_nam}</option>`).join('')}
      </select>
    </td>
    <td>
      <input type="text" class="form-control form-control-sm po-item" placeholder="Item description"
        oninput="poSuggestItem(this)" list="${dlId}">
      <input type="hidden" class="po-iuom" value="0">
      <input type="hidden" class="po-item-code" value="">
    </td>
    <td>
      <select class="form-select form-select-sm text-center po-unit" onchange="poOnUnitChange(this)">
        <option value="">— Unit —</option>
      </select>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end po-qty" min="0.001" step="0.001"
        value="0" oninput="poRecalcLine(this)">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end po-price" min="0" step="0.01"
        value="0" oninput="poRecalcLine(this)">
    </td>
    <td class="text-end pe-3 fw-bold text-primary po-line-total">₱0.00</td>
  `;
  tbody.appendChild(row);
}


function poOnCategoryChange(sel) {
  const row = sel.closest('tr');
  const cat = sel.value;
  const unitInput = row.querySelector('.po-unit');
  const priceInput = row.querySelector('.po-price');
  if (!_poItemMaster) return;

  const listEl = document.getElementById(row.dataset.dlId);
  if (listEl) {
    const pool = cat
      ? _poItemMaster.filter(i => String(i.itms_grp_cod) === cat)
      : _poItemMaster;
    listEl.innerHTML = pool.map(i =>
      `<option value="${i.item_name}" data-code="${i.item_code}" data-unit="${i.invntry_uom}" data-price="${i.avg_price}">`
    ).join('');
  }

  const itemInput = row.querySelector('.po-item');
  if (itemInput) itemInput.value = '';

  const first = cat ? _poItemMaster.find(i => String(i.itms_grp_cod) === cat) : null;
  if (first) {
    poPopulateUnitDropdown(row, first);
    if (priceInput && !parseFloat(priceInput.value)) priceInput.value = first.avg_price ?? 0;
  }
}


function poSuggestItem(input) {
  const row = input.closest('tr');
  const cat = row.querySelector('.po-cat')?.value || '';
  const term = input.value.toLowerCase();
  const listEl = document.getElementById(row.dataset.dlId);
  if (!listEl || !_poItemMaster) return;

  const pool = cat
    ? _poItemMaster.filter(i => String(i.itms_grp_cod) === cat)
    : _poItemMaster;

  const matches = term
    ? pool.filter(i =>
        (i.item_name || '').toLowerCase().includes(term) ||
        (i.item_code || '').toLowerCase().includes(term)
      ).slice(0, 10)
    : pool.slice(0, 10);

  listEl.innerHTML = matches.map(i =>
    `<option value="${i.item_name}" data-code="${i.item_code}" data-unit="${i.invntry_uom}" data-price="${i.avg_price}">`
  ).join('');

  const exact = (_poItemMaster || []).find(
    i => (i.item_name || '').toLowerCase() === term
  );
  if (exact) {
    poPopulateUnitDropdown(row, exact);
    const codeEl = row.querySelector('.po-item-code');
    if (codeEl) codeEl.value = exact.item_code || '';
  }
}


function poOnUnitChange(sel) {
  const row      = sel.closest('tr');
  const itemName = row.querySelector('.po-item')?.value;
  const item     = (_poItemMaster || []).find(i => i.item_name === itemName);
  const price    = resolveUoMPrice(item, sel.value, _poUoMGroups);
  if (price !== null) {
    const priceEl = row.querySelector('.po-price');
    if (priceEl) { priceEl.value = price.toFixed(2); poRecalcLine(priceEl); }
  }
  // Sync po-iuom hidden field to selected unit's entry for DR conversion
  const iuomEl = row.querySelector('.po-iuom');
  if (iuomEl) iuomEl.value = sel.selectedOptions[0]?.dataset.uomEntry || 0;
}

// Populates the unit dropdown for a PO line row.
// Shows all UoMs in the item's UoM group (or falls back to the item's purchase/inventory unit string).
// Embeds data-uom-entry on each option for DR conversion. Syncs the hidden .po-iuom field.
function poPopulateUnitDropdown(row, item) {
  const sel = row.querySelector('.po-unit');
  if (!sel) return;
  const units = item?.ugp_entry
    ? getUoMGroupUnits(item.ugp_entry, _poUoMGroups)
    : [];

  const defaultCode = item?.purch_uom?.uom_code || item?.invntry_uom || '';
  if (units.length === 0 && defaultCode) {
    units.push({ code: defaultCode, name: defaultCode, entry: item?.p_uom_entry || item?.i_uom_entry || 0 });
  }
  sel.innerHTML = units.map(u =>
    `<option value="${u.code}" data-uom-entry="${u.entry || 0}"${u.code === defaultCode ? ' selected' : ''}>${u.name || u.code}</option>`
  ).join('');

  // Sync the hidden po-iuom field to the selected unit's entry (used for DR conversion)
  const selected = sel.selectedOptions[0];
  const iuomEl = row.querySelector('.po-iuom');
  if (iuomEl) iuomEl.value = selected?.dataset.uomEntry || 0;
}


function poRemoveLine(btn) {
  const row = btn.closest('tr');
  if (row) {
    const dl = document.getElementById(row.dataset.dlId);
    if (dl) dl.remove();
    row.remove();
  }
  poRecalcGrandTotal();
}


function poRecalcLine(input) {
  const row = input.closest('tr');
  const qty = parseFloat(row.querySelector('.po-qty').value) || 0;
  const price = parseFloat(row.querySelector('.po-price').value) || 0;
  const total = qty * price;
  row.querySelector('.po-line-total').textContent = formatCurrency(total);
  poRecalcGrandTotal();
}


function poRecalcGrandTotal() {
  let grand = 0;
  document.querySelectorAll('.po-line-row').forEach(row => {
    const qty = parseFloat(row.querySelector('.po-qty').value) || 0;
    const price = parseFloat(row.querySelector('.po-price').value) || 0;
    grand += qty * price;
  });
  const el = document.getElementById('poGrandTotal');
  if (el) el.textContent = formatCurrency(grand);
}


async function submitNewPO() {
  const dateVal = document.getElementById('poDate').value;
  if (!dateVal) { toast('Date is required.', 'warning'); return; }

  const rows = Array.from(document.querySelectorAll('.po-line-row'));
  const lines = [];
  rows.forEach(row => {
    const item = row.querySelector('.po-item').value.trim();
    const qty = parseFloat(row.querySelector('.po-qty').value) || 0;
    const price = parseFloat(row.querySelector('.po-price').value) || 0;
    if (!item || qty <= 0) return;
    lines.push({
      item_code:   row.querySelector('.po-item-code').value.trim(),
      category:    row.querySelector('.po-cat').value,
      item_name:   item,
      unit:        row.querySelector('.po-unit').value.trim(),
      i_uom_entry: parseInt(row.querySelector('.po-iuom').value) || 0,
      quantity:    qty,
      unit_price:  price,
    });
  });

  if (lines.length === 0) { toast('Add at least one line item.', 'warning'); return; }

  const supplierIDRaw = document.getElementById('poSupplierID').value;
  const payload = {
    date:           new Date(dateVal).toISOString(),
    supplier_id:    supplierIDRaw ? parseInt(supplierIDRaw, 10) : null,
    supplier:       document.getElementById('poSupplierName').value.trim(),
    payment_method: document.getElementById('poPaymentMethod').value,
    remarks:        document.getElementById('poRemarks').value.trim(),
    po_number:      document.getElementById('poPONumber').value.trim(),
    lines,
  };

  const btn = document.getElementById('poSaveBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';
  const result = await api.CreatePurchaseHeader(payload);
  if (result) {
    navigate('#/purchasing/purchases');
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-check-lg me-1"></i>Save Purchase Order';
  }
}


async function loadPurchaseDetail(id) {
  showLoading();
  const p = await api.GetPurchaseHeader(id);
  if (!p) {
    showView(`<div class="alert alert-warning m-4">Purchase order not found.</div>`);
    return;
  }
  _poDetail = p;
  const draftDR = (p.delivery_receipts || []).find(dr => dr.status === 'Draft');

  // PO Lines
  const lineRows = (p.lines || []).map(l => `
    <tr>
      <td>${l.category || '—'}</td>
      <td>${l.item_name || '—'}</td>
      <td>${l.unit || '—'}</td>
      <td class="text-end">${formatNumber(l.quantity ?? 0)}</td>
      <td class="text-end ${l.open_qty > 0 ? 'text-success fw-semibold' : 'text-muted'}">${formatNumber(l.open_qty ?? 0)}</td>
      <td class="text-end">${formatCurrency(l.unit_price ?? 0)}</td>
      <td class="text-end fw-semibold">${formatCurrency(l.line_total ?? 0)}</td>
      <td>${l.line_status === 'C' ? '<span class="badge bg-success">Closed</span>' : '<span class="badge bg-primary">Open</span>'}</td>
    </tr>
  `).join('') || `<tr><td colspan="8" class="text-muted text-center py-3">No lines.</td></tr>`;

  // Delivery Receipts
  const drs = (p.delivery_receipts || []).map(dr => `
    <tr style="cursor:pointer" onclick="navigate('#/purchasing/delivery-receipts/${dr.id}')">
      <td class="fw-semibold text-primary">${dr.dr_number || '—'}</td>
      <td>${formatDate(dr.posting_date)}</td>
      <td>${purchStatusBadge(dr.status)}</td>
      <td class="text-end">${formatCurrency(dr.doc_total ?? 0)}</td>
    </tr>
  `).join('') || `<tr><td colspan="4" class="text-muted text-center py-3">No delivery receipts.</td></tr>`;

  // AP Invoices
  const aps = (p.ap_invoices || []).map(inv => `
    <tr style="cursor:pointer" onclick="navigate('#/purchasing/ap-invoices/${inv.id}')">
      <td class="fw-semibold text-primary">${inv.invoice_number || '—'}</td>
      <td>${formatDate(inv.posting_date)}</td>
      <td>${purchStatusBadge(inv.status)}</td>
      <td class="text-end">${formatCurrency(inv.doc_total ?? 0)}</td>
      <td class="text-end">${formatCurrency(inv.amount_paid_stored ?? 0)}</td>
    </tr>
  `).join('') || `<tr><td colspan="5" class="text-muted text-center py-3">No AP invoices.</td></tr>`;

  const canCancel = p.status !== 'Cancelled' && p.status !== 'Closed' &&
    !(p.delivery_receipts || []).some(dr => dr.status === 'Received');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center gap-2 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/purchases')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">Purchase Order</h4>
        <span class="badge bg-secondary ms-1">${p.po_number || '—'}</span>
        ${purchStatusBadge(p.status)}
        <button class="btn btn-sm btn-outline-secondary ms-auto" onclick="printPO()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
        <button class="btn btn-sm btn-outline-info" onclick="showRelationshipMap('purchasing',${p.id},'PO',${p.id})">
          <i class="bi bi-diagram-3 me-1"></i>Document Flow
        </button>
        ${canCancel ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelPurchaseOrder(${p.id})">
          <i class="bi bi-x-circle me-1"></i>Cancel PO</button>` : ''}
      </div>

      <!-- Header info -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-body">
          <div class="row g-3 small">
            <div class="col-md-3"><span class="text-muted">Date</span><div class="fw-semibold">${formatDate(p.posting_date)}</div></div>
            <div class="col-md-3"><span class="text-muted">Supplier</span><div class="fw-semibold">${p.supplier_name || '—'}</div></div>
            <div class="col-md-3"><span class="text-muted">Payment Method</span><div>${p.payment_method || '—'}</div></div>
            <div class="col-md-3"><span class="text-muted">PO Total</span><div class="fw-semibold text-primary fs-5">${formatCurrency(p.doc_total ?? 0)}</div></div>
            ${p.remarks ? `<div class="col-12"><span class="text-muted">Remarks</span><div>${p.remarks}</div></div>` : ''}
          </div>
        </div>
      </div>

      <!-- PO Lines -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-header bg-transparent fw-semibold">
          <i class="bi bi-list-ul me-2"></i>Order Lines
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0 small">
            <thead class="table-light">
              <tr>
                <th>Category</th><th>Item</th><th>Unit</th>
                <th class="text-end">Ordered</th>
                <th class="text-end">Open Qty</th>
                <th class="text-end">Unit Price</th>
                <th class="text-end">Line Total</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>${lineRows}</tbody>
          </table>
        </div>
      </div>

      <!-- Delivery Receipts -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-header bg-transparent d-flex justify-content-between align-items-center fw-semibold">
          <span><i class="bi bi-truck me-2"></i>Delivery Receipts</span>
          ${draftDR
            ? `<button class="btn btn-sm btn-warning" onclick="navigate('#/purchasing/delivery-receipts/${draftDR.id}')">
                 <i class="bi bi-truck me-1"></i>View Draft DR (${draftDR.dr_number})
               </button>`
            : `<button class="btn btn-sm btn-outline-primary" onclick="createDRFromPO(${p.id})">
                 <i class="bi bi-plus-lg me-1"></i>Create Delivery Receipt
               </button>`}
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0 small">
            <thead class="table-light">
              <tr><th>DR #</th><th>Date</th><th>Status</th><th class="text-end">Amount</th></tr>
            </thead>
            <tbody>${drs}</tbody>
          </table>
        </div>
      </div>

      <!-- AP Invoices -->
      <div class="card border-0 shadow-sm">
        <div class="card-header bg-transparent fw-semibold">
          <i class="bi bi-file-earmark-text me-2"></i>AP Invoices
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0 small">
            <thead class="table-light">
              <tr><th>Invoice #</th><th>Date</th><th>Status</th><th class="text-end">Total</th><th class="text-end">Paid</th></tr>
            </thead>
            <tbody>${aps}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}


// "Copy To" shortcut — equivalent to SAP B1's Copy To → Goods Receipt PO.
// Stores the source PO id so the New DR form can auto-select supplier + load open lines.
function createDRFromPO(poId) {
  const existingDraft = (_poDetail && _poDetail.delivery_receipts || []).find(dr => dr.status === 'Draft');
  if (existingDraft) {
    toast(`Draft receipt ${existingDraft.dr_number} already exists — confirm or cancel it first.`, 'warning', 8000);
    navigate(`#/purchasing/delivery-receipts/${existingDraft.id}`);
    return;
  }
  window._drPrefillPOId = poId;
  navigate('#/purchasing/delivery-receipts/new');
}


function printPO() {
  const p = _poDetail;
  if (!p) return;
  const f = window._pfmt;

  const lineRows = (p.lines || []).map(l => `
    <tr>
      <td>${l.item_name || '—'}</td>
      <td>${l.category || '—'}</td>
      <td>${l.unit || '—'}</td>
      <td class="text-end">${f.num(l.quantity)}</td>
      <td class="text-end">${f.currency(l.unit_price)}</td>
      <td class="text-end">${f.currency(l.line_total)}</td>
    </tr>
  `).join('');

  const html = `
    <h2>PURCHASE ORDER</h2>
    <div class="sub">${p.po_number || '—'} &nbsp;·&nbsp; <span class="badge">${p.status || '—'}</span> &nbsp;·&nbsp; ${f.date(p.posting_date)}</div>
    <div class="info-grid">
      <div><span>Supplier</span><br>${p.supplier_name || '—'}</div>
      <div><span>Payment Method</span><br>${p.payment_method || '—'}</div>
      ${p.remarks ? `<div style="grid-column:1/-1"><span>Remarks</span><br>${p.remarks}</div>` : ''}
    </div>
    <table>
      <thead><tr><th>Item</th><th>Category</th><th>Unit</th><th class="text-end">Qty</th><th class="text-end">Unit Price</th><th class="text-end">Total</th></tr></thead>
      <tbody>${lineRows}</tbody>
    </table>
    <div class="totals">
      PO Total: <strong>${f.currency(p.doc_total)}</strong>
    </div>`;
  _printDoc('PO ' + (p.po_number || ''), html);
}

function _resetPurchasingPO() {
  _poDetail = null;
  _poItemMaster = null; _poCategories = []; _poUoMGroups = []; _poLineCounter = 0;
  window._drPrefillPOId = null;
}
