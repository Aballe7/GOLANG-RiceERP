// inventory.js — Modules.Inventory

Modules.Inventory = {
  async load(sub, id, action) {
    const activeTab = sub || 'stock';
    showLoading();

    const tabBar = `
      <ul class="nav nav-tabs mb-4" id="invTabs">
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'stock' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/stock');return false;">
            <i class="bi bi-clipboard-data me-1"></i>Stock Overview
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'items' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/items');return false;">
            <i class="bi bi-list-ul me-1"></i>Item Master
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'categories' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/categories');return false;">
            <i class="bi bi-tags me-1"></i>Item Categories
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'uom' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/uom');return false;">
            <i class="bi bi-rulers me-1"></i>UoM Setup
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'gr' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/gr');return false;">
            <i class="bi bi-arrow-down-circle me-1"></i>Goods Receipt
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'gi' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/gi');return false;">
            <i class="bi bi-arrow-up-circle me-1"></i>Goods Issue
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'warehouses' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/warehouses');return false;">
            <i class="bi bi-building me-1"></i>Warehouses
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'ledger' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/ledger');return false;">
            <i class="bi bi-journal-text me-1"></i>Inventory Ledger
          </a>
        </li>
      </ul>
    `;

    if (activeTab === 'stock') await loadStockOverview(tabBar);
    else if (activeTab === 'items') await loadItemMaster(tabBar);
    else if (activeTab === 'categories') await loadItemCategories(tabBar);
    else if (activeTab === 'uom') await loadUoMSetup(tabBar);
    else if (activeTab === 'gr') {
      if (action === 'new') await loadNewGRForm(tabBar);
      else if (id) await viewGR(id, tabBar);
      else await loadGoodsReceiptList(tabBar);
    }
    else if (activeTab === 'gi') {
      if (action === 'new') await loadNewGIForm(tabBar);
      else if (id) await viewGI(id, tabBar);
      else await loadGoodsIssueList(tabBar);
    }
    else if (activeTab === 'warehouses') await loadWarehouseView(tabBar);
    else if (activeTab === 'ledger') await loadInventoryLedger(tabBar);
    else await loadStockOverview(tabBar);
  },

  reset() {
    _whsModalMode = 'create'; _whsEditCode = '';
    _itemMasterList = []; _uomMasterList = [];
    _itemCategoryList = []; _uomGroupList = [];
    _uomGroupLineIndex = 0; _oivlAllRows = [];
    _resetInventoryGR();
    _resetInventoryGI();
  }
};

// ── Stock Overview ────────────────────────────────────────────────────────────

