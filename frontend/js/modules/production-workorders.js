// production-workorders.js — Work Orders (OWOR / WOR1)

let _woBomMap = null;

// ── Status helpers ─────────────────────────────────────────────────────────────

const WO_STATUS_BADGE = {
  'P': 'secondary',
  'R': 'warning',
  'C': 'success',
  'L': 'danger',
};
const WO_STATUS_LABEL = {
  'P': 'Planned',
  'R': 'Released',
  'C': 'Closed',
  'L': 'Cancelled',
};

function woBadge(status) {
  const cls   = WO_STATUS_BADGE[status] || 'secondary';
  const label = WO_STATUS_LABEL[status] || status;
  return `<span class="badge bg-${cls}">${label}</span>`;
}

// ── List ───────────────────────────────────────────────────────────────────────

async function loadWOList(tabBar) {
  sessionStorage.removeItem('docReturnRoute');
  const wos = await api.ListWorkOrders();
  if (!wos) {
    showView(`<div class="container-fluid p-4">${tabBar}<div class="alert alert-warning">Failed to load work orders.</div></div>`);
    return;
  }

  const rows = wos.length === 0
    ? `<tr><td colspan="7" class="text-center text-muted py-4">No work orders yet. <a href="#" onclick="navigate('#/production/work-orders/new');return false;">Create one</a></td></tr>`
    : wos.map(wo => `
        <tr style="cursor:pointer" onclick="navigate('#/production/work-orders/${wo.doc_entry}')">
          <td class="fw-semibold font-monospace small">${wo.doc_num}</td>
          <td class="fw-semibold">${wo.item_name || wo.item_code}</td>
          <td class="text-end">${formatNumber(wo.planned_qty, 3)}</td>
          <td class="text-end text-success fw-bold">${formatNumber(wo.cmplt_qty, 3)}</td>
          <td class="text-muted small">${formatDate(wo.start_date)}</td>
          <td class="text-muted small">${formatDate(wo.due_date)}</td>
          <td>${woBadge(wo.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary"
              onclick="event.stopPropagation();navigate('#/production/work-orders/${wo.doc_entry}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>`).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-3">
        <h4 class="fw-bold mb-0"><i class="bi bi-clipboard2-check me-2 text-success"></i>Work Orders</h4>
        <button class="btn btn-success" onclick="navigate('#/production/work-orders/new')">
          <i class="bi bi-plus-lg me-1"></i>New Work Order
        </button>
      </div>
      ${tabBar}
      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>WO #</th><th>Finished Item</th>
                <th class="text-end">Planned Qty</th>
                <th class="text-end">Completed</th>
                <th>Start Date</th><th>Due Date</th>
                <th>Status</th><th></th>
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

async function loadWODetail(docEntry) {
  const wo = await api.GetWorkOrder(docEntry);
  if (!wo) { showView('<div class="alert alert-danger">Work order not found.</div>'); return; }

  const lines = (wo.lines || []).map(l => `
    <tr>
      <td class="small font-monospace text-muted">${l.item_code}</td>
      <td class="fw-semibold">${l.item_name}</td>
      <td class="text-center">
        <span class="badge bg-${l.line_dir === 'I' ? 'secondary' : 'success'}">${l.line_dir === 'I' ? 'Input' : 'Output'}</span>
      </td>
      <td class="text-end">${formatNumber(l.planned_qty, 3)}</td>
      <td class="text-end fw-bold ${l.issued_qty > 0 ? 'text-warning' : 'text-muted'}">${formatNumber(l.issued_qty, 3)}</td>
      <td class="text-muted small">${l.issue_method === 'B' ? 'Backflush' : 'Manual'}</td>
      <td class="text-muted small">${l.warehouse || '—'}</td>
    </tr>`).join('');

  let actionButtons = '';
  if (wo.status === 'P') {
    actionButtons = `
      <button class="btn btn-warning me-2" onclick="releaseWO(${wo.doc_entry})">
        <i class="bi bi-play-fill me-1"></i>Release
      </button>
      <button class="btn btn-outline-danger" onclick="cancelWO(${wo.doc_entry})">
        <i class="bi bi-x-circle me-1"></i>Cancel
      </button>`;
  } else if (wo.status === 'R') {
    actionButtons = `
      <button class="btn btn-success me-2" onclick="openCloseWOModal(${wo.doc_entry}, ${wo.planned_qty})">
        <i class="bi bi-check-circle me-1"></i>Close
      </button>
      <button class="btn btn-outline-danger" onclick="cancelWO(${wo.doc_entry})">
        <i class="bi bi-x-circle me-1"></i>Cancel
      </button>`;
  }

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center gap-3 mb-3 flex-wrap">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate(sessionStorage.getItem('docReturnRoute')||'#/production/work-orders');sessionStorage.removeItem('docReturnRoute')">
          <i class="bi bi-arrow-left me-1"></i>Back
        </button>
        <h4 class="fw-bold mb-0"><i class="bi bi-clipboard2-check me-2 text-success"></i>${wo.doc_num}</h4>
        ${woBadge(wo.status)}
        <div class="ms-auto d-flex gap-2">${actionButtons}</div>
      </div>

      <div class="row g-3 mb-4">
        <div class="col-md-6">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body">
              <div class="row g-2 small">
                <div class="col-5 text-muted">Finished Item</div>
                <div class="col-7 fw-semibold">${wo.item_code} — ${wo.item_name}</div>
                <div class="col-5 text-muted">Planned Qty</div>
                <div class="col-7 fw-bold">${formatNumber(wo.planned_qty, 3)}</div>
                <div class="col-5 text-muted">Completed Qty</div>
                <div class="col-7 fw-bold text-success">${formatNumber(wo.cmplt_qty, 3)}</div>
                <div class="col-5 text-muted">Rejected Qty</div>
                <div class="col-7 text-danger">${formatNumber(wo.rjct_qty, 3)}</div>
                <div class="col-5 text-muted">Warehouse</div>
                <div class="col-7">${wo.warehouse || '—'}</div>
              </div>
            </div>
          </div>
        </div>
        <div class="col-md-6">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body">
              <div class="row g-2 small">
                <div class="col-5 text-muted">Start Date</div>
                <div class="col-7">${formatDate(wo.start_date)}</div>
                <div class="col-5 text-muted">Due Date</div>
                <div class="col-7">${formatDate(wo.due_date)}</div>
                ${wo.notes ? `
                <div class="col-5 text-muted">Notes</div>
                <div class="col-7">${wo.notes}</div>` : ''}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3 fw-bold">
          <i class="bi bi-list-ul me-2"></i>Components
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Code</th><th>Item</th><th>Dir</th>
                <th class="text-end">Planned Qty</th>
                <th class="text-end">Issued Qty</th>
                <th>Issue Method</th>
                <th>Warehouse</th>
              </tr>
            </thead>
            <tbody>${lines || `<tr><td colspan="7" class="text-center text-muted py-3">No components.</td></tr>`}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Close WO Modal -->
    <div class="modal fade" id="closeWOModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header border-0 pb-0">
            <h5 class="modal-title fw-bold"><i class="bi bi-check-circle me-2 text-success"></i>Close Work Order</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <div class="mb-3">
              <label class="form-label fw-bold">Completed Qty <span class="text-danger">*</span></label>
              <input type="number" id="woCmpltQty" class="form-control" min="0.001" step="0.001" placeholder="0.000">
            </div>
            <div class="mb-3">
              <label class="form-label fw-bold">Rejected Qty</label>
              <input type="number" id="woRjctQty" class="form-control" min="0" step="0.001" placeholder="0.000" value="0">
            </div>
            <div class="alert alert-info border-0 small mb-0">
              <i class="bi bi-info-circle me-1"></i>
              Completed qty will be received into the finished goods warehouse.
            </div>
          </div>
          <div class="modal-footer border-0">
            <button class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-success fw-bold px-4" id="closeWOBtn" onclick="">
              <i class="bi bi-check-circle me-1"></i>Close Work Order
            </button>
          </div>
        </div>
      </div>
    </div>
  `);
}

// ── New Work Order Form ────────────────────────────────────────────────────────

async function loadNewWOForm() {
  const [items, boms] = await Promise.all([api.ListProductionItems(), api.ListBOMs()]);
  if (!items) { showView('<div class="alert alert-danger">Failed to load production items.</div>'); return; }

  const bomCodes = new Set((boms || []).map(b => b.code));
  const eligible = (items || []).filter(i => bomCodes.has(i.item_code));

  const itemOptions = eligible.map(i =>
    `<option value="${i.item_code}" data-name="${i.item_name}">${i.item_code} — ${i.item_name}</option>`
  ).join('');

  const today = new Date().toISOString().slice(0, 10);

  showView(`
    <div class="container-fluid p-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/production/work-orders')">
          <i class="bi bi-arrow-left me-1"></i>Back
        </button>
        <h4 class="fw-bold mb-0"><i class="bi bi-clipboard2-check me-2 text-success"></i>New Work Order</h4>
      </div>

      <div class="card border-0 shadow-sm mb-4">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-5">
              <label class="form-label fw-bold">Finished Item <span class="text-danger">*</span></label>
              <select id="woItem" class="form-select" onchange="woOnItemChange(this)" required>
                <option value="">— Select item with BOM —</option>
                ${itemOptions}
              </select>
            </div>
            <div class="col-md-2">
              <label class="form-label fw-bold">Planned Qty <span class="text-danger">*</span></label>
              <input type="number" id="woPlannedQty" class="form-control" min="0.001" step="0.001"
                     placeholder="0.000" oninput="woPreviewComponents()">
            </div>
            <div class="col-md-2">
              <label class="form-label fw-bold">Warehouse</label>
              <input type="text" id="woWarehouse" class="form-control" placeholder="WH01">
            </div>
            <div class="col-md-2">
              <label class="form-label fw-bold">Start Date</label>
              <input type="date" id="woStartDate" class="form-control" value="${today}">
            </div>
            <div class="col-md-2">
              <label class="form-label fw-bold">Due Date</label>
              <input type="date" id="woDueDate" class="form-control">
            </div>
            <div class="col-md-6">
              <label class="form-label fw-bold">Notes</label>
              <input type="text" id="woNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <!-- BOM preview -->
      <div class="card border-0 shadow-sm mb-4" id="woComponentsCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3 fw-bold">
          <i class="bi bi-list-ul me-2 text-primary"></i>Components (from BOM — scaled to planned qty)
        </div>
        <div class="table-responsive">
          <table class="table table-sm align-middle mb-0">
            <thead class="table-light">
              <tr><th>Item Code</th><th>Item</th><th class="text-end">Required Qty</th><th>Issue Method</th></tr>
            </thead>
            <tbody id="woComponentsBody"></tbody>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 justify-content-end">
        <button class="btn btn-secondary" onclick="navigate('#/production/work-orders')">Cancel</button>
        <button class="btn btn-success fw-bold" onclick="submitNewWO()">
          <i class="bi bi-save me-1"></i>Create Work Order
        </button>
      </div>
    </div>
  `);

  _woBomMap = {};
  (boms || []).forEach(b => { _woBomMap[b.code] = b; });
}

window.woOnItemChange = function(sel) {
  woPreviewComponents();
};

window.woPreviewComponents = function() {
  const code    = document.getElementById('woItem').value;
  const planned = parseFloat(document.getElementById('woPlannedQty').value) || 0;
  const bom     = _woBomMap?.[code];
  const card    = document.getElementById('woComponentsCard');
  const tbody   = document.getElementById('woComponentsBody');
  if (!bom || !bom.lines || !planned) { card.style.display = 'none'; return; }

  const scale = bom.quantity > 0 ? planned / bom.quantity : planned;
  const rows = bom.lines.map(l => `
    <tr>
      <td class="font-monospace small">${l.item_code}</td>
      <td>${l.item_name || l.item_code}</td>
      <td class="text-end fw-bold">${formatNumber(l.quantity * scale, 3)}</td>
      <td class="text-muted small">${l.issue_method === 'B' ? 'Backflush' : 'Manual'}</td>
    </tr>`).join('');
  tbody.innerHTML = rows;
  card.style.display = '';
};

window.submitNewWO = async function() {
  const itemCode  = document.getElementById('woItem').value;
  const planned   = parseFloat(document.getElementById('woPlannedQty').value) || 0;
  const warehouse = document.getElementById('woWarehouse').value.trim();
  const startDate = document.getElementById('woStartDate').value;
  const dueDate   = document.getElementById('woDueDate').value;
  const notes     = document.getElementById('woNotes').value.trim();

  if (!itemCode)  { toast('Select a finished item.', 'warning'); return; }
  if (planned <= 0) { toast('Planned quantity must be greater than zero.', 'warning'); return; }

  const wo = await api.CreateWorkOrder({ item_code: itemCode, planned_qty: planned, warehouse, start_date: startDate, due_date: dueDate, notes });
  if (wo) navigate(`#/production/work-orders/${wo.doc_entry}`);
};

// ── Actions ───────────────────────────────────────────────────────────────────

window.releaseWO = async function(docEntry) {
  if (!confirm('Release this work order?\n\nBackflush components will be automatically issued from inventory.')) return;
  const ok = await api.ReleaseWorkOrder(docEntry);
  if (ok) navigate(`#/production/work-orders/${docEntry}`);
};

window.openCloseWOModal = function(docEntry, plannedQty) {
  document.getElementById('woCmpltQty').value = plannedQty || '';
  document.getElementById('woRjctQty').value  = 0;
  document.getElementById('closeWOBtn').onclick = () => submitCloseWO(docEntry);
  new bootstrap.Modal(document.getElementById('closeWOModal')).show();
};

window.submitCloseWO = async function(docEntry) {
  const cmpltQty = parseFloat(document.getElementById('woCmpltQty').value) || 0;
  const rjctQty  = parseFloat(document.getElementById('woRjctQty').value)  || 0;
  if (cmpltQty <= 0) { toast('Completed quantity must be greater than zero.', 'warning'); return; }
  bootstrap.Modal.getInstance(document.getElementById('closeWOModal'))?.hide();
  const ok = await api.CloseWorkOrder(docEntry, cmpltQty, rjctQty);
  if (ok) navigate(`#/production/work-orders/${docEntry}`);
};

window.cancelWO = async function(docEntry) {
  if (!confirm('Cancel this work order?\n\nIf Released, issued stock will be restored.')) return;
  const ok = await api.CancelWorkOrder(docEntry);
  if (ok) navigate(`#/production/work-orders/${docEntry}`);
};

function _resetProductionWorkorders() {
  _woBomMap = null;
}
