// sales-customers.js

// ── Customers ─────────────────────────────────────────────────────────────────

let _customerList = [];

function _custRow(c) {
  const pgBadge = c.price_group
    ? `<span class="badge bg-primary bg-opacity-10 text-primary rounded-pill" style="font-size:11px;">${c.price_group.name}</span>`
    : `<span class="text-muted small">—</span>`;
  const typeCls = c.customer_type === 'Account' ? 'bg-primary' : 'bg-secondary';
  return `
    <tr>
      <td class="fw-semibold">${c.name || '—'}</td>
      <td><span class="badge ${typeCls} rounded-pill" style="font-size:10px;">${c.customer_type || 'Walk-in'}</span></td>
      <td>${c.contact_number || '—'}</td>
      <td>${pgBadge}</td>
      <td class="text-end">
        ${c.price_group
          ? `<button class="btn btn-sm btn-outline-info py-0 me-1" onclick="navigate('#/sales/price-groups/${c.price_group.id}')" title="View Price List">
              <i class="bi bi-tag"></i>
            </button>`
          : ''}
        <button class="btn btn-sm btn-outline-secondary py-0" onclick="openCustomerModal(${c.id})">
          <i class="bi bi-pencil"></i>
        </button>
      </td>
    </tr>`;
}

function filterCustomerList() {
  const filterVal = (document.getElementById('custFilterGroup') || {}).value || '';
  let filtered = _customerList;
  if (filterVal === '__none__') {
    filtered = _customerList.filter(c => !c.price_group_id);
  } else if (filterVal) {
    const pgId = parseInt(filterVal, 10);
    filtered = _customerList.filter(c => c.price_group_id === pgId);
  }
  const tbody = document.getElementById('custTableBody');
  const label = document.getElementById('custCountLabel');
  if (tbody) tbody.innerHTML = filtered.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No customers match this filter.</td></tr>`
    : filtered.map(_custRow).join('');
  if (label) label.textContent = `${filtered.length} customer(s)`;
}

async function loadCustomersView(editId, tabBar) {
  const [customers, groups] = await Promise.all([
    api.ListCustomers(false),
    api.ListPriceGroups(false),
  ]);
  _customerList  = customers || [];
  _priceGroupList = groups || [];

  const pgOptions = _priceGroupList
    .filter(g => g.is_active)
    .map(g => `<option value="${g.id}">${g.name}</option>`).join('');

  const rows = _customerList.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No customers found.</td></tr>`
    : _customerList.map(_custRow).join('');

  const pgFilterOptions = _priceGroupList
    .filter(g => g.is_active)
    .map(g => `<option value="${g.id}">${g.name}</option>`).join('');

  showView(wrapSales(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <div class="d-flex align-items-center gap-2">
        <span class="text-muted small" id="custCountLabel">${_customerList.length} customer(s)</span>
        <select class="form-select form-select-sm" id="custFilterGroup"
                style="width:auto;" onchange="filterCustomerList()">
          <option value="">All Groups</option>
          <option value="__none__">Unassigned</option>
          ${pgFilterOptions}
        </select>
      </div>
      <button class="btn btn-primary btn-sm" onclick="openCustomerModal(null)">
        <i class="bi bi-plus-lg me-1"></i>Add Customer
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Name</th><th>Type</th><th>Contact</th><th>Price Group</th><th></th></tr>
          </thead>
          <tbody id="custTableBody">${rows}</tbody>
        </table>
      </div>
    </div>

    <!-- Customer Modal -->
    <div class="modal fade" id="customerModal" tabindex="-1">
      <div class="modal-dialog modal-lg">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="customerModalTitle">Customer</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <input type="hidden" id="customerId">
            <div class="row g-3">
              <div class="col-md-8">
                <label class="form-label fw-bold small">Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="customerName" required>
              </div>
              <div class="col-md-4">
                <label class="form-label fw-bold small">Type</label>
                <select class="form-select" id="customerType">
                  <option value="Walk-in">Walk-in</option>
                  <option value="Account">Account</option>
                </select>
              </div>
              <div class="col-md-6">
                <label class="form-label fw-bold small">Contact Number</label>
                <input type="text" class="form-control" id="customerContactNumber">
              </div>
              <div class="col-md-6">
                <label class="form-label fw-bold small">Email</label>
                <input type="email" class="form-control" id="customerEmail">
              </div>
              <div class="col-12">
                <label class="form-label fw-bold small">Address</label>
                <input type="text" class="form-control" id="customerAddress">
              </div>
              <div class="col-12">
                <label class="form-label fw-bold small">Delivery Address</label>
                <input type="text" class="form-control" id="customerDeliveryAddress" placeholder="If different from above">
              </div>
              <div class="col-md-6">
                <label class="form-label fw-bold small">Price Group</label>
                <select class="form-select" id="customerPriceGroupId" onchange="updateCustPricePreview()">
                  <option value="">— None (manual pricing) —</option>
                  ${pgOptions}
                </select>
                <div class="form-text">Prices auto-fill on new sales orders.</div>
                <div id="custPricePreview" class="mt-2"></div>
              </div>
              <div class="col-12">
                <label class="form-label fw-bold small">Notes</label>
                <textarea class="form-control" id="customerNotes" rows="2"></textarea>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitCustomerForm()">Save</button>
          </div>
        </div>
      </div>
    </div>
  `));
}

function openCustomerModal(id) {
  const isEdit = id !== null;
  document.getElementById('customerModalTitle').textContent = isEdit ? 'Edit Customer' : 'Add Customer';
  document.getElementById('customerId').value = id || '';
  if (isEdit) {
    const c = _customerList.find(x => x.id === id);
    if (c) {
      document.getElementById('customerName').value             = c.name || '';
      document.getElementById('customerType').value             = c.customer_type || 'Walk-in';
      document.getElementById('customerContactNumber').value    = c.contact_number || '';
      document.getElementById('customerEmail').value            = c.email || '';
      document.getElementById('customerAddress').value          = c.address || '';
      document.getElementById('customerDeliveryAddress').value  = c.delivery_address || '';
      document.getElementById('customerPriceGroupId').value     = c.price_group_id || '';
      document.getElementById('customerNotes').value            = c.notes || '';
    }
  } else {
    document.getElementById('customerId').value            = '';
    document.getElementById('customerName').value          = '';
    document.getElementById('customerType').value          = 'Walk-in';
    document.getElementById('customerContactNumber').value = '';
    document.getElementById('customerEmail').value         = '';
    document.getElementById('customerAddress').value       = '';
    document.getElementById('customerDeliveryAddress').value = '';
    document.getElementById('customerPriceGroupId').value  = '';
    document.getElementById('customerNotes').value         = '';
  }
  updateCustPricePreview();
  new bootstrap.Modal(document.getElementById('customerModal')).show();
}

function updateCustPricePreview() {
  const el = document.getElementById('custPricePreview');
  if (!el) return;
  const pgId = parseInt(document.getElementById('customerPriceGroupId').value, 10);
  if (!pgId) { el.innerHTML = ''; return; }
  const pg = _priceGroupList.find(g => g.id === pgId);
  if (!pg || !(pg.items || []).length) {
    el.innerHTML = `<div class="text-muted small fst-italic">No prices set for this group.</div>`;
    return;
  }
  const priceMap = {};
  pg.items.forEach(i => { priceMap[`${i.egg_size}_${i.unit}`] = i.price; });
  const rows = PG_EGG_SIZES.map(size => {
    const tray  = priceMap[`${size}_Tray`];
    const piece = priceMap[`${size}_Piece`];
    if (tray == null && piece == null) return '';
    return `<tr>
      <td class="py-1 ps-2 small fw-semibold">${size}</td>
      <td class="py-1 text-end small">${tray  != null ? '₱' + Number(tray).toFixed(2)  : '—'}</td>
      <td class="py-1 text-end small">${piece != null ? '₱' + Number(piece).toFixed(4) : '—'}</td>
    </tr>`;
  }).join('');
  el.innerHTML = rows ? `
    <div class="border rounded" style="font-size:12px;max-height:160px;overflow-y:auto;">
      <table class="table table-sm mb-0">
        <thead class="table-light"><tr>
          <th class="ps-2 py-1">Egg Size</th>
          <th class="text-end py-1">Tray</th>
          <th class="text-end py-1">Piece</th>
        </tr></thead>
        <tbody>${rows}</tbody>
      </table>
    </div>` : '';
}

async function submitCustomerForm() {
  const id   = document.getElementById('customerId').value;
  const pgRaw = document.getElementById('customerPriceGroupId').value;
  const payload = {
    name:             document.getElementById('customerName').value.trim(),
    customer_type:    document.getElementById('customerType').value,
    contact_number:   document.getElementById('customerContactNumber').value.trim(),
    email:            document.getElementById('customerEmail').value.trim(),
    address:          document.getElementById('customerAddress').value.trim(),
    delivery_address: document.getElementById('customerDeliveryAddress').value.trim(),
    price_group_id:   pgRaw ? parseInt(pgRaw, 10) : null,
    notes:            document.getElementById('customerNotes').value.trim(),
  };
  if (!payload.name) { toast('Customer name is required.', 'warning'); return; }

  const result = id
    ? await api.UpdateCustomer(parseInt(id, 10), payload)
    : await api.CreateCustomer(payload);

  if (result) {
    toast(id ? 'Customer updated.' : 'Customer created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('customerModal')).hide();
    Modules.Sales.load('customers');
  } else {
    toast('Failed to save customer.', 'danger');
  }
}

function _resetSalesCustomers() {
  _customerList = [];
}
