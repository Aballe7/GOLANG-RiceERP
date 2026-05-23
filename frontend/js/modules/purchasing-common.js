// purchasing-common.js

// ── Status Badge Helper ───────────────────────────────────────────────────────

function purchStatusBadge(status) {
  // status now represents delivery state of purchase orders
  const map = {
    Draft:     'secondary',
    Pending:   'warning',
    Partial:   'warning',
    Delivered: 'success',
    Cancelled: 'danger',
    Approved:  'primary',
  };
  const color = map[status] || 'secondary';
  return `<span class="badge bg-${color}">${status || '—'}</span>`;
}

// ── Lookup Helper ─────────────────────────────────────────────────────────────

// loadLookupOptions populates a <select> element with values from purchasing_lookup.
// category: 'supplier_type' | 'payment_terms' | 'payment_method'
// selectedValue: the value to pre-select (optional)
async function loadLookupOptions(selectId, category, selectedValue) {
  const res = await api.ListPurchasingLookup(category);
  const items = res || [];
  const el = document.getElementById(selectId);
  if (!el) return;
  el.innerHTML = items.map(r =>
    `<option value="${r.value}"${r.value === selectedValue ? ' selected' : ''}>${r.value}</option>`
  ).join('');
}

// buildLookupSelectHtml returns an <option> string synchronously from a pre-fetched list.
function buildLookupOptions(items, selectedValue) {
  return items.map(r =>
    `<option value="${r.value}"${r.value === selectedValue ? ' selected' : ''}>${r.value}</option>`
  ).join('');
}

// ── Purchasing Filter Bar ────────────────────────────────────────────────────
// Mirrors sales-common _filterBar but uses "Supplier" label.
// Reuses the same sf-* element IDs so _applyFilters() from sales-common works.

function _purchFilterBar(statusOptions, defaultStatus, onFilter, datalists = {}) {
  const docOpts  = _uniq(datalists.doc  || []).map(v => `<option value="${_esc(v)}">`).join('');
  const suppOpts = _uniq(datalists.supp || []).map(v => `<option value="${_esc(v)}">`).join('');
  const hasStatus = statusOptions && statusOptions.length > 0;
  return `
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-body py-2">
        <div class="row g-2 align-items-end">
          <div class="col-sm-3">
            <label class="form-label small mb-1">Document #</label>
            <input id="sf-doc" class="form-control form-control-sm" placeholder="Search…"
              oninput="${onFilter}" list="sf-doc-list" autocomplete="off">
            <datalist id="sf-doc-list">${docOpts}</datalist>
          </div>
          <div class="col-sm-3">
            <label class="form-label small mb-1">Supplier</label>
            <input id="sf-cust" class="form-control form-control-sm" placeholder="Search…"
              oninput="${onFilter}" list="sf-cust-list" autocomplete="off">
            <datalist id="sf-cust-list">${suppOpts}</datalist>
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Date From</label>
            <input id="sf-date-from" type="date" class="form-control form-control-sm" onchange="${onFilter}">
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Date To</label>
            <input id="sf-date-to" type="date" class="form-control form-control-sm" onchange="${onFilter}">
          </div>
          ${hasStatus ? `
          <div class="col-sm-2">
            <label class="form-label small mb-1">Status</label>
            <select id="sf-status" class="form-select form-select-sm" onchange="${onFilter}">
              ${statusOptions.map(s => `<option value="${s.v}"${s.v === defaultStatus ? ' selected' : ''}>${s.l}</option>`).join('')}
            </select>
          </div>` : ''}
          <div class="col-sm-2">
            <label class="form-label small mb-1">Min Amount</label>
            <input id="sf-amt-min" type="number" min="0" class="form-control form-control-sm"
              placeholder="0" oninput="${onFilter}">
          </div>
          <div class="col-sm-2">
            <label class="form-label small mb-1">Max Amount</label>
            <input id="sf-amt-max" type="number" min="0" class="form-control form-control-sm"
              placeholder="Any" oninput="${onFilter}">
          </div>
          <div class="col-sm-2 d-flex align-items-end">
            <button class="btn btn-outline-secondary btn-sm w-100" onclick="purchClearAll()">
              <i class="bi bi-x-circle me-1"></i>Clear
            </button>
          </div>
        </div>
      </div>
    </div>`;
}

function purchClearAll() {
  ['sf-doc', 'sf-cust', 'sf-date-from', 'sf-date-to', 'sf-amt-min', 'sf-amt-max'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.value = '';
  });
  ['poFilter', 'drFilter', 'apInvFilter', 'apPayFilter'].forEach(fn => {
    if (window[fn]) { try { window[fn](); } catch (e) {} }
  });
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
