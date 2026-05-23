// purchasing-suppliers.js

// ── Suppliers ─────────────────────────────────────────────────────────────────

let _supplierList  = [];
let _supplierTypes = [];
let _supplierTerms = [];

function statusBadge(s) {
  const map = { active: 'success', pending: 'warning text-dark', rejected: 'danger' };
  const cls = map[s] || 'secondary';
  return `<span class="badge bg-${cls}">${s || 'unknown'}</span>`;
}

async function loadSuppliersView(editId, tabBar) {
  const [suppRes, typesRes, termsRes] = await Promise.all([
    api.ListSuppliers(false),
    api.ListPurchasingLookup('supplier_type'),
    api.ListPurchasingLookup('payment_terms'),
  ]);
  _supplierList  = suppRes   || [];
  _supplierTypes = typesRes  || [];
  _supplierTerms = termsRes  || [];

  const isAdmin = currentUser && currentUser.role === 'Admin';

  const rows = _supplierList.length === 0
    ? `<tr><td colspan="7" class="text-center text-muted py-4">No suppliers found.</td></tr>`
    : _supplierList.map(s => {
        const approveRejectBtns = (isAdmin && s.status === 'pending') ? `
          <button class="btn btn-sm btn-success me-1" onclick="approveSupplier(${s.id})" title="Approve">
            <i class="bi bi-check-lg"></i>
          </button>
          <button class="btn btn-sm btn-danger me-1" onclick="rejectSupplier(${s.id})" title="Reject">
            <i class="bi bi-x-lg"></i>
          </button>` : '';
        return `
        <tr>
          <td class="fw-semibold">${s.name || '—'}</td>
          <td>${statusBadge(s.status)}</td>
          <td>${s.tin_number || '—'}</td>
          <td>${s.payment_terms || '—'}</td>
          <td>${s.contact_person || '—'}</td>
          <td>${s.contact_number || '—'}</td>
          <td class="text-end">
            ${approveRejectBtns}
            <button class="btn btn-sm btn-outline-secondary" onclick="openSupplierModal(${s.id})">
              <i class="bi bi-pencil"></i>
            </button>
          </td>
        </tr>`;
      }).join('');

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
            <tr>
              <th>Name</th><th>Status</th><th>TIN</th><th>Payment Terms</th>
              <th>Contact</th><th>Phone</th><th></th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>

    <!-- Supplier Modal -->
    <div class="modal fade" id="supplierModal" tabindex="-1">
      <div class="modal-dialog modal-lg">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="supplierModalTitle">Supplier</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="supplierForm">
              <input type="hidden" id="supplierId">
              <div class="row g-3">
                <div class="col-md-8">
                  <label class="form-label">Company Name <span class="text-danger">*</span></label>
                  <input type="text" class="form-control" id="supplierName" required>
                </div>
                <div class="col-md-4">
                  <label class="form-label">Supplier Type</label>
                  <select class="form-select" id="supplierType">
                    ${buildLookupOptions(_supplierTypes, 'Regular')}
                  </select>
                </div>
                <div class="col-md-6">
                  <label class="form-label">Contact Person</label>
                  <input type="text" class="form-control" id="supplierContact">
                </div>
                <div class="col-md-6">
                  <label class="form-label">Phone</label>
                  <input type="text" class="form-control" id="supplierPhone">
                </div>
                <div class="col-md-6">
                  <label class="form-label">Email</label>
                  <input type="email" class="form-control" id="supplierEmail">
                </div>
                <div class="col-md-6">
                  <label class="form-label">TIN Number</label>
                  <input type="text" class="form-control" id="supplierTIN" placeholder="e.g. 123-456-789-000">
                  <div class="form-text">Philippine TIN format: NNN-NNN-NNN or NNN-NNN-NNN-NNN</div>
                </div>
                <div class="col-md-6">
                  <label class="form-label">Payment Terms</label>
                  <select class="form-select" id="supplierPaymentTerms">
                    ${buildLookupOptions(_supplierTerms, 'COD')}
                  </select>
                </div>
                <div class="col-12"><hr class="my-1"></div>
                <div class="col-12 mb-0 pb-0">
                  <small class="text-muted fw-semibold text-uppercase" style="letter-spacing:.05em">
                    <i class="bi bi-receipt me-1 text-warning"></i>BIR Withholding Tax (EWT) — RR 2-98
                  </small>
                </div>
                <div class="col-md-7">
                  <label class="form-label">WHT Category</label>
                  <select class="form-select" id="supplierWHTCategory">
                    <option value="NONE">None — not subject to EWT</option>
                    <option value="AGRICULTURAL_PRODUCER">Agricultural Producer — palay farmer (1% above ₱300k/yr, WA010)</option>
                    <option value="PALAY_TRADER">Palay Trader / Dealer (2%, WC010)</option>
                    <option value="GOODS_SUPPLIER">Goods Supplier — general (2%, WC158)</option>
                    <option value="SERVICE_PROVIDER">Service Provider — repairs/trucking/milling (2%, WC158)</option>
                    <option value="PROFESSIONAL">Professional — accountant/engineer (5%, WM010)</option>
                    <option value="RENTAL">Rental — land/equipment/warehouse (5%, WB010)</option>
                  </select>
                  <div class="form-text">Used to auto-compute withholding tax on AP invoices.</div>
                </div>
                <div class="col-md-5 d-flex align-items-end pb-1">
                  <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="supplierVATRegistered">
                    <label class="form-check-label" for="supplierVATRegistered">
                      VAT-Registered supplier
                      <div class="form-text mt-0">WHT base = VAT-exclusive amount</div>
                    </label>
                  </div>
                </div>
                <div class="col-12">
                  <label class="form-label">Address</label>
                  <textarea class="form-control" id="supplierAddress" rows="2"></textarea>
                </div>
                <div class="col-12">
                  <label class="form-label">Bank Details</label>
                  <textarea class="form-control" id="supplierBankDetails" rows="2"
                    placeholder="Bank name, account number, account name..."></textarea>
                </div>
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
  const s      = isEdit ? _supplierList.find(x => x.id === id) : null;

  // Rebuild lookup selects with correct selection before showing modal
  document.getElementById('supplierType').innerHTML =
    buildLookupOptions(_supplierTypes, s?.supplier_type || 'Regular');
  document.getElementById('supplierPaymentTerms').innerHTML =
    buildLookupOptions(_supplierTerms, s?.payment_terms || 'COD');

  document.getElementById('supplierModalTitle').textContent = isEdit ? 'Edit Supplier' : 'Add Supplier';
  document.getElementById('supplierId').value = id || '';

  if (s) {
    document.getElementById('supplierName').value        = s.name || '';
    document.getElementById('supplierContact').value     = s.contact_person || '';
    document.getElementById('supplierPhone').value       = s.contact_number || '';
    document.getElementById('supplierEmail').value       = s.email || '';
    document.getElementById('supplierTIN').value         = s.tin_number || '';
    document.getElementById('supplierAddress').value     = s.address || '';
    document.getElementById('supplierBankDetails').value = s.bank_details || '';
    document.getElementById('supplierWHTCategory').value = s.wht_category || 'NONE';
    document.getElementById('supplierVATRegistered').checked = !!s.is_vat_registered;
  } else {
    document.getElementById('supplierForm').reset();
    // Re-apply defaults after reset (reset clears selects too)
    document.getElementById('supplierType').innerHTML =
      buildLookupOptions(_supplierTypes, 'Regular');
    document.getElementById('supplierPaymentTerms').innerHTML =
      buildLookupOptions(_supplierTerms, 'COD');
  }
  new bootstrap.Modal(document.getElementById('supplierModal')).show();
}

