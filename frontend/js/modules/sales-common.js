// sales-common.js

// ── Status Badge ──────────────────────────────────────────────────────────────

function salesStatusBadge(status) {
  const map = {
    Draft:      'secondary',
    Open:       'primary',
    Pending:    'warning',
    Confirmed:  'primary',
    Delivered:  'success',
    Invoiced:   'info',
    Paid:       'success',
    Partial:    'warning',
    Unpaid:     'danger',
    Closed:     'info',
    Cancelled:  'dark',
    Overdue:    'danger',
    Void:       'dark',
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