async function loadStockOverview(tabBar) {
  const items = await api.ListItemMasters();
  if (!items) {
    showView(`<div class="container-fluid p-4">${tabBar}<div class="alert alert-warning">Failed to load stock data.</div></div>`);
    return;
  }

  // Group by category
  const byCategory = {};
  items.forEach(i => {
    const cat = i.category_name || 'Uncategorized';
    if (!byCategory[cat]) byCategory[cat] = [];
    byCategory[cat].push(i);
  });

  const lowItems = items.filter(i => (i.min_level ?? 0) > 0 && (i.on_hand ?? 0) < (i.min_level ?? 0));

  let lowAlert = '';
  if (lowItems.length > 0) {
    const lowByCategory = {};
    lowItems.forEach(i => {
      const cat = i.category_name || 'Uncategorized';
      if (!lowByCategory[cat]) lowByCategory[cat] = [];
      lowByCategory[cat].push(i);
    });
    const lowRows = Object.entries(lowByCategory).map(([cat, catItems]) => {
      const itemRows = catItems.map(i => {
        const isZero = (i.on_hand ?? 0) === 0;
        return `<tr>
          <td class="ps-3 py-1 small fw-semibold">${i.item_name}</td>
          <td class="py-1 text-center small ${isZero ? 'text-danger fw-bold' : 'text-secondary'}">${formatNumber(i.on_hand ?? 0)} ${i.invntry_uom || ''}</td>
          <td class="pe-3 py-1 text-center small text-muted">min: ${formatNumber(i.min_level)}</td>
        </tr>`;
      }).join('');
      return `<tr class="table-warning">
        <td colspan="3" class="ps-3 py-1">
          <span class="fw-bold small text-uppercase text-warning-emphasis" style="letter-spacing:.04em;">${cat}</span>
        </td>
      </tr>${itemRows}`;
    }).join('');

    lowAlert = `
      <div class="card border-warning shadow-sm mb-4">
        <div class="card-header bg-warning bg-opacity-10 border-warning d-flex align-items-center gap-2 py-2"
             style="cursor:pointer;" onclick="this.nextElementSibling.classList.toggle('d-none');this.querySelector('.ls-chevron').classList.toggle('bi-chevron-down');this.querySelector('.ls-chevron').classList.toggle('bi-chevron-up');">
          <i class="bi bi-exclamation-triangle-fill text-warning"></i>
          <span class="fw-bold">Low Stock Alert</span>
          <span class="badge bg-warning text-dark ms-1">${lowItems.length} item${lowItems.length !== 1 ? 's' : ''}</span>
          <i class="bi bi-chevron-down ls-chevron text-warning ms-auto"></i>
        </div>
        <div class="d-none" style="max-height:220px;overflow-y:auto;">
          <table class="table table-sm table-hover mb-0">
            <tbody>${lowRows}</tbody>
          </table>
        </div>
      </div>`;
  }

  const sections = Object.entries(byCategory).map(([cat, catItems]) => {
    const rows = catItems.map(i => {
      const qty = i.on_hand ?? 0;
      const reorder = i.min_level ?? 0;
      const isLow = reorder > 0 && qty < reorder;
      return `
        <tr>
          <td class="text-muted small font-monospace">${i.item_code || '—'}</td>
          <td class="fw-semibold">${i.item_name || '—'}</td>
          <td class="text-end fw-bold ${isLow ? 'text-danger' : ''}">${formatNumber(qty)}</td>
          <td class="text-muted">${i.invntry_uom || '—'}</td>
          <td class="text-end text-muted small">${reorder > 0 ? formatNumber(reorder) : '—'}</td>
          <td>${isLow ? '<span class="badge bg-danger">Low</span>' : '<span class="badge bg-success-subtle text-success border border-success-subtle">OK</span>'}</td>
          <td class="text-end d-flex gap-1 justify-content-end">
            <button class="btn btn-sm btn-outline-info" onclick="showItemWhsModal('${i.item_code}','${(i.item_name||'').replace(/'/g,"\\'")}','${i.invntry_uom||''}')" title="Per-Warehouse Breakdown">
              <i class="bi bi-building"></i>
            </button>
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/inventory/gr/new')" title="Create Goods Receipt">
              <i class="bi bi-arrow-down-circle"></i>
            </button>
          </td>
        </tr>`;
    }).join('');
    return `
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-2 fw-bold text-uppercase small text-muted">
          <i class="bi bi-folder me-2"></i>${cat}
          <span class="badge bg-secondary-subtle text-secondary border border-secondary-subtle ms-2">${catItems.length} items</span>
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Code</th><th>Item</th>
                <th class="text-end">On Hand</th><th>Unit</th>
                <th class="text-end">Reorder Level</th><th>Status</th><th class="text-end">Actions</th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>`;
  }).join('');

  const emptyMsg = items.length === 0
    ? `<div class="alert alert-info">No items found. <a href="#" onclick="navigate('#/inventory/items');return false;">Add items</a> in the Item Master.</div>`
    : '';

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-3 flex-wrap gap-2">
        <h4 class="fw-bold mb-0"><i class="bi bi-clipboard-data me-2 text-warning"></i>Stock Overview</h4>
        <div class="d-flex gap-2">
          <button class="btn btn-sm btn-outline-primary" onclick="navigate('#/inventory/gr/new')">
            <i class="bi bi-arrow-down-circle me-1"></i>Goods Receipt
          </button>
          <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/inventory/gi/new')">
            <i class="bi bi-arrow-up-circle me-1"></i>Goods Issue
          </button>
        </div>
      </div>
      ${tabBar}
      ${lowAlert}
      ${emptyMsg}
      ${sections}
    </div>
  `);
}

// ── Per-Item Warehouse Breakdown Modal ───────────────────────────────────────

window.showItemWhsModal = async function(itemCode, itemName, unit) {
  await _renderItemWhsModal(itemCode, itemName, unit);
};

async function _renderItemWhsModal(itemCode, itemName, unit) {
  const rows = await api.GetItemWarehouseStock(itemCode);
  const data = rows || [];

  const tbody = data.length === 0
    ? `<tr><td colspan="9" class="text-center text-muted py-3 small">No warehouse records found.</td></tr>`
    : data.map(r => {
        const avail  = (r.on_hand || 0) - (r.is_commited || 0) + (r.on_order || 0);
        const isLow  = (r.min_stock || 0) > 0 && (r.on_hand || 0) < (r.min_stock || 0);
        const isZero = (r.on_hand || 0) === 0;
        return `<tr data-item="${r.item_code}" data-whs="${r.whs_code}">
          <td class="fw-semibold small">${r.whs_code}</td>
          <td class="text-end ${isZero ? 'text-danger fw-bold' : ''}">${formatNumber(r.on_hand || 0)}</td>
          <td class="text-end text-muted">${formatNumber(r.is_commited || 0)}</td>
          <td class="text-end text-muted">${formatNumber(r.on_order || 0)}</td>
          <td class="text-end fw-bold ${avail < 0 ? 'text-danger' : 'text-success'}">${formatNumber(avail)}</td>
          <td class="text-end oitw-min-cell">
            <span class="oitw-min-display">${(r.min_stock || 0) > 0 ? formatNumber(r.min_stock) : '—'}</span>
            <input type="number" min="0" step="any" class="form-control form-control-sm d-none oitw-min-input" value="${r.min_stock || 0}" style="width:90px;display:inline-block!important;">
          </td>
          <td class="text-end oitw-max-cell">
            <span class="oitw-max-display">${(r.max_stock || 0) > 0 ? formatNumber(r.max_stock) : '—'}</span>
            <input type="number" min="0" step="any" class="form-control form-control-sm d-none oitw-max-input" value="${r.max_stock || 0}" style="width:90px;display:inline-block!important;">
          </td>
          <td>${isLow ? '<span class="badge bg-danger">Low</span>' : '<span class="badge bg-success-subtle text-success border border-success-subtle">OK</span>'}</td>
          <td>
            <button class="btn btn-outline-warning btn-sm oitw-edit-btn py-0 px-2" style="font-size:.75rem;"
                    onclick="_oitwStartEdit(this)">Edit</button>
            <button class="btn btn-warning btn-sm oitw-save-btn d-none py-0 px-2" style="font-size:.75rem;"
                    onclick="_oitwSaveEdit(this,'${itemCode}','${itemName}','${unit}')">Save</button>
            <button class="btn btn-outline-secondary btn-sm oitw-cancel-btn d-none py-0 px-2" style="font-size:.75rem;"
                    onclick="_oitwCancelEdit(this)">Cancel</button>
          </td>
        </tr>`;
      }).join('');

  const html = `
    <div class="modal fade" id="itemWhsModal" tabindex="-1" aria-hidden="true">
      <div class="modal-dialog modal-xl">
        <div class="modal-content">
          <div class="modal-header">
            <h6 class="modal-title fw-bold">
              <i class="bi bi-building me-2 text-warning"></i>${itemName}
              <span class="text-muted fw-normal small ms-2">— warehouse breakdown</span>
            </h6>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body p-0">
            <div class="alert alert-info mb-0 rounded-0 border-0 border-bottom py-2 px-3 small">
              <i class="bi bi-info-circle me-1"></i>
              <strong>Available = On Hand − Committed + On Order</strong> &nbsp;·&nbsp; Unit: <strong>${unit || 'base UoM'}</strong>
              &nbsp;·&nbsp; Click <strong>Edit</strong> on a row to set Min/Max thresholds.
            </div>
            <div class="table-responsive">
              <table class="table table-hover align-middle mb-0">
                <thead class="table-light">
                  <tr>
                    <th>Warehouse</th>
                    <th class="text-end">On Hand</th>
                    <th class="text-end">Committed</th>
                    <th class="text-end">On Order</th>
                    <th class="text-end">Available</th>
                    <th class="text-end">Min Stock</th>
                    <th class="text-end">Max Stock</th>
                    <th>Status</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>${tbody}</tbody>
              </table>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-sm btn-secondary" data-bs-dismiss="modal">Close</button>
          </div>
        </div>
      </div>
    </div>`;

  document.getElementById('itemWhsModal')?.remove();
  document.body.insertAdjacentHTML('beforeend', html);
  new bootstrap.Modal(document.getElementById('itemWhsModal')).show();
}

window._oitwStartEdit = function(btn) {
  const row = btn.closest('tr');
  row.querySelector('.oitw-min-display').classList.add('d-none');
  row.querySelector('.oitw-max-display').classList.add('d-none');
  row.querySelector('.oitw-min-input').classList.remove('d-none');
  row.querySelector('.oitw-max-input').classList.remove('d-none');
  btn.classList.add('d-none');
  row.querySelector('.oitw-save-btn').classList.remove('d-none');
  row.querySelector('.oitw-cancel-btn').classList.remove('d-none');
  row.querySelector('.oitw-min-input').focus();
};

window._oitwCancelEdit = function(btn) {
  const row = btn.closest('tr');
  row.querySelector('.oitw-min-display').classList.remove('d-none');
  row.querySelector('.oitw-max-display').classList.remove('d-none');
  row.querySelector('.oitw-min-input').classList.add('d-none');
  row.querySelector('.oitw-max-input').classList.add('d-none');
  row.querySelector('.oitw-edit-btn').classList.remove('d-none');
  btn.classList.add('d-none');
  row.querySelector('.oitw-save-btn').classList.add('d-none');
};

window._oitwSaveEdit = async function(btn, itemCode, itemName, unit) {
  const row      = btn.closest('tr');
  const whsCode  = row.dataset.whs;
  const minStock = parseFloat(row.querySelector('.oitw-min-input').value) || 0;
  const maxStock = parseFloat(row.querySelector('.oitw-max-input').value) || 0;
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm"></span>';
  const res = await api.UpdateOITWMinMax(itemCode, whsCode, minStock, maxStock);
  btn.disabled = false;
  btn.innerHTML = 'Save';
  if (!res || !res.success) {
    showToast('error', res?.error || 'Failed to update thresholds.');
    return;
  }
  showToast('success', 'Stock thresholds updated.');
  // Refresh the modal
  document.getElementById('itemWhsModal')?.remove();
  await _renderItemWhsModal(itemCode, itemName, unit);
};

// ── Warehouse View (OITW) ─────────────────────────────────────────────────────

async function loadWarehouseView(tabBar) {
  const [warehouses, oitw] = await Promise.all([
    api.ListWarehouses(),
    api.ListAllWarehouseStock(),
  ]);

  const whsList = warehouses || [];
  const rows    = oitw || [];

  const whsSummary = whsList.map(w => {
    const whsRows = rows.filter(r => r.whs_code === w.whs_code);
    const total   = whsRows.reduce((s, r) => s + (r.on_hand || 0), 0);
    const lowCnt  = whsRows.filter(r => (r.min_stock || 0) > 0 && (r.on_hand || 0) < r.min_stock).length;
    const isInactive = w.inactive === 'Y';
    return `
      <div class="col-sm-6 col-xl-3">
        <div class="card border-0 shadow-sm h-100"
             style="cursor:pointer;"
             onclick="showWhsItemsModal('${w.whs_code}','${(w.whs_name||'').replace(/'/g,"\\'")}')">
          <div class="card-body">
            <div class="d-flex align-items-start justify-content-between mb-1">
              <div>
                <div class="fw-bold">${w.whs_name}</div>
                <div class="text-muted small font-monospace">${w.whs_code}</div>
                ${w.location ? `<div class="text-muted small"><i class="bi bi-geo-alt me-1"></i>${w.location}</div>` : ''}
                ${w.city || w.state ? `<div class="text-muted small"><i class="bi bi-map me-1"></i>${[w.city, w.state].filter(Boolean).join(', ')}</div>` : ''}
                ${w.phone ? `<div class="text-muted small"><i class="bi bi-telephone me-1"></i>${w.phone}</div>` : ''}
              </div>
              <span class="badge ${isInactive ? 'bg-secondary' : 'bg-success-subtle text-success border border-success-subtle'} ms-2">
                ${isInactive ? 'Inactive' : 'Active'}
              </span>
            </div>
            <div class="mt-2 pt-2 border-top d-flex gap-3">
              <div>
                <div class="small text-muted">Items</div>
                <div class="fw-bold">${whsRows.length}</div>
              </div>
              <div>
                <div class="small text-muted">Total On Hand</div>
                <div class="fw-bold">${formatNumber(total)}</div>
              </div>
              ${lowCnt > 0 ? `<div class="ms-auto"><span class="badge bg-danger">${lowCnt} low</span></div>` : ''}
            </div>
            <div class="d-flex gap-2 mt-2 pt-2 border-top">
              <button class="btn btn-sm btn-outline-secondary flex-fill"
                      onclick="openWarehouseModal('${w.whs_code}');event.stopPropagation()">
                <i class="bi bi-pencil me-1"></i>Edit
              </button>
              <button class="btn btn-sm ${isInactive ? 'btn-outline-success' : 'btn-outline-danger'} flex-fill"
                      onclick="toggleWhsInactive('${w.whs_code}','${w.inactive}');event.stopPropagation()">
                ${isInactive ? '<i class="bi bi-check-circle me-1"></i>Activate' : '<i class="bi bi-x-circle me-1"></i>Deactivate'}
              </button>
            </div>
          </div>
        </div>
      </div>`;
  }).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-3">
        <h4 class="fw-bold mb-0"><i class="bi bi-building me-2 text-warning"></i>Warehouses</h4>
        <button class="btn btn-warning btn-sm" onclick="openWarehouseModal()">
          <i class="bi bi-plus-lg me-1"></i>New Warehouse
        </button>
      </div>
      ${tabBar}

      <div class="alert alert-info py-2 px-3 small mb-3">
        <i class="bi bi-info-circle me-1"></i>
        Click a warehouse card to view its items and stock levels.
      </div>

      <div class="row g-3 mb-4">
        ${whsSummary}
      </div>
      ${whsList.length === 0 ? '<div class="text-center text-muted py-5">No warehouses found. Click <strong>New Warehouse</strong> to create one.</div>' : ''}
    </div>

    <!-- Warehouse Create/Edit Modal -->
    <div class="modal fade" id="whsModal" tabindex="-1">
      <div class="modal-dialog modal-lg">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title" id="whsModalTitle">New Warehouse</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <div class="row g-3">
              <div class="col-md-4">
                <label class="form-label fw-semibold">Warehouse Code <span class="text-danger">*</span></label>
                <input type="text" id="whsCode" class="form-control text-uppercase" maxlength="10" placeholder="e.g. WH01">
              </div>
              <div class="col-md-8">
                <label class="form-label fw-semibold">Warehouse Name <span class="text-danger">*</span></label>
                <input type="text" id="whsName" class="form-control" placeholder="e.g. Main Warehouse">
              </div>
              <div class="col-12">
                <label class="form-label fw-semibold">Location / Notes</label>
                <input type="text" id="whsLocation" class="form-control" placeholder="General location description">
              </div>
              <div class="col-md-8">
                <label class="form-label fw-semibold">Street Address</label>
                <input type="text" id="whsStreet" class="form-control" placeholder="Street and number">
              </div>
              <div class="col-md-4">
                <label class="form-label fw-semibold">ZIP / Postal Code</label>
                <input type="text" id="whsZipCode" class="form-control" placeholder="ZIP">
              </div>
              <div class="col-md-6">
                <label class="form-label fw-semibold">City</label>
                <input type="text" id="whsCity" class="form-control" placeholder="City">
              </div>
              <div class="col-md-6">
                <label class="form-label fw-semibold">State / Province</label>
                <input type="text" id="whsState" class="form-control" placeholder="State">
              </div>
              <div class="col-md-6">
                <label class="form-label fw-semibold">Phone</label>
                <input type="text" id="whsPhone" class="form-control" placeholder="Contact phone">
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-warning" onclick="saveWarehouse()">
              <i class="bi bi-floppy me-1"></i>Save
            </button>
          </div>
        </div>
      </div>
    </div>
  `);
}

