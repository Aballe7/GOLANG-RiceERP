// inventory-gr.js — Goods Receipt (OIGN/IGN1) views

// ── Helpers ───────────────────────────────────────────────────────────────────

function grStatusBadge(status) {
  switch (status) {
    case 'Open':      return `<span class="badge bg-success">Open</span>`;
    case 'Cancelled': return `<span class="badge bg-secondary">Cancelled</span>`;
    default:          return `<span class="badge bg-light text-dark">${status || '—'}</span>`;
  }
}

function wrapInv(tabBar, inner) {
  return `<div class="container-fluid p-4">${tabBar}${inner}</div>`;
}

let _grDetail = null;
let _grItemMaster = null, _grCategories = [], _grWarehouses = [], _grLineCounter = 0;

// ── List View ─────────────────────────────────────────────────────────────────

async function loadGoodsReceiptList(tabBar) {
  sessionStorage.removeItem('docReturnRoute');
  const list = await api.ListGoodsReceipts();
  if (!list) {
    showView(wrapInv(tabBar, `<div class="alert alert-warning">Failed to load goods receipts.</div>`));
    return;
  }

  const rows = list.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No goods receipts found.</td></tr>`
    : list.map(gr => `
        <tr style="cursor:pointer" onclick="navigate('#/inventory/gr/${gr.id}')">
          <td class="fw-semibold">${gr.gr_number || '—'}</td>
          <td>${formatDate(gr.posting_date)}</td>
          <td>${gr.doc_due_date ? formatDate(gr.doc_due_date) : '—'}</td>
          <td class="text-end">${formatCurrency(gr.doc_total ?? 0)}</td>
          <td>${grStatusBadge(gr.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="event.stopPropagation();navigate('#/inventory/gr/${gr.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapInv(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small">${list.length} record(s)</span>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/inventory/gr/new')">
        <i class="bi bi-plus-lg me-1"></i>New Goods Receipt
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr>
              <th>GR Number</th>
              <th>Posting Date</th>
              <th>Due Date</th>
              <th class="text-end">Doc Total</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `));
}

// ── New Form ──────────────────────────────────────────────────────────────────

async function loadNewGRForm(tabBar) {
  const [items, cats, whs] = await Promise.all([api.ListInventoryItems(), api.ListItemCategoriesForModule('inventory'), api.ListWarehouses()]);
  _grItemMaster = items || [];
  _grCategories = cats  || [];
  _grWarehouses = (whs || []).filter(w => w.inactive !== 'Y');

  const today = new Date().toISOString().slice(0, 10);

  showView(wrapInv(tabBar, `
    <div style="max-width:960px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/inventory/gr')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">New Goods Receipt</h4>
      </div>

      <!-- Header -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">GR Number</label>
              <input type="text" class="form-control" id="grNumber" readonly placeholder="Auto-generated">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Posting Date <span class="text-danger">*</span></label>
              <input type="date" class="form-control" id="grPostingDate" required value="${today}">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" class="form-control" id="grDueDate">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Receive Into Warehouse <span class="text-danger">*</span></label>
              <select class="form-select" id="grWarehouse" required>
                <option value="">— Select Warehouse —</option>
                ${(_grWarehouses||[]).map(w => `<option value="${w.whs_code}">${w.whs_name} (${w.whs_code})</option>`).join('')}
              </select>
            </div>
            <div class="col-md-12">
              <label class="form-label small fw-bold">Remarks</label>
              <textarea class="form-control" id="grRemarks" rows="2"
                placeholder="e.g. Opening balance, Stock correction…"></textarea>
            </div>
          </div>
        </div>
      </div>

      <!-- Line Items -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-box-seam me-2"></i>Line Items</span>
          <button class="btn btn-sm btn-outline-primary" type="button" onclick="grAddLine()">
            <i class="bi bi-plus-lg me-1"></i>Add Line
          </button>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0" id="grLinesTable">
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
            <tbody id="grLinesBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="6" class="text-end pe-3">Doc Total</td>
                <td class="text-end pe-3 text-primary fs-5" id="grGrandTotal">₱0.00</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 mt-3 justify-content-end">
        <button type="button" class="btn btn-outline-secondary"
          onclick="navigate('#/inventory/gr')">Cancel</button>
        <button type="button" class="btn btn-primary px-4" id="grSaveBtn" onclick="submitNewGR()">
          <i class="bi bi-check-lg me-1"></i>Save Goods Receipt
        </button>
      </div>
    </div>
  `));

  _grLineCounter = 0;
  grAddLine(); // start with one empty line
}


function grAddLine() {
  const tbody = document.getElementById('grLinesBody');
  if (!tbody) return;
  const dlId = 'grItemSuggestions_' + (_grLineCounter++);
  const dl = document.createElement('datalist');
  dl.id = dlId;
  document.body.appendChild(dl);

  const row = document.createElement('tr');
  row.className = 'gr-line-row';
  row.dataset.dlId = dlId;
  row.innerHTML = `
    <td class="ps-3">
      <button type="button" class="btn btn-sm btn-outline-danger" onclick="grRemoveLine(this)">
        <i class="bi bi-x"></i>
      </button>
    </td>
    <td>
      <select class="form-select form-select-sm gr-cat" onchange="grOnCategoryChange(this)">
        <option value="">— All —</option>
        ${(_grCategories||[]).map(c=>`<option value="${c.itms_grp_cod}">${c.itms_grp_nam}</option>`).join('')}
      </select>
    </td>
    <td>
      <input type="text" class="form-control form-control-sm gr-item" placeholder="Type to search item…"
        oninput="grSuggestItem(this)" onchange="grOnItemSelect(this)" list="${dlId}">
    </td>
    <td>
      <input type="text" class="form-control form-control-sm text-center gr-unit" placeholder="kg…" readonly>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end gr-qty" min="0.001" step="0.001"
        value="1" oninput="grRecalcLine(this)">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end gr-price" min="0" step="0.01"
        value="0" oninput="grRecalcLine(this)">
    </td>
    <td class="text-end pe-3 fw-bold text-primary gr-line-total">₱0.00</td>
  `;
  row.dataset.itemId = '0';
  row.dataset.uomEntry = '0';
  tbody.appendChild(row);
  grRecalcGrandTotal();
}

function grOnCategoryChange(sel) {
  const row = sel.closest('tr');
  if (!_grItemMaster) return;
  const cat = sel.value;
  const listEl = document.getElementById(row.dataset.dlId);
  if (listEl) {
    const pool = cat ? _grItemMaster.filter(i => String(i.itms_grp_cod) === cat) : _grItemMaster;
    listEl.innerHTML = pool.map(i =>
      `<option value="${i.item_name}" data-id="${i.id}" data-code="${i.item_code}" data-unit="${i.invntry_uom}" data-price="${i.avg_price}" data-uom-entry="${i.i_uom_entry || 0}">`
    ).join('');
  }
  const itemInput = row.querySelector('.gr-item');
  if (itemInput) itemInput.value = '';
  row.dataset.itemId = '0';
}

function grSuggestItem(input) {
  const row = input.closest('tr');
  const cat = row.querySelector('.gr-cat')?.value || '';
  const term = input.value.toLowerCase();
  const listEl = document.getElementById(row.dataset.dlId);
  if (!listEl || !_grItemMaster) return;
  const pool = cat ? _grItemMaster.filter(i => String(i.itms_grp_cod) === cat) : _grItemMaster;
  const matches = term
    ? pool.filter(i =>
        (i.item_name || '').toLowerCase().includes(term) ||
        (i.item_code || '').toLowerCase().includes(term)
      ).slice(0, 10)
    : pool.slice(0, 10);
  listEl.innerHTML = matches.map(i =>
    `<option value="${i.item_name}" data-id="${i.id}" data-code="${i.item_code}" data-unit="${i.invntry_uom}" data-price="${i.avg_price}" data-uom-entry="${i.i_uom_entry || 0}">`
  ).join('');
}

function grOnItemSelect(input) {
  // When the user picks a value from the datalist, auto-fill unit & price
  const row = input.closest('tr');
  const listEl = document.getElementById(row.dataset.dlId);
  if (!listEl) return;
  const opt = Array.from(listEl.options).find(o => o.value === input.value);
  if (!opt) return;
  row.dataset.itemId = opt.dataset.id || '0';
  row.dataset.uomEntry = opt.dataset.uomEntry || '0';
  const unitInput = row.querySelector('.gr-unit');
  const priceInput = row.querySelector('.gr-price');
  if (unitInput) unitInput.value = opt.dataset.unit || '';
  if (priceInput && parseFloat(priceInput.value) === 0) priceInput.value = opt.dataset.price || '0';
  grRecalcLine(input);
}

function grRemoveLine(btn) {
  const row = btn.closest('tr');
  if (row) {
    const dl = document.getElementById(row.dataset.dlId);
    if (dl) dl.remove();
    row.remove();
  }
  grRecalcGrandTotal();
}

function grRecalcLine(input) {
  const row = input.closest('tr');
  const qty = parseFloat(row.querySelector('.gr-qty').value) || 0;
  const price = parseFloat(row.querySelector('.gr-price').value) || 0;
  row.querySelector('.gr-line-total').textContent = formatCurrency(qty * price);
  grRecalcGrandTotal();
}

function grRecalcGrandTotal() {
  let total = 0;
  document.querySelectorAll('#grLinesBody .gr-line-row').forEach(row => {
    const qty = parseFloat(row.querySelector('.gr-qty').value) || 0;
    const price = parseFloat(row.querySelector('.gr-price').value) || 0;
    total += qty * price;
  });
  const el = document.getElementById('grGrandTotal');
  if (el) el.textContent = formatCurrency(total);
}

async function submitNewGR() {
  const postingDate = document.getElementById('grPostingDate')?.value;
  const dueDate = document.getElementById('grDueDate')?.value || '';
  const remarks = document.getElementById('grRemarks')?.value || '';
  const warehouseCode = document.getElementById('grWarehouse')?.value || '';

  if (!postingDate) { alert('Posting Date is required.'); return; }
  if (!warehouseCode) { alert('Please select a warehouse.'); return; }

  const lines = [];
  let hasError = false;
  document.querySelectorAll('#grLinesBody .gr-line-row').forEach((row, i) => {
    const itemId = parseInt(row.dataset.itemId || '0', 10);
    const qty = parseFloat(row.querySelector('.gr-qty').value) || 0;
    const price = parseFloat(row.querySelector('.gr-price').value) || 0;
    if (itemId === 0) { hasError = true; return; }
    if (qty <= 0) { hasError = true; return; }
    lines.push({
      item_id:        itemId,
      uom_entry:      parseInt(row.dataset.uomEntry || '0') || 0,
      quantity:       qty,
      price:          price,
      warehouse_code: warehouseCode,
      account_code:   '',
      project:        '',
    });
  });

  if (hasError || lines.length === 0) {
    alert('Please ensure all lines have a valid item and quantity > 0.');
    return;
  }

  const btn = document.getElementById('grSaveBtn');
  if (btn) { btn.disabled = true; btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…'; }

  const result = await api.CreateGoodsReceipt({
    posting_date: postingDate,
    doc_due_date: dueDate,
    remarks:      remarks,
    lines:        lines,
  });

  if (btn) { btn.disabled = false; btn.innerHTML = '<i class="bi bi-check-lg me-1"></i>Save Goods Receipt'; }

  if (result) {
    // Same-hash reload safety: call load directly
    const tabBarEl = document.getElementById('invTabs')?.outerHTML;
    Modules.Inventory.load('gr');
  }
}

// ── Detail View ───────────────────────────────────────────────────────────────

async function viewGR(id, tabBar) {
  const gr = await api.GetGoodsReceipt(id);
  _grDetail = gr || null;
  if (!gr) {
    if (tabBar) showView(wrapInv(tabBar, `<div class="alert alert-danger">Goods receipt not found.</div>`));
    return;
  }

  const lineRows = (gr.lines || []).length === 0
    ? `<tr><td colspan="7" class="text-center text-muted py-3">No lines.</td></tr>`
    : (gr.lines || []).map((line, i) => `
        <tr>
          <td class="text-muted small">${line.line_num + 1}</td>
          <td class="fw-semibold small">${line.item_code || '—'}</td>
          <td>${line.description || '—'}</td>
          <td class="text-center">${formatNumber(line.quantity)}</td>
          <td class="text-center text-muted small">${line.unit || '—'}</td>
          <td class="text-end">${formatCurrency(line.price)}</td>
          <td class="text-end fw-semibold">${formatCurrency(line.line_total)}</td>
        </tr>
      `).join('');

  const cancelBtn = gr.status === 'Open'
    ? `<button class="btn btn-outline-danger btn-sm" onclick="cancelGR(${gr.id})">
         <i class="bi bi-x-circle me-1"></i>Cancel GR
       </button>`
    : '';

  const content = `
    <div style="max-width:960px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate(sessionStorage.getItem('docReturnRoute')||'#/inventory/gr');sessionStorage.removeItem('docReturnRoute')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">Goods Receipt — ${gr.gr_number}</h4>
        <div class="ms-2">${grStatusBadge(gr.status)}</div>
        <button class="btn btn-sm btn-outline-secondary ms-auto" onclick="printGR(${gr.id})">
          <i class="bi bi-printer me-1"></i>Print
        </button>
      </div>

      <!-- Header Info -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">GR Number</div>
              <div class="fw-semibold">${gr.gr_number}</div>
            </div>
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">Posting Date</div>
              <div class="fw-semibold">${formatDate(gr.posting_date)}</div>
            </div>
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">Due Date</div>
              <div class="fw-semibold">${gr.doc_due_date ? formatDate(gr.doc_due_date) : '—'}</div>
            </div>
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">Doc Total</div>
              <div class="fw-semibold text-primary fs-5">${formatCurrency(gr.doc_total ?? 0)}</div>
            </div>
            ${gr.remarks ? `
            <div class="col-md-12">
              <div class="text-muted small fw-bold mb-1">Remarks</div>
              <div>${gr.remarks}</div>
            </div>` : ''}
          </div>
        </div>
      </div>

      <!-- Lines -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-light fw-bold py-2">
          <i class="bi bi-box-seam me-2"></i>Line Items
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>#</th>
                <th>Item Code</th>
                <th>Description</th>
                <th class="text-center">Qty</th>
                <th class="text-center">Unit</th>
                <th class="text-end">Price</th>
                <th class="text-end">Line Total</th>
              </tr>
            </thead>
            <tbody>${lineRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="6" class="text-end pe-3">Doc Total</td>
                <td class="text-end pe-3 text-primary">${formatCurrency(gr.doc_total ?? 0)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 mt-3">
        ${cancelBtn}
      </div>
    </div>
  `;

  if (tabBar) {
    showView(wrapInv(tabBar, content));
  } else {
    showView(`<div class="container-fluid p-4">${content}</div>`);
  }
}

async function printGR(id) {
  const gr = (_grDetail && _grDetail.id === id) ? _grDetail : await api.GetGoodsReceipt(id);
  if (!gr) return;
  const lineRows = (gr.lines || []).map((l, i) => `<tr>
    <td>${i + 1}</td>
    <td>${l.item_code || '—'}</td>
    <td>${l.description || '—'}</td>
    <td class="text-end">${_pfmt.num(l.quantity)}</td>
    <td>${l.unit || '—'}</td>
    <td class="text-end">${_pfmt.currency(l.price)}</td>
    <td class="text-end">${_pfmt.currency(l.line_total)}</td>
  </tr>`).join('') || `<tr><td colspan="7" class="text-center">No lines.</td></tr>`;

  const html = `
    <h2>Goods Receipt</h2>
    <div class="sub">${gr.gr_number} &nbsp;·&nbsp; ${_pfmt.date(gr.posting_date)}</div>
    <div class="info-grid">
      <div><span>GR Number</span><br>${gr.gr_number}</div>
      <div><span>Posting Date</span><br>${_pfmt.date(gr.posting_date)}</div>
      <div><span>Due Date</span><br>${gr.doc_due_date ? _pfmt.date(gr.doc_due_date) : '—'}</div>
      <div><span>Status</span><br>${gr.status || '—'}</div>
      ${gr.remarks ? `<div><span>Remarks</span><br>${gr.remarks}</div>` : ''}
    </div>
    <div class="section-title">Line Items</div>
    <table>
      <thead><tr><th>#</th><th>Item Code</th><th>Description</th><th class="text-end">Qty</th><th>Unit</th><th class="text-end">Price</th><th class="text-end">Line Total</th></tr></thead>
      <tbody>${lineRows}</tbody>
    </table>
    <div class="totals"><strong>Doc Total: ${_pfmt.currency(gr.doc_total ?? 0)}</strong></div>`;

  _printDoc('GR ' + gr.gr_number, html);
}

async function cancelGR(id) {
  if (!confirm('Are you sure you want to cancel this Goods Receipt? This will reverse the stock increase.')) return;
  const ok = await api.CancelGoodsReceipt(id);
  if (ok !== null) {
    // reload detail
    const tabBar = document.getElementById('invTabs')?.outerHTML || '';
    await viewGR(id, tabBar || undefined);
  }
}

function _resetInventoryGR() {
  _grDetail = null;
  _grItemMaster = null; _grCategories = []; _grWarehouses = []; _grLineCounter = 0;
}
