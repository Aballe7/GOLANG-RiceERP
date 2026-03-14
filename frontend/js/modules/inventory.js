// inventory.js — Modules.Inventory

const INV_EGG_SIZES = ['Jumbo', 'XL', 'Large', 'Medium', 'Small', 'Peewee'];

Modules.Inventory = {
  async load(sub) {
    const activeTab = sub || 'eggs';
    showLoading();

    const tabBar = `
      <ul class="nav nav-tabs mb-4" id="invTabs">
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'eggs' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/eggs');return false;">
            <i class="bi bi-egg me-1"></i>Egg Inventory
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'supplies' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/supplies');return false;">
            <i class="bi bi-box-seam me-1"></i>Supply Inventory
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${activeTab === 'feed' ? 'active' : ''}" href="#"
            onclick="navigate('#/inventory/feed');return false;">
            <i class="bi bi-bag me-1"></i>Feed Dashboard
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
      </ul>
    `;

    if (activeTab === 'eggs') await loadEggInventory(tabBar);
    else if (activeTab === 'supplies') await loadSupplyInventory(tabBar);
    else if (activeTab === 'feed') await loadFeedDashboard(tabBar);
    else if (activeTab === 'items') await loadItemMaster(tabBar);
    else if (activeTab === 'categories') await loadItemCategories(tabBar);
    else if (activeTab === 'uom') await loadUoMSetup(tabBar);
    else await loadEggInventory(tabBar);
  }
};

// ── Egg Inventory ─────────────────────────────────────────────────────────────

async function loadEggInventory(tabBar) {
  const data = await api.GetEggInventory();
  if (!data) {
    showView(`<div class="container-fluid p-4">${tabBar}<div class="alert alert-warning">Failed to load egg inventory.</div></div>`);
    return;
  }

  const inventory = data.by_size || {};
  const gridCells = INV_EGG_SIZES.map(size => {
    const count = inventory[size] ?? 0;
    const trays = (count / 30).toFixed(1);
    return `
      <div class="egg-grid-cell text-center border rounded p-3">
        <div class="fw-semibold text-secondary small text-uppercase mb-1">${size}</div>
        <div class="fs-4 fw-bold">${formatNumber(count)}</div>
        <div class="text-muted small">${trays} trays</div>
      </div>
    `;
  }).join('');

  const tableRows = INV_EGG_SIZES.map(size => {
    const count = inventory[size] ?? 0;
    const trays = (count / 30).toFixed(2);
    return `
      <tr>
        <td>${size}</td>
        <td>${formatNumber(count)}</td>
        <td>${trays}</td>
      </tr>
    `;
  }).join('');

  const totalPieces = INV_EGG_SIZES.reduce((s, sz) => s + (inventory[sz] ?? 0), 0);

  showView(`
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-egg me-2"></i>Inventory</h4>
      ${tabBar}

      <div class="egg-grid d-grid gap-3 mb-4"
        style="grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));">
        ${gridCells}
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-transparent fw-semibold">Egg Inventory Detail</div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Size</th><th>Pieces</th><th>Trays (30s)</th></tr>
            </thead>
            <tbody>${tableRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr><td>Total</td><td>${formatNumber(totalPieces)}</td><td>${(totalPieces/30).toFixed(2)}</td></tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>
  `);
}

// ── Supply Inventory ──────────────────────────────────────────────────────────

async function loadSupplyInventory(tabBar) {
  const items = await api.ListInventory();
  if (!items) {
    showView(`<div class="container-fluid p-4">${tabBar}<div class="alert alert-warning">Failed to load supply inventory.</div></div>`);
    return;
  }

  const rows = items.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No supply inventory records.</td></tr>`
    : items.map(i => `
        <tr>
          <td>${i.item_name || '—'}</td>
          <td>${i.category || '—'}</td>
          <td>${formatNumber(i.quantity ?? 0)}</td>
          <td>${i.unit || '—'}</td>
          <td class="${(i.quantity ?? 0) < (i.reorder_point ?? 0) ? 'text-danger fw-semibold' : ''}">
            ${formatNumber(i.reorder_point ?? 0)}
          </td>
        </tr>
      `).join('');

  showView(`
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-box-seam me-2"></i>Inventory</h4>
      ${tabBar}
      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Item</th><th>Category</th><th>Qty</th><th>Unit</th><th>Reorder Point</th></tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}

// ── Feed Dashboard ────────────────────────────────────────────────────────────