window.showWhsItemsModal = async function(whsCode, whsName) {
  const [stock, items] = await Promise.all([
    api.GetWarehouseStock(whsCode),
    api.ListItemMasters(),
  ]);

  const data     = stock || [];
  const itemsMap = Object.fromEntries((items || []).map(i => [i.item_code, i]));

  const totalOnHand = data.reduce((s, r) => s + (r.on_hand || 0), 0);
  const lowCount    = data.filter(r => (r.min_stock || 0) > 0 && (r.on_hand || 0) < r.min_stock).length;

  const tbody = data.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No items in this warehouse.</td></tr>`
    : data.map(r => {
        const item  = itemsMap[r.item_code] || {};
        const avail = (r.on_hand || 0) - (r.is_commited || 0) + (r.on_order || 0);
        const isLow = (r.min_stock || 0) > 0 && (r.on_hand || 0) < r.min_stock;
        const isZero = (r.on_hand || 0) === 0;
        return `<tr>
          <td class="text-muted small font-monospace">${r.item_code}</td>
          <td class="fw-semibold">${item.item_name || r.item_code}</td>
          <td class="text-muted small">${item.invntry_uom || ''}</td>
          <td class="text-end ${isZero ? 'text-danger fw-bold' : ''}">${formatNumber(r.on_hand || 0)}</td>
          <td class="text-end text-muted">${formatNumber(r.is_commited || 0)}</td>
          <td class="text-end text-muted">${formatNumber(r.on_order || 0)}</td>
          <td class="text-end fw-bold ${avail < 0 ? 'text-danger' : ''}">${formatNumber(avail)}</td>
          <td>${isLow ? '<span class="badge bg-danger">Low</span>' : '<span class="badge bg-success-subtle text-success border border-success-subtle">OK</span>'}</td>
        </tr>`;
      }).join('');

  const html = `
    <div class="modal fade" id="whsItemsModal" tabindex="-1" aria-hidden="true">
      <div class="modal-dialog modal-xl">
        <div class="modal-content">
          <div class="modal-header">
            <h6 class="modal-title fw-bold">
              <i class="bi bi-building me-2 text-warning"></i>${whsName}
              <span class="text-muted fw-normal small ms-2 font-monospace">${whsCode}</span>
            </h6>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body p-0">
            <div class="d-flex gap-4 px-3 py-2 bg-light border-bottom small">
              <div><span class="text-muted">Items:</span> <strong>${data.length}</strong></div>
              <div><span class="text-muted">Total On Hand:</span> <strong>${formatNumber(totalOnHand)}</strong></div>
              ${lowCount > 0 ? `<div><span class="badge bg-danger">${lowCount} low stock</span></div>` : ''}
            </div>
            <div class="alert alert-info mb-0 rounded-0 border-0 border-bottom py-2 px-3 small">
              <i class="bi bi-info-circle me-1"></i>
              <strong>Available = On Hand - Committed + On Order</strong>
            </div>
            <div class="table-responsive">
              <table class="table table-hover align-middle mb-0">
                <thead class="table-light">
                  <tr>
                    <th>Item Code</th>
                    <th>Item Name</th>
                    <th>Unit</th>
                    <th class="text-end">On Hand</th>
                    <th class="text-end">Committed</th>
                    <th class="text-end">On Order</th>
                    <th class="text-end">Available</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>${tbody}</tbody>
              </table>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-sm btn-secondary" data-bs-dismiss="modal">Close</button>
          </div>
        </div>
      </div>
    </div>`;

  document.getElementById('whsItemsModal')?.remove();
  document.body.insertAdjacentHTML('beforeend', html);
  new bootstrap.Modal(document.getElementById('whsItemsModal')).show();
};

let _whsModalMode = 'create'; // 'create' | 'edit'
let _whsEditCode  = '';

window.openWarehouseModal = async function(whsCode) {
  _whsModalMode = whsCode ? 'edit' : 'create';
  _whsEditCode  = whsCode || '';

  ['whsCode','whsName','whsLocation','whsStreet','whsZipCode','whsCity','whsState','whsPhone']
    .forEach(id => { const el = document.getElementById(id); if (el) el.value = ''; });

  document.getElementById('whsModalTitle').textContent =
    _whsModalMode === 'edit' ? 'Edit Warehouse' : 'New Warehouse';
  document.getElementById('whsCode').disabled = (_whsModalMode === 'edit');

  if (_whsModalMode === 'edit') {
    const res = await api.GetWarehouse(whsCode);
    if (res) {
      document.getElementById('whsCode').value     = res.whs_code || '';
      document.getElementById('whsName').value     = res.whs_name || '';
      document.getElementById('whsLocation').value = res.location || '';
      document.getElementById('whsStreet').value   = res.street   || '';
      document.getElementById('whsZipCode').value  = res.zip_code || '';
      document.getElementById('whsCity').value     = res.city     || '';
      document.getElementById('whsState').value    = res.state    || '';
      document.getElementById('whsPhone').value    = res.phone    || '';
    }
  }

  new bootstrap.Modal(document.getElementById('whsModal')).show();
};

window.saveWarehouse = async function() {
  const code = document.getElementById('whsCode').value.trim().toUpperCase();
  const name = document.getElementById('whsName').value.trim();
  if (!code) { toast('Warehouse Code is required.', 'warning'); return; }
  if (!name) { toast('Warehouse Name is required.',  'warning'); return; }

  const payload = {
    whs_code: code,
    whs_name: name,
    location: document.getElementById('whsLocation').value.trim(),
    street:   document.getElementById('whsStreet').value.trim(),
    zip_code: document.getElementById('whsZipCode').value.trim(),
    city:     document.getElementById('whsCity').value.trim(),
    state:    document.getElementById('whsState').value.trim(),
    phone:    document.getElementById('whsPhone').value.trim(),
    inactive: 'N',
  };

  let ok;
  if (_whsModalMode === 'create') {
    ok = await api.CreateWarehouse(payload);
  } else {
    const { whs_code, ...updates } = payload;
    ok = await api.UpdateWarehouse(_whsEditCode, updates);
  }

  if (ok) {
    toast(_whsModalMode === 'create' ? 'Warehouse created.' : 'Warehouse updated.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('whsModal')).hide();
    navigate('#/inventory/warehouses');
  }
};

window.toggleWhsInactive = async function(whsCode, currentInactive) {
  const newVal = currentInactive === 'Y' ? 'N' : 'Y';
  const label  = newVal === 'Y' ? 'deactivate' : 'activate';
  if (!confirm(`Are you sure you want to ${label} warehouse ${whsCode}?`)) return;
  const ok = await api.UpdateWarehouse(whsCode, { inactive: newVal });
  if (ok) {
    toast(`Warehouse ${whsCode} ${label}d.`, 'success');
    navigate('#/inventory/warehouses');
  }
};

// ── Item Master ───────────────────────────────────────────────────────────────

const IM_CAT_COLOR = { Feed:'success', Vaccine:'primary', Medicine:'danger', Packaging:'warning', Supplies:'secondary', Fuel:'dark' };

let _itemMasterList = [];
let _uomMasterList  = [];

function _imRows(list) {
  if (list.length === 0)
    return `<tr><td colspan="8" class="text-center text-muted py-4">No items match the filter.</td></tr>`;
  return list.map(i => {
    const badgeClass = IM_CAT_COLOR[i.category_name] || 'secondary';
    const uomLabel   = i.base_uom ? i.base_uom.uom_code : (i.i_uom_entry ? `#${i.i_uom_entry}` : '<span class="text-danger small">Not set</span>');
    const suomLabel  = i.sales_uom  ? `S:${i.sales_uom.uom_code}`  : '';
    const puomLabel  = i.purch_uom  ? `P:${i.purch_uom.uom_code}`  : '';
    const uomTitle   = [suomLabel, puomLabel].filter(Boolean).join(' / ');
    return `
      <tr>
        <td class="text-muted small font-monospace">${i.item_code || '—'}</td>
        <td class="fw-semibold">${i.item_name || '—'}</td>
        <td><span class="badge bg-${badgeClass}-subtle text-${badgeClass} border border-${badgeClass}-subtle">${i.category_name || '—'}</span></td>
        <td>${i.invntry_uom || '—'}</td>
        <td title="${uomTitle}">${uomLabel}${uomTitle ? `<span class="text-muted small ms-1">(${uomTitle})</span>` : ''}</td>
        <td class="text-end">₱${formatNumber(i.avg_price ?? 0)}</td>
        <td class="text-end">${formatNumber(i.min_level ?? 0)}</td>
        <td>
          <button class="btn btn-sm btn-outline-secondary" onclick="openItemModal(${i.id})">
            <i class="bi bi-pencil"></i>
          </button>
        </td>
      </tr>`;
  }).join('');
}

