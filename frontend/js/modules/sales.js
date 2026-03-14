// sales.js — Modules.Sales

Modules.Sales = {
  async load(sub, id, action) {
    const active = sub || 'orders';
    showLoading();

    // Detail / form views (no tab bar needed)
    if (active === 'orders' && action === 'new')           { await loadNewSOForm(); return; }
    if (active === 'orders' && id)                         { await loadSODetail(id); return; }
    if (active === 'delivery-orders' && action === 'new')  { await loadNewDOForm(); return; }
    if (active === 'delivery-orders' && id)                { await loadDODetail(id); return; }
    if (active === 'ar-invoices' && action === 'new')      { await loadNewARInvoiceForm(); return; }
    if (active === 'ar-invoices' && id)                    { await loadARInvoiceDetail(id); return; }
    if (active === 'collections' && action === 'new')      { await loadNewCollectionForm(); return; }
    if (active === 'collections' && id)                    { await loadCollectionDetail(id); return; }
    if (active === 'price-groups' && id)                   { await loadPriceGroupDetail(id); return; }

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'orders' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/orders');return false;">
            <i class="bi bi-bag me-1"></i>Sales Orders
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'delivery-orders' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/delivery-orders');return false;">
            <i class="bi bi-truck me-1"></i>Delivery Orders
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ar-invoices' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/ar-invoices');return false;">
            <i class="bi bi-file-earmark-text me-1"></i>AR Invoices
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'collections' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/collections');return false;">
            <i class="bi bi-cash-coin me-1"></i>Collections
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'customers' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/customers');return false;">
            <i class="bi bi-people me-1"></i>Customers
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'price-groups' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/price-groups');return false;">
            <i class="bi bi-tags me-1"></i>Price Groups
          </a>
        </li>
      </ul>
    `;

    if (active === 'orders') await loadSalesOrdersList(tabBar);
    else if (active === 'delivery-orders') await loadDeliveryOrdersList(tabBar);
    else if (active === 'ar-invoices') await loadArInvoicesList(tabBar);
    else if (active === 'collections') await loadCollectionsList(tabBar);
    else if (active === 'customers') await this.loadCustomers(id, tabBar);
    else if (active === 'price-groups') await this.loadPriceGroups(id, tabBar);
    else await loadSalesOrdersList(tabBar);
  },

  async loadCustomers(id, tabBar) {
    await loadCustomersView(id, tabBar);
  },

  async loadPriceGroups(id, tabBar) {
    await loadPriceGroupsView(id, tabBar);
  }
};

// ── Status Badge ──────────────────────────────────────────────────────────────

function salesStatusBadge(status) {
  const map = {
    Draft:      'secondary',
    Pending:    'warning',
    Confirmed:  'primary',
    Delivered:  'success',
    Invoiced:   'info',
    Paid:       'success',
    Partial:    'warning',
    Cancelled:  'danger',
    Overdue:    'danger',
  };
  const color = map[status] || 'secondary';
  return `<span class="badge bg-${color}">${status || '—'}</span>`;
}

// ── Filter helpers ────────────────────────────────────────────────────────────

function _sfVal(id) { return (document.getElementById(id) || {}).value || ''; }
function _uniq(arr) { return [...new Set(arr.filter(Boolean))].sort(); }
function _esc(s) { return String(s).replace(/&/g,'&amp;').replace(/"/g,'&quot;').replace(/</g,'&lt;'); }

function _applyFilters(list, { docField, customerField, amountField, dateField, statusField, defaultStatuses }) {
  const doc     = _sfVal('sf-doc').toLowerCase();
  const cust    = _sfVal('sf-cust').toLowerCase();
  const dateFrom = _sfVal('sf-date-from');
  const dateTo   = _sfVal('sf-date-to');
  const amtMin   = parseFloat(_sfVal('sf-amt-min')) || 0;
  const amtMax   = parseFloat(_sfVal('sf-amt-max')) || Infinity;
  const status   = _sfVal('sf-status');

  return list.filter(r => {
    if (doc && !(r[docField] || '').toLowerCase().includes(doc)) return false;
    const custName = (customerField ? (r[customerField] || '') : (r.customer_name_snapshot || (r.customer && r.customer.name) || '')).toLowerCase();
    if (cust && !custName.includes(cust)) return false;
    const rowDate = (r[dateField || 'date'] || '').slice(0, 10);
    if (dateFrom && rowDate < dateFrom) return false;
    if (dateTo   && rowDate > dateTo)   return false;
    const amt = r[amountField] ?? 0;
    if (amt < amtMin || amt > amtMax) return false;
    const rowStatus = statusField ? r[statusField] : r.status;
    if (status) {
      const allowed = status.split(',');
      if (!allowed.includes(rowStatus)) return false;
    }
    return true;
  });
}

function _filterBar(statusOptions, defaultStatus, onFilter, datalists = {}) {
  const docOpts  = _uniq(datalists.doc  || []).map(v => `<option value="${_esc(v)}">`).join('');
  const custOpts = _uniq(datalists.cust || []).map(v => `<option value="${_esc(v)}">`).join('');
  return `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-body py-2">
        <div class="row g-2 align-items-end">
          <div class="col-sm-3">
            <label class="form-label small mb-1">Document #</label>
            <input id="sf-doc" class="form-control form-control-sm" placeholder="Search…" oninput="${onFilter}" list="sf-doc-list" autocomplete="off">
            <datalist id="sf-doc-list">${docOpts}</datalist>
          </div>
          <div class="col-sm-3">
            <label class="form-label small mb-1">Customer</label>
            <input id="sf-cust" class="form-control form-control-sm" placeholder="Search…" oninput="${onFilter}" list="sf-cust-list" autocomplete="off">
            <datalist id="sf-cust-list">${custOpts}</datalist>
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Date From</label>
            <input id="sf-date-from" type="date" class="form-control form-control-sm" onchange="${onFilter}">
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Date To</label>
            <input id="sf-date-to" type="date" class="form-control form-control-sm" onchange="${onFilter}">
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Status</label>
            <select id="sf-status" class="form-select form-select-sm" onchange="${onFilter}">
              ${statusOptions.map(s => `<option value="${s.v}" ${s.v === defaultStatus ? 'selected' : ''}>${s.l}</option>`).join('')}
            </select>
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Min Amount</label>
            <input id="sf-amt-min" type="number" min="0" class="form-control form-control-sm" placeholder="0" oninput="${onFilter}">
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Max Amount</label>
            <input id="sf-amt-max" type="number" min="0" class="form-control form-control-sm" placeholder="Any" oninput="${onFilter}">
          </div>
          <div class="col-sm-2 d-flex align-items-end">
            <button class="btn btn-outline-secondary btn-sm w-100" onclick="${onFilter.replace('soFilter()', 'sfClearAll()')
              .replace('doFilter()', 'sfClearAll()')
              .replace('arFilter()', 'sfClearAll()')
              .replace('crFilter()', 'sfClearAll()')}sfClearAll()">
              <i class="bi bi-x-circle me-1"></i>Clear
            </button>
          </div>
        </div>
      </div>
    </div>`;
}

function sfClearAll() {
  ['sf-doc','sf-cust','sf-date-from','sf-date-to','sf-amt-min','sf-amt-max'].forEach(id => {
    const el = document.getElementById(id); if (el) el.value = '';
  });
  // re-trigger whichever filter is active
  ['soFilter','doFilter','arFilter','crFilter'].forEach(fn => {
    if (window[fn]) { try { window[fn](); } catch(e) {} }
  });
}

// ── Sales Orders ──────────────────────────────────────────────────────────────

let _soList = [];

async function loadSalesOrdersList(tabBar) {
  _soList = await api.ListSalesOrders() || [];
  if (!_soList) {
    showView(wrapSales(tabBar, `<div class="alert alert-warning">Failed to load sales orders.</div>`));
    return;
  }

  const statusOpts = [
    { v: 'Unpaid,Partial', l: 'Unpaid / Partial' },
    { v: 'Unpaid',  l: 'Unpaid' },
    { v: 'Partial', l: 'Partial' },
    { v: 'Paid',    l: 'Paid' },
    { v: 'Void',    l: 'Void' },
    { v: '',        l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="navigate('#/sales/orders/new')">
        <i class="bi bi-plus-lg me-1"></i>New Sales Order
      </button>
    </div>
    ${_filterBar(statusOpts, 'Unpaid,Partial', 'soFilter()', {
      doc:  _soList.map(o => o.sales_order_number),
      cust: _soList.map(o => o.customer_name_snapshot || (o.customer && o.customer.name) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>SO #</th><th>Date</th><th>Customer</th><th>Amount</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="so-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  soFilter();
}

window.soFilter = function() {
  const filtered = _applyFilters(_soList, {
    docField: 'sales_order_number', amountField: 'grand_total', statusField: 'payment_status',
    defaultStatuses: ['Unpaid','Partial'],
  });
  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No sales orders match the filter.</td></tr>`
    : filtered.map(o => `
        <tr>
          <td class="fw-semibold">${o.sales_order_number || '—'}</td>
          <td>${formatDate(o.date)}</td>
          <td>${o.customer_name_snapshot || (o.customer && o.customer.name) || '—'}</td>
          <td>${formatCurrency(o.grand_total ?? 0)}</td>
          <td>${salesStatusBadge(o.payment_status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary me-1" onclick="navigate('#/sales/orders/${o.id}')">
              <i class="bi bi-eye"></i>
            </button>
            ${o.payment_status !== 'Void' ? `
              <button class="btn btn-sm btn-outline-primary" title="Create Delivery Order" onclick="newDOFromSO(${o.id})">
                <i class="bi bi-truck"></i>
              </button>` : ''}
          </td>
        </tr>`).join('');
  const el = document.getElementById('so-tbody');
  if (el) el.innerHTML = rows;
};

// ── Delivery Orders ───────────────────────────────────────────────────────────

let _doList = [];

