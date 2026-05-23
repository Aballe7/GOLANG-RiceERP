// inventory-gi.js — Goods Issue (OIGE/IGE1) views

// ── Helpers ───────────────────────────────────────────────────────────────────

function giStatusBadge(status) { return grStatusBadge(status); }
function wrapGI(tabBar, inner)  { return wrapInv(tabBar, inner); }

let _giDetail = null;
let _giItemMaster = null, _giCategories = [], _giWarehouses = [], _giLineCounter = 0;

// ── List View ─────────────────────────────────────────────────────────────────

async function loadGoodsIssueList(tabBar) {
  sessionStorage.removeItem('docReturnRoute');
  const list = await api.ListGoodsIssues();
  if (!list) {
    showView(wrapGI(tabBar, `<div class="alert alert-warning">Failed to load goods issues.</div>`));
    return;
  }

  const rows = list.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No goods issues found.</td></tr>`
    : list.map(gi => `
        <tr style="cursor:pointer" onclick="navigate('#/inventory/gi/${gi.id}')">
          <td class="fw-semibold">${gi.gi_number || '—'}</td>
          <td>${formatDate(gi.posting_date)}</td>
          <td>${gi.doc_due_date ? formatDate(gi.doc_due_date) : '—'}</td>
          <td class="text-end">${formatCurrency(gi.doc_total ?? 0)}</td>
          <td>${giStatusBadge(gi.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="event.stopPropagation();navigate('#/inventory/gi/${gi.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapGI(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small">${list.length} record(s)</span>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/inventory/gi/new')">
        <i class="bi bi-plus-lg me-1"></i>New Goods Issue
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr>
              <th>GI Number</th>
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

async function loadNewGIForm(tabBar) {
  const [items, cats, whs] = await Promise.all([api.ListInventoryItems(), api.ListItemCategoriesForModule('inventory'), api.ListWarehouses()]);
  _giItemMaster = items || [];
  _giCategories = cats  || [];
  _giWarehouses = (whs || []).filter(w => w.inactive !== 'Y');

  const today = new Date().toISOString().slice(0, 10);

  showView(wrapGI(tabBar, `
    <div style="max-width:960px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/inventory/gi')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">New Goods Issue</h4>
      </div>

      <!-- Header -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">GI Number</label>
              <input type="text" class="form-control" id="giNumber" readonly placeholder="Auto-generated">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Posting Date <span class="text-danger">*</span></label>
              <input type="date" class="form-control" id="giPostingDate" required value="${today}">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" class="form-control" id="giDueDate">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Issue From Warehouse <span class="text-danger">*</span></label>
              <select class="form-select" id="giWarehouse" required>
                <option value="">— Select Warehouse —</option>
                ${(_giWarehouses||[]).map(w => `<option value="${w.whs_code}">${w.whs_name} (${w.whs_code})</option>`).join('')}
              </select>
            </div>
            <div class="col-md-12">
              <label class="form-label small fw-bold">Remarks</label>
              <textarea class="form-control" id="giRemarks" rows="2"
                placeholder="e.g. Internal use, Breakage, Sample…"></textarea>
            </div>
          </div>
        </div>
      </div>

      <!-- Line Items -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-box-seam me-2"></i>Line Items</span>
          <button class="btn btn-sm btn-outline-primary" type="button" onclick="giAddLine()">
            <i class="bi bi-plus-lg me-1"></i>Add Line
          </button>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0" id="giLinesTable">
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
            <tbody id="giLinesBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="6" class="text-end pe-3">Doc Total</td>
                <td class="text-end pe-3 text-primary fs-5" id="giGrandTotal">₱0.00</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 mt-3 justify-content-end">
        <button type="button" class="btn btn-outline-secondary"
          onclick="navigate('#/inventory/gi')">Cancel</button>
        <button type="button" class="btn btn-primary px-4" id="giSaveBtn" onclick="submitNewGI()">
          <i class="bi bi-check-lg me-1"></i>Save Goods Issue
        </button>
      </div>
    </div>
  `));

  _giLineCounter = 0;
  giAddLine(); // start with one empty line
}

function giAddLine() {
  const tbody = document.getElementById('giLinesBody');
  if (!tbody) return;
  const dlId = 'giItemSuggestions_' + (_giLineCounter++);
  const dl = document.createElement('datalist');
  dl.id = dlId;
  document.body.appendChild(dl);

  const row = document.createElement('tr');
  row.className = 'gi-line-row';
  row.dataset.dlId = dlId;
  row.innerHTML = `
    <td class="ps-3">
      <button type="button" class="btn btn-sm btn-outline-danger" onclick="giRemoveLine(this)">
        <i class="bi bi-x"></i>
      </button>
    </td>
    <td>
      <select class="form-select form-select-sm gi-cat" onchange="giOnCategoryChange(this)">
        <option value="">— All —</option>
        ${(_giCategories||[]).map(c=>`<option value="${c.itms_grp_cod}">${c.itms_grp_nam}</option>`).join('')}
      </select>
    </td>
    <td>
      <input type="text" class="form-control form-control-sm gi-item" placeholder="Type to search item…"
        oninput="giSuggestItem(this)" onchange="giOnItemSelect(this)" list="${dlId}">
    </td>
    <td>
      <input type="text" class="form-control form-control-sm text-center gi-unit" placeholder="kg…" readonly>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end gi-qty" min="0.001" step="0.001"
        value="1" oninput="giRecalcLine(this)">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end gi-price" min="0" step="0.01"
        value="0" oninput="giRecalcLine(this)">
    </td>
    <td class="text-end pe-3 fw-bold text-primary gi-line-total">₱0.00</td>
  `;
  row.dataset.itemId = '0';
  row.dataset.uomEntry = '0';
  tbody.appendChild(row);
  giRecalcGrandTotal();
}

function giOnCategoryChange(sel) {
  const row = sel.closest('tr');
  if (!_giItemMaster) return;
  const cat = sel.value;
  const listEl = document.getElementById(row.dataset.dlId);
  if (listEl) {
    const pool = cat ? _giItemMaster.filter(i => String(i.itms_grp_cod) === cat) : _giItemMaster;
    listEl.innerHTML = pool.map(i =>
      `<option value="${i.item_name}" data-id="${i.id}" data-code="${i.item_code}" data-unit="${i.invntry_uom}" data-price="${i.avg_price}" data-uom-entry="${i.i_uom_entry || 0}">`
    ).join('');
  }
  const itemInput = row.querySelector('.gi-item');
  if (itemInput) itemInput.value = '';
  row.dataset.itemId = '0';
}

function giSuggestItem(input) {
  const row = input.closest('tr');
  const cat = row.querySelector('.gi-cat')?.value || '';
  const term = input.value.toLowerCase();
  const listEl = document.getElementById(row.dataset.dlId);
  if (!listEl || !_giItemMaster) return;
  const pool = cat ? _giItemMaster.filter(i => String(i.itms_grp_cod) === cat) : _giItemMaster;
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

function giOnItemSelect(input) {
  const row = input.closest('tr');
  const listEl = document.getElementById(row.dataset.dlId);
  if (!listEl) return;
  const opt = Array.from(listEl.options).find(o => o.value === input.value);
  if (!opt) return;
  row.dataset.itemId = opt.dataset.id || '0';
  row.dataset.uomEntry = opt.dataset.uomEntry || '0';
  const unitInput = row.querySelector('.gi-unit');
  const priceInput = row.querySelector('.gi-price');
  if (unitInput) unitInput.value = opt.dataset.unit || '';
  if (priceInput && parseFloat(priceInput.value) === 0) priceInput.value = opt.dataset.price || '0';
  giRecalcLine(input);
}

function giRemoveLine(btn) {
  const row = btn.closest('tr');
  if (row) {
    const dl = document.getElementById(row.dataset.dlId);
    if (dl) dl.remove();
    row.remove();
  }
  giRecalcGrandTotal();
}

function giRecalcLine(input) {
  const row = input.closest('tr');
  const qty = parseFloat(row.querySelector('.gi-qty').value) || 0;
  const price = parseFloat(row.querySelector('.gi-price').value) || 0;
  row.querySelector('.gi-line-total').textContent = formatCurrency(qty * price);
  giRecalcGrandTotal();
}

function giRecalcGrandTotal() {
  let total = 0;
  document.querySelectorAll('#giLinesBody .gi-line-row').forEach(row => {
    const qty = parseFloat(row.querySelector('.gi-qty').value) || 0;
    const price = parseFloat(row.querySelector('.gi-price').value) || 0;
    total += qty * price;
  });
  const el = document.getElementById('giGrandTotal');
  if (el) el.textContent = formatCurrency(total);
}

async function submitNewGI() {
  const postingDate = document.getElementById('giPostingDate')?.value;
  const dueDate = document.getElementById('giDueDate')?.value || '';
  const remarks = document.getElementById('giRemarks')?.value || '';
  const warehouseCode = document.getElementById('giWarehouse')?.value || '';

  if (!postingDate) { alert('Posting Date is required.'); return; }
  if (!warehouseCode) { alert('Please select a warehouse.'); return; }

  const lines = [];
  let hasError = false;
  document.querySelectorAll('#giLinesBody .gi-line-row').forEach((row) => {
    const itemId = parseInt(row.dataset.itemId || '0', 10);
    const qty = parseFloat(row.querySelector('.gi-qty').value) || 0;
    const price = parseFloat(row.querySelector('.gi-price').value) || 0;
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

  const btn = document.getElementById('giSaveBtn');
  if (btn) { btn.disabled = true; btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…'; }

  const result = await api.CreateGoodsIssue({
    posting_date: postingDate,
    doc_due_date: dueDate,
    remarks:      remarks,
    lines:        lines,
  });

  if (btn) { btn.disabled = false; btn.innerHTML = '<i class="bi bi-check-lg me-1"></i>Save Goods Issue'; }

  if (result) {
    // Same-hash reload safety: call load directly
    Modules.Inventory.load('gi');
  }
}

// ── Detail View ───────────────────────────────────────────────────────────────

async function viewGI(id, tabBar) {
  const gi = await api.GetGoodsIssue(id);
  _giDetail = gi || null;
  if (!gi) {
    if (tabBar) showView(wrapGI(tabBar, `<div class="alert alert-danger">Goods issue not found.</div>`));
    return;
  }

  const lineRows = (gi.lines || []).length === 0
    ? `<tr><td colspan="7" class="text-center text-muted py-3">No lines.</td></tr>`
    : (gi.lines || []).map((line) => `
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

  const cancelBtn = gi.status === 'Open'
    ? `<button class="btn btn-outline-danger btn-sm" onclick="cancelGI(${gi.id})">
         <i class="bi bi-x-circle me-1"></i>Cancel GI
       </button>`
    : '';

  const content = `
    <div style="max-width:960px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate(sessionStorage.getItem('docReturnRoute')||'#/inventory/gi');sessionStorage.removeItem('docReturnRoute')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">Goods Issue — ${gi.gi_number}</h4>
        <div class="ms-2">${giStatusBadge(gi.status)}</div>
        <button class="btn btn-sm btn-outline-secondary ms-auto" onclick="printGI(${gi.id})">
          <i class="bi bi-printer me-1"></i>Print
        </button>
      </div>

      <!-- Header Info -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">GI Number</div>
              <div class="fw-semibold">${gi.gi_number}</div>
            </div>
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">Posting Date</div>
              <div class="fw-semibold">${formatDate(gi.posting_date)}</div>
            </div>
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">Due Date</div>
              <div class="fw-semibold">${gi.doc_due_date ? formatDate(gi.doc_due_date) : '—'}</div>
            </div>
            <div class="col-md-3">
              <div class="text-muted small fw-bold mb-1">Doc Total</div>
              <div class="fw-semibold text-primary fs-5">${formatCurrency(gi.doc_total ?? 0)}</div>
            </div>
            ${gi.remarks ? `
            <div class="col-md-12">
              <div class="text-muted small fw-bold mb-1">Remarks</div>
              <div>${gi.remarks}</div>
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
                <td class="text-end pe-3 text-primary">${formatCurrency(gi.doc_total ?? 0)}</td>
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
    showView(wrapGI(tabBar, content));
  } else {
    showView(`<div class="container-fluid p-4">${content}</div>`);
  }
}

async function printGI(id) {
  const gi = (_giDetail && _giDetail.id === id) ? _giDetail : await api.GetGoodsIssue(id);
  if (!gi) return;
  const lineRows = (gi.lines || []).map((l, i) => `<tr>
    <td>${i + 1}</td>
    <td>${l.item_code || '—'}</td>
    <td>${l.description || '—'}</td>
    <td class="text-end">${_pfmt.num(l.quantity)}</td>
    <td>${l.unit || '—'}</td>
    <td class="text-end">${_pfmt.currency(l.price)}</td>
    <td class="text-end">${_pfmt.currency(l.line_total)}</td>
  </tr>`).join('') || `<tr><td colspan="7" class="text-center">No lines.</td></tr>`;

  const html = `
    <h2>Goods Issue</h2>
    <div class="sub">${gi.gi_number} &nbsp;·&nbsp; ${_pfmt.date(gi.posting_date)}</div>
    <div class="info-grid">
      <div><span>GI Number</span><br>${gi.gi_number}</div>
      <div><span>Posting Date</span><br>${_pfmt.date(gi.posting_date)}</div>
      <div><span>Due Date</span><br>${gi.doc_due_date ? _pfmt.date(gi.doc_due_date) : '—'}</div>
      <div><span>Status</span><br>${gi.status || '—'}</div>
      ${gi.remarks ? `<div><span>Remarks</span><br>${gi.remarks}</div>` : ''}
    </div>
    <div class="section-title">Line Items</div>
    <table>
      <thead><tr><th>#</th><th>Item Code</th><th>Description</th><th class="text-end">Qty</th><th>Unit</th><th class="text-end">Price</th><th class="text-end">Line Total</th></tr></thead>
      <tbody>${lineRows}</tbody>
    </table>
    <div class="totals"><strong>Doc Total: ${_pfmt.currency(gi.doc_total ?? 0)}</strong></div>`;

  _printDoc('GI ' + gi.gi_number, html);
}

async function cancelGI(id) {
  if (!confirm('Are you sure you want to cancel this Goods Issue? This will restore the stock decrease.')) return;
  const ok = await api.CancelGoodsIssue(id);
  if (ok !== null) {
    // reload detail
    const tabBar = document.getElementById('invTabs')?.outerHTML || '';
    await viewGI(id, tabBar || undefined);
  }
}

function _resetInventoryGI() {
  _giDetail = null;
  _giItemMaster = null; _giCategories = []; _giWarehouses = []; _giLineCounter = 0;
}