window.imFilter = function() {
  const search = (document.getElementById('im-search')?.value || '').toLowerCase();
  const cat    = (document.getElementById('im-cat')?.value || '');
  const unit   = (document.getElementById('im-unit')?.value || '').toLowerCase();

  const filtered = _itemMasterList.filter(i => {
    if (search && !(i.item_name || '').toLowerCase().includes(search) &&
                  !(i.item_code || '').toLowerCase().includes(search) &&
                  !(i.description || '').toLowerCase().includes(search)) return false;
    if (cat  && i.category_name !== cat) return false;
    if (unit && !(i.invntry_uom || '').toLowerCase().includes(unit)) return false;
    return true;
  });

  const tbody = document.getElementById('im-tbody');
  if (tbody) tbody.innerHTML = _imRows(filtered);

  const countEl = document.getElementById('im-count');
  if (countEl) countEl.textContent = `${filtered.length} of ${_itemMasterList.length} item${_itemMasterList.length !== 1 ? 's' : ''}`;
};

window.imClearFilter = function() {
  ['im-search', 'im-unit'].forEach(id => { const el = document.getElementById(id); if (el) el.value = ''; });
  const cat = document.getElementById('im-cat'); if (cat) cat.value = '';
  window.imFilter();
};

async function loadItemMaster(tabBar) {
  const [items, uoms, cats, groups] = await Promise.all([
    api.ListItemMasters(),
    api.ListUoMMasters(),
    api.ListItemCategories(),
    api.ListUoMGroups(),
  ]);
  _itemMasterList    = items  || [];
  _uomMasterList     = uoms   || [];
  _itemCategoryList  = cats   || [];
  _uomGroupList      = groups || [];

  // Build dynamic filter options from the loaded data
  const filterCats  = [...new Set(_itemMasterList.map(i => i.category_name).filter(Boolean))].sort();
  const filterUnits = [...new Set(_itemMasterList.map(i => i.invntry_uom).filter(Boolean))].sort();

  const catOptions  = `<option value="">All Categories</option>` + filterCats.map(c => `<option value="${c}">${c}</option>`).join('');
  const unitOptions = `<option value="">All Units</option>` + filterUnits.map(u => `<option value="${u}">${u}</option>`).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-4">
        <h4 class="fw-bold mb-0"><i class="bi bi-list-ul me-2"></i>Inventory</h4>
        <button class="btn btn-primary" onclick="openItemModal(null)">
          <i class="bi bi-plus-lg me-1"></i>Add Item
        </button>
      </div>
      ${tabBar}

      <!-- Filter Bar -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body py-2">
          <div class="row g-2 align-items-end">
            <div class="col-sm-4">
              <label class="form-label small mb-1">Search</label>
              <input id="im-search" class="form-control form-control-sm" placeholder="Name, code, or description…"
                     oninput="imFilter()" autocomplete="off">
            </div>
            <div class="col-sm-3">
              <label class="form-label small mb-1">Category</label>
              <select id="im-cat" class="form-select form-select-sm" onchange="imFilter()">
                ${catOptions}
              </select>
            </div>
            <div class="col-sm-3">
              <label class="form-label small mb-1">Unit</label>
              <select id="im-unit" class="form-select form-select-sm" onchange="imFilter()">
                ${unitOptions}
              </select>
            </div>
            <div class="col-sm-2 d-flex align-items-end gap-2">
              <button class="btn btn-outline-secondary btn-sm w-100" onclick="imClearFilter()">
                <i class="bi bi-x-circle me-1"></i>Clear
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="d-flex justify-content-end mb-2">
        <small class="text-muted" id="im-count">${_itemMasterList.length} item${_itemMasterList.length !== 1 ? 's' : ''}</small>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Code</th>
                <th>Name</th>
                <th>Category</th>
                <th>Unit</th>
                <th>Inv. UoM <span class="text-danger">*</span></th>
                <th class="text-end">Unit Price</th>
                <th class="text-end">Reorder Lvl</th>
                <th></th>
              </tr>
            </thead>
            <tbody id="im-tbody">${_imRows(_itemMasterList)}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Item Modal -->
    <div class="modal fade" id="itemModal" tabindex="-1">
      <div class="modal-dialog modal-lg modal-dialog-scrollable">
        <div class="modal-content border-0 shadow">
          <div class="modal-header border-0 pb-0">
            <div>
              <h5 class="modal-title fw-bold" id="itemModalTitle">Item</h5>
              <p class="text-muted small mb-0" id="itemModalSubtitle">Fill in the item details below</p>
            </div>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body pt-3">
            <form id="itemForm">
              <input type="hidden" id="itemId">

              <!-- Row 1: Code + Category -->
              <div class="row g-3 mb-3">
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Item Code</label>
                  <input type="text" class="form-control" id="itemCode"
                         placeholder="e.g. FEED-001" style="font-family:monospace">
                  <div class="form-text">Leave blank to auto-assign</div>
                </div>
                <div class="col-md-8">
                  <label class="form-label small fw-bold">Category <span class="text-danger">*</span></label>
                  <input type="text" class="form-control" id="itemCategory"
                         list="itemCategoryList" placeholder="Select or type a category" required>
                  <datalist id="itemCategoryList">
                    ${_itemCategoryList.filter(c => c.itms_grp_nam).map(c => `<option value="${c.itms_grp_nam}">`).join('')}
                  </datalist>
                </div>
              </div>

              <!-- Row 2: Name (full width) -->
              <div class="mb-3">
                <label class="form-label small fw-bold">Item Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="itemName"
                       placeholder="e.g. Layer Mash (Commercial)" required>
              </div>

              <!-- Row 3: Unit + Inventory UoM -->
              <div class="row g-3 mb-3">
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Unit <span class="text-danger">*</span></label>
                  <input type="text" class="form-control" id="itemUnit"
                         list="itemUnitList" placeholder="e.g. kg, bag, piece" required>
                  <datalist id="itemUnitList">
                    <option value="kg">
                    <option value="bag (50kg)">
                    <option value="sack">
                    <option value="piece">
                    <option value="liter">
                    <option value="dose">
                    <option value="roll">
                    <option value="cylinder">
                    <option value="pack">
                    <option value="box">
                  </datalist>
                </div>
                <div class="col-md-8">
                  <label class="form-label small fw-bold">
                    Inventory UoM <span class="text-danger">*</span>
                    <span class="text-muted fw-normal ms-1 small">— the base unit for stock valuation</span>
                  </label>
                  <select class="form-select" id="itemIUoMEntry" required>
                    <option value="">— Select a UoM —</option>
                    ${_uomMasterList.map(u => `<option value="${u.uom_entry}">${u.uom_code}${u.uom_name && u.uom_name !== u.uom_code ? ' — ' + u.uom_name : ''}</option>`).join('')}
                  </select>
                  <div class="form-text">Determines what "1" in stock means (e.g. 1 Piece, 1 Box).</div>
                </div>
              </div>

              <!-- Row 3b: Sales UoM + Purchase UoM + UoM Group -->
              <div class="row g-3 mb-3">
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Sales UoM</label>
                  <select class="form-select" id="itemSUoMEntry">
                    <option value="0">— Same as Inventory —</option>
                    ${_uomMasterList.map(u => `<option value="${u.uom_entry}">${u.uom_code}${u.uom_name && u.uom_name !== u.uom_code ? ' — ' + u.uom_name : ''}</option>`).join('')}
                  </select>
                  <div class="form-text">Default UoM on sales documents</div>
                </div>
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Purchase UoM</label>
                  <select class="form-select" id="itemPUoMEntry">
                    <option value="0">— Same as Inventory —</option>
                    ${_uomMasterList.map(u => `<option value="${u.uom_entry}">${u.uom_code}${u.uom_name && u.uom_name !== u.uom_code ? ' — ' + u.uom_name : ''}</option>`).join('')}
                  </select>
                  <div class="form-text">Default UoM on purchase documents</div>
                </div>
                <div class="col-md-4">
                  <label class="form-label small fw-bold">UoM Group</label>
                  <select class="form-select" id="itemUgpEntry" onchange="itemUgpChanged(this)">
                    <option value="0">— None —</option>
                    ${_uomGroupList.map(g => `<option value="${g.ugp_entry}">${g.ugp_code} — ${g.ugp_name}</option>`).join('')}
                  </select>
                  <div class="form-text">Conversion rules for this item</div>
                </div>
              </div>

              <!-- Row 4: Unit Price + Reorder Level -->
              <div class="row g-3 mb-3">
                <div class="col-md-6">
                  <label class="form-label small fw-bold">Unit Price (₱)</label>
                  <div class="input-group">
                    <span class="input-group-text">₱</span>
                    <input type="number" class="form-control" id="itemUnitPrice"
                           min="0" step="0.01" placeholder="0.00">
                  </div>
                </div>
                <div class="col-md-6">
                  <label class="form-label small fw-bold">Reorder Level</label>
                  <input type="number" class="form-control" id="itemReorderLevel"
                         min="0" step="0.001" placeholder="0">
                  <div class="form-text">Alert when stock falls below this</div>
                </div>
              </div>

              <!-- Row 4: Description -->
              <div class="mb-3">
                <label class="form-label small fw-bold">Description</label>
                <textarea class="form-control" id="itemDescription" rows="2"
                          placeholder="Brief description of the item…"></textarea>
              </div>

              <!-- Flags -->
              <div class="d-flex gap-4">
                <div class="form-check form-switch">
                  <input class="form-check-input" type="checkbox" id="itemIsProductionItem">
                  <label class="form-check-label small" for="itemIsProductionItem">
                    <i class="bi bi-gear me-1 text-warning"></i>Production Item
                    <span class="text-muted fw-normal">— available as raw material / output in milling orders</span>
                  </label>
                </div>
              </div>

              <!-- UoM Prices (ITM9) -->
              <div id="itemUoMPricesSection" class="d-none mt-3">
                <hr class="my-3">
                <h6 class="fw-semibold mb-2">
                  <i class="bi bi-tag me-1"></i>UoM Prices
                  <small class="text-muted fw-normal ms-2">— explicit price per alternate unit</small>
                </h6>
                <div id="itemUoMPricesTable"></div>
              </div>

            </form>
          </div>
          <div class="modal-footer border-0">
            <button class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary px-4" onclick="submitItemForm()">
              <i class="bi bi-check-lg me-1"></i>Save Item
            </button>
          </div>
        </div>
      </div>
    </div>
  `);
}

function openItemModal(id) {
  const isEdit = id !== null;
  document.getElementById('itemModalTitle').textContent = isEdit ? 'Edit Item' : 'New Item';
  document.getElementById('itemModalSubtitle').textContent = isEdit
    ? 'Update the item details below'
    : 'Fill in the item details below';
  document.getElementById('itemId').value = id || '';
  if (isEdit) {
    const item = _itemMasterList.find(i => i.id === id);
    if (item) {
      document.getElementById('itemCode').value         = item.item_code || '';
      document.getElementById('itemName').value         = item.item_name || '';
      document.getElementById('itemCategory').value     = item.category_name || '';
      document.getElementById('itemUnit').value         = item.invntry_uom || '';
      document.getElementById('itemIUoMEntry').value    = item.i_uom_entry || '';
      document.getElementById('itemSUoMEntry').value    = item.s_uom_entry || 0;
      document.getElementById('itemPUoMEntry').value    = item.p_uom_entry || 0;
      document.getElementById('itemUgpEntry').value     = item.ugp_entry   || 0;
      document.getElementById('itemUnitPrice').value          = item.avg_price ?? '';
      document.getElementById('itemReorderLevel').value        = item.min_level ?? '';
      document.getElementById('itemDescription').value         = item.description || '';
      document.getElementById('itemIsProductionItem').checked  = item.mak_item === 'Y';
      renderItemUoMPricesTable(item.ugp_entry || 0, item.uom_prices || []);
    }
  } else {
    document.getElementById('itemForm').reset();
    document.getElementById('itemIsProductionItem').checked = false;
    renderItemUoMPricesTable(0, []);
  }
  new bootstrap.Modal(document.getElementById('itemModal')).show();
}

async function submitItemForm() {
  const id         = document.getElementById('itemId').value;
  const iUoMEntry  = parseInt(document.getElementById('itemIUoMEntry').value, 10) || 0;
  const categoryName = document.getElementById('itemCategory').value.trim();
  const categoryObj  = _itemCategoryList.find(c => c.itms_grp_nam === categoryName);
  const payload = {
    item_code:    document.getElementById('itemCode').value.trim(),
    item_name:    document.getElementById('itemName').value.trim(),
    itms_grp_cod: categoryObj ? categoryObj.itms_grp_cod : 0,
    invntry_uom:  document.getElementById('itemUnit').value.trim(),
    i_uom_entry:  iUoMEntry,
    s_uom_entry:  parseInt(document.getElementById('itemSUoMEntry').value, 10) || 0,
    p_uom_entry:  parseInt(document.getElementById('itemPUoMEntry').value, 10) || 0,
    ugp_entry:    parseInt(document.getElementById('itemUgpEntry').value,  10) || 0,
    avg_price:    parseFloat(document.getElementById('itemUnitPrice').value) || 0,
    min_level:    parseFloat(document.getElementById('itemReorderLevel').value) || 0,
    description:  document.getElementById('itemDescription').value.trim(),
    mak_item:     document.getElementById('itemIsProductionItem').checked ? 'Y' : 'N',
  };
  if (!payload.item_name)    { toast('Item name is required.', 'warning'); return; }
  if (!payload.itms_grp_cod) { toast('Category is required.', 'warning'); return; }
  if (!payload.invntry_uom)  { toast('Unit is required.', 'warning'); return; }
  if (!payload.i_uom_entry)  { toast('Inventory UoM is required — select the base unit for stock valuation.', 'warning'); return; }

  const result = id
    ? await api.UpdateItemMaster(parseInt(id, 10), payload)
    : await api.CreateItemMaster(payload);

  if (result) {
    // Save UoM prices (ITM9) keyed by item_code
    const itemCode = document.getElementById('itemCode').value.trim();
    if (itemCode) {
      const priceInputs = document.querySelectorAll('#itemUoMPricesTable .itm9-price');
      for (const input of priceInputs) {
        const uomEntry = parseInt(input.dataset.uomEntry, 10);
        const price    = parseFloat(input.value) || 0;
        const factorEl = document.querySelector(`#itemUoMPricesTable .itm9-factor[data-uom-entry="${uomEntry}"]`);
        const factor   = parseFloat(factorEl?.value) || 0;
        await api.UpsertItemUoMPrice({ item_code: itemCode, uom_entry: uomEntry, price, factor });
      }
    }
    toast(id ? 'Item updated.' : 'Item created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('itemModal')).hide();
    Modules.Inventory.load('items');
  }
  // On failure, api.js already shows the specific server error for 8 seconds
}