async function loadDeliveryOrdersList(tabBar) {
  _doList = await api.ListDeliveryOrders(0) || [];

  const statusOpts = [
    { v: 'Draft',     l: 'Draft' },
    { v: 'Delivered', l: 'Delivered' },
    { v: '',          l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    ${_filterBar(statusOpts, 'Draft', 'doFilter()', {
      doc:  _doList.map(d => d.delivery_number),
      cust: _doList.map(d => (d.sales_order && d.sales_order.customer_name_snapshot) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>DO #</th><th>Delivery Date</th><th>Customer</th><th>SO #</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="do-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  doFilter();
}

window.doFilter = function() {
  const doc    = _sfVal('sf-doc').toLowerCase();
  const cust   = _sfVal('sf-cust').toLowerCase();
  const dateFrom = _sfVal('sf-date-from');
  const dateTo   = _sfVal('sf-date-to');
  const status   = _sfVal('sf-status');

  const filtered = _doList.filter(d => {
    if (doc  && !(d.delivery_number || '').toLowerCase().includes(doc)) return false;
    const custName = ((d.sales_order && d.sales_order.customer_name_snapshot) || '').toLowerCase();
    if (cust && !custName.includes(cust)) return false;
    const rowDate = (d.date || '').slice(0, 10);
    if (dateFrom && rowDate < dateFrom) return false;
    if (dateTo   && rowDate > dateTo)   return false;
    if (status) {
      const allowed = status.split(',');
      if (!allowed.includes(d.status)) return false;
    }
    return true;
  });

  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No delivery orders match the filter.</td></tr>`
    : filtered.map(d => `
        <tr>
          <td class="fw-semibold">${d.delivery_number || '—'}</td>
          <td>${formatDate(d.date)}</td>
          <td>${(d.sales_order && d.sales_order.customer_name_snapshot) || '—'}</td>
          <td>${(d.sales_order && d.sales_order.sales_order_number) || '—'}</td>
          <td>${salesStatusBadge(d.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/delivery-orders/${d.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>`).join('');
  const el = document.getElementById('do-tbody');
  if (el) el.innerHTML = rows;
};

// ── AR Invoices ───────────────────────────────────────────────────────────────

let _arList = [];

async function loadArInvoicesList(tabBar) {
  _arList = await api.ListARInvoices(null, '') || [];

  const statusOpts = [
    { v: 'Open,Partial', l: 'Open / Partial' },
    { v: 'Open',    l: 'Open' },
    { v: 'Partial', l: 'Partial' },
    { v: 'Paid',    l: 'Paid' },
    { v: 'Cancelled', l: 'Cancelled' },
    { v: '',        l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="navigate('#/sales/ar-invoices/new')">
        <i class="bi bi-plus-lg me-1"></i>New AR Invoice
      </button>
    </div>
    ${_filterBar(statusOpts, 'Open,Partial', 'arFilter()', {
      doc:  _arList.map(i => i.invoice_number),
      cust: _arList.map(i => i.customer_name_snapshot || (i.customer && i.customer.name) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>AR Invoice #</th><th>Date</th><th>Due</th><th>Customer</th><th>Amount</th><th>Balance</th><th>Status</th><th></th></tr>
          </thead>
          <tbody id="ar-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  arFilter();
}

window.arFilter = function() {
  const filtered = _applyFilters(_arList, {
    docField: 'invoice_number', amountField: 'total_amount',
    defaultStatuses: ['Open','Partial'],
  });
  const rows = filtered.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No AR invoices match the filter.</td></tr>`
    : filtered.map(i => {
        const balance = (i.total_amount ?? 0) - (i.amount_collected ?? 0);
        return `
        <tr>
          <td class="fw-semibold">${i.invoice_number || '—'}</td>
          <td>${formatDate(i.date)}</td>
          <td>${formatDate(i.due_date)}</td>
          <td>${i.customer_name_snapshot || (i.customer && i.customer.name) || '—'}</td>
          <td>${formatCurrency(i.total_amount ?? 0)}</td>
          <td class="${balance > 0 ? 'text-danger fw-semibold' : ''}">${formatCurrency(balance)}</td>
          <td>${salesStatusBadge(i.status)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary me-1" onclick="navigate('#/sales/ar-invoices/${i.id}')">
              <i class="bi bi-eye"></i>
            </button>
            ${(i.status === 'Open' || i.status === 'Partial') ? `
              <button class="btn btn-sm btn-outline-success" title="Record Collection" onclick="newCollectionFromARInv(${i.id})">
                <i class="bi bi-cash"></i>
              </button>` : ''}
          </td>
        </tr>`;
      }).join('');
  const el = document.getElementById('ar-tbody');
  if (el) el.innerHTML = rows;
};

// ── Collections ───────────────────────────────────────────────────────────────

let _crList = [];

async function loadCollectionsList(tabBar) {
  _crList = await api.ListCollections() || [];

  const statusOpts = [
    { v: '', l: 'All' },
  ];

  showView(wrapSales(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="navigate('#/sales/collections/new')">
        <i class="bi bi-plus-lg me-1"></i>Record Collection
      </button>
    </div>
    ${_filterBar(statusOpts, '', 'crFilter()', {
      doc:  _crList.map(c => c.collection_number),
      cust: _crList.map(c => c.customer_name_snapshot || (c.customer && c.customer.name) || ''),
    })}
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Collection #</th><th>Date</th><th>Customer</th><th>Method</th><th>Amount</th><th></th></tr>
          </thead>
          <tbody id="cr-tbody"></tbody>
        </table>
      </div>
    </div>
  `));
  crFilter();
}

window.crFilter = function() {
  const doc    = _sfVal('sf-doc').toLowerCase();
  const cust   = _sfVal('sf-cust').toLowerCase();
  const dateFrom = _sfVal('sf-date-from');
  const dateTo   = _sfVal('sf-date-to');
  const amtMin   = parseFloat(_sfVal('sf-amt-min')) || 0;
  const amtMax   = parseFloat(_sfVal('sf-amt-max')) || Infinity;

  const filtered = _crList.filter(c => {
    if (doc  && !(c.collection_number || '').toLowerCase().includes(doc)) return false;
    const custName = (c.customer_name_snapshot || (c.customer && c.customer.name) || '').toLowerCase();
    if (cust && !custName.includes(cust)) return false;
    const rowDate = (c.date || '').slice(0, 10);
    if (dateFrom && rowDate < dateFrom) return false;
    if (dateTo   && rowDate > dateTo)   return false;
    const amt = c.total_amount ?? 0;
    if (amt < amtMin || amt > amtMax) return false;
    return true;
  });

  const rows = filtered.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No collections match the filter.</td></tr>`
    : filtered.map(c => `
        <tr>
          <td class="fw-semibold">${c.collection_number || '—'}</td>
          <td>${formatDate(c.date)}</td>
          <td>${c.customer_name_snapshot || (c.customer && c.customer.name) || '—'}</td>
          <td>${c.payment_method || '—'}</td>
          <td>${formatCurrency(c.total_amount ?? 0)}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/collections/${c.id}')">
              <i class="bi bi-eye"></i>
            </button>
          </td>
        </tr>`).join('');
  const el = document.getElementById('cr-tbody');
  if (el) el.innerHTML = rows;
};

// ── Customers ─────────────────────────────────────────────────────────────────

let _customerList = [];

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
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No customers found.</td></tr>`
    : _customerList.map(c => {
        const pgName = c.price_group ? c.price_group.name : '—';
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
      }).join('');

  showView(wrapSales(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small">${_customerList.length} customer(s)</span>
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
          <tbody>${rows}</tbody>
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
                <select class="form-select" id="customerPriceGroupId">
                  <option value="">— None (manual pricing) —</option>
                  ${pgOptions}
                </select>
                <div class="form-text">Prices auto-fill on new sales orders.</div>
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
  new bootstrap.Modal(document.getElementById('customerModal')).show();
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
    navigate('#/sales/customers');
  } else {
    toast('Failed to save customer.', 'danger');
  }
}

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

// ── Sales Order Detail ────────────────────────────────────────────────────────

async function loadSODetail(id) {
  showLoading();
  const so = await api.GetSalesOrder(id);
  if (!so) { showView(`<div class="alert alert-warning m-4">Sales order not found.</div>`); return; }

  const items     = so.items || [];
  const deliveries = so.deliveries || [];
  const arInvoices = so.ar_invoices || [];

  const itemRows = items.map(i => `
    <tr>
      <td class="ps-4 fw-semibold">${i.sku}</td>
      <td class="text-center text-muted">${i.unit}</td>
      <td class="text-center">${formatNumber(i.quantity)}</td>
      <td class="text-center">${formatCurrency(i.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.line_total)}</td>
    </tr>`).join('') || `<tr><td colspan="5" class="text-center text-muted py-3">No items.</td></tr>`;

  const doRows = deliveries.map(d => `
    <tr>
      <td class="ps-4 fw-semibold">
        <a href="#" onclick="navigate('#/sales/delivery-orders/${d.id}');return false;" class="text-decoration-none">${d.delivery_number}</a>
      </td>
      <td class="text-muted small">${formatDate(d.date)}</td>
      <td>${salesStatusBadge(d.status)}</td>
      <td class="text-end pe-3">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/delivery-orders/${d.id}')">View</button>
      </td>
    </tr>`).join('') || `<tr><td colspan="4" class="text-center text-muted py-3">No delivery orders.</td></tr>`;

  const invRows = arInvoices.map(i => {
    const bal = Math.max(0, (i.total_amount || 0) - (i.amount_collected || 0));
    return `<tr>
      <td class="ps-4 fw-semibold">
        <a href="#" onclick="navigate('#/sales/ar-invoices/${i.id}');return false;" class="text-decoration-none">${i.invoice_number}</a>
      </td>
      <td class="text-muted small">${formatDate(i.date)}</td>
      <td>${formatCurrency(i.total_amount || 0)}</td>
      <td class="fw-bold ${bal <= 0.005 ? 'text-success' : 'text-danger'}">${bal <= 0.005 ? 'Paid' : formatCurrency(bal)}</td>
      <td>${salesStatusBadge(i.status)}</td>
    </tr>`;
  }).join('') || `<tr><td colspan="5" class="text-center text-muted py-3">No AR invoices.</td></tr>`;

  const canVoid = so.payment_status !== 'Void' && deliveries.length === 0 && arInvoices.length === 0;
  const custName = so.customer_name_snapshot || (so.customer && so.customer.name) || '—';

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/orders')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-bag me-2 text-primary"></i>${so.sales_order_number}</h4>
          <small class="text-muted">Sales Order · ${formatDate(so.date)}</small>
        </div>
        ${salesStatusBadge(so.payment_status)}
        <button class="btn btn-sm btn-outline-primary" onclick="newDOFromSO(${so.id})">
          <i class="bi bi-truck me-1"></i>Create Delivery Order
        </button>
        <button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromSO(${so.id})">
          <i class="bi bi-file-earmark-plus me-1"></i>Create AR Invoice
        </button>
        ${canVoid ? `<button class="btn btn-sm btn-outline-danger" onclick="voidSO(${so.id})">Void</button>` : ''}
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted fw-bold">CUSTOMER</div><div class="fw-semibold">${custName}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">DATE</div><div>${formatDate(so.date)}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">TERMS</div><div>${so.terms || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">PAYMENT METHOD</div><div>${so.payment_method || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">GRAND TOTAL</div><div class="fw-bold text-primary">${formatCurrency(so.grand_total)}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">AMOUNT COLLECTED</div><div class="fw-bold text-success">${formatCurrency(so.amount_collected || 0)}</div></div>
            ${so.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${so.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-egg me-2 text-warning"></i>Line Items</span></div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Egg Size</th><th class="text-center">Unit</th><th class="text-center">Qty</th><th class="text-center">Price/Unit</th><th class="text-end pe-4">Total</th></tr></thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="4" class="text-end pe-3 ps-4">Grand Total</td><td class="text-end pe-4 text-primary fs-5">${formatCurrency(so.grand_total)}</td></tr></tfoot>
          </table>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-truck me-2"></i>Delivery Orders</span>
          <button class="btn btn-sm btn-outline-primary" onclick="newDOFromSO(${so.id})">+ Create DO</button>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">DO #</th><th>Date</th><th>Status</th><th></th></tr></thead>
            <tbody>${doRows}</tbody>
          </table>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices</span>
          <button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromSO(${so.id})">+ Create AR Invoice</button>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Invoice #</th><th>Date</th><th>Total</th><th>Balance</th><th>Status</th></tr></thead>
            <tbody>${invRows}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}

async function voidSO(id) {
  if (!confirm('Void this sales order?')) return;
  const r = await api.VoidSalesOrder(id);
  if (r) navigate(`#/sales/orders/${id}`);
}

function newDOFromSO(soId) {
  window._newDOPrefillSOId = soId;
  navigate('#/sales/delivery-orders/new');
}

function newARInvoiceFromSO(soId) {
  window._newARInvoicePrefillSOId = soId;
  navigate('#/sales/ar-invoices/new');
}

function newCollectionFromARInv(invId) {
  window._newCollectionPrefillInvId = invId;
  navigate('#/sales/collections/new');
}

// ── Delivery Order Detail ─────────────────────────────────────────────────────

async function loadDODetail(id) {
  showLoading();
  const doRec = await api.GetDeliveryOrder(id);
  if (!doRec) { showView(`<div class="alert alert-warning m-4">Delivery order not found.</div>`); return; }

  const so      = doRec.sales_order || {};
  const items   = doRec.items || [];
  const arInvs  = doRec.ar_invoices || [];
  const total   = items.reduce((s, i) => s + (i.quantity_delivered * i.price_per_unit), 0);
  const uninvoiced = total - (doRec.amount_invoiced || 0);

  const statusColor = { Delivered: 'success', Cancelled: 'secondary', Draft: 'warning' };
  const badge = `<span class="badge bg-${statusColor[doRec.status] || 'secondary'} fs-6">${doRec.status}</span>`;

  const itemRows = items.map(i => `
    <tr>
      <td class="ps-4 fw-semibold">${i.sku}</td>
      <td class="text-center text-muted">${i.unit}</td>
      <td class="text-end text-muted">${formatNumber(i.quantity_ordered)}</td>
      <td class="text-end fw-bold">${formatNumber(i.quantity_delivered)}</td>
      <td class="text-end">${formatCurrency(i.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity_delivered * i.price_per_unit)}</td>
    </tr>`).join('') || `<tr><td colspan="6" class="text-center text-muted py-3">No items.</td></tr>`;

  const arRows = arInvs.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between">
        <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices</span>
        ${uninvoiced > 0.005 ? `<button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromDO(${doRec.id})">+ Create AR Invoice</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light"><tr><th class="ps-4">Invoice #</th><th>Date</th><th>Status</th><th class="text-end pe-4">Total</th></tr></thead>
          <tbody>
            ${arInvs.map(inv => `<tr>
              <td class="ps-4 fw-semibold"><a href="#" onclick="navigate('#/sales/ar-invoices/${inv.id}');return false;" class="text-decoration-none">${inv.invoice_number}</a></td>
              <td class="text-muted small">${formatDate(inv.date)}</td>
              <td>${salesStatusBadge(inv.status)}</td>
              <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(inv.total_amount || 0)}</td>
            </tr>`).join('')}
          </tbody>
        </table>
      </div>
    </div>`;

  const confirmBtn = doRec.status === 'Draft' ? `
    <button class="btn btn-sm btn-success" onclick="confirmDO(${doRec.id})">
      <i class="bi bi-check-lg me-1"></i>Confirm Delivery
    </button>` : '';
  const createARBtn = doRec.status === 'Delivered' && uninvoiced > 0.005 ? `
    <button class="btn btn-sm btn-outline-success" onclick="newARInvoiceFromDO(${doRec.id})">
      <i class="bi bi-file-earmark-plus me-1"></i>Create AR Invoice
    </button>` : '';

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/delivery-orders')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-truck me-2" style="color:#ec4899"></i>${doRec.delivery_number}</h4>
          <small class="text-muted">Delivery Order · ${formatDate(doRec.date)}</small>
        </div>
        ${badge}
        ${confirmBtn}
        ${createARBtn}
      </div>

      ${doRec.status === 'Delivered' ? `
        <div class="card border-0 shadow-sm mb-3">
          <div class="card-header bg-white border-bottom py-2"><span class="small fw-bold text-muted">INVOICE STATUS</span></div>
          <div class="card-body">
            <div class="row g-2 text-center">
              <div class="col"><div class="small text-muted mb-1">DO Total</div><div class="fw-bold">${formatCurrency(total)}</div></div>
              <div class="col-auto d-flex align-items-center text-muted">→</div>
              <div class="col"><div class="small text-muted mb-1">AR Invoiced</div><div class="fw-bold text-primary">${formatCurrency(doRec.amount_invoiced || 0)}</div></div>
              <div class="col-auto d-flex align-items-center text-muted">→</div>
              <div class="col"><div class="small text-muted mb-1">Uninvoiced</div>
                <div class="fw-bold ${uninvoiced > 0.005 ? 'text-danger' : 'text-success'}">${uninvoiced > 0.005 ? formatCurrency(uninvoiced) : '<i class="bi bi-check-circle-fill me-1"></i>Fully Invoiced'}</div>
              </div>
            </div>
          </div>
        </div>` : ''}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-info-circle me-2"></i>Delivery Details</span></div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted">Sales Order</div>
              <a href="#" onclick="navigate('#/sales/orders/${doRec.sales_order_id}');return false;" class="text-decoration-none fw-semibold">${so.sales_order_number || '—'}</a>
            </div>
            <div class="col-md-4"><div class="small text-muted">Customer</div><div class="fw-semibold">${so.customer_name_snapshot || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted">Delivered By</div><div>${doRec.delivered_by || '—'}</div></div>
            ${doRec.notes ? `<div class="col-12"><div class="small text-muted">Notes</div><div>${doRec.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-egg me-2 text-warning"></i>Items Delivered</span></div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Egg Size</th><th class="text-center">Unit</th><th class="text-end">Ordered</th><th class="text-end">Delivered</th><th class="text-end">Price/Unit</th><th class="text-end pe-4">Line Total</th></tr></thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="5" class="text-end pe-3 ps-4">Total</td><td class="text-end pe-4 text-primary fs-5">${formatCurrency(total)}</td></tr></tfoot>
          </table>
        </div>
      </div>

      ${arRows}
    </div>
  `);
}

async function confirmDO(id) {
  if (!confirm('Confirm delivery and update inventory?')) return;
  const r = await api.ConfirmDeliveryOrder(id);
  if (r) await loadDODetail(id);
}

function newARInvoiceFromDO(doId) {
  window._newARInvoicePrefillDOId = doId;
  navigate('#/sales/ar-invoices/new');
}

// ── New Delivery Order Form ───────────────────────────────────────────────────

let _newDOSalesOrders = [];

async function loadNewDOForm() {
  showLoading();
  const soId = window._newDOPrefillSOId || null;
  window._newDOPrefillSOId = null;

  let soData = null;
  if (soId) {
    soData = await api.GetSalesOrder(soId);
  }
  if (!soData) {
    // Show SO selector first
    const soList = await api.ListSalesOrders();
    _newDOSalesOrders = (soList || []).filter(o => o.payment_status !== 'Void');
    const opts = _newDOSalesOrders.map(o =>
      `<option value="${o.id}">${o.sales_order_number} — ${o.customer_name_snapshot || 'Walk-in'} (${formatCurrency(o.grand_total)})</option>`
    ).join('');
    showView(`
      <div class="container p-4" style="max-width:600px">
        <div class="d-flex align-items-center gap-2 mb-4">
          <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/delivery-orders')"><i class="bi bi-arrow-left"></i></button>
          <h4 class="fw-bold mb-0">New Delivery Order</h4>
        </div>
        <div class="card border-0 shadow-sm">
          <div class="card-body">
            <label class="form-label fw-bold">Select Sales Order <span class="text-danger">*</span></label>
            <select class="form-select mb-3" id="doSOPicker">
              <option value="">— Select Sales Order —</option>${opts}
            </select>
            <button class="btn btn-primary" onclick="doPickSO()">Continue</button>
          </div>
        </div>
      </div>`);
    return;
  }
  await renderNewDOForm(soData);
}

async function doPickSO() {
  const id = parseInt(document.getElementById('doSOPicker').value);
  if (!id) { toast('Please select a sales order.', 'warning'); return; }
  const so = await api.GetSalesOrder(id);
  if (so) await renderNewDOForm(so);
}

async function renderNewDOForm(so) {
  const today = new Date().toISOString().slice(0, 10);
  const custName = so.customer_name_snapshot || (so.customer && so.customer.name) || 'Walk-in';
  const items = so.items || [];

  const itemRows = items.map((item, idx) => `
    <tr class="do-item-row">
      <td class="ps-4">
        <input type="checkbox" class="form-check-input do-row-check" checked onchange="doToggleRow(this)">
        <input type="hidden" class="do-sku" value="${item.sku}">
        <input type="hidden" class="do-unit" value="${item.unit}">
        <input type="hidden" class="do-price" value="${item.price_per_unit}">
      </td>
      <td class="fw-semibold">${item.sku}</td>
      <td class="text-center text-muted">${item.unit}</td>
      <td class="text-center text-muted">${formatNumber(item.quantity)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-center do-qty"
          value="${item.quantity}" min="0" max="${item.quantity}" step="0.5"
          oninput="doUpdateTotal()">
      </td>
      <td class="text-center text-muted">${formatCurrency(item.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold do-line-total text-primary">${formatCurrency(item.line_total)}</td>
    </tr>`).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/orders/${so.id}')"><i class="bi bi-arrow-left"></i> Back to SO</button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-truck me-2" style="color:#ec4899"></i>New Delivery Order</h4>
          <div class="text-muted small">Copied from ${so.sales_order_number} — ${custName}</div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Delivery Date <span class="text-danger">*</span></label>
              <input type="date" id="doDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Delivered By</label>
              <input type="text" id="doDeliveredBy" class="form-control" placeholder="Driver / staff name">
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="doNotes" class="form-control" placeholder="e.g. Partial delivery">
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
          <span class="fw-bold"><i class="bi bi-egg me-2 text-warning"></i>Items to Deliver</span>
          <span class="text-muted small">Adjust <strong>Qty to Deliver</strong> for partial delivery</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:5%">
                  <input type="checkbox" class="form-check-input" id="doCheckAll" checked onchange="doToggleAll(this)">
                </th>
                <th>Egg Size</th><th class="text-center">Unit</th><th class="text-center">Ordered</th>
                <th class="text-center">Qty to Deliver</th><th class="text-center">Price/Unit</th><th class="text-end pe-4">Line Total</th>
              </tr>
            </thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold">
              <tr><td colspan="6" class="text-end pe-3 fw-bold">Delivery Total</td>
                  <td class="text-end pe-4 fw-bold text-primary fs-5" id="doGrandTotal">${formatCurrency(so.grand_total)}</td></tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 justify-content-end">
        <button class="btn btn-outline-secondary px-4" onclick="navigate('#/sales/orders/${so.id}')">Cancel</button>
        <button class="btn btn-primary px-4" onclick="submitNewDO(${so.id})">
          <i class="bi bi-truck me-1"></i>Create Delivery Order
        </button>
      </div>
    </div>
  `);
}

function doUpdateTotal() {
  let grand = 0;
  document.querySelectorAll('.do-item-row').forEach(row => {
    if (!row.querySelector('.do-row-check').checked) return;
    const qty   = parseFloat(row.querySelector('.do-qty').value) || 0;
    const price = parseFloat(row.querySelector('.do-price').value) || 0;
    const line  = qty * price;
    row.querySelector('.do-line-total').textContent = formatCurrency(line);
    grand += line;
  });
  const el = document.getElementById('doGrandTotal');
  if (el) el.textContent = formatCurrency(grand);
}

function doToggleRow(cb) {
  const row = cb.closest('tr');
  const qty = row.querySelector('.do-qty');
  row.style.opacity = cb.checked ? '1' : '0.4';
  if (!cb.checked) { qty.value = 0; }
  doUpdateTotal();
}

function doToggleAll(master) {
  document.querySelectorAll('.do-row-check').forEach(cb => { cb.checked = master.checked; doToggleRow(cb); });
}

async function submitNewDO(soId) {
  const items = [];
  document.querySelectorAll('.do-item-row').forEach(row => {
    if (!row.querySelector('.do-row-check').checked) return;
    const qty = parseFloat(row.querySelector('.do-qty').value) || 0;
    if (qty <= 0) return;
    items.push({
      sku:                row.querySelector('.do-sku').value,
      unit:               row.querySelector('.do-unit').value,
      quantity_ordered:   qty,
      quantity_delivered: qty,
      price_per_unit:     parseFloat(row.querySelector('.do-price').value) || 0,
    });
  });
  if (items.length === 0) { toast('No items selected.', 'warning'); return; }

  const result = await api.CreateDeliveryOrder({
    sales_order_id: soId,
    date:           new Date(document.getElementById('doDate').value).toISOString(),
    delivered_by:   document.getElementById('doDeliveredBy').value,
    notes:          document.getElementById('doNotes').value,
    items,
  });
  if (result) navigate('#/sales/delivery-orders');
}

// ── AR Invoice Detail ─────────────────────────────────────────────────────────

async function loadARInvoiceDetail(id) {
  showLoading();
  const inv = await api.GetARInvoice(id);
  if (!inv) { showView(`<div class="alert alert-warning m-4">AR Invoice not found.</div>`); return; }

  const items   = inv.items || [];
  const colLines = inv.collection_lines || [];
  const collected = inv.amount_collected || 0;
  const balance   = Math.max(0, (inv.total_amount || 0) - collected);
  const so  = inv.sales_order || null;
  const doRec = inv.delivery_order || null;
  const custName = inv.customer_name_snapshot || (inv.customer && inv.customer.name) || '—';

  const statusColor = { Paid: 'success', Partial: 'warning', Open: 'primary', Cancelled: 'secondary' };
  const badge = `<span class="badge bg-${statusColor[inv.status] || 'secondary'} fs-6">${inv.status}</span>`;

  const canCollect = inv.status === 'Open' || inv.status === 'Partial';
  const canCancel  = (inv.status === 'Open' || inv.status === 'Partial') && colLines.length === 0;

  const statusBanner = inv.status === 'Paid'
    ? `<div class="alert alert-success border-0 mb-3 d-flex align-items-center gap-2"><i class="bi bi-check-circle-fill fs-5"></i><strong>Fully Collected</strong> — ${formatCurrency(inv.total_amount)} received</div>`
    : inv.status === 'Partial'
    ? `<div class="alert alert-warning border-0 mb-3 d-flex align-items-center gap-2"><i class="bi bi-clock-fill fs-5"></i>Balance: <strong>${formatCurrency(balance)}</strong> remaining</div>`
    : inv.status === 'Open'
    ? `<div class="alert alert-primary border-0 mb-3">${formatCurrency(inv.total_amount)} outstanding</div>`
    : '';

  const itemRows = items.map(i => `
    <tr>
      <td class="ps-4 fw-semibold">${i.sku}</td>
      <td class="text-center text-muted">${i.unit}</td>
      <td class="text-center">${formatNumber(i.quantity)}</td>
      <td class="text-center">${formatCurrency(i.price_per_unit)}</td>
      <td class="text-end pe-4 fw-bold text-primary">${formatCurrency(i.quantity * i.price_per_unit)}</td>
    </tr>`).join('') || `<tr><td colspan="5" class="text-center text-muted py-3">No items.</td></tr>`;

  const colRowsHtml = colLines.length === 0 ? '' : `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center">
        <span class="fw-bold"><i class="bi bi-cash-coin me-2 text-success"></i>Collections Applied</span>
        ${canCollect ? `<button class="btn btn-sm btn-outline-success" onclick="newCollectionFromARInv(${inv.id})">+ Add Collection</button>` : ''}
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th class="ps-4">Collection #</th><th>Date</th><th>Method</th><th>Reference</th><th class="text-end pe-4">Amount Applied</th></tr>
          </thead>
          <tbody>
            ${colLines.map(cl => {
              const col = cl.collection || {};
              return `<tr>
                <td class="ps-4 fw-bold">${col.collection_number || '—'}</td>
                <td class="text-muted small">${formatDate(col.date)}</td>
                <td>${col.payment_method || '—'}</td>
                <td class="text-muted">${col.reference_number || '—'}</td>
                <td class="text-end pe-4 fw-bold text-success">${formatCurrency(cl.amount_applied)}</td>
              </tr>`;
            }).join('')}
          </tbody>
          <tfoot class="table-light fw-bold small">
            <tr><td colspan="4" class="ps-4 text-muted">Invoice Total / Collected / Balance</td><td class="text-end pe-4 text-primary">${formatCurrency(inv.total_amount)}</td></tr>
            <tr><td colspan="4"></td><td class="text-end pe-4 text-success">${formatCurrency(collected)} collected</td></tr>
            <tr><td colspan="4"></td>
              <td class="text-end pe-4 ${balance <= 0.005 ? 'text-success' : 'text-danger fw-bold'}">
                ${balance <= 0.005 ? '<i class="bi bi-check-circle-fill me-1"></i>Fully Collected' : formatCurrency(balance) + ' remaining'}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/ar-invoices')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-file-earmark-text me-2 text-info"></i>${inv.invoice_number}</h4>
          <small class="text-muted">AR Invoice · ${formatDate(inv.date)}</small>
        </div>
        ${badge}
        ${canCollect ? `<button class="btn btn-sm btn-success text-white fw-bold" onclick="newCollectionFromARInv(${inv.id})">
          <i class="bi bi-cash-coin me-1"></i>Record Collection</button>` : ''}
        ${canCancel ? `<button class="btn btn-sm btn-outline-danger" onclick="cancelARInv(${inv.id})">Cancel</button>` : ''}
      </div>

      ${statusBanner}

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted fw-bold">CUSTOMER</div><div class="fw-semibold">${custName}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">INVOICE DATE</div><div>${formatDate(inv.date)}</div></div>
            ${inv.due_date ? `<div class="col-md-4"><div class="small text-muted fw-bold">DUE DATE</div><div>${formatDate(inv.due_date)}</div></div>` : ''}
            <div class="col-md-4"><div class="small text-muted fw-bold">TERMS</div><div>${inv.terms || '—'}</div></div>
            ${so ? `<div class="col-md-4"><div class="small text-muted fw-bold">SALES ORDER</div>
              <a href="#" onclick="navigate('#/sales/orders/${inv.sales_order_id}');return false;" class="text-decoration-none fw-semibold">${so.sales_order_number}</a></div>` : ''}
            ${doRec ? `<div class="col-md-4"><div class="small text-muted fw-bold">DELIVERY ORDER</div>
              <a href="#" onclick="navigate('#/sales/delivery-orders/${inv.delivery_order_id}');return false;" class="text-decoration-none fw-semibold">${doRec.delivery_number}</a></div>` : ''}
            ${inv.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${inv.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-egg me-2 text-warning"></i>Line Items</span></div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Egg Size</th><th class="text-center">Unit</th><th class="text-center">Qty</th><th class="text-center">Price/Unit</th><th class="text-end pe-4">Total</th></tr></thead>
            <tbody>${itemRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="4" class="text-end pe-3 ps-4">Total Amount</td><td class="text-end pe-4 text-primary fs-5">${formatCurrency(inv.total_amount)}</td></tr></tfoot>
          </table>
        </div>
      </div>

      ${colRowsHtml}
    </div>
  `);
}

async function cancelARInv(id) {
  if (!confirm('Cancel this AR Invoice? This will reverse the accounting entry.')) return;
  const r = await api.CancelARInvoice(id);
  if (r) navigate(`#/sales/ar-invoices/${id}`);
}

// ── New AR Invoice Form ───────────────────────────────────────────────────────

let _arInvSOList = [];
let _arInvDOList = [];

async function loadNewARInvoiceForm() {
  showLoading();
  const [soList, doList] = await Promise.all([
    api.ListSalesOrders(),
    api.ListDeliveryOrders(0),
  ]);
  _arInvSOList = (soList || []).filter(o => o.payment_status !== 'Void');
  _arInvDOList = (doList || []).filter(d => d.status === 'Delivered');

  const prefillSOId = window._newARInvoicePrefillSOId || null;
  const prefillDOId = window._newARInvoicePrefillDOId || null;
  window._newARInvoicePrefillSOId = null;
  window._newARInvoicePrefillDOId = null;

  const today = new Date().toISOString().slice(0, 10);

  const soOpts = _arInvSOList.map(o =>
    `<option value="${o.id}">${o.sales_order_number} — ${o.customer_name_snapshot || 'Walk-in'} (${formatCurrency(o.grand_total)})</option>`
  ).join('');

  const doOpts = _arInvDOList.map(d =>
    `<option value="${d.id}" data-sales-order-id="${d.sales_order_id}">${d.delivery_number} (${formatDate(d.date)})</option>`
  ).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/ar-invoices')"><i class="bi bi-arrow-left"></i> Back</button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>New AR Invoice</h4>
          <div class="text-muted small">Copy from Sales Order or Delivery Order</div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold">Copy From</span></div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-6">
              <label class="form-label small fw-bold">Sales Order <span class="text-danger">*</span></label>
              <select id="arSOSelect" class="form-select" onchange="arInvOnSOChange()">
                <option value="">— Select Sales Order —</option>${soOpts}
              </select>
            </div>
            <div class="col-md-6">
              <label class="form-label small fw-bold">Delivery Order <span class="text-muted fw-normal">(optional)</span></label>
              <select id="arDOSelect" class="form-select" onchange="arInvOnDOChange()">
                <option value="">— Skip (copy from SO directly) —</option>${doOpts}
              </select>
              <div class="form-text">If selected, items will be copied from the Delivery Order.</div>
            </div>
          </div>
          <div class="row g-3 mt-1" id="arCustInfo" style="display:none">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Customer</label>
              <input type="text" id="arCustName" class="form-control bg-light" disabled>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">SO/DO Reference</label>
              <input type="text" id="arSORef" class="form-control bg-light" disabled>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Terms</label>
              <select id="arTerms" class="form-select">
                <option>COD</option><option>7 days</option><option>15 days</option><option>30 days</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Invoice Date <span class="text-danger">*</span></label>
              <input type="date" id="arDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="arDueDate" class="form-control">
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Notes</label>
              <input type="text" id="arNotes" class="form-control" placeholder="Optional">
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3" id="arItemsCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between">
          <span class="fw-bold"><i class="bi bi-egg me-2 text-warning"></i>Invoice Items</span>
          <span class="text-muted small">Adjust qty for partial invoice</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:5%">
                  <input type="checkbox" class="form-check-input" id="arCheckAll" checked onchange="arToggleAll(this)">
                </th>
                <th>Egg Size</th><th class="text-center">Unit</th><th class="text-center">Available</th>
                <th class="text-center">Qty to Invoice</th><th class="text-center">Price/Unit ₱</th><th class="text-end pe-4">Line Total</th>
              </tr>
            </thead>
            <tbody id="arItemsBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr><td colspan="6" class="text-end pe-3">Invoice Total</td>
                  <td class="text-end pe-4 text-primary fs-5" id="arGrandTotal">₱0.00</td></tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div class="d-flex gap-2 justify-content-end">
        <button class="btn btn-outline-secondary px-4" onclick="navigate('#/sales/ar-invoices')">Cancel</button>
        <button class="btn btn-primary px-4" id="arSubmitBtn" disabled onclick="submitNewARInvoice()">
          <i class="bi bi-file-earmark-text me-1"></i>Create AR Invoice
        </button>
      </div>
    </div>
  `);

  if (prefillDOId) {
    const doSel = document.getElementById('arDOSelect');
    if (doSel) { doSel.value = prefillDOId; await arInvOnDOChange(); }
  } else if (prefillSOId) {
    const soSel = document.getElementById('arSOSelect');
    if (soSel) { soSel.value = prefillSOId; await arInvOnSOChange(); }
  }
}

async function arInvOnSOChange() {
  const doSel = document.getElementById('arDOSelect');
  if (doSel && doSel.value) return;
  const soId = parseInt(document.getElementById('arSOSelect').value);
  if (!soId) { arHideItems(); return; }
  const so = _arInvSOList.find(o => o.id === soId);
  if (!so) return;
  const custName = so.customer_name_snapshot || 'Walk-in';
  arRenderItems(
    (so.items || []).map(i => ({ sku: i.sku, unit: i.unit, quantity: i.quantity, price_per_unit: i.price_per_unit })),
    custName, so.sales_order_number
  );
}

async function arInvOnDOChange() {
  const doId = document.getElementById('arDOSelect').value;
  if (!doId) { await arInvOnSOChange(); return; }
  const doRec = await api.GetDeliveryOrder(parseInt(doId));
  if (!doRec) return;
  const soSel = document.getElementById('arSOSelect');
  if (soSel) soSel.value = doRec.sales_order_id;
  const so = doRec.sales_order || {};
  arRenderItems(
    (doRec.items || []).map(i => ({ sku: i.sku, unit: i.unit, quantity: i.quantity_delivered, price_per_unit: i.price_per_unit })),
    so.customer_name_snapshot || 'Walk-in', doRec.delivery_number
  );
}

function arRenderItems(items, custName, refNumber) {
  const tbody = document.getElementById('arItemsBody');
  tbody.innerHTML = items.map(item => `
    <tr class="ar-item-row">
      <td class="ps-4">
        <input type="checkbox" class="form-check-input ar-row-check" checked onchange="arToggleRow(this)">
        <input type="hidden" class="ar-sku" value="${item.sku}">
        <input type="hidden" class="ar-unit" value="${item.unit}">
      </td>
      <td class="fw-semibold">${item.sku}</td>
      <td class="text-center text-muted">${item.unit}</td>
      <td class="text-center text-muted">${formatNumber(item.quantity)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-center ar-qty"
          value="${item.quantity}" min="0" step="0.5" oninput="arUpdateTotal()">
      </td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-center ar-price"
          value="${item.price_per_unit.toFixed(2)}" min="0" step="0.01" oninput="arUpdateTotal()">
      </td>
      <td class="text-end pe-4 fw-bold ar-line-total text-primary">${formatCurrency(item.quantity * item.price_per_unit)}</td>
    </tr>`).join('');

  document.getElementById('arCustInfo').style.display = '';
  document.getElementById('arCustName').value = custName;
  document.getElementById('arSORef').value = refNumber;
  document.getElementById('arItemsCard').style.display = '';
  document.getElementById('arSubmitBtn').disabled = false;
  arUpdateTotal();
}

function arUpdateTotal() {
  let grand = 0;
  document.querySelectorAll('.ar-item-row').forEach(row => {
    if (!row.querySelector('.ar-row-check').checked) return;
    const qty = parseFloat(row.querySelector('.ar-qty').value) || 0;
    const price = parseFloat(row.querySelector('.ar-price').value) || 0;
    row.querySelector('.ar-line-total').textContent = formatCurrency(qty * price);
    grand += qty * price;
  });
  const el = document.getElementById('arGrandTotal');
  if (el) el.textContent = formatCurrency(grand);
}

function arToggleRow(cb) {
  const row = cb.closest('tr');
  row.style.opacity = cb.checked ? '1' : '0.4';
  if (!cb.checked) row.querySelector('.ar-qty').value = 0;
  arUpdateTotal();
}

function arToggleAll(master) {
  document.querySelectorAll('.ar-row-check').forEach(cb => { cb.checked = master.checked; arToggleRow(cb); });
}

function arHideItems() {
  const card = document.getElementById('arItemsCard');
  const info = document.getElementById('arCustInfo');
  const btn  = document.getElementById('arSubmitBtn');
  if (card) card.style.display = 'none';
  if (info) info.style.display = 'none';
  if (btn)  btn.disabled = true;
}

async function submitNewARInvoice() {
  const soId = parseInt(document.getElementById('arSOSelect').value) || null;
  const doRaw = document.getElementById('arDOSelect').value;
  const doId  = doRaw ? parseInt(doRaw) : null;
  if (!soId) { toast('Please select a Sales Order.', 'warning'); return; }

  const items = [];
  document.querySelectorAll('.ar-item-row').forEach(row => {
    if (!row.querySelector('.ar-row-check').checked) return;
    const qty = parseFloat(row.querySelector('.ar-qty').value) || 0;
    if (qty <= 0) return;
    items.push({
      sku:           row.querySelector('.ar-sku').value,
      unit:          row.querySelector('.ar-unit').value,
      quantity:      qty,
      price_per_unit: parseFloat(row.querySelector('.ar-price').value) || 0,
    });
  });
  if (items.length === 0) { toast('No items selected.', 'warning'); return; }

  const so = _arInvSOList.find(o => o.id === soId) || {};
  const custName = so.customer_name_snapshot || 'Walk-in';
  const dueDateVal = document.getElementById('arDueDate').value;
  const dueDate = dueDateVal ? new Date(dueDateVal).toISOString() : null;

  const btn = document.getElementById('arSubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Creating…';

  const result = await api.CreateARInvoice({
    sales_order_id:    soId,
    delivery_order_id: doId,
    customer_id:       so.customer_id || null,
    customer_name:     custName,
    customer_addr:     so.customer_address_snapshot || '',
    customer_contact:  so.customer_contact_snapshot || '',
    date:              new Date(document.getElementById('arDate').value).toISOString(),
    terms:             document.getElementById('arTerms').value,
    notes:             document.getElementById('arNotes').value,
    due_date:          dueDate,
    items,
  });

  if (result) {
    navigate('#/sales/ar-invoices');
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-file-earmark-text me-1"></i>Create AR Invoice';
  }
}

// ── Collection Detail ─────────────────────────────────────────────────────────

async function loadCollectionDetail(id) {
  showLoading();
  const col = await api.GetCollection(id);
  if (!col) { showView(`<div class="alert alert-warning m-4">Collection not found.</div>`); return; }

  const lines = col.lines || [];
  const custName = col.customer_name_snapshot || (col.customer && col.customer.name) || '—';

  const lineRows = lines.map(l => {
    const inv = l.ar_invoice || {};
    const bal = Math.max(0, (inv.total_amount || 0) - (inv.amount_collected || 0));
    return `<tr>
      <td class="ps-4 fw-semibold">
        <a href="#" onclick="navigate('#/sales/ar-invoices/${l.ar_invoice_id}');return false;" class="text-decoration-none">
          ${inv.invoice_number || '—'}
        </a>
      </td>
      <td class="text-muted small">${formatDate(inv.date)}</td>
      <td>${salesStatusBadge(inv.status)}</td>
      <td class="text-end pe-4 fw-bold text-success">${formatCurrency(l.amount_applied)}</td>
    </tr>`;
  }).join('') || `<tr><td colspan="4" class="text-center text-muted py-3">No invoice lines.</td></tr>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:800px">
      <div class="d-flex align-items-center gap-3 mb-4 flex-wrap">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/collections')"><i class="bi bi-arrow-left"></i></button>
        <div class="flex-grow-1">
          <h4 class="fw-bold mb-0"><i class="bi bi-cash-coin me-2 text-success"></i>${col.collection_number}</h4>
          <small class="text-muted">Collection · ${formatDate(col.date)}</small>
        </div>
        <span class="badge bg-success fs-6">Posted</span>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4"><div class="small text-muted fw-bold">CUSTOMER</div><div class="fw-semibold">${custName}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">DATE</div><div>${formatDate(col.date)}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">PAYMENT METHOD</div><div>${col.payment_method || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">REFERENCE #</div><div>${col.reference_number || '—'}</div></div>
            <div class="col-md-4"><div class="small text-muted fw-bold">TOTAL AMOUNT</div><div class="fw-bold fs-5 text-success">${formatCurrency(col.total_amount)}</div></div>
            ${col.notes ? `<div class="col-12"><div class="small text-muted fw-bold">NOTES</div><div class="text-muted">${col.notes}</div></div>` : ''}
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-3">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices Settled</span>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light"><tr><th class="ps-4">Invoice #</th><th>Date</th><th>Status</th><th class="text-end pe-4">Amount Applied</th></tr></thead>
            <tbody>${lineRows}</tbody>
            <tfoot class="table-light fw-bold"><tr><td colspan="3" class="text-end pe-3 ps-4">Total Collection</td><td class="text-end pe-4 text-success fs-5">${formatCurrency(col.total_amount)}</td></tr></tfoot>
          </table>
        </div>
      </div>
    </div>
  `);
}

// ── New Collection Form ───────────────────────────────────────────────────────

let _colAllInvoices = [];

async function loadNewCollectionForm() {
  showLoading();
  const [customers, allInvoices] = await Promise.all([
    api.ListCustomers(true),
    api.ListARInvoices(null, ''),
  ]);
  const custList = customers || [];
  _colAllInvoices = (allInvoices || []).filter(i => i.status === 'Open' || i.status === 'Partial');

  const prefillInvId = window._newCollectionPrefillInvId || null;
  window._newCollectionPrefillInvId = null;

  const today = new Date().toISOString().slice(0, 10);
  const custOpts = custList.map(c => `<option value="${c.id}" data-name="${c.name}">${c.name}</option>`).join('');

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:960px">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-sm btn-outline-secondary" onclick="navigate('#/sales/collections')"><i class="bi bi-arrow-left"></i> Back</button>
        <div>
          <h4 class="mb-0 fw-bold"><i class="bi bi-cash-coin me-2 text-success"></i>Record Collection</h4>
          <div class="text-muted small">One receipt — applies to one or more AR Invoices</div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-person me-2 text-success"></i>Customer</span></div>
        <div class="card-body">
          <div class="row g-3 align-items-end">
            <div class="col-md-6">
              <label class="form-label small fw-bold">Select Customer <span class="text-danger">*</span></label>
              <select id="colCustSel" class="form-select" onchange="colOnCustChange()">
                <option value="">— Select a customer —</option>${custOpts}
              </select>
            </div>
            <div class="col-md-6" id="colCustSummary" style="display:none">
              <div class="p-3 rounded d-flex gap-4" style="background:#f0fdf4;border:1px solid #bbf7d0">
                <div><div class="small text-muted">Open Invoices</div><div class="fw-bold fs-5 text-success" id="colSummCount">—</div></div>
                <div><div class="small text-muted">Total Outstanding</div><div class="fw-bold fs-5 text-danger" id="colSummTotal">—</div></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3" id="colInvCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3 d-flex justify-content-between align-items-center flex-wrap gap-2">
          <span class="fw-bold"><i class="bi bi-file-earmark-text me-2 text-info"></i>AR Invoices to Collect</span>
          <div class="d-flex gap-2">
            <button type="button" class="btn btn-sm btn-outline-success" onclick="colSelectAll(true)"><i class="bi bi-check-all me-1"></i>Select All</button>
            <button type="button" class="btn btn-sm btn-outline-secondary" onclick="colSelectAll(false)">Deselect All</button>
            <button type="button" class="btn btn-sm btn-outline-warning" onclick="colFillBalances()"><i class="bi bi-arrow-down-circle me-1"></i>Fill Balances</button>
          </div>
        </div>
        <div class="table-responsive">
          <table class="table align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th class="ps-4" style="width:40px"><input type="checkbox" class="form-check-input" id="colCheckAll" checked onchange="colToggleAll(this)"></th>
                <th>Invoice #</th><th>Date</th>
                <th class="text-end">Invoice Total</th><th class="text-end">Collected</th>
                <th class="text-end">Balance Due</th><th class="text-center" style="width:160px">Amount to Collect ₱</th>
              </tr>
            </thead>
            <tbody id="colInvBody"></tbody>
            <tfoot class="table-light fw-bold">
              <tr>
                <td colspan="5" class="text-end pe-3 ps-4 text-muted">TOTAL TO COLLECT</td>
                <td class="text-end text-danger" id="colTotalBal">—</td>
                <td class="text-center fw-bold text-success fs-6" id="colTotalApply">—</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div id="colNoInvMsg" style="display:none" class="card border-0 shadow-sm mb-3">
        <div class="card-body text-center py-4 text-muted">
          <i class="bi bi-check-circle-fill text-success" style="font-size:2rem"></i>
          <div class="mt-2 fw-semibold">No open AR invoices for this customer</div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3" id="colDetailsCard" style="display:none">
        <div class="card-header bg-white border-bottom py-3"><span class="fw-bold"><i class="bi bi-credit-card me-2"></i>Payment Details</span></div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3"><label class="form-label small fw-bold">Date <span class="text-danger">*</span></label><input type="date" id="colDate" class="form-control" value="${today}" required></div>
            <div class="col-md-3"><label class="form-label small fw-bold">Payment Method</label>
              <select id="colMethod" class="form-select"><option>Cash</option><option>Bank Transfer</option><option>GCash</option><option>Check</option></select>
            </div>
            <div class="col-md-3"><label class="form-label small fw-bold">Reference #</label><input type="text" id="colRef" class="form-control" placeholder="Optional"></div>
            <div class="col-md-3"><label class="form-label small fw-bold">Notes</label><input type="text" id="colNotes" class="form-control" placeholder="Optional"></div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm" id="colSubmitCard" style="display:none">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col"><div class="small text-muted mb-1">Selected</div><div class="fw-semibold" id="colSelCount">0 invoice(s)</div></div>
            <div class="col text-center">
              <div class="small text-muted mb-1">Total Collection Amount</div>
              <div class="fw-bold fs-4 text-success" id="colGrandTotal">₱0.00</div>
            </div>
            <div class="col text-end d-flex gap-2 justify-content-end">
              <button class="btn btn-outline-secondary px-4" onclick="navigate('#/sales/collections')">Cancel</button>
              <button class="btn btn-success px-5 text-white fw-bold" id="colSubmitBtn" disabled onclick="submitCollection()">
                <i class="bi bi-cash-coin me-2"></i>Post Collection
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `);

  if (prefillInvId) {
    const inv = _colAllInvoices.find(i => i.id === prefillInvId);
    if (inv) {
      const custSel = document.getElementById('colCustSel');
      const opt = Array.from(custSel.options).find(o => {
        const c = customers.find(c => c.id === parseInt(o.value));
        return c && c.name === inv.customer_name_snapshot;
      });
      if (opt) { custSel.value = opt.value; colOnCustChange(prefillInvId); }
    }
  }
}

function colOnCustChange(prefillInvId) {
  const sel = document.getElementById('colCustSel');
  const custName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  if (!sel.value) {
    ['colInvCard','colDetailsCard','colSubmitCard','colCustSummary','colNoInvMsg']
      .forEach(id => { const el = document.getElementById(id); if (el) el.style.display = 'none'; });
    return;
  }
  const openInvs = _colAllInvoices.filter(i =>
    i.customer_name_snapshot === custName || (i.customer && i.customer.name === custName)
  );
  colRenderInvoices(openInvs, prefillInvId);
}

function colRenderInvoices(invoices, prefillInvId) {
  const tbody = document.getElementById('colInvBody');
  if (!tbody) return;
  tbody.innerHTML = '';

  if (invoices.length === 0) {
    ['colInvCard','colDetailsCard','colSubmitCard'].forEach(id => { const el = document.getElementById(id); if (el) el.style.display = 'none'; });
    document.getElementById('colNoInvMsg').style.display = '';
    document.getElementById('colCustSummary').style.display = 'none';
    return;
  }

  let totalBal = 0;
  invoices.forEach(inv => {
    const balance = Math.max(0, (inv.total_amount || 0) - (inv.amount_collected || 0));
    totalBal += balance;
    const isPrefill = prefillInvId && inv.id !== prefillInvId;
    const row = document.createElement('tr');
    row.className = 'col-inv-row';
    row.innerHTML = `
      <td class="ps-4"><input type="checkbox" class="form-check-input col-inv-check" data-inv-id="${inv.id}"
        ${isPrefill ? '' : 'checked'} onchange="colToggleRow(this)"></td>
      <td class="fw-semibold">${inv.invoice_number}${inv.status === 'Partial' ? ' <span class="badge bg-warning text-dark ms-1" style="font-size:10px">Partial</span>' : ''}</td>
      <td class="text-muted small">${formatDate(inv.date)}</td>
      <td class="text-end text-muted">${formatCurrency(inv.total_amount || 0)}</td>
      <td class="text-end text-success small">${formatCurrency(inv.amount_collected || 0)}</td>
      <td class="text-end fw-bold text-danger">${formatCurrency(balance)}</td>
      <td class="text-center">
        <input type="number" class="form-control form-control-sm text-end col-amt"
          value="${isPrefill ? '0.00' : balance.toFixed(2)}"
          min="0" max="${balance}" step="0.01" data-max="${balance}"
          ${isPrefill ? 'disabled' : ''}
          oninput="colUpdateTotals()">
      </td>`;
    tbody.appendChild(row);
  });

  document.getElementById('colTotalBal').textContent   = formatCurrency(totalBal);
  document.getElementById('colSummCount').textContent  = invoices.length;
  document.getElementById('colSummTotal').textContent  = formatCurrency(totalBal);
  document.getElementById('colCustSummary').style.display = '';
  document.getElementById('colInvCard').style.display     = '';
  document.getElementById('colNoInvMsg').style.display    = 'none';
  document.getElementById('colDetailsCard').style.display = '';
  document.getElementById('colSubmitCard').style.display  = '';
  document.getElementById('colCheckAll').checked = true;
  colUpdateTotals();
}

function colToggleAll(master) {
  document.querySelectorAll('.col-inv-check').forEach(cb => { cb.checked = master.checked; colToggleRow(cb); });
}
function colToggleRow(cb) {
  const row = cb.closest('tr'), amt = row.querySelector('.col-amt');
  if (!cb.checked) { amt.disabled = true; amt.value = '0'; row.style.opacity = '0.4'; }
  else             { amt.disabled = false; amt.value = amt.dataset.max; row.style.opacity = '1'; }
  colUpdateTotals();
}
function colSelectAll(state) {
  document.querySelectorAll('.col-inv-check').forEach(cb => { cb.checked = state; colToggleRow(cb); });
  const master = document.getElementById('colCheckAll');
  if (master) master.checked = state;
}
function colFillBalances() {
  document.querySelectorAll('.col-inv-row').forEach(row => {
    const cb = row.querySelector('.col-inv-check'), amt = row.querySelector('.col-amt');
    if (cb && cb.checked && amt) amt.value = amt.dataset.max;
  });
  colUpdateTotals();
}
function colUpdateTotals() {
  let total = 0, count = 0;
  document.querySelectorAll('.col-inv-row').forEach(row => {
    const cb = row.querySelector('.col-inv-check'), amt = row.querySelector('.col-amt');
    if (cb && cb.checked && amt) { const v = parseFloat(amt.value) || 0; total += v; if (v > 0) count++; }
  });
  const applyEl = document.getElementById('colTotalApply');
  const grandEl = document.getElementById('colGrandTotal');
  const countEl = document.getElementById('colSelCount');
  const btn     = document.getElementById('colSubmitBtn');
  if (applyEl) applyEl.textContent = formatCurrency(total);
  if (grandEl) grandEl.textContent = formatCurrency(total);
  if (countEl) countEl.textContent = count + ' invoice(s)';
  if (btn)     btn.disabled = total <= 0;
}

async function submitCollection() {
  const sel      = document.getElementById('colCustSel');
  const custId   = parseInt(sel.value) || null;
  const custName = sel.options[sel.selectedIndex]?.dataset?.name || '';
  const lines    = [];

  document.querySelectorAll('.col-inv-row').forEach(row => {
    const cb  = row.querySelector('.col-inv-check');
    const amt = row.querySelector('.col-amt');
    const id  = parseInt(cb.dataset.invId);
    const val = parseFloat(amt.value) || 0;
    if (cb.checked && val > 0) lines.push({ ar_invoice_id: id, amount_applied: val });
  });

  if (lines.length === 0) { toast('No invoices selected.', 'warning'); return; }

  const btn = document.getElementById('colSubmitBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Posting…';

  const result = await api.CreateCollection({
    date:           new Date(document.getElementById('colDate').value).toISOString(),
    customer_id:    custId,
    customer_name:  custName,
    payment_method: document.getElementById('colMethod').value,
    ref_num:        document.getElementById('colRef').value,
    notes:          document.getElementById('colNotes').value,
    lines,
  });

  if (result) {
    navigate('#/sales/collections');
  } else {
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-cash-coin me-2"></i>Post Collection';
  }
}

// ── New Sales Order Form ───────────────────────────────────────────────────────

const SO_EGG_SIZES = ['Jumbo', 'Extra Large', 'Large', 'Medium', 'Small', 'Pullet', 'Peewee'];
let _soCustomerPrices = {}; // {size_unit: price}
let _soCustomers = [];

async function loadNewSOForm() {
  showLoading();
  const customers = await api.ListCustomers(false);
  _soCustomers = customers || [];
  _soCustomerPrices = {};

  const today = new Date().toISOString().slice(0, 10);

  const custOptions = _soCustomers.map(c => {
    const hasPg = c.price_group_id ? ' ★' : '';
    const type  = c.customer_type === 'Account' ? ' (Account)' : '';
    return `<option value="${c.id}"
      data-name="${c.name}"
      data-address="${c.delivery_address || c.address || ''}"
      data-contact="${c.contact_number || ''}"
      data-pgid="${c.price_group_id || ''}"
    >${c.name}${type}${hasPg}</option>`;
  }).join('');

  const sizeOptions = SO_EGG_SIZES.map(s => `<option value="${s}">${s}</option>`).join('');

  showView(`
    <div class="container py-4" style="max-width:900px;">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/sales/orders')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div>
          <h4 class="fw-bold mb-0"><i class="bi bi-receipt me-2 text-primary"></i>New Sales Order</h4>
          <small class="text-muted">Fill in customer details and add egg line items below</small>
        </div>
      </div>

      <!-- Date -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label small fw-bold">Order Date</label>
              <input type="date" id="soDate" class="form-control" value="${today}" required>
            </div>
            <div class="col-md-8">
              <label class="form-label small fw-bold">Invoice # <span class="text-muted fw-normal">(auto-generated on save)</span></label>
              <input type="text" class="form-control bg-light" disabled placeholder="SO-YYYYMMDD-XXXX">
            </div>
          </div>
        </div>
      </div>

      <!-- Customer -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-light border-0 py-3">
          <h6 class="fw-bold mb-0"><i class="bi bi-person me-2"></i>Customer</h6>
        </div>
        <div class="card-body">
          <div class="mb-3">
            <label class="form-label small fw-bold">
              Existing Account <span class="text-muted fw-normal">(optional — leave blank for walk-in)</span>
              <span id="soPriceBadge" class="badge ms-2 rounded-pill d-none"
                    style="background:#dcfce7;color:#16a34a;font-size:11px;">
                <i class="bi bi-tag-fill me-1"></i>Price list loaded
              </span>
            </label>
            <select id="soCustomerSelect" class="form-select" onchange="soFillCustomer(this)">
              <option value="">— Walk-in / New Customer —</option>
              ${custOptions}
            </select>
          </div>
          <div class="row g-3">
            <div class="col-md-5">
              <label class="form-label small fw-bold">Customer Name <span class="text-danger">*</span></label>
              <input type="text" id="soCustName" class="form-control" placeholder="Full name" required>
            </div>
            <div class="col-md-4">
              <label class="form-label small fw-bold">Contact Number</label>
              <input type="text" id="soCustContact" class="form-control" placeholder="09XX XXX XXXX">
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Terms</label>
              <select id="soTerms" class="form-select">
                <option value="COD">COD</option>
                <option value="7 days">7 days</option>
                <option value="15 days">15 days</option>
                <option value="30 days">30 days</option>
              </select>
            </div>
            <div class="col-12">
              <label class="form-label small fw-bold">Delivery Address</label>
              <input type="text" id="soCustAddress" class="form-control" placeholder="Street, Barangay, Municipality">
            </div>
          </div>
        </div>
      </div>

      <!-- Line Items -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-warning bg-opacity-10 border-0 py-3 d-flex justify-content-between align-items-center">
          <h6 class="fw-bold mb-0"><i class="bi bi-egg me-2 text-warning"></i>Order Items</h6>
          <button type="button" class="btn btn-sm btn-outline-warning" onclick="soAddRow()">
            <i class="bi bi-plus-lg me-1"></i> Add Item
          </button>
        </div>
        <div class="card-body p-0">
          <div class="table-responsive">
            <table class="table mb-0">
              <thead class="table-light">
                <tr>
                  <th class="ps-3" style="width:30%;">Egg Size (SKU)</th>
                  <th style="width:15%;">Unit</th>
                  <th style="width:15%;">Qty</th>
                  <th style="width:18%;">Price / Unit (₱)</th>
                  <th style="width:17%;">Line Total</th>
                  <th style="width:5%;"></th>
                </tr>
              </thead>
              <tbody id="soItemsBody"></tbody>
              <tfoot>
                <tr class="table-light">
                  <td colspan="4" class="text-end fw-bold pe-3">Grand Total</td>
                  <td class="fw-bold text-primary fs-5" id="soGrandTotal">₱0.00</td>
                  <td></td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>

      <!-- Payment -->
      <div class="card shadow-sm border-0 mb-3">
        <div class="card-header bg-light border-0 py-3">
          <h6 class="fw-bold mb-0"><i class="bi bi-cash-coin me-2"></i>Payment</h6>
        </div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-md-3">
              <label class="form-label small fw-bold">Method</label>
              <select id="soPayMethod" class="form-select">
                <option value="Cash">Cash</option>
                <option value="GCash">GCash</option>
                <option value="Bank Transfer">Bank Transfer</option>
                <option value="Cheque">Cheque</option>
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Status</label>
              <select id="soPayStatus" class="form-select" onchange="soToggleDueDate()">
                <option value="Unpaid" selected>Unpaid</option>
                <option value="Partial">Partial</option>
                <option value="Paid">Paid</option>
              </select>
            </div>
            <div class="col-md-3">
              <label class="form-label small fw-bold">Amount Paid (₱)</label>
              <input type="number" step="0.01" id="soAmountPaid" class="form-control" placeholder="0.00" min="0" value="0">
            </div>
            <div class="col-md-3" id="soDueDateGroup">
              <label class="form-label small fw-bold">Due Date</label>
              <input type="date" id="soDueDate" class="form-control">
            </div>
          </div>
        </div>
      </div>

      <!-- Notes -->
      <div class="card shadow-sm border-0 mb-4">
        <div class="card-body">
          <label class="form-label small fw-bold">Remarks / Notes</label>
          <textarea id="soNotes" class="form-control" rows="2" placeholder="Delivery instructions, special requests…"></textarea>
        </div>
      </div>

      <div class="d-flex gap-2">
        <button class="btn btn-primary btn-lg fw-bold flex-grow-1" onclick="submitNewSOForm()">
          <i class="bi bi-floppy me-2"></i>Save Order
        </button>
        <button class="btn btn-outline-secondary btn-lg" onclick="navigate('#/sales/orders')">Cancel</button>
      </div>
    </div>
  `);

  soAddRow(); // start with one empty row
  soToggleDueDate();
}

function soFillCustomer(sel) {
  const opt = sel.options[sel.selectedIndex];
  document.getElementById('soCustName').value    = opt.dataset.name    || '';
  document.getElementById('soCustAddress').value = opt.dataset.address || '';
  document.getElementById('soCustContact').value = opt.dataset.contact || '';

  _soCustomerPrices = {};
  document.getElementById('soPriceBadge').classList.add('d-none');

  const pgId = opt.dataset.pgid ? parseInt(opt.dataset.pgid, 10) : null;
  if (!pgId) return;

  api.GetPriceGroup(pgId).then(pg => {
    if (!pg || !pg.items) return;
    pg.items.forEach(item => {
      _soCustomerPrices[`${item.egg_size}|${item.unit}`] = item.price;
    });
    if (Object.keys(_soCustomerPrices).length > 0) {
      document.getElementById('soPriceBadge').classList.remove('d-none');
      // Apply prices to existing rows
      document.querySelectorAll('#soItemsBody tr').forEach(row => soApplyPriceToRow(row));
    }
  });
}

function soAddRow() {
  const tbody = document.getElementById('soItemsBody');
  const sizeOptions = SO_EGG_SIZES.map(s => `<option value="${s}">${s}</option>`).join('');
  const tr = document.createElement('tr');
  tr.innerHTML = `
    <td class="ps-3">
      <select class="form-select form-select-sm so-sku" required
              onchange="soUpdateRow(this); soApplyPriceToRow(this.closest('tr'));">
        <option value="">— Select size —</option>
        ${sizeOptions}
      </select>
    </td>
    <td>
      <select class="form-select form-select-sm so-unit"
              onchange="soUpdateRow(this); soApplyPriceToRow(this.closest('tr'));">
        <option value="Tray">Tray (30 pcs)</option>
        <option value="Piece">Piece</option>
      </select>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-center so-qty"
             placeholder="0" min="0.5" step="0.5" required oninput="soUpdateRow(this)">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm text-center so-price"
             placeholder="0.00" min="0" step="0.01" required oninput="soUpdateRow(this)">
    </td>
    <td class="fw-bold so-line-total text-primary">₱0.00</td>
    <td>
      <button type="button" class="btn btn-sm btn-outline-danger py-0"
              onclick="this.closest('tr').remove(); soUpdateGrandTotal();">
        <i class="bi bi-x"></i>
      </button>
    </td>`;
  tbody.appendChild(tr);
  soApplyPriceToRow(tr);
}

function soApplyPriceToRow(row) {
  const sku   = row.querySelector('.so-sku')?.value;
  const unit  = row.querySelector('.so-unit')?.value;
  const priceEl = row.querySelector('.so-price');
  if (!sku || !unit || !priceEl) return;
  const key = `${sku}|${unit}`;
  if (_soCustomerPrices[key] !== undefined) {
    priceEl.value = Number(_soCustomerPrices[key]).toFixed(2);
    priceEl.style.background = '#f0fdf4';
    soUpdateRow(priceEl);
  } else {
    priceEl.style.background = '';
  }
}

function soUpdateRow(el) {
  const row   = el.closest('tr');
  const qty   = parseFloat(row.querySelector('.so-qty')?.value)   || 0;
  const price = parseFloat(row.querySelector('.so-price')?.value) || 0;
  row.querySelector('.so-line-total').textContent = formatCurrency(qty * price);
  soUpdateGrandTotal();
}

function soUpdateGrandTotal() {
  let grand = 0;
  document.querySelectorAll('#soItemsBody tr').forEach(row => {
    const qty   = parseFloat(row.querySelector('.so-qty')?.value)   || 0;
    const price = parseFloat(row.querySelector('.so-price')?.value) || 0;
    grand += qty * price;
  });
  document.getElementById('soGrandTotal').textContent = formatCurrency(grand);
}

function soToggleDueDate() {
  const status = document.getElementById('soPayStatus').value;
  const show   = status === 'Unpaid' || status === 'Partial';
  document.getElementById('soDueDateGroup').style.display = show ? '' : 'none';
}

async function submitNewSOForm() {
  const customerIdRaw = document.getElementById('soCustomerSelect').value;
  const custName      = document.getElementById('soCustName').value.trim();
  if (!custName) { toast('Customer name is required.', 'warning'); return; }

  // Collect items
  const items = [];
  let valid = true;
  document.querySelectorAll('#soItemsBody tr').forEach(row => {
    const sku   = row.querySelector('.so-sku')?.value;
    const unit  = row.querySelector('.so-unit')?.value;
    const qty   = parseFloat(row.querySelector('.so-qty')?.value)   || 0;
    const price = parseFloat(row.querySelector('.so-price')?.value) || 0;
    if (!sku) { valid = false; return; }
    items.push({ sku, unit, quantity: qty, price_per_unit: price, line_total: qty * price });
  });
  if (!valid) { toast('Please select an egg size for all rows.', 'warning'); return; }
  if (items.length === 0) { toast('Add at least one item.', 'warning'); return; }

  const dueDateVal = document.getElementById('soDueDate').value;
  const payStatus  = document.getElementById('soPayStatus').value;

  const payload = {
    date:             new Date(document.getElementById('soDate').value).toISOString(),
    customer_id:      customerIdRaw ? parseInt(customerIdRaw, 10) : null,
    customer_name:    custName,
    customer_addr:    document.getElementById('soCustAddress').value.trim(),
    customer_contact: document.getElementById('soCustContact').value.trim(),
    payment_method:   document.getElementById('soPayMethod').value,
    terms:            document.getElementById('soTerms').value,
    due_date:         (payStatus !== 'Paid' && dueDateVal) ? dueDateVal : null,
    notes:            document.getElementById('soNotes').value.trim(),
    items,
  };

  const btn = document.querySelector('[onclick="submitNewSOForm()"]');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Saving…';

  const result = await api.CreateSalesOrder(payload);
  if (result) {
    toast('Sales order created.', 'success');
    navigate('#/sales/orders/' + result.id);
  } else {
    toast('Failed to create sales order.', 'danger');
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-floppy me-2"></i>Save Order';
  }
}

// ── Layout Helper ─────────────────────────────────────────────────────────────

function wrapSales(tabBar, content) {
  return `
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-bag3 me-2"></i>Sales</h4>
      ${tabBar}
      ${content}
    </div>
  `;
}
