// production-milling.js — Milling Order list, detail, create, start, complete, cancel

// ── Status helpers ─────────────────────────────────────────────────────────────

const MO_STATUS_BADGE = {
  'Draft':       'secondary',
  'In Progress': 'warning',
  'Completed':   'success',
  'Cancelled':   'danger',
};

function moBadge(status) {
  const cls = MO_STATUS_BADGE[status] || 'secondary';
  return `<span class="badge bg-${cls}">${status}</span>`;
}

// ── List ───────────────────────────────────────────────────────────────────────

async function loadMillingOrdersList(tabBar) {
  const orders = await api.ListMillingOrders();
  if (!orders) { showView(`<div class="container-fluid p-4">${tabBar}<div class="alert alert-warning">Failed to load milling orders.</div></div>`); return; }

  const rows = orders.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No milling orders yet. <a href="#" onclick="navigate('#/production/milling-orders/new');return false;">Create one</a></td></tr>`
    : orders.map(mo => `
        <tr style="cursor:pointer" onclick="navigate('#/production/milling-orders/${mo.id}')">
          <td class="fw-semibold font-monospace small">${mo.mo_number}</td>
          <td class="text-muted small">${formatDate(mo.posting_date)}</td>
          <td>${mo.input_item_name || '—'}</td>
          <td class="text-end">${formatNumber(mo.input_qty, 3)} ${mo.input_unit || ''}</td>
          <td>${moBadge(mo.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="event.stopPropagation();navigate('#/production/milling-orders/${mo.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>`).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-3">
        <h4 class="fw-bold mb-0"><i class="bi bi-gear-wide-connected me-2 text-warning"></i>Milling Orders</h4>
        <button class="btn btn-warning" onclick="navigate('#/production/milling-orders/new')">
          <i class="bi bi-plus-lg me-1"></i>New Milling Order
        </button>
      </div>
      ${tabBar}
      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>MO #</th><th>Date</th><th>Input Item</th>
                <th class="text-end">Input Qty</th><th>Status</th><th></th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}

// ── Detail ─────────────────────────────────────────────────────────────────────

async function loadMillingOrderDetail(id) {
  const [mo, items] = await Promise.all([api.GetMillingOrder(id), api.ListProductionItems()]);
  if (!mo) { showView('<div class="alert alert-danger">Milling order not found.</div>'); return; }
  _moItemList = items || [];

  const lines = (mo.lines || []).map(l => `
    <tr>
      <td class="text-muted small font-monospace">${l.output_item_code || '—'}</td>
      <td class="fw-semibold">${l.output_item_name || '—'}</td>
      <td class="text-end fw-bold text-success">${formatNumber(l.actual_qty, 3)} ${l.unit || ''}</td>
    </tr>`).join('');

  const giLink = mo.goods_issue_number
    ? `<a href="#" onclick="navigate('#/inventory/gi/${mo.goods_issue_id}');return false;" class="text-warning fw-semibold">
        <i class="bi bi-arrow-up-circle me-1"></i>${mo.goods_issue_number}</a>`
    : '—';

  const grLink = mo.goods_receipt_number
    ? `<a href="#" onclick="navigate('#/inventory/gr/${mo.goods_receipt_id}');return false;" class="text-success fw-semibold">
        <i class="bi bi-arrow-down-circle me-1"></i>${mo.goods_receipt_number}</a>`
    : '—';

  // Action buttons based on status
  let actionButtons = '';
  if (mo.status === 'Draft') {
    actionButtons = `
      <button class="btn btn-warning me-2" onclick="startMillingOrder(${mo.id})">
        <i class="bi bi-play-fill me-1"></i>Start Milling
      </button>
      <button class="btn btn-outline-danger" onclick="cancelMillingOrder(${mo.id})">
        <i class="bi bi-x-circle me-1"></i>Cancel
      </button>`;
  } else if (mo.status === 'In Progress') {
    actionButtons = `
      <button class="btn btn-success me-2" onclick="openCompleteModal(${mo.id})">
        <i class="bi bi-check-circle me-1"></i>Complete
      </button>
      <button class="btn btn-outline-danger" onclick="cancelMillingOrder(${mo.id})">
        <i class="bi bi-x-circle me-1"></i>Cancel
      </button>`;
  }

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center gap-3 mb-3 flex-wrap">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/production/milling-orders')">
          <i class="bi bi-arrow-left me-1"></i>Back
        </button>
        <h4 class="fw-bold mb-0"><i class="bi bi-gear-wide-connected me-2 text-warning"></i>${mo.mo_number}</h4>
        ${moBadge(mo.status)}
        <div class="ms-auto d-flex gap-2">${actionButtons}</div>
      </div>

      <!-- Header info -->
      <div class="row g-3 mb-4">
        <div class="col-md-6">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body">
              <div class="row g-2 small">
                <div class="col-5 text-muted">Posting Date</div>
                <div class="col-7 fw-semibold">${formatDate(mo.posting_date)}</div>
                ${mo.expected_date ? `
                <div class="col-5 text-muted">Expected Date</div>
                <div class="col-7">${formatDate(mo.expected_date)}</div>` : ''}
                <div class="col-5 text-muted">Input Item</div>
                <div class="col-7 fw-semibold">${mo.input_item_name}</div>
                <div class="col-5 text-muted">Input Qty</div>
                <div class="col-7 fw-bold">${formatNumber(mo.input_qty, 3)} ${mo.input_unit || ''}</div>
                <div class="col-5 text-muted">Unit Cost</div>
                <div class="col-7">${formatCurrency(mo.input_unit_cost)}</div>
                <div class="col-5 text-muted">Total Input Cost</div>
                <div class="col-7 fw-bold text-warning">${formatCurrency(mo.doc_total)}</div>
                ${mo.remarks ? `
                <div class="col-5 text-muted">Remarks</div>
                <div class="col-7">${mo.remarks}</div>` : ''}
              </div>
            </div>
          </div>
        </div>
        <div class="col-md-6">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body">
              <div class="row g-2 small">
                <div class="col-5 text-muted">Goods Issue</div>
                <div class="col-7">${giLink}</div>
                <div class="col-5 text-muted">Goods Receipt</div>
                <div class="col-7">${grLink}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Output lines -->
      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3 fw-bold">
          <i class="bi bi-list-ul me-2"></i>Output Products
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Code</th><th>Item</th>
                <th class="text-end">Actual Qty</th>
              </tr>
            </thead>
            <tbody>${lines || `<tr><td colspan="3" class="text-center text-muted py-3">No output recorded yet — complete the milling order to record output.</td></tr>`}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Complete Modal -->
    <div class="modal fade" id="completeModal" tabindex="-1">
      <div class="modal-dialog modal-lg">
        <div class="modal-content">
          <div class="modal-header border-0 pb-0">
            <div>
              <h5 class="modal-title fw-bold"><i class="bi bi-check-circle me-2 text-success"></i>Complete Milling Order</h5>
              <div class="text-muted small">${mo.mo_number}</div>
            </div>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">

            <!-- Input Summary -->
            <div class="card bg-light border-0 mb-3">
              <div class="card-body py-2 px-3">
                <div class="small text-muted fw-bold text-uppercase mb-1">Input Summary</div>
                <div class="d-flex gap-4 small">
                  <span><span class="text-muted">Item:</span> <strong>${mo.input_item_name}</strong></span>
                  <span><span class="text-muted">Qty In:</span> <strong>${formatNumber(mo.input_qty, 3)} ${mo.input_unit || ''}</strong></span>
                </div>
              </div>
            </div>

            <!-- Output Lines -->
            <div class="d-flex justify-content-between align-items-center mb-2">
              <div class="fw-bold small"><i class="bi bi-arrow-down-circle me-1 text-success"></i>Milling Output</div>
              <button class="btn btn-sm btn-outline-success" onclick="moCompleteAddLine()">
                <i class="bi bi-plus me-1"></i>Add Output Line
              </button>
            </div>
            <table class="table table-sm align-middle mb-2" id="completeOutputTable">
              <thead class="table-light">
                <tr>
                  <th>Output Item</th>
                  <th style="width:120px">Actual Qty</th>
                  <th style="width:80px">Unit</th>
                  <th style="width:36px"></th>
                </tr>
              </thead>
              <tbody id="completeOutputLines"></tbody>
            </table>

            <!-- Recovery Rate (MRR) -->
            <div class="d-flex align-items-center gap-3 mb-3 p-2 rounded bg-light">
              <span class="small text-muted" title="Milling Recovery Rate = milled white rice output ÷ palay input. Rice hull, bran, and polishings are excluded (PSA/IRRI standard).">
                Milling Recovery Rate <i class="bi bi-info-circle text-muted"></i>:
              </span>
              <div class="progress flex-grow-1" style="height:8px;">
                <div id="moRecoveryBar" class="progress-bar bg-success" style="width:0%"></div>
              </div>
              <span id="moRecoveryPct" class="small fw-bold text-success" style="min-width:40px">0%</span>
            </div>

            <!-- Remarks -->
            <div>
              <label class="form-label small fw-bold">Completion Remarks</label>
              <input type="text" id="moCompletionNote" class="form-control form-control-sm"
                     placeholder="e.g. moisture content, machine issues…">
            </div>

          </div>
          <div class="modal-footer border-0">
            <button class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-success fw-bold px-4" onclick="submitCompleteMillingOrder(${mo.id})">
              <i class="bi bi-check-circle me-1"></i>Mark as Completed
            </button>
          </div>
        </div>
      </div>
    </div>
  `);

  // Store data needed by the complete modal
  _moLines    = mo.lines || [];
  _moInputQty = mo.input_qty || 0;
}

// ── New Milling Order Form ────────────────────────────────────────────────────

let _moItemList = [];
let _moLines = [], _moInputQty = 0;

async function loadNewMillingOrderForm() {
  const items = await api.ListProductionItems();
  if (!items) { showView('<div class="alert alert-danger">Failed to load production items. Make sure items are marked as production items in Item Master.</div>'); return; }
  _moItemList = items;

  const itemOptions = items.map(i =>
    `<option value="${i.id}" data-code="${i.item_code}" data-name="${i.item_name}" data-unit="${i.invntry_uom}" data-price="${i.avg_price || 0}" data-uom="${i.i_uom_entry || 0}">${i.item_code} — ${i.item_name}</option>`
  ).join('');

  const today = new Date().toISOString().slice(0, 10);

  showView(`
    <div class="container-fluid p-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/production/milling-orders')">
          <i class="bi bi-arrow-left me-1"></i>Back
        </button>
        <h4 class="fw-bold mb-0"><i class="bi bi-gear-wide-connected me-2 text-warning"></i>New Milling Order</h4>
      </div>

      <!-- Header -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label fw-bold">Posting Date <span class="text-danger">*</span></label>
              <input type="date" id="moPostingDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-4">
              <label class="form-label fw-bold">Expected Completion</label>
              <input type="date" id="moExpectedDate" class="form-control">
            </div>
            <div class="col-md-8">
              <label class="form-label fw-bold">Remarks</label>
              <input type="text" id="moRemarks" class="form-control" placeholder="Optional notes">
            </div>
          </div>
        </div>
      </div>

      <!-- Input -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-header bg-warning bg-opacity-10 border-bottom py-3 fw-bold">
          <i class="bi bi-arrow-up-circle me-2 text-warning"></i>Input (Raw Material)
        </div>
        <div class="card-body">
          <div class="row g-3 align-items-end">
            <div class="col-md-5">
              <label class="form-label fw-bold">Item <span class="text-danger">*</span></label>
              <select id="moInputItem" class="form-select" onchange="moOnInputItemChange(this)" required>
                <option value="">— Select item —</option>
                ${itemOptions}
              </select>
            </div>
            <div class="col-md-2">
              <label class="form-label fw-bold">Quantity <span class="text-danger">*</span></label>
              <input type="number" id="moInputQty" class="form-control" min="0.001" step="0.001" placeholder="0.000"
                oninput="moRecalcOutputs()">
            </div>
            <div class="col-md-2">
              <label class="form-label fw-bold">Unit</label>
              <input type="text" id="moInputUnit" class="form-control" readonly placeholder="—">
            </div>
            <div class="col-md-3">
              <label class="form-label fw-bold">Unit Cost (₱)</label>
              <input type="number" id="moInputCost" class="form-control" min="0" step="0.01" placeholder="0.00" readonly>
            </div>
          </div>
        </div>
      </div>

      <div class="alert alert-info border-0 small">
        <i class="bi bi-info-circle me-2"></i>
        Output products (milled rice, bran, ipa, etc.) are recorded when you <strong>complete</strong> the milling order after milling is done.
      </div>

      <div class="d-flex gap-2 justify-content-end">
        <button class="btn btn-secondary" onclick="navigate('#/production/milling-orders')">Cancel</button>
        <button class="btn btn-warning fw-bold" onclick="submitMillingOrder()">
          <i class="bi bi-save me-1"></i>Create Milling Order
        </button>
      </div>
    </div>
  `);

  // Add two default output lines
  moAddOutputLine();
  moAddOutputLine();
}

window.moOnInputItemChange = function(sel) {
  const opt = sel.options[sel.selectedIndex];
  document.getElementById('moInputUnit').value = opt.dataset.unit || '';
  document.getElementById('moInputCost').value = opt.dataset.price || '';
};

window.submitMillingOrder = async function() {
  const postingDate   = document.getElementById('moPostingDate').value;
  const expectedDate  = document.getElementById('moExpectedDate').value;
  const remarks       = document.getElementById('moRemarks').value.trim();
  const inputItemSel  = document.getElementById('moInputItem');
  const inputItemID   = parseInt(inputItemSel.value) || 0;
  const inputQty      = parseFloat(document.getElementById('moInputQty').value) || 0;
  const inputUomEntry = parseInt(inputItemSel.options[inputItemSel.selectedIndex]?.dataset?.uom) || 0;

  if (!postingDate)   { toast('Posting date is required.', 'warning'); return; }
  if (!inputItemID)   { toast('Input item is required.', 'warning'); return; }
  if (inputQty <= 0)  { toast('Input quantity must be greater than zero.', 'warning'); return; }

  const req = {
    posting_date:    postingDate,
    expected_date:   expectedDate,
    remarks,
    input_item_id:   inputItemID,
    input_qty:       inputQty,
    input_uom_entry: inputUomEntry,
  };

  const mo = await api.CreateMillingOrder(req);
  if (mo) navigate(`#/production/milling-orders/${mo.id}`);
};

// ── Start ──────────────────────────────────────────────────────────────────────

window.startMillingOrder = async function(id) {
  if (!confirm('Start this milling order?\n\nThis will create a Goods Issue to consume the paddy from inventory.')) return;
  const ok = await api.StartMillingOrder(id);
  if (ok) navigate(`#/production/milling-orders/${id}`);
};

// ── Complete ───────────────────────────────────────────────────────────────────

let _moCompleteLineIdx = 0;
let _moInputQtyForRecovery = 0;

window.openCompleteModal = function(id) {
  _moCompleteLineIdx = 0;
  _moInputQtyForRecovery = _moInputQty || 0;
  document.getElementById('completeOutputLines').innerHTML = '';
  document.getElementById('moCompletionNote').value = '';
  moUpdateRecovery();
  // Add 3 default lines
  moCompleteAddLine();
  moCompleteAddLine();
  moCompleteAddLine();
  new bootstrap.Modal(document.getElementById('completeModal')).show();
};

window.moCompleteAddLine = function() {
  const idx = _moCompleteLineIdx++;
  const itemOptions = _moItemList.map(i =>
    `<option value="${i.id}" data-unit="${i.invntry_uom}" data-uom="${i.i_uom_entry || 0}">${i.item_code} — ${i.item_name}</option>`
  ).join('');
  const tr = document.createElement('tr');
  tr.dataset.idx = idx;
  tr.innerHTML = `
    <td>
      <select class="form-select form-select-sm mo-cmp-item" onchange="moCompleteOnItemChange(this, ${idx})">
        <option value="">— Select item —</option>
        ${itemOptions}
      </select>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm mo-cmp-qty text-center"
             min="0" step="0.001" placeholder="0.000" oninput="moUpdateRecovery()">
    </td>
    <td>
      <input type="text" class="form-control form-control-sm mo-cmp-unit text-center" readonly placeholder="—">
    </td>
    <td>
      <button class="btn btn-sm btn-outline-danger py-0" onclick="this.closest('tr').remove();moUpdateRecovery();">
        <i class="bi bi-x"></i>
      </button>
    </td>`;
  document.getElementById('completeOutputLines').appendChild(tr);
};

window.moCompleteOnItemChange = function(sel, idx) {
  const opt = sel.options[sel.selectedIndex];
  const row = document.querySelector(`#completeOutputLines tr[data-idx="${idx}"]`);
  if (row) row.querySelector('.mo-cmp-unit').value = opt.dataset.unit || '';
  moUpdateRecovery();
};

window.moUpdateRecovery = function() {
  const inputQty = _moInputQtyForRecovery;
  // MRR (Milling Recovery Rate) = milled white rice output only ÷ palay input × 100
  // Rice hull (ipa), bran, and polishings are excluded per PSA/IRRI standard.
  let milledRiceQty = 0;
  document.querySelectorAll('#completeOutputLines tr').forEach(row => {
    const sel = row.querySelector('.mo-cmp-item');
    const qty = parseFloat(row.querySelector('.mo-cmp-qty')?.value) || 0;
    const itemId = parseInt(sel?.value) || 0;
    if (!itemId || qty <= 0) return;
    const item = _moItemList.find(i => i.id === itemId);
    if (item && item.category === 'Milled Rice') milledRiceQty += qty;
  });
  const pct = inputQty > 0 ? Math.min((milledRiceQty / inputQty) * 100, 999) : 0;
  const bar = Math.min(pct, 100);
  document.getElementById('moRecoveryBar').style.width = bar.toFixed(1) + '%';
  document.getElementById('moRecoveryPct').textContent = pct.toFixed(1) + '%';
};

window.submitCompleteMillingOrder = async function(id) {
  const lines = [];
  document.querySelectorAll('#completeOutputLines tr').forEach(row => {
    const itemID  = parseInt(row.querySelector('.mo-cmp-item')?.value) || 0;
    const qty     = parseFloat(row.querySelector('.mo-cmp-qty')?.value) || 0;
    const uomEntry = parseInt(row.querySelector('.mo-cmp-item')?.selectedOptions[0]?.dataset?.uom) || 0;
    if (itemID > 0 && qty > 0) {
      lines.push({ output_item_id: itemID, actual_qty: qty, uom_entry: uomEntry });
    }
  });

  if (lines.length === 0) { toast('Add at least one output item with a quantity.', 'warning'); return; }

  const note = document.getElementById('moCompletionNote').value.trim();
  bootstrap.Modal.getInstance(document.getElementById('completeModal'))?.hide();

  const ok = await api.CompleteMillingOrder(id, { lines, completion_note: note });
  if (ok) navigate(`#/production/milling-orders/${id}`);
};

// ── Cancel ─────────────────────────────────────────────────────────────────────

window.cancelMillingOrder = async function(id) {
  if (!confirm('Cancel this milling order?\n\nIf it is In Progress, the Goods Issue will be reversed and stock restored.')) return;
  const ok = await api.CancelMillingOrder(id);
  if (ok) navigate(`#/production/milling-orders/${id}`);
};

function _resetProductionMilling() {
  _moItemList = []; _moCompleteLineIdx = 0; _moInputQtyForRecovery = 0;
  _moLines = []; _moInputQty = 0;
}