function itemUgpChanged(sel) {
  renderItemUoMPricesTable(parseInt(sel.value) || 0, []);
}

function renderItemUoMPricesTable(ugpEntry, existingPrices) {
  const section  = document.getElementById('itemUoMPricesSection');
  const tableDiv = document.getElementById('itemUoMPricesTable');
  if (!section || !tableDiv) return;

  if (!ugpEntry) {
    section.classList.add('d-none');
    return;
  }

  const group = (_uomGroupList || []).find(g => g.ugp_entry === ugpEntry);
  if (!group || !(group.lines || []).length) {
    section.classList.add('d-none');
    return;
  }
  section.classList.remove('d-none');

  const baseCode = group.base_unit?.uom_code || '—';
  const baseName = group.base_unit?.uom_name || baseCode;

  const altRows = (group.lines || []).map(line => {
    const uomEntry = line.uom_entry;
    const uomCode  = line.unit?.uom_code || '—';
    const uomName  = line.unit?.uom_name || uomCode;
    const existing = (existingPrices || []).find(p => p.uom_entry === uomEntry);
    const price    = existing ? existing.price : '';
    const factor   = existing ? existing.factor : 0;
    return `
      <tr>
        <td>${uomName} <small class="text-muted">(${uomCode})</small></td>
        <td>
          <div class="input-group input-group-sm">
            <span class="input-group-text">₱</span>
            <input type="number" class="form-control itm9-price"
                   data-uom-entry="${uomEntry}"
                   value="${price}" min="0" step="0.01" placeholder="0.00">
          </div>
        </td>
        <td>
          <div class="input-group input-group-sm">
            <input type="number" class="form-control itm9-factor"
                   data-uom-entry="${uomEntry}"
                   value="${factor}" min="0" max="100" step="0.01" placeholder="0">
            <span class="input-group-text">%</span>
          </div>
        </td>
      </tr>`;
  }).join('');

  tableDiv.innerHTML = `
    <table class="table table-sm table-bordered align-middle mb-0">
      <thead class="table-light">
        <tr>
          <th style="width:35%">Unit of Measure</th>
          <th>Price (₱) <small class="fw-normal text-muted">— leave 0 to auto-calculate</small></th>
          <th style="width:22%">Reduce By %</th>
        </tr>
      </thead>
      <tbody>
        <tr class="table-light">
          <td>
            <span class="badge bg-secondary me-1">${baseCode}</span>${baseName}
            <small class="text-muted ms-1">(Base)</small>
          </td>
          <td colspan="2" class="text-muted small fst-italic">
            <i class="bi bi-link me-1"></i>Linked to "Unit Price" field above
          </td>
        </tr>
        ${altRows}
      </tbody>
    </table>`;
}

// ── Item Categories ────────────────────────────────────────────────────────────

let _itemCategoryList = [];

