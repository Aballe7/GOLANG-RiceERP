// production-bom.js — Bill of Materials (OITT / ITT1)

// ── List ───────────────────────────────────────────────────────────────────────

async function loadBOMList(tabBar) {
  const boms = await api.ListBOMs();
  if (!boms) {
    showView(`<div class="container-fluid p-4">${tabBar}<div class="alert alert-warning">Failed to load BOMs.</div></div>`);
    return;
  }

  const TREE_TYPE_LABEL = {
    P: '<span class="badge bg-primary">BOM</span>',
    M: '<span class="badge bg-warning text-dark">Milling Template</span>',
    D: '<span class="badge bg-info text-dark">Drying Template</span>',
  };
  const rows = boms.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No BOMs yet. Create one to enable Work Orders.</td></tr>`
    : boms.map(b => `
        <tr>
          <td class="font-monospace fw-semibold small">${b.code}</td>
          <td>${TREE_TYPE_LABEL[b.tree_type] || TREE_TYPE_LABEL.P}</td>
          <td class="text-muted small">${formatNumber(b.quantity, 3)}</td>
          <td>${(b.lines || []).length} line${(b.lines || []).length !== 1 ? 's' : ''}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-primary me-1"
              onclick="openBOMModal(${JSON.stringify(JSON.stringify(b))})">
              <i class="bi bi-pencil"></i>
            </button>
            <button class="btn btn-sm btn-outline-danger"
              onclick="deleteBOM('${b.code}')">
              <i class="bi bi-trash"></i>
            </button>
          </td>
        </tr>`).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-3">
        <h4 class="fw-bold mb-0"><i class="bi bi-diagram-3 me-2 text-primary"></i>Bill of Materials</h4>
        <button class="btn btn-primary" onclick="openBOMModal(null)">
          <i class="bi bi-plus-lg me-1"></i>New BOM
        </button>
      </div>
      ${tabBar}
      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Item Code</th>
                <th>Type</th>
                <th>Batch Qty</th>
                <th>Lines</th>
                <th></th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- BOM Modal -->
    <div class="modal fade" id="bomModal" tabindex="-1">
      <div class="modal-dialog modal-xl">
        <div class="modal-content">
          <div class="modal-header border-0 pb-0">
            <h5 class="modal-title fw-bold" id="bomModalTitle">
              <i class="bi bi-diagram-3 me-2 text-primary"></i>Bill of Materials
            </h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <div class="row g-3 mb-4">
              <div class="col-md-3">
                <label class="form-label fw-bold">Type</label>
                <select id="bomTreeType" class="form-select" onchange="bomOnTypeChange()">
                  <option value="P" selected>Production BOM</option>
                  <option value="M">Milling Yield Template</option>
                  <option value="D">Drying Yield Template</option>
                </select>
              </div>
              <div class="col-md-3">
                <label class="form-label fw-bold"><span id="bomCodeLabel">Finished Item Code</span> <span class="text-danger">*</span></label>
                <input type="text" id="bomCode" class="form-control font-monospace" placeholder="e.g. MILLED-RICE-STD">
                <div class="form-text" id="bomCodeHint">Must match an Item Master code with mak_item = Yes</div>
              </div>
              <div class="col-md-2">
                <label class="form-label fw-bold">Batch Quantity</label>
                <input type="number" id="bomQty" class="form-control" value="1" min="0.001" step="0.001">
                <div class="form-text" id="bomQtyHint"></div>
              </div>
              <div class="col-md-2">
                <label class="form-label fw-bold">Default Warehouse</label>
                <input type="text" id="bomWarehouse" class="form-control" placeholder="WH01">
              </div>
              <div class="col-md-2">
                <label class="form-label fw-bold">Notes</label>
                <input type="text" id="bomNotes" class="form-control" placeholder="Optional">
              </div>
            </div>

            <div class="d-flex justify-content-between align-items-center mb-2">
              <div class="fw-bold small"><i class="bi bi-list-ul me-1 text-primary"></i><span id="bomLinesLabel">Components</span></div>
              <button class="btn btn-sm btn-outline-primary" onclick="bomAddLine()">
                <i class="bi bi-plus me-1"></i>Add Line
              </button>
            </div>
            <div class="table-responsive">
              <table class="table table-sm align-middle" id="bomLinesTable">
                <thead class="table-light">
                  <tr>
                    <th>Item Code</th>
                    <th style="width:120px">Qty per Batch</th>
                    <th style="width:100px">Warehouse</th>
                    <th style="width:100px" class="bom-col-issue">Issue Method</th>
                    <th style="width:130px" class="bom-col-outtype">Output Type</th>
                    <th style="width:36px"></th>
                  </tr>
                </thead>
                <tbody id="bomLinesBody"></tbody>
              </table>
            </div>
          </div>
          <div class="modal-footer border-0">
            <button class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary fw-bold px-4" onclick="saveBOM()">
              <i class="bi bi-save me-1"></i>Save BOM
            </button>
          </div>
        </div>
      </div>
    </div>
  `);
}

// ── Modal helpers ──────────────────────────────────────────────────────────────

let _bomLineIdx = 0;
let _bomIsEdit  = false;

