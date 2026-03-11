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
      </ul>
    `;

    if (activeTab === 'eggs') await loadEggInventory(tabBar);
    else if (activeTab === 'supplies') await loadSupplyInventory(tabBar);
    else if (activeTab === 'feed') await loadFeedDashboard(tabBar);
    else if (activeTab === 'items') await loadItemMaster(tabBar);
    else if (activeTab === 'categories') await loadItemCategories(tabBar);
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

let _itemMasterList = [];

async function loadItemMaster(tabBar) {
  const items = await api.ListItemMasters();
  _itemMasterList = items || [];

  const rows = _itemMasterList.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No items in master list.</td></tr>`
    : _itemMasterList.map(i => `
        <tr>
          <td>${i.item_code || '—'}</td>
          <td>${i.item_name || '—'}</td>
          <td>${i.category || '—'}</td>
          <td>${i.unit || '—'}</td>
          <td>
            <button class="btn btn-sm btn-outline-secondary" onclick="openItemModal(${i.id})">
              <i class="bi bi-pencil"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-4">
        <h4 class="fw-bold mb-0"><i class="bi bi-list-ul me-2"></i>Inventory</h4>
        <button class="btn btn-primary" onclick="openItemModal(null)">
          <i class="bi bi-plus-lg me-1"></i>Add Item
        </button>
      </div>
      ${tabBar}
      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Code</th><th>Name</th><th>Category</th><th>Unit</th><th></th></tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Item Modal -->
    <div class="modal fade" id="itemModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="itemModalTitle">Item</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="itemForm">
              <input type="hidden" id="itemId">
              <div class="mb-3">
                <label class="form-label">Item Code</label>
                <input type="text" class="form-control" id="itemCode" placeholder="e.g. FEED-001">
              </div>
              <div class="mb-3">
                <label class="form-label">Item Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="itemName" required>
              </div>
              <div class="mb-3">
                <label class="form-label">Category</label>
                <input type="text" class="form-control" id="itemCategory" placeholder="e.g. Feed, Supplies">
              </div>
              <div class="mb-3">
                <label class="form-label">Unit</label>
                <input type="text" class="form-control" id="itemUnit" placeholder="e.g. kg, pcs, sack">
              </div>
            </form>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitItemForm()">Save</button>
          </div>
        </div>
      </div>
    </div>
  `);
}

function openItemModal(id) {
  const isEdit = id !== null;
  document.getElementById('itemModalTitle').textContent = isEdit ? 'Edit Item' : 'Add Item';
  document.getElementById('itemId').value = id || '';
  if (isEdit) {
    const item = _itemMasterList.find(i => i.id === id);
    if (item) {
      document.getElementById('itemCode').value = item.item_code || '';
      document.getElementById('itemName').value = item.item_name || '';
      document.getElementById('itemCategory').value = item.category || '';
      document.getElementById('itemUnit').value = item.unit || '';
    }
  } else {
    document.getElementById('itemForm').reset();
  }
  new bootstrap.Modal(document.getElementById('itemModal')).show();
}

async function submitItemForm() {
  const id = document.getElementById('itemId').value;
  const payload = {
    item_code: document.getElementById('itemCode').value.trim(),
    item_name: document.getElementById('itemName').value.trim(),
    category:  document.getElementById('itemCategory').value.trim(),
    unit:      document.getElementById('itemUnit').value.trim(),
  };
  if (!payload.item_name) { toast('Item name is required.', 'warning'); return; }

  const result = id
    ? await api.UpdateItemMaster({ ...payload, id: parseInt(id, 10) })
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