async function loadItemCategories(tabBar) {
  const cats = await api.ListItemCategories();
  _itemCategoryList = cats || [];

  const rows = _itemCategoryList.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No categories found.</td></tr>`
    : _itemCategoryList.map(c => `
        <tr>
          <td class="fw-semibold">${c.itms_grp_nam || '—'}</td>
          <td class="text-muted">${c.description || '—'}</td>
          <td>
            ${c.for_sales !== false ? '<span class="badge bg-primary-subtle text-primary border border-primary-subtle me-1">Sales</span>' : ''}
            ${c.for_purchasing !== false ? '<span class="badge bg-info-subtle text-info border border-info-subtle me-1">Purchasing</span>' : ''}
            ${c.for_inventory !== false ? '<span class="badge bg-success-subtle text-success border border-success-subtle me-1">Inventory</span>' : ''}
            ${c.for_production !== false ? '<span class="badge bg-warning-subtle text-warning border border-warning-subtle me-1">Production</span>' : ''}
          </td>
          <td>${c.is_active
            ? `<span class="badge bg-success-subtle text-success border border-success-subtle">Active</span>`
            : `<span class="badge bg-secondary-subtle text-secondary border border-secondary-subtle">Inactive</span>`
          }</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary me-1" onclick="openCategoryModal(${c.itms_grp_cod})">
              <i class="bi bi-pencil"></i>
            </button>
            <button class="btn btn-sm btn-outline-danger" onclick="deleteCategory(${c.itms_grp_cod}, '${(c.itms_grp_nam || '').replace(/'/g, "\\'")}')">
              <i class="bi bi-trash"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-4">
        <h4 class="fw-bold mb-0"><i class="bi bi-tags me-2"></i>Inventory</h4>
        <button class="btn btn-primary" onclick="openCategoryModal(null)">
          <i class="bi bi-plus-lg me-1"></i>Add Category
        </button>
      </div>
      ${tabBar}
      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Name</th><th>Description</th><th>Visible In</th><th>Status</th><th></th></tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Category Modal -->
    <div class="modal fade" id="categoryModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="categoryModalTitle">Item Category</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="categoryForm">
              <input type="hidden" id="categoryId">
              <div class="mb-3">
                <label class="form-label">Category Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="categoryName" required
                  placeholder="e.g. Feeds, Vaccines, Supplies">
              </div>
              <div class="mb-3">
                <label class="form-label">Description</label>
                <input type="text" class="form-control" id="categoryDescription"
                  placeholder="Optional description">
              </div>
              <div class="mb-3">
                <label class="form-label">Visible In Modules</label>
                <div class="d-flex flex-wrap gap-3">
                  <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="catForSales" checked>
                    <label class="form-check-label" for="catForSales">Sales</label>
                  </div>
                  <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="catForPurchasing" checked>
                    <label class="form-check-label" for="catForPurchasing">Purchasing</label>
                  </div>
                  <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="catForInventory" checked>
                    <label class="form-check-label" for="catForInventory">Inventory</label>
                  </div>
                  <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="catForProduction" checked>
                    <label class="form-check-label" for="catForProduction">Production</label>
                  </div>
                </div>
              </div>
            </form>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitCategoryForm()">Save</button>
          </div>
        </div>
      </div>
    </div>
  `);
}

function openCategoryModal(id) {
  const isEdit = id !== null;
  document.getElementById('categoryModalTitle').textContent = isEdit ? 'Edit Category' : 'Add Category';
  document.getElementById('categoryId').value = id || '';
  if (isEdit) {
    const cat = _itemCategoryList.find(c => c.itms_grp_cod === id);
    if (cat) {
      document.getElementById('categoryName').value = cat.itms_grp_nam || '';
      document.getElementById('categoryDescription').value = cat.description || '';
      document.getElementById('catForSales').checked = cat.for_sales !== false;
      document.getElementById('catForPurchasing').checked = cat.for_purchasing !== false;
      document.getElementById('catForInventory').checked = cat.for_inventory !== false;
      document.getElementById('catForProduction').checked = cat.for_production !== false;
    }
  } else {
    document.getElementById('categoryForm').reset();
  }
  new bootstrap.Modal(document.getElementById('categoryModal')).show();
}

async function submitCategoryForm() {
  const id = document.getElementById('categoryId').value;
  const name = document.getElementById('categoryName').value.trim();
  const description = document.getElementById('categoryDescription').value.trim();

  if (!name) { toast('Category name is required.', 'warning'); return; }

  const forSales      = document.getElementById('catForSales').checked;
  const forPurchasing = document.getElementById('catForPurchasing').checked;
  const forInventory  = document.getElementById('catForInventory').checked;
  const forProduction = document.getElementById('catForProduction').checked;

  const result = id
    ? await api.UpdateItemCategory(parseInt(id, 10), { itms_grp_nam: name, description, for_sales: forSales, for_purchasing: forPurchasing, for_inventory: forInventory, for_production: forProduction })
    : await api.CreateItemCategory({ itms_grp_nam: name, description, for_sales: forSales, for_purchasing: forPurchasing, for_inventory: forInventory, for_production: forProduction });

  if (result) {
    toast(id ? 'Category updated.' : 'Category created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('categoryModal')).hide();
    navigate('#/inventory/categories');
  } else {
    toast('Failed to save category.', 'danger');
  }
}

async function deleteCategory(id, name) {
  if (!confirm(`Deactivate category "${name}"?`)) return;
  const result = await api.DeleteItemCategory(id);
  if (result) {
    toast('Category deactivated.', 'success');
    navigate('#/inventory/categories');
  } else {
    toast('Failed to deactivate category.', 'danger');
  }
}

// ── UoM Setup (OUOM / OUGP / UGP1) ───────────────────────────────────────────

// _uomMasterList is shared with the Item Master section (declared above)
let _uomGroupList  = [];

async function loadUoMSetup(tabBar) {
  const [masters, groups] = await Promise.all([
    api.ListUoMMasters(),
    api.ListUoMGroups(),
  ]);
  _uomMasterList = masters || [];
  _uomGroupList  = groups  || [];

  const uomMasterOptions = _uomMasterList.map(u =>
    `<option value="${u.uom_entry}">${u.uom_code} — ${u.uom_name}</option>`
  ).join('');

  showView(`
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-rulers me-2"></i>Inventory</h4>
      ${tabBar}

      <!-- UoM Master (collapsible) -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-header bg-white border-bottom py-2 d-flex align-items-center justify-content-between"
             style="cursor:pointer"
             onclick="document.getElementById('ouomCollapse').classList.toggle('show'); this.querySelector('.ouom-chevron').classList.toggle('bi-chevron-right'); this.querySelector('.ouom-chevron').classList.toggle('bi-chevron-down');">
          <h6 class="fw-bold mb-0 text-muted text-uppercase small">
            <i class="bi bi-list-check me-1"></i>Unit of Measure Master (OUOM)
          </h6>
          <div class="d-flex align-items-center gap-2" onclick="event.stopPropagation()">
            <button class="btn btn-sm btn-primary" onclick="openUomMasterModal(null)">
              <i class="bi bi-plus-lg me-1"></i>Add UoM
            </button>
            <i class="bi bi-chevron-right ouom-chevron text-muted"></i>
          </div>
        </div>
        <div class="collapse" id="ouomCollapse">
          <div class="table-responsive">
            <table class="table table-hover align-middle mb-0">
              <thead class="table-light">
                <tr>
                  <th>Code</th><th>Name</th>
                  <th class="text-end">Weight</th><th class="text-end">Volume</th>
                  <th></th>
                </tr>
              </thead>
              <tbody id="uom-master-tbody">${_uomMasterRows()}</tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- UoM Groups -->
      <div class="d-flex align-items-center justify-content-between mb-2">
        <h6 class="fw-bold mb-0 text-muted text-uppercase small">
          <i class="bi bi-collection me-1"></i>UoM Groups &amp; Conversions (OUGP)
        </h6>
        <button class="btn btn-sm btn-primary" onclick="openUomGroupModal(null)">
          <i class="bi bi-plus-lg me-1"></i>Add Group
        </button>
      </div>
      <div class="card border-0 shadow-sm mb-4">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Group Code</th><th>Group Name</th>
                <th>Base Unit</th><th>Conversions</th><th></th>
              </tr>
            </thead>
            <tbody id="uom-group-tbody">${_uomGroupRows()}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- UoM Master Modal -->
    <div class="modal fade" id="uomMasterModal" tabindex="-1">
      <div class="modal-dialog modal-lg">
        <div class="modal-content border-0 shadow">
          <div class="modal-header border-0 pb-0">
            <div>
              <h5 class="modal-title fw-bold" id="uomMasterModalTitle">Unit of Measure</h5>
              <p class="text-muted small mb-0">Define a unit and its physical properties</p>
            </div>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="uomMasterForm">
              <input type="hidden" id="uomMasterEntry">
              <div class="row g-3 mb-3">
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Code <span class="text-danger">*</span></label>
                  <input type="text" class="form-control font-monospace" id="uomMasterCode"
                         placeholder="e.g. PCS, BOX, TRAY" required maxlength="20">
                </div>
                <div class="col-md-8">
                  <label class="form-label small fw-bold">Name <span class="text-danger">*</span></label>
                  <input type="text" class="form-control" id="uomMasterName"
                         placeholder="e.g. Piece, Box of 24 Pieces" required maxlength="100">
                </div>
              </div>
              <div class="row g-3 mb-3">
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Weight</label>
                  <input type="number" class="form-control" id="uomMasterWeight" min="0" step="any" placeholder="0">
                </div>
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Volume</label>
                  <input type="number" class="form-control" id="uomMasterVolume" min="0" step="any" placeholder="0">
                </div>
              </div>
              <div class="row g-3 mb-3">
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Length</label>
                  <input type="number" class="form-control" id="uomMasterLength" min="0" step="any" placeholder="0">
                </div>
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Width</label>
                  <input type="number" class="form-control" id="uomMasterWidth" min="0" step="any" placeholder="0">
                </div>
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Height</label>
                  <input type="number" class="form-control" id="uomMasterHeight" min="0" step="any" placeholder="0">
                </div>
              </div>
            </form>
          </div>
          <div class="modal-footer border-0">
            <button class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-danger me-auto d-none" id="uomMasterDeleteBtn" onclick="deleteUomMaster()">
              <i class="bi bi-trash me-1"></i>Delete
            </button>
            <button class="btn btn-primary px-4" onclick="submitUomMasterForm()">
              <i class="bi bi-check-lg me-1"></i>Save
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- UoM Group Modal -->
    <div class="modal fade" id="uomGroupModal" tabindex="-1">
      <div class="modal-dialog modal-lg modal-dialog-scrollable">
        <div class="modal-content border-0 shadow">
          <div class="modal-header border-0 pb-0">
            <div>
              <h5 class="modal-title fw-bold" id="uomGroupModalTitle">UoM Group</h5>
              <p class="text-muted small mb-0">Define a conversion group between units</p>
            </div>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="uomGroupForm">
              <input type="hidden" id="uomGroupEntry">
              <div class="row g-3 mb-3">
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Group Code <span class="text-danger">*</span></label>
                  <input type="text" class="form-control font-monospace" id="uomGroupCode"
                         placeholder="e.g. EGG-GRP" required maxlength="20">
                </div>
                <div class="col-md-8">
                  <label class="form-label small fw-bold">Group Name <span class="text-danger">*</span></label>
                  <input type="text" class="form-control" id="uomGroupName"
                         placeholder="e.g. Egg Packaging Units" required maxlength="100">
                </div>
              </div>
              <div class="mb-3">
                <label class="form-label small fw-bold">Base Unit <span class="text-danger">*</span></label>
                <select class="form-select" id="uomGroupBaseUom" required>
                  <option value="">— Select base unit —</option>
                  ${uomMasterOptions}
                </select>
                <div class="form-text">The smallest unit all conversions express quantities in (e.g. Piece).</div>
              </div>
              <hr class="my-3">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <h6 class="fw-semibold mb-0 small text-uppercase text-muted">Conversion Lines (UGP1)</h6>
                <button type="button" class="btn btn-sm btn-outline-primary" onclick="uomGroupAddLine()">
                  <i class="bi bi-plus-lg me-1"></i>Add Line
                </button>
              </div>
              <div class="table-responsive">
                <table class="table table-sm align-middle mb-0">
                  <thead class="table-light">
                    <tr>
                      <th>Unit</th>
                      <th class="text-end" style="width:110px">Alt Qty</th>
                      <th class="text-center px-1" style="width:24px">=</th>
                      <th class="text-end" style="width:110px">Base Qty</th>
                      <th style="width:36px"></th>
                    </tr>
                  </thead>
                  <tbody id="uomGroupLines"></tbody>
                </table>
              </div>
              <div class="form-text mt-2 text-muted">
                Example: 1 Box = 24 Pieces &rarr; Alt Qty=1, Unit=Box, Base Qty=24
              </div>
            </form>
          </div>
          <div class="modal-footer border-0">
            <button class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-danger me-auto d-none" id="uomGroupDeleteBtn" onclick="deleteUomGroup()">
              <i class="bi bi-trash me-1"></i>Delete
            </button>
            <button class="btn btn-primary px-4" onclick="submitUomGroupForm()">
              <i class="bi bi-check-lg me-1"></i>Save
            </button>
          </div>
        </div>
      </div>
    </div>
  `);
}

// ── UoM Master helpers ────────────────────────────────────────────────────────

function _uomMasterRows() {
  if (_uomMasterList.length === 0)
    return `<tr><td colspan="5" class="text-center text-muted py-4">No units defined yet.</td></tr>`;
  return _uomMasterList.map(u => `
    <tr>
      <td class="font-monospace fw-semibold">${u.uom_code || '—'}</td>
      <td>${u.uom_name || '—'}</td>
      <td class="text-end text-muted small">${u.weight || '—'}</td>
      <td class="text-end text-muted small">${u.volume || '—'}</td>
      <td class="text-end">
        <button class="btn btn-sm btn-outline-secondary" onclick="openUomMasterModal(${u.uom_entry})">
          <i class="bi bi-pencil"></i>
        </button>
      </td>
    </tr>`).join('');
}

function openUomMasterModal(entry) {
  const isEdit = entry !== null;
  document.getElementById('uomMasterModalTitle').textContent = isEdit ? 'Edit Unit of Measure' : 'New Unit of Measure';
  document.getElementById('uomMasterEntry').value = entry || '';
  const delBtn = document.getElementById('uomMasterDeleteBtn');
  if (isEdit) delBtn.classList.remove('d-none'); else delBtn.classList.add('d-none');

  if (isEdit) {
    const u = _uomMasterList.find(x => x.uom_entry === entry);
    if (u) {
      document.getElementById('uomMasterCode').value   = u.uom_code || '';
      document.getElementById('uomMasterName').value   = u.uom_name || '';
      document.getElementById('uomMasterWeight').value = u.weight  || '';
      document.getElementById('uomMasterVolume').value = u.volume  || '';
      document.getElementById('uomMasterLength').value = u.length  || '';
      document.getElementById('uomMasterWidth').value  = u.width   || '';
      document.getElementById('uomMasterHeight').value = u.height  || '';
    }
  } else {
    document.getElementById('uomMasterForm').reset();
  }
  new bootstrap.Modal(document.getElementById('uomMasterModal')).show();
}

async function submitUomMasterForm() {
  const entry = document.getElementById('uomMasterEntry').value;
  const code  = document.getElementById('uomMasterCode').value.trim().toUpperCase();
  const name  = document.getElementById('uomMasterName').value.trim();
  if (!code) { toast('UoM Code is required.', 'warning'); return; }
  if (!name) { toast('UoM Name is required.', 'warning'); return; }

  const payload = {
    uom_code: code,
    uom_name: name,
    weight:   parseFloat(document.getElementById('uomMasterWeight').value) || 0,
    volume:   parseFloat(document.getElementById('uomMasterVolume').value) || 0,
    length:   parseFloat(document.getElementById('uomMasterLength').value) || 0,
    width:    parseFloat(document.getElementById('uomMasterWidth').value)  || 0,
    height:   parseFloat(document.getElementById('uomMasterHeight').value) || 0,
  };

  const result = entry
    ? await api.UpdateUoMMaster(parseInt(entry, 10), payload)
    : await api.CreateUoMMaster(payload);

  if (result) {
    toast(entry ? 'UoM updated.' : 'UoM created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('uomMasterModal')).hide();
    Modules.Inventory.load('uom');
  } else {
    toast('Failed to save UoM.', 'danger');
  }
}

async function deleteUomMaster() {
  const entry = document.getElementById('uomMasterEntry').value;
  const code  = document.getElementById('uomMasterCode').value;
  if (!confirm('Delete UoM "' + code + '"? This cannot be undone.')) return;
  const result = await api.DeleteUoMMaster(parseInt(entry, 10));
  if (result) {
    toast('UoM deleted.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('uomMasterModal')).hide();
    Modules.Inventory.load('uom');
  } else {
    toast('Failed to delete UoM.', 'danger');
  }
}

// ── UoM Group helpers ─────────────────────────────────────────────────────────

function _uomGroupRows() {
  if (_uomGroupList.length === 0)
    return `<tr><td colspan="5" class="text-center text-muted py-4">No UoM groups defined yet.</td></tr>`;
  return _uomGroupList.map(g => {
    const baseUnit = _uomMasterList.find(u => u.uom_entry === g.base_uom);
    const lines = g.lines || [];
    const conversionBadges = lines.map(l => {
      const u = _uomMasterList.find(x => x.uom_entry === l.uom_entry);
      const bu = baseUnit ? baseUnit.uom_code : '';
      return `<span class="badge bg-light text-dark border me-1">${l.alt_qty} ${u ? u.uom_code : '?'} = ${l.base_qty} ${bu}</span>`;
    }).join('') || `<span class="text-muted small">${lines.length} line${lines.length !== 1 ? 's' : ''}</span>`;
    return `
      <tr>
        <td class="font-monospace fw-semibold">${g.ugp_code || '—'}</td>
        <td>${g.ugp_name || '—'}</td>
        <td>${baseUnit
          ? '<span class="badge bg-primary-subtle text-primary border border-primary-subtle">' + baseUnit.uom_code + '</span>'
          : '—'}</td>
        <td><div class="d-flex flex-wrap gap-1">${conversionBadges}</div></td>
        <td class="text-end">
          <button class="btn btn-sm btn-outline-secondary" onclick="openUomGroupModal(${g.ugp_entry})">
            <i class="bi bi-pencil"></i>
          </button>
        </td>
      </tr>`;
  }).join('');
}

let _uomGroupLineIndex = 0;

function uomGroupAddLine(uomEntry, altQty, baseQty) {
  const idx = _uomGroupLineIndex++;
  const options = _uomMasterList.map(u =>
    `<option value="${u.uom_entry}" ${u.uom_entry === uomEntry ? 'selected' : ''}>${u.uom_code} — ${u.uom_name}</option>`
  ).join('');
  const tr = document.createElement('tr');
  tr.id = 'ugl-' + idx;
  tr.innerHTML = `
    <td>
      <select class="form-select form-select-sm" id="ugl-unit-${idx}">
        <option value="">— select —</option>
        ${options}
      </select>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-end" id="ugl-alt-${idx}"
             min="0" step="any" value="${altQty || 1}">
    </td>
    <td class="text-center px-1 fw-bold">=</td>
    <td>
      <input type="number" class="form-control form-control-sm text-end" id="ugl-base-${idx}"
             min="0" step="any" value="${baseQty || 1}">
    </td>
    <td>
      <button type="button" class="btn btn-sm btn-outline-danger"
              onclick="document.getElementById('ugl-${idx}').remove()">
        <i class="bi bi-x"></i>
      </button>
    </td>`;
  document.getElementById('uomGroupLines').appendChild(tr);
}

function openUomGroupModal(entry) {
  const isEdit = entry !== null;
  document.getElementById('uomGroupModalTitle').textContent = isEdit ? 'Edit UoM Group' : 'New UoM Group';
  document.getElementById('uomGroupEntry').value = entry || '';
  document.getElementById('uomGroupLines').innerHTML = '';
  _uomGroupLineIndex = 0;

  const delBtn = document.getElementById('uomGroupDeleteBtn');
  if (isEdit) delBtn.classList.remove('d-none'); else delBtn.classList.add('d-none');

  if (isEdit) {
    const g = _uomGroupList.find(x => x.ugp_entry === entry);
    if (g) {
      document.getElementById('uomGroupCode').value    = g.ugp_code || '';
      document.getElementById('uomGroupName').value    = g.ugp_name || '';
      document.getElementById('uomGroupBaseUom').value = g.base_uom || '';
      (g.lines || []).forEach(l => uomGroupAddLine(l.uom_entry, l.alt_qty, l.base_qty));
    }
  } else {
    document.getElementById('uomGroupForm').reset();
    document.getElementById('uomGroupLines').innerHTML = '';
  }
  new bootstrap.Modal(document.getElementById('uomGroupModal')).show();
}

async function submitUomGroupForm() {
  const entry   = document.getElementById('uomGroupEntry').value;
  const ugpCode = document.getElementById('uomGroupCode').value.trim().toUpperCase();
  const ugpName = document.getElementById('uomGroupName').value.trim();
  const baseUom = parseInt(document.getElementById('uomGroupBaseUom').value, 10);

  if (!ugpCode) { toast('Group Code is required.', 'warning'); return; }
  if (!ugpName) { toast('Group Name is required.', 'warning'); return; }
  if (!baseUom) { toast('Base Unit is required.', 'warning'); return; }

  const lineRows = document.querySelectorAll('#uomGroupLines tr');
  const lines = [];
  for (const row of lineRows) {
    const idx      = row.id.replace('ugl-', '');
    const uomEntry = parseInt(document.getElementById('ugl-unit-' + idx)?.value || '0', 10);
    const altQty   = parseFloat(document.getElementById('ugl-alt-'  + idx)?.value)  || 1;
    const baseQty  = parseFloat(document.getElementById('ugl-base-' + idx)?.value) || 1;
    if (!uomEntry) { toast('Each line must have a unit selected.', 'warning'); return; }
    lines.push({ uom_entry: uomEntry, alt_qty: altQty, base_qty: baseQty });
  }

  let result;
  if (entry) {
    result = await api.UpdateUoMGroup(parseInt(entry, 10), ugpCode, ugpName, baseUom, lines);
  } else {
    result = await api.CreateUoMGroup({ ugp_code: ugpCode, ugp_name: ugpName, base_uom: baseUom, lines });
  }

  if (result) {
    toast(entry ? 'UoM Group updated.' : 'UoM Group created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('uomGroupModal')).hide();
    Modules.Inventory.load('uom');
  } else {
    toast('Failed to save UoM Group.', 'danger');
  }
}

async function deleteUomGroup() {
  const entry = document.getElementById('uomGroupEntry').value;
  const code  = document.getElementById('uomGroupCode').value;
  if (!confirm('Delete UoM Group "' + code + '" and all its conversion lines?')) return;
  const result = await api.DeleteUoMGroup(parseInt(entry, 10));
  if (result) {
    toast('UoM Group deleted.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('uomGroupModal')).hide();
    Modules.Inventory.load('uom');
  } else {
    toast('Failed to delete UoM Group.', 'danger');
  }
}

// ── Inventory Ledger (OIVL) ───────────────────────────────────────────────────

let _oivlAllRows = [];

const OIVL_TRANS_BADGE = {
  // Standalone Goods Receipt
  'GR':         { cls: 'bg-success',                                                   label: 'GR' },
  'GR_CANCEL':  { cls: 'bg-success-subtle text-success border border-success-subtle',  label: 'GR Cancel' },
  // Standalone Goods Issue
  'GI':         { cls: 'bg-danger',                                                    label: 'GI' },
  'GI_CANCEL':  { cls: 'bg-danger-subtle text-danger border border-danger-subtle',     label: 'GI Cancel' },
  // Sales Delivery Order
  'DO':         { cls: 'bg-primary',                                                   label: 'Delivery' },
  'DO_CANCEL':  { cls: 'bg-primary-subtle text-primary border border-primary-subtle',  label: 'DO Cancel' },
  // Purchasing Delivery Receipt
  'DR':         { cls: 'bg-info text-dark',                                            label: 'DR' },
  'DR_CANCEL':  { cls: 'bg-info-subtle text-info border border-info-subtle',           label: 'DR Cancel' },
  // Work Orders (Production)
  'WO_ISSUE':   { cls: 'bg-warning text-dark',                                        label: 'WO Issue' },
  'WO_RECEIPT': { cls: 'bg-success',                                                  label: 'WO Receipt' },
  'WO_CANCEL':  { cls: 'bg-warning-subtle text-warning border border-warning-subtle', label: 'WO Cancel' },
};

function _oivlBadge(t) {
  const b = OIVL_TRANS_BADGE[t] || { cls: 'bg-secondary', label: t };
  return `<span class="badge ${b.cls}">${b.label}</span>`;
}

// Stock-in types → subtle green row; all others → subtle red
const OIVL_IN_TYPES = new Set(['GR', 'GI_CANCEL', 'DR', 'DO_CANCEL', 'WO_RECEIPT', 'WO_CANCEL']);

// Returns the hash route for the source document, or null if not navigable.
function _oivlNavRoute(r) {
  const id = r.doc_num;
  if (!id) return null;
  switch (r.trans_type) {
    case 'GR': case 'GR_CANCEL':
      return `#/inventory/gr/${id}`;
    case 'GI': case 'GI_CANCEL':
      return `#/inventory/gi/${id}`;
    case 'DO': case 'DO_CANCEL':
      return `#/sales/delivery-orders/${id}`;
    case 'DR': case 'DR_CANCEL':
      return `#/purchasing/delivery-receipts/${id}`;
    case 'WO_ISSUE': case 'WO_RECEIPT': case 'WO_CANCEL':
      return `#/production/work-orders/${id}`;
    default:
      return null;
  }
}