async function loadFeedDashboard(tabBar) {
  const stocks = await api.ListFeedStocks();
  if (!stocks) {
    showView(`<div class="container-fluid p-4">${tabBar}<div class="alert alert-warning">Failed to load feed stocks.</div></div>`);
    return;
  }

  const cards = stocks.length === 0
    ? `<div class="col-12"><div class="alert alert-info">No feed stocks available.</div></div>`
    : stocks.map(s => {
        const sacks = s.remaining_sacks ?? 0;
        const kg = s.remaining_kg ?? 0;
        const isLow = s.is_low_stock === true;
        return `
          <div class="col-sm-6 col-xl-4">
            <div class="card border-0 shadow-sm h-100 ${isLow ? 'border-danger border' : ''}">
              <div class="card-body">
                <div class="d-flex align-items-start justify-content-between mb-2">
                  <h6 class="fw-bold mb-0">${s.feed_name || 'Unknown Feed'}</h6>
                  ${isLow ? `<span class="badge bg-danger">Low Stock</span>` : `<span class="badge bg-success">OK</span>`}
                </div>
                <div class="text-muted small mb-1">${s.feed_type || ''} · ${s.supplier || ''}</div>
                <hr class="my-2">
                <div class="row text-center">
                  <div class="col">
                    <div class="fs-5 fw-bold">${formatNumber(sacks)}</div>
                    <div class="text-muted small">Sacks</div>
                  </div>
                  <div class="col">
                    <div class="fs-5 fw-bold">${formatNumber(kg)}</div>
                    <div class="text-muted small">Kg</div>
                  </div>
                </div>
                ${isLow ? `<div class="text-danger small mt-2"><i class="bi bi-exclamation-triangle-fill me-1"></i>Below reorder level</div>` : ''}
              </div>
            </div>
          </div>
        `;
      }).join('');

  showView(`
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-bag me-2"></i>Inventory</h4>
      ${tabBar}
      <div class="row g-3">${cards}</div>
    </div>
  `);
}

// ── Item Master ───────────────────────────────────────────────────────────────

const IM_CAT_COLOR = { Feed:'success', Vaccine:'primary', Medicine:'danger', Packaging:'warning', Supplies:'secondary', Fuel:'dark' };

let _itemMasterList = [];

