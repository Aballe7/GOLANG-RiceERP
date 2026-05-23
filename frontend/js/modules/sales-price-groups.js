// sales-price-groups.js

// ── Price Groups ──────────────────────────────────────────────────────────────

const PG_EGG_SIZES = ['Jumbo', 'Extra Large', 'Large', 'Medium', 'Small', 'Pullet', 'Peewee'];
const PG_POPULAR   = ['Medium', 'Large', 'Extra Large'];
let _priceGroupList = [];

async function loadPriceGroupsView(editId, tabBar) {
  const groups = await api.ListPriceGroups(false);
  _priceGroupList = groups || [];

  const cards = _priceGroupList.length === 0
    ? `<div class="col-12">
        <div class="card border-0 shadow-sm">
          <div class="card-body text-center py-5 text-muted">
            <i class="bi bi-tags" style="font-size:3rem;opacity:0.2"></i>
            <div class="mt-3 fw-semibold">No price groups yet</div>
            <div class="small mb-3">Create groups like Wholesale, Retail, Distributor</div>
            <button class="btn btn-primary" data-bs-toggle="modal" data-bs-target="#addGroupModal">
              <i class="bi bi-plus-lg me-1"></i> Create First Price Group
            </button>
          </div>
        </div>
      </div>`
    : _priceGroupList.map(g => {
        const items = g.items || [];
        const trayItems = items.filter(i => i.unit === 'Tray');
        const priceBadges = trayItems.length
          ? trayItems.map(i => `
              <span class="badge rounded-pill me-1 mb-1"
                    style="background:#eff6ff;color:#1d4ed8;font-size:11px;">
                ${i.egg_size}: ₱${i.price.toFixed(2)}
              </span>`).join('')
          : `<span class="text-muted small fst-italic">No prices set yet</span>`;
        const custCount = (g.customers || []).length;
        const activeBadge = g.is_active
          ? `<span class="badge rounded-pill bg-success">Active</span>`
          : `<span class="badge rounded-pill bg-secondary">Inactive</span>`;
        return `
          <div class="col-md-6">
            <div class="card border-0 shadow-sm h-100">
              <div class="card-body">
                <div class="d-flex justify-content-between align-items-start mb-2">
                  <div>
                    <h6 class="fw-bold mb-0">${g.name}</h6>
                    ${g.description ? `<div class="text-muted small">${g.description}</div>` : ''}
                  </div>
                  ${activeBadge}
                </div>
                <div class="mb-3">${priceBadges}</div>
                <div class="d-flex align-items-center justify-content-between">
                  <div class="text-muted small">
                    <i class="bi bi-people me-1"></i>${custCount} customer${custCount !== 1 ? 's' : ''}
                  </div>
                  <button class="btn btn-sm btn-primary" onclick="navigate('#/sales/price-groups/${g.id}')">
                    <i class="bi bi-pencil me-1"></i> Edit
                  </button>
                </div>
              </div>
            </div>
          </div>`;
      }).join('');

  showView(wrapSales(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-4">
      <div>
        <h5 class="fw-bold mb-0"><i class="bi bi-tags-fill me-2 text-primary"></i>Price Groups</h5>
        <div class="text-muted small">Define price lists and assign customers to a group</div>
      </div>
      <button class="btn btn-primary" data-bs-toggle="modal" data-bs-target="#addGroupModal">
        <i class="bi bi-plus-lg me-1"></i> New Price Group
      </button>
    </div>
    <div class="row g-3">${cards}</div>

    <!-- Add Group Modal -->
    <div class="modal fade" id="addGroupModal" tabindex="-1">
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content border-0 shadow">
          <div class="modal-header border-0 pb-0">
            <h5 class="modal-title fw-bold">New Price Group</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <div class="mb-3">
              <label class="form-label fw-bold small">Group Name <span class="text-danger">*</span></label>
              <input type="text" class="form-control" id="newPgName"
                     placeholder="e.g. Wholesale, Retail, Distributor" required>
              <div class="form-text">This name will appear on the customer profile.</div>
            </div>
            <div class="mb-3">
              <label class="form-label fw-bold small">Description</label>
              <input type="text" class="form-control" id="newPgDescription"
                     placeholder="e.g. For bulk buyers 500+ trays/week">
            </div>
          </div>
          <div class="modal-footer border-0">
            <button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary px-4" onclick="submitNewPriceGroup()">
              Create &amp; Set Prices
            </button>
          </div>
        </div>
      </div>
    </div>
  `));
}

async function submitNewPriceGroup() {
  const name = document.getElementById('newPgName').value.trim();
  const desc = document.getElementById('newPgDescription').value.trim();
  if (!name) { toast('Group name is required.', 'warning'); return; }
  const result = await api.CreatePriceGroup({ name, description: desc, is_active: true });
  if (result) {
    bootstrap.Modal.getInstance(document.getElementById('addGroupModal')).hide();
    toast('Price group created.', 'success');
    // Navigate to the new group's detail page to set prices
    navigate('#/sales/price-groups/' + result.id);
  } else {
    toast('Failed to create price group.', 'danger');
  }
}

// ── Price Group Detail ────────────────────────────────────────────────────────

async function loadPriceGroupDetail(pgId) {
  showLoading();
  const [pg, allCustomers] = await Promise.all([
    api.GetPriceGroup(pgId),
    api.ListCustomers(false),
  ]);
  if (!pg) { showView(`<div class="alert alert-warning m-4">Price group not found.</div>`); return; }

  const items = pg.items || [];
  const assignedCustomers = pg.customers || [];
  const assignedIds = new Set(assignedCustomers.map(c => c.id));
  const unassigned = (allCustomers || []).filter(c => !assignedIds.has(c.id));

  // Build price map: {size_unit: price}
  const priceMap = {};
  items.forEach(i => { priceMap[`${i.egg_size}_${i.unit}`] = i.price; });

  const priceRows = PG_EGG_SIZES.map(size => {
    const trayVal  = priceMap[`${size}_Tray`]  ?? '';
    const pieceVal = priceMap[`${size}_Piece`] ?? '';
    const pop = PG_POPULAR.includes(size)
      ? `<span class="badge bg-success-subtle text-success ms-1" style="font-size:9px;">Popular</span>`
      : '';
    return `
      <tr>
        <td class="ps-3 fw-semibold small">${size}${pop}</td>
        <td class="text-center">
          <div class="input-group input-group-sm justify-content-center" style="max-width:130px;margin:0 auto;">
            <span class="input-group-text">₱</span>
            <input type="number" class="form-control text-end tray-inp" data-size="${size}"
                   placeholder="0.00" min="0" step="0.01" value="${trayVal !== '' ? Number(trayVal).toFixed(2) : ''}">
          </div>
        </td>
        <td class="text-center">
          <div class="input-group input-group-sm justify-content-center" style="max-width:130px;margin:0 auto;">
            <span class="input-group-text">₱</span>
            <input type="number" class="form-control text-end piece-inp" data-size="${size}"
                   placeholder="0.00" min="0" step="0.01" value="${pieceVal !== '' ? Number(pieceVal).toFixed(4) : ''}">
          </div>
        </td>
      </tr>`;
  }).join('');

  const assignedRows = assignedCustomers.length
    ? assignedCustomers.map(c => `
        <li class="list-group-item d-flex align-items-center justify-content-between py-2 px-3">
          <div>
            <div class="fw-semibold small">${c.name}</div>
            <div class="text-muted" style="font-size:11px;">${c.customer_type || ''}${c.contact_number ? ' · ' + c.contact_number : ''}</div>
          </div>
          <button class="btn btn-sm btn-outline-danger py-0 px-1"
                  onclick="pgRemoveCustomer(${pgId}, ${c.id})" title="Remove">
            <i class="bi bi-x-lg"></i>
          </button>
        </li>`).join('')
    : `<li class="list-group-item text-center text-muted py-4 small">
        <i class="bi bi-person-x d-block mb-1" style="font-size:1.5rem;opacity:0.3;"></i>
        No customers assigned yet
      </li>`;

  const unassignedOptions = unassigned.length
    ? unassigned.map(c => {
        const cur = c.price_group ? ` (currently: ${c.price_group.name})` : '';
        return `<option value="${c.id}">${c.name}${cur}</option>`;
      }).join('')
    : '';

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px;">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/price-groups')">
          <i class="bi bi-arrow-left"></i> Back
        </button>
        <div class="flex-grow-1">
          <h4 class="mb-0 fw-bold"><i class="bi bi-tag-fill me-2 text-primary"></i>${pg.name}</h4>
          ${pg.description ? `<div class="text-muted small">${pg.description}</div>` : ''}
        </div>
        <button class="btn btn-sm btn-outline-danger"
                onclick="pgDeleteGroup(${pgId}, ${JSON.stringify(pg.name)})">
          <i class="bi bi-trash me-1"></i>Delete Group
        </button>
      </div>

      <div class="row g-4">
        <!-- LEFT: Prices -->
        <div class="col-lg-7">
          <div class="card border-0 shadow-sm mb-3">
            <div class="card-header bg-white border-bottom py-3">
              <div class="d-flex justify-content-between align-items-center">
                <span class="fw-bold">Price List</span>
                <div class="d-flex gap-2">
                  <button type="button" class="btn btn-sm btn-outline-secondary" onclick="pgDivideByThirty()">
                    <i class="bi bi-calculator me-1"></i>÷30 Piece
                  </button>
                  <button type="button" class="btn btn-sm btn-outline-danger" onclick="pgClearPrices()">
                    <i class="bi bi-x me-1"></i>Clear
                  </button>
                </div>
              </div>
            </div>

            <!-- Edit name/description -->
            <div class="px-3 pt-3 pb-2 border-bottom bg-light">
              <div class="row g-2">
                <div class="col-5">
                  <label class="form-label small fw-bold mb-1">Group Name</label>
                  <input type="text" class="form-control form-control-sm" id="pgEditName" value="${pg.name}" required>
                </div>
                <div class="col-7">
                  <label class="form-label small fw-bold mb-1">Description</label>
                  <input type="text" class="form-control form-control-sm" id="pgEditDesc" value="${pg.description || ''}" placeholder="Optional">
                </div>
              </div>
            </div>

            <div class="table-responsive">
              <table class="table table-hover align-middle mb-0">
                <thead class="table-light">
                  <tr>
                    <th class="ps-3" style="width:35%;">Egg Size</th>
                    <th class="text-center"><i class="bi bi-basket2 me-1 text-warning"></i>Tray ₱</th>
                    <th class="text-center"><i class="bi bi-egg me-1 text-warning"></i>Piece ₱</th>
                  </tr>
                </thead>
                <tbody id="pgPriceRows">${priceRows}</tbody>
              </table>
            </div>
          </div>
          <div class="d-flex justify-content-end mb-4">
            <button class="btn btn-primary px-4" onclick="pgSavePrices(${pgId})">
              <i class="bi bi-floppy-fill me-1"></i> Save Prices
            </button>
          </div>
        </div>

        <!-- RIGHT: Customers -->
        <div class="col-lg-5">
          <!-- Assigned customers -->
          <div class="card border-0 shadow-sm mb-3">
            <div class="card-header bg-white border-bottom py-3">
              <span class="fw-bold">
                <i class="bi bi-people-fill me-1 text-primary"></i>
                Assigned Customers
                <span class="badge bg-primary rounded-pill ms-1">${assignedCustomers.length}</span>
              </span>
            </div>
            <ul class="list-group list-group-flush" id="pgAssignedList">${assignedRows}</ul>
          </div>

          <!-- Assign new customer -->
          <div class="card border-0 shadow-sm">
            <div class="card-header bg-white border-bottom py-3">
              <span class="fw-bold"><i class="bi bi-person-plus-fill me-1 text-success"></i>Assign Customer</span>
            </div>
            <div class="card-body">
              ${unassigned.length ? `
              <div class="d-flex gap-2">
                <select class="form-select form-select-sm" id="pgAssignSelect">
                  <option value="">— Select customer —</option>
                  ${unassignedOptions}
                </select>
                <button class="btn btn-sm btn-success px-3" onclick="pgAssignCustomer(${pgId})">
                  <i class="bi bi-plus-lg"></i>
                </button>
              </div>
              <div class="form-text mt-1">Assigning will move customer from their current group.</div>
              ` : `<div class="text-muted small fst-italic">All customers are already in this group.</div>`}
            </div>
          </div>
        </div>
      </div>
    </div>
  `);
}

function pgDivideByThirty() {
  document.querySelectorAll('.tray-inp').forEach(inp => {
    const val = parseFloat(inp.value);
    if (!isNaN(val) && val > 0) {
      const size = inp.dataset.size;
      const pieceInp = document.querySelector(`.piece-inp[data-size="${size}"]`);
      if (pieceInp && !pieceInp.value) {
        pieceInp.value = (val / 30).toFixed(4);
      }
    }
  });
}

function pgClearPrices() {
  if (!confirm('Clear all prices in this group?')) return;
  document.querySelectorAll('.tray-inp, .piece-inp').forEach(inp => { inp.value = ''; });
}

function _resetSalesPriceGroups() {
  _priceGroupList = [];
}

async function pgSavePrices(pgId) {
  const name = document.getElementById('pgEditName').value.trim();
  const desc = document.getElementById('pgEditDesc').value.trim();
  if (!name) { toast('Group name is required.', 'warning'); return; }

  // Update name/description
  const metaResult = await api.UpdatePriceGroup(pgId, { name, description: desc });
  if (!metaResult) { toast('Failed to update group name.', 'danger'); return; }

  // Upsert each price row
  const rows = document.querySelectorAll('#pgPriceRows tr');
  const tasks = [];
  rows.forEach(row => {
    const trayInp  = row.querySelector('.tray-inp');
    const pieceInp = row.querySelector('.piece-inp');
    const size = trayInp?.dataset.size;
    if (!size) return;
    const trayVal  = parseFloat(trayInp?.value);
    const pieceVal = parseFloat(pieceInp?.value);
    if (!isNaN(trayVal)  && trayVal  > 0) tasks.push(api.UpsertPriceGroupItem({ price_group_id: pgId, egg_size: size, unit: 'Tray',  price: trayVal }));
    if (!isNaN(pieceVal) && pieceVal > 0) tasks.push(api.UpsertPriceGroupItem({ price_group_id: pgId, egg_size: size, unit: 'Piece', price: pieceVal }));
  });

  await Promise.all(tasks);
  toast('Prices saved.', 'success');
  navigate('#/sales/price-groups/' + pgId);
}

async function pgAssignCustomer(pgId) {
  const sel = document.getElementById('pgAssignSelect');
  const custId = parseInt(sel.value, 10);
  if (!custId) { toast('Please select a customer.', 'warning'); return; }
  const result = await api.UpdateCustomer(custId, { price_group_id: pgId });
  if (result) {
    toast('Customer assigned.', 'success');
    navigate('#/sales/price-groups/' + pgId);
  } else {
    toast('Failed to assign customer.', 'danger');
  }
}

async function pgRemoveCustomer(pgId, custId) {
  if (!confirm('Remove this customer from the group?')) return;
  const result = await api.UpdateCustomer(custId, { price_group_id: null });
  if (result) {
    toast('Customer removed from group.', 'success');
    navigate('#/sales/price-groups/' + pgId);
  } else {
    toast('Failed to remove customer.', 'danger');
  }
}

async function pgDeleteGroup(pgId, name) {
  if (!confirm(`Delete "${name}"? Customers will lose their price group assignment.`)) return;
  const result = await api.DeletePriceGroup(pgId);
  if (result) {
    toast('Price group deleted.', 'success');
    navigate('#/sales/price-groups');
  } else {
    toast('Failed to delete price group.', 'danger');
  }
}