function _oivlBuildRows(rows) {
  if (rows.length === 0)
    return `<tr><td colspan="10" class="text-center text-muted py-4">No ledger entries found.</td></tr>`;
  return rows.map(r => {
    const inQty  = r.in_qty  || 0;
    const outQty = r.out_qty || 0;
    const net    = inQty - outQty;
    const rowCls = OIVL_IN_TYPES.has(r.trans_type) ? 'table-success' : 'table-danger';
    const route  = _oivlNavRoute(r);
    const trAttrs = route
      ? `class="${rowCls} bg-opacity-10" style="cursor:pointer" ondblclick="sessionStorage.setItem('docReturnRoute','#/inventory/ledger');navigate('${route}')" title="Double-click to view source document"`
      : `class="${rowCls} bg-opacity-10"`;
    const docNumCell = r.doc_num
      ? (route ? `<u>${r.doc_num}</u>` : r.doc_num)
      : '—';
    return `
      <tr ${trAttrs}>
        <td class="small text-muted">${formatDate(r.doc_date)}</td>
        <td>${_oivlBadge(r.trans_type)}</td>
        <td class="small font-monospace ${route ? 'text-primary' : 'text-muted'}">${docNumCell}</td>
        <td class="small font-monospace">${r.item_code}</td>
        <td class="small">${r.item_name || '—'}</td>
        <td class="small text-muted">${r.warehouse || '—'}</td>
        <td class="text-end small ${inQty > 0 ? 'text-success fw-bold' : 'text-muted'}">${inQty > 0 ? '+' + formatNumber(inQty) : '—'}</td>
        <td class="text-end small ${outQty > 0 ? 'text-danger fw-bold' : 'text-muted'}">${outQty > 0 ? '−' + formatNumber(outQty) : '—'}</td>
        <td class="text-end small ${net >= 0 ? 'text-success' : 'text-danger'}">${net >= 0 ? '+' : ''}${formatNumber(net)}</td>
        <td class="text-end small">${formatCurrency(r.value || 0)}</td>
      </tr>`;
  }).join('');
}