window.openBOMModal = function(bomJson) {
  _bomLineIdx = 0;
  _bomIsEdit = !!bomJson;
  document.getElementById('bomLinesBody').innerHTML = '';

  const bom = bomJson ? JSON.parse(bomJson) : null;
  document.getElementById('bomModalTitle').innerHTML =
    `<i class="bi bi-diagram-3 me-2 text-primary"></i>${bom ? 'Edit BOM — ' + bom.code : 'New BOM'}`;
  document.getElementById('bomTreeType').value  = bom?.tree_type || 'P';
  document.getElementById('bomTreeType').disabled = !!bom;
  document.getElementById('bomCode').value      = bom?.code      || '';
  document.getElementById('bomCode').readOnly   = !!bom;
  document.getElementById('bomQty').value       = bom?.quantity  ?? 1;
  document.getElementById('bomWarehouse').value = bom?.warehouse || '';
  document.getElementById('bomNotes').value     = bom?.notes     || '';
  bomOnTypeChange();

  (bom?.lines || []).forEach(l => bomAddLine(l));
  if (!bom) bomAddLine();

  new bootstrap.Modal(document.getElementById('bomModal')).show();
};

// bomOnTypeChange adapts labels and column visibility to the tree type:
// P (production BOM) shows Issue Method; M/D (yield templates) show Output Type.
window.bomOnTypeChange = function() {
  const type = document.getElementById('bomTreeType').value;
  const isTemplate = type === 'M' || type === 'D';
  document.getElementById('bomCodeLabel').textContent = isTemplate ? 'Input Item Code (Paddy)' : 'Finished Item Code';
  document.getElementById('bomCodeHint').textContent = isTemplate
    ? 'The raw material this template applies to — expected outputs are created automatically on new orders'
    : 'Must match an Item Master code with mak_item = Yes';
  document.getElementById('bomQtyHint').textContent = isTemplate ? 'e.g. 100 for per-100-kg yields' : '';
  document.getElementById('bomLinesLabel').textContent = isTemplate ? 'Expected Outputs' : 'Components';
  document.querySelectorAll('.bom-col-issue').forEach(el => el.style.display = isTemplate ? 'none' : '');
  document.querySelectorAll('.bom-col-outtype').forEach(el => el.style.display = isTemplate ? '' : 'none');
};

window.bomAddLine = function(prefill) {
  const idx = _bomLineIdx++;
  const isTemplate = ['M', 'D'].includes(document.getElementById('bomTreeType').value);
  const tr = document.createElement('tr');
  tr.dataset.idx = idx;
  tr.innerHTML = `
    <td>
      <input type="text" class="form-control form-control-sm bom-item-code font-monospace"
             placeholder="Item Code" value="${prefill?.item_code || ''}">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm bom-qty text-center"
             min="0.001" step="0.001" placeholder="0.000" value="${prefill?.quantity || ''}">
    </td>
    <td>
      <input type="text" class="form-control form-control-sm bom-whs"
             placeholder="WH01" value="${prefill?.warehouse || ''}">
    </td>
    <td class="bom-col-issue" style="display:${isTemplate ? 'none' : ''}">
      <select class="form-select form-select-sm bom-issue">
        <option value="M" ${!prefill || prefill.issue_method === 'M' ? 'selected' : ''}>Manual</option>
        <option value="B" ${prefill?.issue_method === 'B' ? 'selected' : ''}>Backflush</option>
      </select>
    </td>
    <td class="bom-col-outtype" style="display:${isTemplate ? '' : 'none'}">
      <select class="form-select form-select-sm bom-outtype">
        <option value="" ${!prefill?.output_type ? 'selected' : ''}>—</option>
        <option value="H" ${prefill?.output_type === 'H' ? 'selected' : ''}>Head Rice</option>
        <option value="B" ${prefill?.output_type === 'B' ? 'selected' : ''}>Brokens</option>
        <option value="Y" ${prefill?.output_type === 'Y' ? 'selected' : ''}>By-product</option>
      </select>
    </td>
    <td>
      <button class="btn btn-sm btn-outline-danger py-0" onclick="this.closest('tr').remove()">
        <i class="bi bi-x"></i>
      </button>
    </td>`;
  document.getElementById('bomLinesBody').appendChild(tr);
};

window.saveBOM = async function() {
  const code = document.getElementById('bomCode').value.trim();
  if (!code) { toast('Item code is required.', 'warning'); return; }

  const lines = [];
  document.querySelectorAll('#bomLinesBody tr').forEach(tr => {
    const itemCode = tr.querySelector('.bom-item-code')?.value.trim();
    const qty      = parseFloat(tr.querySelector('.bom-qty')?.value) || 0;
    const whs      = tr.querySelector('.bom-whs')?.value.trim();
    const method   = tr.querySelector('.bom-issue')?.value || 'M';
    const outType  = tr.querySelector('.bom-outtype')?.value || '';
    if (itemCode && qty > 0) {
      lines.push({ item_code: itemCode, quantity: qty, warehouse: whs, issue_method: method, output_type: outType });
    }
  });

  const payload = {
    code,
    tree_type: document.getElementById('bomTreeType').value || 'P',
    quantity:  parseFloat(document.getElementById('bomQty').value) || 1,
    warehouse: document.getElementById('bomWarehouse').value.trim(),
    notes:     document.getElementById('bomNotes').value.trim(),
    lines,
  };

  bootstrap.Modal.getInstance(document.getElementById('bomModal'))?.hide();
  const ok = await api.UpsertBOM(payload);
  if (ok) navigate('#/production/bom');
};

window.deleteBOM = async function(code) {
  if (!confirm(`Delete BOM for "${code}"?\n\nThis will remove all component lines. Work Orders already created will not be affected.`)) return;
  const ok = await api.DeleteBOM(code);
  if (ok) navigate('#/production/bom');
};

function _resetProductionBOM() {
  _bomLineIdx = 0; _bomIsEdit = false;
}
