// purchasing.js — Modules.Purchasing

Modules.Purchasing = {
  async load(sub, id, action) {
    const active = sub || 'purchases';
    showLoading();

    // Detail / form views (no tab bar needed)
    if (active === 'purchases' && action === 'new')    { await loadNewPurchaseForm(); return; }
    if (active === 'purchases' && id)                  { await loadPurchaseDetail(id); return; }
    if (active === 'delivery-receipts' && id)          { await loadDRDetail(id); return; }
    if (active === 'ap-invoices' && action === 'new')  { await loadNewAPInvoiceForm(); return; }
    if (active === 'ap-invoices' && id)                { await loadAPInvoiceDetail(id); return; }
    if (active === 'ap-payments' && action === 'new')  { await loadNewAPPaymentForm(); return; }
    if (active === 'ap-payments' && id)                { await loadAPPaymentDetail(id); return; }

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'purchases' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/purchases');return false;">
            <i class="bi bi-cart me-1"></i>Purchases
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'delivery-receipts' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/delivery-receipts');return false;">
            <i class="bi bi-truck me-1"></i>Delivery Receipts
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ap-invoices' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/ap-invoices');return false;">
            <i class="bi bi-file-earmark-text me-1"></i>AP Invoices
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ap-payments' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/ap-payments');return false;">
            <i class="bi bi-cash me-1"></i>AP Payments
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'suppliers' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/suppliers');return false;">
            <i class="bi bi-building me-1"></i>Suppliers
          </a>
        </li>
      </ul>
    `;

    if (active === 'purchases')         await loadPurchasesList(tabBar);
    else if (active === 'delivery-receipts') await loadDeliveryReceiptsList(tabBar);
    else if (active === 'ap-invoices')  await loadApInvoicesList(tabBar);
    else if (active === 'ap-payments')  await loadApPaymentsList(tabBar);
    else if (active === 'suppliers')    await this.loadSuppliers(id, tabBar);
    else await loadPurchasesList(tabBar);
  },

  async loadSuppliers(id, tabBar) {
    await loadSuppliersView(id, tabBar);
  }
};

// ── Status Badge Helper ───────────────────────────────────────────────────────

function purchStatusBadge(status) {
  const map = {
    Draft:     'secondary',
    Pending:   'warning',
    Received:  'success',
    Partial:   'info',
    Paid:      'success',
    Cancelled: 'danger',
    Approved:  'primary',
    Overdue:   'danger',
  };
  const color = map[status] || 'secondary';
  return `<span class="badge bg-${color}">${status || '—'}</span>`;
}

// ── Purchases List ────────────────────────────────────────────────────────────

async function loadPurchasesList(tabBar) {
  const list = await api.ListPurchases();
  if (!list) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load purchases.</div>`));
    return;
  }

  const rows = list.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No purchases found.</td></tr>`
    : list.map(p => `
        <tr>
          <td class="fw-semibold">${p.po_number || '—'}</td>
          <td>${formatDate(p.order_date)}</td>
          <td>${p.supplier_name || '—'}</td>
          <td>${formatCurrency(p.total_amount ?? 0)}</td>
          <td>${purchStatusBadge(p.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="viewPurchase(${p.id})">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapPurch(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small">${list.length} record(s)</span>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/purchasing/purchases/new')">
        <i class="bi bi-plus-lg me-1"></i>New Purchase Order
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>PO #</th><th>Date</th><th>Supplier</th><th>Amount</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `));
}

function viewPurchase(id) { navigate(`#/purchasing/purchases/${id}`); }

// ── Delivery Receipts ─────────────────────────────────────────────────────────

async function loadDeliveryReceiptsList(tabBar) {
  const list = await api.ListDeliveryReceipts(0);
  if (!list) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load delivery receipts.</div>`));
    return;
  }

  const rows = list.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No delivery receipts found.</td></tr>`
    : list.map(r => `
        <tr>
          <td class="fw-semibold">${r.dr_number || '—'}</td>
          <td>${formatDate(r.delivery_date)}</td>
          <td>${r.supplier_name || '—'}</td>
          <td>${r.po_number || '—'}</td>
          <td>${purchStatusBadge(r.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/delivery-receipts/${r.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapPurch(tabBar, `
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>DR #</th><th>Delivery Date</th><th>Supplier</th><th>PO #</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `));
}

// ── AP Invoices ───────────────────────────────────────────────────────────────

async function loadApInvoicesList(tabBar) {
  const list = await api.ListAPInvoices(null, '');
  if (!list) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load AP invoices.</div>`));
    return;
  }

  const rows = list.length === 0
    ? `<tr><td colspan="7" class="text-center text-muted py-4">No AP invoices found.</td></tr>`
    : list.map(i => `
        <tr>
          <td class="fw-semibold">${i.invoice_number || '—'}</td>
          <td>${formatDate(i.invoice_date)}</td>
          <td>${formatDate(i.due_date)}</td>
          <td>${i.supplier_name || '—'}</td>
          <td>${formatCurrency(i.total_amount ?? 0)}</td>
          <td>${purchStatusBadge(i.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-invoices/${i.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapPurch(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="navigate('#/purchasing/ap-invoices/new')">
        <i class="bi bi-plus-lg me-1"></i>New AP Invoice
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Invoice #</th><th>Date</th><th>Due</th><th>Supplier</th><th>Amount</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `));
}

// ── AP Payments ───────────────────────────────────────────────────────────────

async function loadApPaymentsList(tabBar) {
  const list = await api.ListAPPayments();
  if (!list) {
    showView(wrapPurch(tabBar, `<div class="alert alert-warning">Failed to load AP payments.</div>`));
    return;
  }

  const rows = list.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No AP payments found.</td></tr>`
    : list.map(p => `
        <tr>
          <td class="fw-semibold">${p.payment_number || '—'}</td>
          <td>${formatDate(p.date)}</td>
          <td>${p.supplier_name_snapshot || '—'}</td>
          <td>${p.payment_method || '—'}</td>
          <td>${formatCurrency(p.total_amount ?? 0)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-payments/${p.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapPurch(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="navigate('#/purchasing/ap-payments/new')">
        <i class="bi bi-plus-lg me-1"></i>Record Payment
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Payment #</th><th>Date</th><th>Supplier</th><th>Method</th><th>Amount</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `));
}

// ── Suppliers ─────────────────────────────────────────────────────────────────

let _supplierList = [];

async function loadSuppliersView(editId, tabBar) {
  const suppliers = await api.ListSuppliers(false);
  _supplierList = suppliers || [];

  const rows = _supplierList.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No suppliers found.</td></tr>`
    : _supplierList.map(s => `
        <tr>
          <td class="fw-semibold">${s.name || '—'}</td>
          <td>${s.contact_person || '—'}</td>
          <td>${s.contact_number || '—'}</td>
          <td>${s.email || '—'}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="openSupplierModal(${s.id})">
              <i class="bi bi-pencil"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapPurch(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small">${_supplierList.length} supplier(s)</span>
      <button class="btn btn-primary btn-sm" onclick="openSupplierModal(null)">
        <i class="bi bi-plus-lg me-1"></i>Add Supplier
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Name</th><th>Contact</th><th>Phone</th><th>Email</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>

    <!-- Supplier Modal -->
    <div class="modal fade" id="supplierModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="supplierModalTitle">Supplier</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="supplierForm">
              <input type="hidden" id="supplierId">
              <div class="mb-3">
                <label class="form-label">Company Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="supplierName" required>
              </div>
              <div class="mb-3">
                <label class="form-label">Contact Person</label>
                <input type="text" class="form-control" id="supplierContact">
              </div>
              <div class="row g-3">
                <div class="col-md-6">
                  <label class="form-label">Phone</label>
                  <input type="text" class="form-control" id="supplierPhone">
                </div>
                <div class="col-md-6">
                  <label class="form-label">Email</label>
                  <input type="email" class="form-control" id="supplierEmail">
                </div>
              </div>
              <div class="mb-3 mt-3">
                <label class="form-label">Address</label>
                <textarea class="form-control" id="supplierAddress" rows="2"></textarea>
              </div>
            </form>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitSupplierForm()">Save</button>
          </div>
        </div>
      </div>
    </div>
  `));
}

function openSupplierModal(id) {
  const isEdit = id !== null;
  document.getElementById('supplierModalTitle').textContent = isEdit ? 'Edit Supplier' : 'Add Supplier';
  document.getElementById('supplierId').value = id || '';
  if (isEdit) {
    const s = _supplierList.find(x => x.id === id);
    if (s) {
      document.getElementById('supplierName').value = s.name || '';
      document.getElementById('supplierContact').value = s.contact_person || '';
      document.getElementById('supplierPhone').value = s.contact_number || '';
      document.getElementById('supplierEmail').value = s.email || '';
      document.getElementById('supplierAddress').value = s.address || '';
    }
  } else {
    document.getElementById('supplierForm').reset();
  }
  new bootstrap.Modal(document.getElementById('supplierModal')).show();
}

async function submitSupplierForm() {
  const id = document.getElementById('supplierId').value;
  const payload = {
    name:           document.getElementById('supplierName').value.trim(),
    contact_person: document.getElementById('supplierContact').value.trim(),
    contact_number: document.getElementById('supplierPhone').value.trim(),
    email:          document.getElementById('supplierEmail').value.trim(),
    address:        document.getElementById('supplierAddress').value.trim(),
  };
  if (!payload.name) { toast('Supplier name is required.', 'warning'); return; }

  const result = id
    ? await api.UpdateSupplier(parseInt(id, 10), payload)
    : await api.CreateSupplier(payload);

  if (result) {
    toast(id ? 'Supplier updated.' : 'Supplier created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('supplierModal')).hide();
    navigate('#/purchasing/suppliers');
  } else {
    toast('Failed to save supplier.', 'danger');
  }
}

// ── New Purchase Order Form ───────────────────────────────────────────────────

async function loadNewPurchaseForm() {
  const suppliers = await api.ListSuppliers(true) || [];
  const supplierOptions = suppliers.map(s =>
    `<option value="${s.id}" data-name="${s.name}">${s.name}</option>`
  ).join('');

  const today = new Date().toISOString().slice(0, 10);

  showView(`
    <div class="container p-4" style="max-width:720px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/purchases')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">New Purchase Order</h4>
      </div>
      <div class="card border-0 shadow-sm">
        <div class="card-body">
          <form id="newPurchaseForm">
            <div class="row g-3">
              <div class="col-md-6">
                <label class="form-label">Date <span class="text-danger">*</span></label>
                <input type="date" class="form-control" id="poDate" required value="${today}">
              </div>
              <div class="col-md-6">
                <label class="form-label">Supplier</label>
                <select class="form-select" id="poSupplierID" onchange="poFillSupplierName()">
                  <option value="">— Select supplier —</option>
                  ${supplierOptions}
                </select>
              </div>
              <div class="col-md-6">
                <label class="form-label">Supplier Name (snapshot)</label>
                <input type="text" class="form-control" id="poSupplierName" placeholder="Auto-filled or type manually">
              </div>
              <div class="col-md-6">
                <label class="form-label">Category</label>
                <select class="form-select" id="poCategory">
                  <option value="">— Select —</option>
                  <option>Feeds</option>
                  <option>Medicine</option>
                  <option>Supplies</option>
                  <option>Equipment</option>
                  <option>Other</option>
                </select>
              </div>
              <div class="col-md-6">
                <label class="form-label">Item Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="poItemName" required placeholder="e.g. Layer Feeds 50kg">
              </div>
              <div class="col-md-3">
                <label class="form-label">Quantity <span class="text-danger">*</span></label>
                <input type="number" class="form-control" id="poQty" required min="0.001" step="0.001" placeholder="0"
                  oninput="poCalcTotal()">
              </div>
              <div class="col-md-3">
                <label class="form-label">Unit</label>
                <input type="text" class="form-control" id="poUnit" placeholder="sacks, pcs, kg…">
              </div>
              <div class="col-md-4">
                <label class="form-label">Unit Price</label>
                <input type="number" class="form-control" id="poUnitPrice" min="0" step="0.01" placeholder="0.00"
                  oninput="poCalcTotal()">
              </div>
              <div class="col-md-4">
                <label class="form-label">Total Cost</label>
                <input type="number" class="form-control" id="poTotalCost" min="0" step="0.01" placeholder="0.00" readonly>
              </div>
              <div class="col-md-4">
                <label class="form-label">Payment Method</label>
                <select class="form-select" id="poPaymentMethod">
                  <option>Cash</option>
                  <option>Bank Transfer</option>
                  <option>GCash</option>
                  <option>Check</option>
                  <option>Credit</option>
                </select>
              </div>
              <div class="col-md-4">
                <label class="form-label">Payment Status</label>
                <select class="form-select" id="poPaymentStatus">
                  <option value="Unpaid">Unpaid</option>
                  <option value="Paid" selected>Paid</option>
                  <option value="Partial">Partial</option>
                </select>
              </div>
              <div class="col-md-4">
                <label class="form-label">Amount Paid</label>
                <input type="number" class="form-control" id="poAmountPaid" min="0" step="0.01" placeholder="0.00">
              </div>
              <div class="col-md-4">
                <label class="form-label">Received By</label>
                <input type="text" class="form-control" id="poReceivedBy">
              </div>
              <div class="col-md-6">
                <label class="form-label">PO Number</label>
                <input type="text" class="form-control" id="poPONumber" placeholder="Auto or manual">
              </div>
              <div class="col-md-6">
                <label class="form-label">Invoice Number</label>
                <input type="text" class="form-control" id="poInvoiceNumber">
              </div>
              <div class="col-12">
                <label class="form-label">Remarks</label>
                <textarea class="form-control" id="poRemarks" rows="2"></textarea>
              </div>
            </div>
            <div class="d-flex gap-2 mt-4">
              <button type="submit" class="btn btn-primary px-4">
                <i class="bi bi-check-lg me-1"></i>Save Purchase Order
              </button>
              <button type="button" class="btn btn-outline-secondary"
                onclick="navigate('#/purchasing/purchases')">Cancel</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  `);

  document.getElementById('newPurchaseForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const qty = parseFloat(document.getElementById('poQty').value) || 0;
    const unitPrice = parseFloat(document.getElementById('poUnitPrice').value) || 0;
    const totalCost = parseFloat(document.getElementById('poTotalCost').value) || (qty * unitPrice);
    const supplierIDRaw = document.getElementById('poSupplierID').value;
    const payload = {
      date:           document.getElementById('poDate').value,
      category:       document.getElementById('poCategory').value,
      item_name:      document.getElementById('poItemName').value.trim(),
      quantity:       qty,
      unit:           document.getElementById('poUnit').value.trim(),
      unit_price:     unitPrice,
      total_cost:     totalCost,
      supplier:       document.getElementById('poSupplierName').value.trim(),
      supplier_id:    supplierIDRaw ? parseInt(supplierIDRaw, 10) : null,
      po_number:      document.getElementById('poPONumber').value.trim(),
      invoice_number: document.getElementById('poInvoiceNumber').value.trim(),
      received_by:    document.getElementById('poReceivedBy').value.trim(),
      payment_status: document.getElementById('poPaymentStatus').value,
      amount_paid:    parseFloat(document.getElementById('poAmountPaid').value) || 0,
      payment_method: document.getElementById('poPaymentMethod').value,
      remarks:        document.getElementById('poRemarks').value.trim(),
    };
    const btn = e.target.querySelector('[type=submit]');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';
    const result = await api.CreatePurchase(payload);
    if (result) {
      toast('Purchase order saved.', 'success');
      navigate('#/purchasing/purchases');
    } else {
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-check-lg me-1"></i>Save Purchase Order';
    }
  });
}

function poFillSupplierName() {
  const sel = document.getElementById('poSupplierID');
  const opt = sel.options[sel.selectedIndex];
  if (opt && opt.dataset.name) {
    document.getElementById('poSupplierName').value = opt.dataset.name;
  }
}

function poCalcTotal() {
  const qty = parseFloat(document.getElementById('poQty').value) || 0;
  const price = parseFloat(document.getElementById('poUnitPrice').value) || 0;
  document.getElementById('poTotalCost').value = (qty * price).toFixed(2);
}

// ── Purchase Detail View ──────────────────────────────────────────────────────

async function loadPurchaseDetail(id) {
  showLoading();
  const p = await api.GetPurchase(id);
  if (!p) {
    showView(`<div class="alert alert-warning m-4">Purchase not found.</div>`);
    return;
  }

  const drs = (p.delivery_receipts || []).map(dr => `
    <tr>
      <td class="fw-semibold">${dr.dr_number || '—'}</td>
      <td>${formatDate(dr.date)}</td>
      <td>${purchStatusBadge(dr.status)}</td>
      <td>${formatCurrency(dr.amount_invoiced ?? 0)}</td>
    </tr>
  `).join('') || `<tr><td colspan="4" class="text-muted text-center py-3">No delivery receipts.</td></tr>`;

  const aps = (p.ap_invoices || []).map(inv => `
    <tr>
      <td class="fw-semibold">${inv.invoice_number || '—'}</td>
      <td>${formatDate(inv.date)}</td>
      <td>${purchStatusBadge(inv.status)}</td>
      <td>${formatCurrency(inv.total_amount ?? 0)}</td>
      <td>${formatCurrency(inv.amount_paid_stored ?? 0)}</td>
    </tr>
  `).join('') || `<tr><td colspan="5" class="text-muted text-center py-3">No AP invoices.</td></tr>`;

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/purchases')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">Purchase Order</h4>
        <span class="badge bg-secondary ms-1">${p.po_number || '—'}</span>
        ${purchStatusBadge(p.payment_status)}
      </div>

      <div class="row g-4 mb-4">
        <div class="col-md-6">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body">
              <h6 class="text-muted mb-3">Purchase Details</h6>
              <dl class="row mb-0 small">
                <dt class="col-sm-5">Date</dt><dd class="col-sm-7">${formatDate(p.date)}</dd>
                <dt class="col-sm-5">Supplier</dt><dd class="col-sm-7">${p.supplier || '—'}</dd>
                <dt class="col-sm-5">Category</dt><dd class="col-sm-7">${p.category || '—'}</dd>
                <dt class="col-sm-5">Item</dt><dd class="col-sm-7">${p.item_name || '—'}</dd>
                <dt class="col-sm-5">Quantity</dt><dd class="col-sm-7">${formatNumber(p.quantity ?? 0)} ${p.unit || ''}</dd>
                <dt class="col-sm-5">Unit Price</dt><dd class="col-sm-7">${formatCurrency(p.unit_price ?? 0)}</dd>
                <dt class="col-sm-5">Total Cost</dt><dd class="col-sm-7 fw-semibold">${formatCurrency(p.total_cost ?? 0)}</dd>
              </dl>
            </div>
          </div>
        </div>
        <div class="col-md-6">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body">
              <h6 class="text-muted mb-3">Payment</h6>
              <dl class="row mb-0 small">
                <dt class="col-sm-5">Status</dt><dd class="col-sm-7">${purchStatusBadge(p.payment_status)}</dd>
                <dt class="col-sm-5">Method</dt><dd class="col-sm-7">${p.payment_method || '—'}</dd>
                <dt class="col-sm-5">Amount Paid</dt><dd class="col-sm-7">${formatCurrency(p.amount_paid ?? 0)}</dd>
                <dt class="col-sm-5">Amount Received</dt><dd class="col-sm-7">${formatCurrency(p.amount_received ?? 0)}</dd>
                <dt class="col-sm-5">Invoiced</dt><dd class="col-sm-7">${formatCurrency(p.amount_invoiced ?? 0)}</dd>
                <dt class="col-sm-5">Settled</dt><dd class="col-sm-7">${formatCurrency(p.amount_settled ?? 0)}</dd>
                <dt class="col-sm-5">Received By</dt><dd class="col-sm-7">${p.received_by || '—'}</dd>
              </dl>
            </div>
          </div>
        </div>
      </div>

      <!-- Delivery Receipts -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-header bg-transparent d-flex justify-content-between align-items-center fw-semibold">
          <span><i class="bi bi-truck me-2"></i>Delivery Receipts</span>
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>DR #</th><th>Date</th><th>Status</th><th>Amount Invoiced</th></tr>
            </thead>
            <tbody>${drs}</tbody>
          </table>
        </div>
      </div>

      <!-- AP Invoices -->
      <div class="card border-0 shadow-sm">
        <div class="card-header bg-transparent fw-semibold">
          <i class="bi bi-file-earmark-text me-2"></i>AP Invoices
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Invoice #</th><th>Date</th><th>Status</th><th>Total</th><th>Paid</th></tr>
            </thead>
            <tbody>${aps}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}

// ── Delivery Receipt Detail ───────────────────────────────────────────────────

async function loadDRDetail(id) {
  showLoading();
  const dr = await api.GetDeliveryReceipt(id);
  if (!dr) {
    showView(`<div class="alert alert-warning m-4">Delivery receipt not found.</div>`);
    return;
  }

  const po     = dr.purchase || {};
  const items  = dr.items || [];
  const apInvs = dr.ap_invoices || [];

  const total = items.reduce((s, i) => s + (i.quantity_received * i.unit_price), 0);
  const uninvoiced = total - (dr.amount_invoiced || 0);

  const statusColor = { Received: 'success', Cancelled: 'secondary', Draft: 'warning' };
  const badge = `<span class="badge bg-${statusColor[dr.status] || 'secondary'} fs-6">${dr.status}</span>`;

  const actionBtns = dr.status === 'Draft' ? `
    <button class="btn btn-sm btn-success" onclick="confirmDR(${dr.id})">
      <i class="bi bi-check-lg me-1"></i>Confirm Receipt
    </button>
    <button class="btn btn-sm btn-outline-danger" onclick="cancelDR(${dr.id})">Cancel</button>
  ` : (dr.status === 'Received' && uninvoiced > 0.005 ? `
    <button class="btn btn-sm btn-primary" onclick="newAPInvoiceFromDR(${dr.id})">
      <i class="bi bi-file-earmark-plus me-1"></i>Create AP Invoice
    </button>
  ` : '');

  const invoiceStatusCard = dr.status === 'Received' ? `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-2">
        <span class="small fw-bold text-muted">INVOICE STATUS</span>
      </div>
      <div class="card-body">
        <div class="row g-2 text-center">
          <div class="col">
            <div class="small text-muted mb-1">DR Total</div>
            <div class="fw-bold">${formatCurrency(total)}</div>
          </div>
          <div class="col-auto d-flex align-items-center text-muted">→</div>
          <div class="col">
            <div class="small text-muted mb-1">AP Invoiced</div>
            <div class="fw-bold text-primary">${formatCurrency(dr.amount_invoiced || 0)}</div>
          </div>
          <div class="col-auto d-flex align-items-center text-muted">→</div>
          <div class="col">
            <div class="small text-muted mb-1">Uninvoiced</div>
            <div class="fw-bold ${uninvoiced > 0.005 ? 'text-danger' : 'text-success'}">
              ${uninvoiced > 0.005 ? formatCurrency(uninvoiced) : '<i class="bi bi-check-circle-fill me-1"></i>Fully Invoiced'}
            </div>
          </div>
        </div>
      </div>
    </div>
  ` : '';

  const itemRows = items.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-3">No items.</td></tr>`
    : items.map(i => `
      <tr>
        <td class="ps-4"><span class="badge bg-light text-dark border">${i.category || '—'}</span></td>
        <td class="fw-semibold">${i.item_name}</td>
        <td class="text-end text-muted">${formatNumber(i.quantity_ordered)} ${i.unit}</td>
        <td class="text-end fw-bold">${formatNumber(i.quantity_received)} ${i.unit}</td>
        <td class="text-end">${formatCurrency(i.unit_price)}</td>
        <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity_received * i.unit_price)}</td>
      </tr>`).join('');

  const apRows = apInvs.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between">
        <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-primary"></i>AP Invoices</span>
        ${uninvoiced > 0.005 ? `<button class="btn btn-sm btn-outline-primary" onclick="newAPInvoiceFromDR(${dr.id})">+ Add AP Invoice</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th class="ps-4">Invoice #</th><th>Date</th><th class="text-end">Total</th>
                <th class="text-end">Paid</th><th class="text-end">Balance</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            ${apInvs.map(inv => {
              const bal = (inv.total_amount || 0) - (inv.amount_paid_stored || 0);
              return `<tr>
                <td class="ps-4 fw-semibold">${inv.invoice_number}</td>
                <td class="text-muted small">${formatDate(inv.date)}</td>
                <td class="text-end">${formatCurrency(inv.total_amount || 0)}</td>
                <td class="text-end text-success small">${formatCurrency(inv.amount_paid_stored || 0)}</td>
                <td class="text-end fw-bold ${bal <= 0.005 ? 'text-success' : 'text-danger'}">${bal <= 0.005 ? 'Paid' : formatCurrency(bal)}</td>
                <td>${purchStatusBadge(inv.status)}</td>
                <td class="text-end pe-3">
                  <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-invoices/${inv.id}')">View</button>
                </td>
              </tr>`;
            }).join('')}
          </tbody>
        </table>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/delivery-receipts')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-box-seam me-2 text-success"></i>${dr.dr_number}</h4>
          <small class="text-muted">Delivery Receipt · ${formatDate(dr.date)}</small>
        </div>
        ${badge}
        ${actionBtns}
      </div>

      ${invoiceStatusCard}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-info-circle me-2"></i>Receipt Details</span>
        </div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <div class="small text-muted">Purchase Order</div>
              <div class="fw-semibold">
                <a href="#" onclick="navigate('#/purchasing/purchases/${dr.purchase_id}');return false;" class="text-decoration-none">
                  ${po.po_number || ('PO-' + String(dr.purchase_id).padStart(5, '0'))}
                </a>
              </div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted">Supplier</div>
              <div class="fw-semibold">${po.supplier || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted">Supplier DR Ref #</div>
              <div>${dr.supplier_dr_ref || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted">Received By</div>
              <div>${dr.received_by || '—'}</div>
            </div>
            ${dr.notes ? `<div class="col-12"><div class="small text-muted">Notes</div><div>${dr.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-box-seam me-2"></i>Items Received</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr><th class="ps-4">Category</th><th>Item</th><th class="text-end">Ordered</th>
                  <th class="text-end">Received</th><th class="text-end">Unit Price</th><th class="text-end pe-4">Line Total</th></tr>
            </thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4">Total</td>
                <td class="text-end pe-4 text-primary fs-5">${formatCurrency(total)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      ${apRows}
    </div>
  `);
}

async function confirmDR(id) {
  if (!confirm('Confirm delivery and update inventory?')) return;
  const r = await api.ConfirmDeliveryReceipt(id);
  if (r) await loadDRDetail(id);
}

async function cancelDR(id) {
  if (!confirm('Cancel this delivery receipt?')) return;
  const r = await api.CancelDeliveryReceipt(id);
  if (r) navigate(`#/purchasing/delivery-receipts/${id}`);
}

function newAPInvoiceFromDR(drId) {
  window._apInvoicePrefillDRId = drId;
  navigate('#/purchasing/ap-invoices/new');
}

// ── AP Invoice Detail ─────────────────────────────────────────────────────────

async function loadAPInvoiceDetail(id) {
  showLoading();
  const inv = await api.GetAPInvoice(id);
  if (!inv) {
    showView(`<div class="alert alert-warning m-4">AP Invoice not found.</div>`);
    return;
  }

  const items        = inv.items || [];
  const paymentLines = inv.payment_lines || [];
  const totalPaid    = inv.amount_paid_stored || 0;
  const balance      = Math.max(0, (inv.total_amount || 0) - totalPaid);
  const po           = inv.purchase || {};
  const dr           = inv.delivery_receipt || null;

  const statusColor  = { Paid: 'success', Partial: 'warning', Open: 'primary', Cancelled: 'secondary' };
  const badge        = `<span class="badge bg-${statusColor[inv.status] || 'secondary'} fs-6">${inv.status}</span>`;

  const canPay    = inv.status === 'Open' || inv.status === 'Partial';
  const canCancel = (inv.status === 'Open' || inv.status === 'Partial') && paymentLines.length === 0;

  const statusBanner = inv.status === 'Paid'
    ? `<div class="alert alert-success border-0 mb-3 d-flex align-items-center gap-2">
        <i class="bi bi-check-circle-fill fs-5"></i><strong>Fully Paid</strong> — ${formatCurrency(inv.total_amount)} settled</div>`
    : inv.status === 'Partial'
    ? `<div class="alert alert-warning border-0 mb-3 d-flex align-items-center gap-2">
        <i class="bi bi-clock-fill fs-5"></i>Balance: <strong>${formatCurrency(balance)}</strong> remaining</div>`
    : inv.status === 'Open'
    ? `<div class="alert alert-primary border-0 mb-3">${formatCurrency(inv.total_amount)} outstanding</div>`
    : '';

  const itemRows = items.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-3">No items.</td></tr>`
    : items.map(i => `
      <tr>
        <td class="ps-4"><span class="badge bg-light text-dark border">${i.category || '—'}</span></td>
        <td class="fw-semibold">${i.item_name}</td>
        <td class="text-center">${formatNumber(i.quantity)}</td>
        <td class="text-center text-muted">${i.unit}</td>
        <td class="text-center">${formatCurrency(i.unit_price)}</td>
        <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity * i.unit_price)}</td>
      </tr>`).join('');

  const payRows = paymentLines.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
        <span class="fw-bold"><i class="bi bi-cash-stack me-2 text-warning"></i>Payments Applied</span>
        ${canPay ? `<button class="btn btn-sm btn-outline-warning" onclick="newAPPaymentFromInvoice(${inv.id})">+ Add Payment</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th class="ps-4">Payment #</th><th>Date</th><th>Method</th><th>Reference</th>
                <th class="text-end pe-4">Amount Applied</th></tr>
          </thead>
          <tbody>
            ${paymentLines.map(pl => {
              const pay = pl.payment || {};
              return `<tr>
                <td class="ps-4 fw-bold">${pay.payment_number || '—'}</td>
                <td class="text-muted small">${formatDate(pay.date)}</td>
                <td>${pay.payment_method || '—'}</td>
                <td class="text-muted">${pay.reference_number || '—'}</td>
                <td class="text-end pe-4 fw-bold text-warning">${formatCurrency(pl.amount_applied)}</td>
              </tr>`;
            }).join('')}
          </tbody>
          <tfoot class="table-light fw-bold small">
            <tr>
              <td colspan="4" class="ps-4 text-muted">Invoice Total / Paid / Balance</td>
              <td class="text-end pe-4 text-primary">${formatCurrency(inv.total_amount)}</td>
            </tr>
            <tr>
              <td colspan="4"></td>
              <td class="text-end pe-4 text-success">${formatCurrency(totalPaid)} paid</td>
            </tr>
            <tr>
              <td colspan="4"></td>
              <td class="text-end pe-4 ${balance <= 0.005 ? 'text-success' : 'text-danger fw-bold'}">
                ${balance <= 0.005 ? '<i class="bi bi-check-circle-fill me-1"></i>Fully Paid' : formatCurrency(balance) + ' remaining'}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/ap-invoices')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-file-earmark-text me-2 text-primary"></i>${inv.invoice_number}</h4>
          <small class="text-muted">AP Invoice · ${formatDate(inv.date)}</small>
        </div>
        ${badge}
        ${canPay ? `<button class="btn btn-sm btn-warning text-white fw-bold" onclick="newAPPaymentFromInvoice(${inv.id})">
          <i class="bi bi-cash-stack me-1"></i>Record Payment</button>` : ''}
        ${canCancel ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelAPInv(${inv.id})">Cancel</button>` : ''}
      </div>

      ${statusBanner}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <div class="small text-muted fw-bold">SUPPLIER</div>
              <div class="fw-semibold">${inv.supplier_name || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">INVOICE DATE</div>
              <div>${formatDate(inv.date)}</div>
            </div>
            ${inv.due_date ? `<div class="col-md-4"><div class="small text-muted fw-bold">DUE DATE</div><div>${formatDate(inv.due_date)}</div></div>` : ''}
            <div class="col-md-4">
              <div class="small text-muted fw-bold">TERMS</div>
              <div>${inv.terms || '—'}</div>
            </div>
            ${inv.supplier_invoice_ref ? `<div class="col-md-4"><div class="small text-muted fw-bold">SUPPLIER INV REF #</div><div>${inv.supplier_invoice_ref}</div></div>` : ''}
            <div class="col-md-4">
              <div class="small text-muted fw-bold">PURCHASE ORDER</div>
              <a href="#" onclick="navigate('#/purchasing/purchases/${inv.purchase_id}');return false;" class="text-decoration-none fw-semibold">
                ${po.po_number || ('PO-' + String(inv.purchase_id).padStart(5, '0'))}
              </a>
            </div>
            ${dr ? `<div class="col-md-4"><div class="small text-muted fw-bold">DELIVERY RECEIPT</div>
              <a href="#" onclick="navigate('#/purchasing/delivery-receipts/${inv.delivery_receipt_id}');return false;" class="text-decoration-none fw-semibold">${dr.dr_number}</a></div>` : ''}
            ${inv.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${inv.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-box-seam me-2"></i>Line Items</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr><th class="ps-4">Category</th><th>Item</th><th class="text-center">Qty</th>
                  <th class="text-center">Unit</th><th class="text-center">Unit Price</th><th class="text-end pe-4">Total</th></tr>
            </thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4">Total Amount</td>
                <td class="text-end pe-4 text-primary fs-5">${formatCurrency(inv.total_amount)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      ${payRows}
    </div>
  `);
}

async function cancelAPInv(id) {
  if (!confirm('Cancel this AP Invoice? This will reverse the accounting entry.')) return;
  const r = await api.CancelAPInvoice(id);
  if (r) navigate(`#/purchasing/ap-invoices/${id}`);
}

function newAPPaymentFromInvoice(invId) {
  window._apPaymentPrefillInvId = invId;
  navigate('#/purchasing/ap-payments/new');
}

// ── New AP Invoice Form ───────────────────────────────────────────────────────

let _apInvPurchases = [];
let _apInvDRs       = [];

async function loadNewAPInvoiceForm() {
  showLoading();
  const [purchases, drs] = await Promise.all([
    api.ListPurchases(),
    api.ListDeliveryReceipts(0),
  ]);
  _apInvPurchases = purchases || [];
  _apInvDRs       = drs || [];

  const today = new Date().toISOString().slice(0, 10);
  const prefillDRId = window._apInvoicePrefillDRId || null;
  window._apInvoicePrefillDRId = null;

  const poOptions = _apInvPurchases.map(p =>
    `<option value="${p.id}" data-supplier="${p.supplier || ''}">${p.po_number || ('PO-' + String(p.id).padStart(5,'0'))} — ${p.supplier || 'Unknown'} — ${p.item_name || ''} (${formatCurrency(p.total_cost || 0)})</option>`
  ).join('');

  const drOptions = _apInvDRs.filter(r => r.status === 'Received').map(r =>
    `<option value="${r.id}" data-purchase-id="${r.purchase_id}">${r.dr_number} (${formatDate(r.date)})</option>`
  ).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-invoices')">
          <i class="bi bi-arrow-left"></i> Back
        </button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-file-earmark-text me-2" style="color:#8b5cf6"></i>New AP Invoice</h4>
          <div class="text-muted small">Copy from Purchase Order or Delivery Receipt</div>
        </div>
      </div>

      <!-- Source Selection -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold">Copy From</span></div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-6">
              <label class="form-label small fw-bold">Purchase Order <span class="text-danger">*</span></label>
              <select id="apPOSelect" class="form-select" onchange="apInvOnPOChange()">
                <option value="">— Select Purchase Order —</option>
                ${poOptions}
              </select>
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Delivery Receipt
                <span class="text-muted fw-normal">(optional)</span></label>
              <select id="apDRSelect" class="form-select" onchange="apInvOnDRChange()">
                <option value="">— Skip (copy from PO directly) —</option>
                ${drOptions}
              </select>
              <div class="form-text"><i class="bi bi-info-circle me-1"></i>If no DR: inventory updated on invoice creation.</div>
            </div>
          </div>
          <div class="row g-3 mt-1" id="apSupplierInfo" style="display:none">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Supplier</label>
              <input type="text" id="apSupplierName" class="form-control bg-light" disabled>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">PO Reference</label>
              <input type="text" id="apPORef" class="form-control bg-light" disabled>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Supplier Invoice Ref</label>
              <input type="text" id="apSupplierInvRef" class="form-control" placeholder="Supplier's invoice number">
            </div>
          </div>
        </div>
      </div>

      <!-- Dates & Terms -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">Invoice Date <span class="text-danger">*</span></label>
              <input type="date" id="apDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="apDueDate" class="form-control">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Terms</label>
              <select id="apTerms" class="form-select">
                <option>COD</option><option>7 days</option><option>15 days</option><option>30 days</option>
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="apNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <!-- Line Items -->
      <div class="card border-0 shadow-sm mb-3" id="apItemsCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between">
          <span class="fw-bold"><i class="bi bi-box me-2 text-warning"></i>Invoice Items</span>
          <span class="text-muted small">Adjust quantity for partial invoice</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:5%">
                  <input type="checkbox" class="form-check-input" id="apCheckAll" checked onchange="apToggleAll(this)">
                </th>
                <th>Item</th><th class="text-center">Category</th><th class="text-center">Unit</th>
                <th class="text-center">Available</th><th class="text-center">Qty to Invoice</th>
                <th class="text-center">Unit Price ₱</th><th class="text-end pe-4">Line Total</th>
              </tr>
            </thead>
            <tbody id="apItemsBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="7" class="text-end pe-3">Invoice Total</td>
                <td class="text-end pe-4 text-primary fs-5" id="apGrandTotal">₱0.00</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 justify-content-end">
        <button class="btn btn-outline-secondary px-4" onclick="navigate('#/purchasing/ap-invoices')">Cancel</button>
        <button class="btn btn-primary px-4" id="apSubmitBtn" disabled onclick="submitNewAPInvoice()">
          <i class="bi bi-file-earmark-text me-1"></i>Create AP Invoice
        </button>
      </div>
    </div>
  `);

  if (prefillDRId) {
    const drSel = document.getElementById('apDRSelect');
    if (drSel) {
      drSel.value = prefillDRId;
      await apInvOnDRChange();
    }
  }
}

async function apInvOnPOChange() {
  const drSel = document.getElementById('apDRSelect');
  if (drSel.value) return; // DR takes precedence
  const poId = document.getElementById('apPOSelect').value;
  if (!poId) { apHideItems(); return; }
  const po = _apInvPurchases.find(p => p.id === parseInt(poId));
  if (!po) return;
  apRenderItems([{
    item_name: po.item_name, category: po.category, unit: po.unit || '',
    quantity: po.quantity, unit_price: po.unit_price,
  }], po.supplier, po.po_number || ('PO-' + String(po.id).padStart(5,'0')));
}

async function apInvOnDRChange() {
  const drId = document.getElementById('apDRSelect').value;
  if (!drId) {
    await apInvOnPOChange();
    return;
  }
  const dr = await api.GetDeliveryReceipt(parseInt(drId));
  if (!dr) return;
  // Auto-select PO
  const poSel = document.getElementById('apPOSelect');
  if (poSel) poSel.value = dr.purchase_id;
  const po = dr.purchase || {};
  const items = (dr.items || []).map(i => ({
    item_name: i.item_name, category: i.category, unit: i.unit,
    quantity: i.quantity_received, unit_price: i.unit_price,
  }));
  apRenderItems(items, po.supplier || '', dr.dr_number);
}

function apRenderItems(items, supplier, refNumber) {
  const tbody = document.getElementById('apItemsBody');
  tbody.innerHTML = items.map(item => `
    <tr class="ap-item-row">
      <td class="ps-4">
        <input type="checkbox" class="form-check-input ap-row-check" checked onchange="apToggleRow(this)">
        <input type="hidden" class="ap-item-name" value="${item.item_name}">
        <input type="hidden" class="ap-item-cat" value="${item.category || ''}">
        <input type="hidden" class="ap-item-unit" value="${item.unit}">
      </td>
      <td class="fw-semibold">${item.item_name}</td>
      <td class="text-center"><span class="badge bg-light text-dark border">${item.category || '—'}</span></td>
      <td class="text-center text-muted">${item.unit}</td>
      <td class="text-center text-muted">${formatNumber(item.quantity)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-center ap-qty"
          value="${item.quantity}" min="0" step="0.5" oninput="apUpdateRow(this, ${item.unit_price})">
      </td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-center ap-price"
          value="${item.unit_price.toFixed(2)}" min="0" step="0.01" oninput="apUpdateGrandTotal()">
      </td>
      <td class="text-end pe-4 fw-bold ap-line-total text-primary">${formatCurrency(item.quantity * item.unit_price)}</td>
    </tr>`).join('');

  document.getElementById('apSupplierInfo').style.display = '';
  document.getElementById('apSupplierName').value = supplier;
  document.getElementById('apPORef').value = refNumber;
  document.getElementById('apItemsCard').style.display = '';
  document.getElementById('apSubmitBtn').disabled = false;
  apUpdateGrandTotal();
}

function apUpdateRow(qtyInput) {
  const row   = qtyInput.closest('tr');
  const qty   = parseFloat(qtyInput.value) || 0;
  const price = parseFloat(row.querySelector('.ap-price').value) || 0;
  row.querySelector('.ap-line-total').textContent = formatCurrency(qty * price);
  apUpdateGrandTotal();
}

function apUpdateGrandTotal() {
  let grand = 0;
  document.querySelectorAll('.ap-item-row').forEach(row => {
    if (!row.querySelector('.ap-row-check').checked) return;
    const qty   = parseFloat(row.querySelector('.ap-qty').value) || 0;
    const price = parseFloat(row.querySelector('.ap-price').value) || 0;
    grand += qty * price;
  });
  document.getElementById('apGrandTotal').textContent = formatCurrency(grand);
}

function apToggleRow(cb) {
  const row = cb.closest('tr');
  const qty = row.querySelector('.ap-qty');
  row.style.opacity = cb.checked ? '1' : '0.4';
  if (!cb.checked) qty.value = 0;
  apUpdateGrandTotal();
}

function apToggleAll(master) {
  document.querySelectorAll('.ap-row-check').forEach(cb => { cb.checked = master.checked; apToggleRow(cb); });
}

function apHideItems() {
  const card = document.getElementById('apItemsCard');
  const info = document.getElementById('apSupplierInfo');
  const btn  = document.getElementById('apSubmitBtn');
  if (card) card.style.display = 'none';
  if (info) info.style.display = 'none';
  if (btn)  btn.disabled = true;
}

async function submitNewAPInvoice() {
  const poId  = parseInt(document.getElementById('apPOSelect').value);
  const drRaw = document.getElementById('apDRSelect').value;
  const drId  = drRaw ? parseInt(drRaw) : null;

  if (!poId) { toast('Please select a Purchase Order.', 'warning'); return; }

  const items = [];
  document.querySelectorAll('.ap-item-row').forEach(row => {
    if (!row.querySelector('.ap-row-check').checked) return;
    const qty = parseFloat(row.querySelector('.ap-qty').value) || 0;
    if (qty <= 0) return;
    items.push({
      item_name:  row.querySelector('.ap-item-name').value,
      category:   row.querySelector('.ap-item-cat').value,
      unit:       row.querySelector('.ap-item-unit').value,
      quantity:   qty,
      unit_price: parseFloat(row.querySelector('.ap-price').value) || 0,
    });
  });

  if (items.length === 0) { toast('No items selected.', 'warning'); return; }

  const po      = _apInvPurchases.find(p => p.id === poId) || {};
  const dueDate = document.getElementById('apDueDate').value || null;

  const btn = document.getElementById('apSubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Creating…';

  const result = await api.CreateAPInvoice({
    purchase_id:      poId,
    dr_id:            drId,
    date:             document.getElementById('apDate').value,
    terms:            document.getElementById('apTerms').value,
    supplier_name:    document.getElementById('apSupplierName').value,
    supplier_inv_ref: document.getElementById('apSupplierInvRef').value,
    notes:            document.getElementById('apNotes').value,
    due_date:         dueDate,
    items,
  });

  if (result) {
    navigate('#/purchasing/ap-invoices');
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-file-earmark-text me-1"></i>Create AP Invoice';
  }
}

// ── AP Payment Detail ─────────────────────────────────────────────────────────

async function loadAPPaymentDetail(id) {
  showLoading();
  const pay = await api.GetAPPayment(id);
  if (!pay) {
    showView(`<div class="alert alert-warning m-4">AP Payment not found.</div>`);
    return;
  }

  const lines = pay.lines || [];
  const lineRows = lines.length === 0
    ? `<tr><td colspan="4" class="text-center text-muted py-3">No invoice lines.</td></tr>`
    : lines.map(l => {
        const inv = l.ap_invoice || {};
        const bal = (inv.total_amount || 0) - (inv.amount_paid_stored || 0);
        return `<tr>
          <td class="ps-4 fw-semibold">
            <a href="#" onclick="navigate('#/purchasing/ap-invoices/${l.ap_invoice_id}');return false;" class="text-decoration-none">
              ${inv.invoice_number || '—'}
            </a>
          </td>
          <td class="text-muted small">${formatDate(inv.date)}</td>
          <td>${purchStatusBadge(inv.status)}</td>
          <td class="text-end pe-4 fw-bold text-warning">${formatCurrency(l.amount_applied)}</td>
        </tr>`;
      }).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:800px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/purchasing/ap-payments')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-cash-stack me-2 text-warning"></i>${pay.payment_number}</h4>
          <small class="text-muted">AP Payment · ${formatDate(pay.date)}</small>
        </div>
        <span class="badge bg-success fs-6">Posted</span>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <div class="small text-muted fw-bold">SUPPLIER</div>
              <div class="fw-semibold">${pay.supplier_name_snapshot || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">DATE</div>
              <div>${formatDate(pay.date)}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">PAYMENT METHOD</div>
              <div>${pay.payment_method || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">REFERENCE #</div>
              <div>${pay.reference_number || '—'}</div>
            </div>
            <div class="col-md-4">
              <div class="small text-muted fw-bold">TOTAL AMOUNT</div>
              <div class="fw-bold fs-5 text-warning">${formatCurrency(pay.total_amount)}</div>
            </div>
            ${pay.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${pay.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-primary"></i>AP Invoices Settled</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr><th class="ps-4">Invoice #</th><th>Date</th><th>Status</th><th class="text-end pe-4">Amount Applied</th></tr>
            </thead>
            <tbody>${lineRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="3" class="text-end pe-3 ps-4">Total Payment</td>
                <td class="text-end pe-4 text-warning fs-5">${formatCurrency(pay.total_amount)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>
  `);
}

// ── New AP Payment Form ───────────────────────────────────────────────────────

let _apPayAllInvoices = [];

async function loadNewAPPaymentForm() {
  showLoading();
  const [suppliers, allInvoices] = await Promise.all([
    api.ListSuppliers(true),
    api.ListAPInvoices(null, ''),
  ]);
  const suppList = suppliers || [];
  _apPayAllInvoices = (allInvoices || []).filter(i => i.status === 'Open' || i.status === 'Partial');

  const prefillInvId = window._apPaymentPrefillInvId || null;
  window._apPaymentPrefillInvId = null;

  const today = new Date().toISOString().slice(0, 10);

  const suppOptions = suppList.map(s =>
    `<option value="${s.id}" data-name="${s.name}">${s.name}</option>`
  ).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/purchasing/ap-payments')">
          <i class="bi bi-arrow-left"></i> Back
        </button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-cash-stack me-2 text-warning"></i>New AP Payment</h4>
          <div class="text-muted small">One payment — applies to one or more AP Invoices</div>
        </div>
      </div>

      <!-- Step 1: Supplier -->
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-truck me-2 text-warning"></i>Supplier</span>
        </div>
        <div class="card-body">
          <div class="row g-3 align-items-end">
            <div class="col-md-6">
              <label class="form-label small fw-bold">Select Supplier <span class="text-danger">*</span></label>
              <select id="apPaySupplierSel" class="form-select" onchange="apPayOnSupplierChange()">
                <option value="">— Select a supplier —</option>
                ${suppOptions}
              </select>
            </div>
            <div class="col-md-6" id="apPaySuppSummary" style="display:none">
              <div class="p-3 rounded d-flex gap-4" style="background:#fffbeb;border:1px solid #fde68a">
                <div>
                  <div class="small text-muted">Open Invoices</div>
                  <div class="fw-bold fs-5 text-warning" id="apPaySuppCount">—</div>
                </div>
                <div>
                  <div class="small text-muted">Total Outstanding</div>
                  <div class="fw-bold fs-5 text-danger" id="apPaySuppTotal">—</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Step 2: Invoices -->
      <div class="card border-0 shadow-sm mb-3" id="apPayInvCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center flex-wrap gap-2">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-primary"></i>AP Invoices to Pay</span>
          <div class="d-flex gap-2">
            <button type="button" class="btn btn-sm btn-outline-success" onclick="apPaySelectAll(true)">
              <i class="bi bi-check-all me-1"></i>Select All
            </button>
            <button type="button" class="btn btn-sm btn-outline-secondary" onclick="apPaySelectAll(false)">Deselect All</button>
            <button type="button" class="btn btn-sm btn-outline-warning" onclick="apPayFillBalances()">
              <i class="bi bi-arrow-down-circle me-1"></i>Fill Balances
            </button>
          </div>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:40px">
                  <input type="checkbox" class="form-check-input" id="apPayCheckAll" checked onchange="apPayToggleAll(this)">
                </th>
                <th>Invoice #</th><th>Date</th>
                <th class="text-end">Invoice Total</th><th class="text-end">Paid</th>
                <th class="text-end">Balance Due</th><th class="text-center" style="width:160px">Amount to Pay ₱</th>
              </tr>
            </thead>
            <tbody id="apPayInvBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4 text-muted">TOTAL TO PAY</td>
                <td class="text-end text-danger" id="apPayTotalBal">—</td>
                <td class="text-center fw-bold text-warning fs-6" id="apPayTotalApply">—</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <!-- No invoices -->
      <div id="apPayNoInvMsg" style="display:none" class="card border-0 shadow-sm mb-3">
        <div class="card-body text-center py-4 text-muted">
          <i class="bi bi-check-circle-fill text-success" style="font-size:2rem"></i>
          <div class="mt-2 fw-semibold">No open AP invoices for this supplier</div>
        </div>
      </div>

      <!-- Step 3: Payment Details -->
      <div class="card border-0 shadow-sm mb-3" id="apPayDetailsCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-credit-card me-2"></i>Payment Details</span>
        </div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">Date <span class="text-danger">*</span></label>
              <input type="date" id="apPayDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Payment Method</label>
              <select id="apPayMethod" class="form-select">
                <option>Cash</option><option>Bank Transfer</option><option>GCash</option><option>Check</option>
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Reference #</label>
              <input type="text" id="apPayRef" class="form-control" placeholder="Optional">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="apPayNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <!-- Submit Bar -->
      <div class="card border-0 shadow-sm" id="apPaySubmitCard" style="display:none">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col">
              <div class="small text-muted mb-1">Selected</div>
              <div class="fw-semibold" id="apPaySelCount">0 invoice(s)</div>
            </div>
            <div class="col text-center">
              <div class="small text-muted mb-1">Total Payment Amount</div>
              <div class="fw-bold fs-4 text-warning" id="apPayGrandTotal">₱0.00</div>
            </div>
            <div class="col text-end d-flex gap-2 justify-content-end">
              <button class="btn btn-outline-secondary px-4" onclick="navigate('#/purchasing/ap-payments')">Cancel</button>
              <button class="btn btn-warning px-5 text-white fw-bold" id="apPaySubmitBtn" disabled onclick="submitAPPayment()">
                <i class="bi bi-cash-stack me-2"></i>Post Payment
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `);

  // If prefill: auto-select supplier from invoice
  if (prefillInvId) {
    const inv = _apPayAllInvoices.find(i => i.id === prefillInvId);
    if (inv) {
      const suppSel = document.getElementById('apPaySupplierSel');
      // find by name match
      const opt = Array.from(suppSel.options).find(o => o.dataset.name === inv.supplier_name);
      if (opt) {
        suppSel.value = opt.value;
        apPayOnSupplierChange(prefillInvId);
      }
    }
  }
}

function apPayOnSupplierChange(prefillInvId) {
  const sel  = document.getElementById('apPaySupplierSel');
  const suppName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  if (!sel.value) {
    ['apPayInvCard','apPayDetailsCard','apPaySubmitCard','apPaySuppSummary','apPayNoInvMsg']
      .forEach(id => { const el = document.getElementById(id); if (el) el.style.display = 'none'; });
    return;
  }
  const openInvs = _apPayAllInvoices.filter(i => i.supplier_name === suppName);
  apPayRenderInvoices(openInvs, prefillInvId);
}

function apPayRenderInvoices(invoices, prefillInvId) {
  const tbody = document.getElementById('apPayInvBody');
  if (!tbody) return;
  tbody.innerHTML = '';

  if (invoices.length === 0) {
    ['apPayInvCard','apPayDetailsCard','apPaySubmitCard'].forEach(id => {
      const el = document.getElementById(id); if (el) el.style.display = 'none';
    });
    document.getElementById('apPayNoInvMsg').style.display = '';
    document.getElementById('apPaySuppSummary').style.display = 'none';
    return;
  }

  let totalBal = 0;
  invoices.forEach(inv => {
    const balance = Math.max(0, (inv.total_amount || 0) - (inv.amount_paid_stored || 0));
    totalBal += balance;
    const isPrefill = prefillInvId && inv.id !== prefillInvId;
    const row = document.createElement('tr');
    row.className = 'apPay-inv-row';
    row.innerHTML = `
      <td class="ps-4">
        <input type="checkbox" class="form-check-input apPay-inv-check" data-inv-id="${inv.id}"
          ${isPrefill ? '' : 'checked'} onchange="apPayToggleRow(this)">
      </td>
      <td class="fw-semibold">${inv.invoice_number}${inv.status === 'Partial' ? ' <span class="badge bg-warning text-dark ms-1" style="font-size:10px">Partial</span>' : ''}</td>
      <td class="text-muted small">${formatDate(inv.date)}</td>
      <td class="text-end text-muted">${formatCurrency(inv.total_amount || 0)}</td>
      <td class="text-end text-success small">${formatCurrency(inv.amount_paid_stored || 0)}</td>
      <td class="text-end fw-bold text-danger">${formatCurrency(balance)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-end apPay-amt"
          value="${isPrefill ? '0.00' : balance.toFixed(2)}"
          min="0" max="${balance}" step="0.01" data-max="${balance}"
          ${isPrefill ? 'disabled' : ''}
          oninput="apPayUpdateTotals()">
      </td>`;
    tbody.appendChild(row);
  });

  document.getElementById('apPayTotalBal').textContent  = formatCurrency(totalBal);
  document.getElementById('apPaySuppCount').textContent = invoices.length;
  document.getElementById('apPaySuppTotal').textContent = formatCurrency(totalBal);
  document.getElementById('apPaySuppSummary').style.display  = '';
  document.getElementById('apPayInvCard').style.display      = '';
  document.getElementById('apPayNoInvMsg').style.display     = 'none';
  document.getElementById('apPayDetailsCard').style.display  = '';
  document.getElementById('apPaySubmitCard').style.display   = '';
  document.getElementById('apPayCheckAll').checked = true;
  apPayUpdateTotals();
}

function apPayToggleAll(master) {
  document.querySelectorAll('.apPay-inv-check').forEach(cb => { cb.checked = master.checked; apPayToggleRow(cb); });
}
function apPayToggleRow(cb) {
  const row = cb.closest('tr');
  const amt = row.querySelector('.apPay-amt');
  if (!cb.checked) { amt.disabled = true; amt.value = '0'; row.style.opacity = '0.4'; }
  else             { amt.disabled = false; amt.value = amt.dataset.max; row.style.opacity = '1'; }
  apPayUpdateTotals();
}
function apPaySelectAll(state) {
  document.querySelectorAll('.apPay-inv-check').forEach(cb => { cb.checked = state; apPayToggleRow(cb); });
  const master = document.getElementById('apPayCheckAll');
  if (master) master.checked = state;
}
function apPayFillBalances() {
  document.querySelectorAll('.apPay-inv-row').forEach(row => {
    const cb = row.querySelector('.apPay-inv-check');
    const amt = row.querySelector('.apPay-amt');
    if (cb && cb.checked && amt) amt.value = amt.dataset.max;
  });
  apPayUpdateTotals();
}
function apPayUpdateTotals() {
  let total = 0, count = 0;
  document.querySelectorAll('.apPay-inv-row').forEach(row => {
    const cb = row.querySelector('.apPay-inv-check');
    const amt = row.querySelector('.apPay-amt');
    if (cb && cb.checked && amt) { const v = parseFloat(amt.value) || 0; total += v; if (v > 0) count++; }
  });
  const applyEl = document.getElementById('apPayTotalApply');
  const grandEl = document.getElementById('apPayGrandTotal');
  const countEl = document.getElementById('apPaySelCount');
  const btn     = document.getElementById('apPaySubmitBtn');
  if (applyEl) applyEl.textContent = formatCurrency(total);
  if (grandEl) grandEl.textContent = formatCurrency(total);
  if (countEl) countEl.textContent = count + ' invoice(s)';
  if (btn)     btn.disabled = total <= 0;
}

async function submitAPPayment() {
  const sel      = document.getElementById('apPaySupplierSel');
  const suppId   = parseInt(sel.value) || null;
  const suppName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  const lines    = [];

  document.querySelectorAll('.apPay-inv-row').forEach(row => {
    const cb  = row.querySelector('.apPay-inv-check');
    const amt = row.querySelector('.apPay-amt');
    const id  = parseInt(cb.dataset.invId);
    const val = parseFloat(amt.value) || 0;
    if (cb.checked && val > 0) lines.push({ ap_invoice_id: id, amount_applied: val });
  });

  if (lines.length === 0) { toast('No invoices selected.', 'warning'); return; }

  const btn = document.getElementById('apPaySubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Posting…';

  const result = await api.CreateAPPayment({
    date:           document.getElementById('apPayDate').value,
    supplier_id:    suppId,
    supplier_name:  suppName,
    payment_method: document.getElementById('apPayMethod').value,
    ref_num:        document.getElementById('apPayRef').value,
    notes:          document.getElementById('apPayNotes').value,
    lines,
  });

  if (result) {
    navigate('#/purchasing/ap-payments');
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-cash-stack me-2"></i>Post Payment';
  }
}

// ── Layout Helper ─────────────────────────────────────────────────────────────

function wrapPurch(tabBar, content) {
  return `
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-cart3 me-2"></i>Purchasing</h4>
      ${tabBar}
      ${content}
    </div>
  `;
}