// Called by filter inputs — only updates tbody and count without touching the page shell
window._oivlFilter = function() {
  const search     = (document.getElementById('oivl-search')?.value || '').toLowerCase();
  const typeFilter =  document.getElementById('oivl-type')?.value   || '';
  const filtered   = _oivlAllRows.filter(r => {
    if (typeFilter && r.trans_type !== typeFilter) return false;
    if (search && !r.item_code.toLowerCase().includes(search) && !(r.item_name || '').toLowerCase().includes(search)) return false;
    return true;
  });
  const tbody = document.getElementById('oivlTbody');
  if (tbody) tbody.innerHTML = _oivlBuildRows(filtered);
  const cnt = document.getElementById('oivlCount');
  if (cnt) cnt.textContent = filtered.length + ' entries';
};

async function loadInventoryLedger(tabBar) {
  const raw    = await api.ListOIVL('');
  _oivlAllRows = raw || [];

  const typeOptions = ['GR', 'GR_CANCEL', 'GI', 'GI_CANCEL', 'DO', 'DO_CANCEL', 'DR', 'DR_CANCEL', 'WO_ISSUE', 'WO_RECEIPT', 'WO_CANCEL']
    .map(t => `<option value="${t}">${OIVL_TRANS_BADGE[t]?.label || t}</option>`)
    .join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-3">
        <h4 class="fw-bold mb-0"><i class="bi bi-journal-text me-2 text-warning"></i>Inventory Ledger</h4>
        <span class="text-muted small">Last 500 entries · newest first</span>
      </div>
      ${tabBar}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body py-2 d-flex gap-2 flex-wrap align-items-center">
          <input type="text" id="oivl-search" class="form-control form-control-sm"
                 placeholder="Filter by item code or name…" style="max-width:240px;" oninput="_oivlFilter()">
          <select id="oivl-type" class="form-select form-select-sm" style="max-width:190px;" onchange="_oivlFilter()">
            <option value="">All transaction types</option>
            ${typeOptions}
          </select>
          <span id="oivlCount" class="ms-auto text-muted small">${_oivlAllRows.length} entries</span>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="table-responsive" style="max-height:600px;overflow-y:auto;">
          <table class="table table-sm table-hover align-middle mb-0">
            <thead class="table-light sticky-top">
              <tr>
                <th>Date</th>
                <th>Type</th>
                <th>Doc #</th>
                <th>Item Code</th>
                <th>Item Name</th>
                <th>Warehouse</th>
                <th class="text-end">In Qty</th>
                <th class="text-end">Out Qty</th>
                <th class="text-end">Net</th>
                <th class="text-end">Value</th>
              </tr>
            </thead>
            <tbody id="oivlTbody">${_oivlBuildRows(_oivlAllRows)}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}