function _imRows(list) {
  if (list.length === 0)
    return `<tr><td colspan="7" class="text-center text-muted py-4">No items match the filter.</td></tr>`;
  return list.map(i => {
    const badgeClass = IM_CAT_COLOR[i.category] || 'secondary';
    return `
      <tr>
        <td class="text-muted small font-monospace">${i.item_code || '—'}</td>
        <td class="fw-semibold">${i.name || '—'}</td>
        <td><span class="badge bg-${badgeClass}-subtle text-${badgeClass} border border-${badgeClass}-subtle">${i.category || '—'}</span></td>
        <td>${i.unit || '—'}</td>
        <td class="text-end">₱${formatNumber(i.unit_price ?? 0)}</td>
        <td class="text-end">${formatNumber(i.reorder_level ?? 0)}</td>
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
    if (search && !(i.name || '').toLowerCase().includes(search) &&
                  !(i.item_code || '').toLowerCase().includes(search) &&
                  !(i.description || '').toLowerCase().includes(search)) return false;
    if (cat  && i.category !== cat) return false;
    if (unit && !(i.unit || '').toLowerCase().includes(unit)) return false;
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
  const items = await api.ListItemMasters();
  _itemMasterList = items || [];

  // Build dynamic filter options from the loaded data
  const cats  = [...new Set(_itemMasterList.map(i => i.category).filter(Boolean))].sort();
  const units = [...new Set(_itemMasterList.map(i => i.unit).filter(Boolean))].sort();

  const catOptions  = `<option value="">All Categories</option>` + cats.map(c => `<option value="${c}">${c}</option>`).join('');
  const unitOptions = `<option value="">All Units</option>` + units.map(u => `<option value="${u}">${u}</option>`).join('');

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
                    <option value="Feed">
                    <option value="Vaccine">
                    <option value="Medicine">
                    <option value="Packaging">
                    <option value="Supplies">
                    <option value="Fuel">
                    <option value="Equipment">
                  </datalist>
                </div>
              </div>

              <!-- Row 2: Name (full width) -->
              <div class="mb-3">
                <label class="form-label small fw-bold">Item Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="itemName"
                       placeholder="e.g. Layer Mash (Commercial)" required>
              </div>

              <!-- Row 3: Unit + Unit Price -->
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
                <div class="col-md-4">
                  <label class="form-label small fw-bold">Unit Price (₱)</label>
                  <div class="input-group">
                    <span class="input-group-text">₱</span>
                    <input type="number" class="form-control" id="itemUnitPrice"
                           min="0" step="0.01" placeholder="0.00">
                  </div>
                </div>
                <div class="col-md-4">
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
      document.getElementById('itemName').value         = item.name || '';
      document.getElementById('itemCategory').value     = item.category || '';
      document.getElementById('itemUnit').value         = item.unit || '';
      document.getElementById('itemUnitPrice').value    = item.unit_price ?? '';
      document.getElementById('itemReorderLevel').value = item.reorder_level ?? '';
      document.getElementById('itemDescription').value  = item.description || '';
    }
  } else {
    document.getElementById('itemForm').reset();
  }
  new bootstrap.Modal(document.getElementById('itemModal')).show();
}

async function submitItemForm() {
  const id = document.getElementById('itemId').value;
  const payload = {
    item_code:     document.getElementById('itemCode').value.trim(),
    name:          document.getElementById('itemName').value.trim(),
    category:      document.getElementById('itemCategory').value.trim(),
    unit:          document.getElementById('itemUnit').value.trim(),
    unit_price:    parseFloat(document.getElementById('itemUnitPrice').value) || 0,
    reorder_level: parseFloat(document.getElementById('itemReorderLevel').value) || 0,
    description:   document.getElementById('itemDescription').value.trim(),
  };
  if (!payload.name)     { toast('Item name is required.', 'warning'); return; }
  if (!payload.category) { toast('Category is required.', 'warning'); return; }
  if (!payload.unit)     { toast('Unit is required.', 'warning'); return; }

  const result = id
    ? await api.UpdateItemMaster(parseInt(id, 10), payload)
    : await api.CreateItemMaster(payload);

  if (result) {
    toast(id ? 'Item updated.' : 'Item created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('itemModal')).hide();
    navigate('#/inventory/items');
  } else {
    toast('Failed to save item.', 'danger');
  }
}

// ── Item Categories ────────────────────────────────────────────────────────────

let _itemCategoryList = [];

async function loadItemCategories(tabBar) {
  const cats = await api.ListItemCategories();
  _itemCategoryList = cats || [];

  const rows = _itemCategoryList.length === 0
    ? `<tr><td colspan="4" class="text-center text-muted py-4">No categories found.</td></tr>`
    : _itemCategoryList.map(c => `
        <tr>
          <td class="fw-semibold">${c.name || '—'}</td>
          <td class="text-muted">${c.description || '—'}</td>
          <td>${c.is_active
            ? `<span class="badge bg-success-subtle text-success border border-success-subtle">Active</span>`
            : `<span class="badge bg-secondary-subtle text-secondary border border-secondary-subtle">Inactive</span>`
          }</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary me-1" onclick="openCategoryModal(${c.id})">
              <i class="bi bi-pencil"></i>
            </button>
            <button class="btn btn-sm btn-outline-danger" onclick="deleteCategory(${c.id}, '${(c.name || '').replace(/'/g, "\\'")}')">
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
              <tr><th>Name</th><th>Description</th><th>Status</th><th></th></tr>
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
    const cat = _itemCategoryList.find(c => c.id === id);
    if (cat) {
      document.getElementById('categoryName').value = cat.name || '';
      document.getElementById('categoryDescription').value = cat.description || '';
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

  const result = id
    ? await api.UpdateItemCategory(parseInt(id, 10), { name, description })
    : await api.CreateItemCategory({ name, description });

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

let _uomMasterList = [];
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

      <!-- UoM Master -->
      <div class="d-flex align-items-center justify-content-between mb-2 mt-1">
        <h6 class="fw-bold mb-0 text-muted text-uppercase small">
          <i class="bi bi-list-check me-1"></i>Unit of Measure Master (OUOM)
        </h6>
        <button class="btn btn-sm btn-primary" onclick="openUomMasterModal(null)">
          <i class="bi bi-plus-lg me-1"></i>Add UoM
        </button>
      </div>
      <div class="card border-0 shadow-sm mb-4">
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
    navigate('#/inventory/uom');
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
    navigate('#/inventory/uom');
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
    navigate('#/inventory/uom');
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
    navigate('#/inventory/uom');
  } else {
    toast('Failed to delete UoM Group.', 'danger');
  }
}