async function submitSupplierForm() {
  const id = document.getElementById('supplierId').value;
  const name = document.getElementById('supplierName').value.trim();
  if (!name) { toast('Supplier name is required.', 'warning'); return; }

  const payload = {
    name,
    supplier_type:     document.getElementById('supplierType').value,
    contact_person:    document.getElementById('supplierContact').value.trim(),
    contact_number:    document.getElementById('supplierPhone').value.trim(),
    email:             document.getElementById('supplierEmail').value.trim(),
    tin_number:        document.getElementById('supplierTIN').value.trim(),
    payment_terms:     document.getElementById('supplierPaymentTerms').value,
    address:           document.getElementById('supplierAddress').value.trim(),
    bank_details:      document.getElementById('supplierBankDetails').value.trim(),
    wht_category:      document.getElementById('supplierWHTCategory').value,
    is_vat_registered: document.getElementById('supplierVATRegistered').checked,
  };

  const result = id
    ? await api.UpdateSupplier(parseInt(id, 10), payload)
    : await api.CreateSupplier(payload);

  if (result) {
    bootstrap.Modal.getInstance(document.getElementById('supplierModal')).hide();
    Modules.Purchasing.load('suppliers');
  }
}

async function approveSupplier(id) {
  if (!confirm('Approve this supplier and mark them as active?')) return;
  const result = await api.ApproveSupplier(id);
  if (result) Modules.Purchasing.load('suppliers');
}

async function rejectSupplier(id) {
  if (!confirm('Reject this supplier?')) return;
  const result = await api.RejectSupplier(id);
  if (result) Modules.Purchasing.load('suppliers');
}

function _resetPurchasingSuppliers() {
  _supplierList = []; _supplierTypes = []; _supplierTerms = [];
}
