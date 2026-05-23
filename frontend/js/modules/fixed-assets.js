// fixed-assets.js — Fixed Asset Management (PAS 16)

// ── Helpers ───────────────────────────────────────────────────────────────────

function faStatusBadge(status) {
  switch (status) {
    case 'ACTIVE':             return `<span class="badge bg-success">Active</span>`;
    case 'FULLY_DEPRECIATED':  return `<span class="badge bg-secondary">Fully Depreciated</span>`;
    case 'DISPOSED':           return `<span class="badge bg-dark">Disposed</span>`;
    default:                   return `<span class="badge bg-light text-dark">${status || '—'}</span>`;
  }
}

function faDepMethodLabel(m) {
  return m === 'DECLINING_BALANCE' ? 'Double Declining Balance' : 'Straight-Line';
}

function faUsefulLife(months) {
  const yrs = Math.floor(months / 12);
  const mo  = months % 12;
  if (mo === 0) return `${yrs} yr${yrs !== 1 ? 's' : ''}`;
  return yrs > 0 ? `${yrs} yr${yrs !== 1 ? 's' : ''} ${mo} mo` : `${mo} mo`;
}

function faPctDepreciated(fa) {
  if (!fa.acquisition_cost) return 0;
  return Math.min(100, Math.round(fa.accum_depreciation / fa.acquisition_cost * 1000) / 10);
}

let _faGLAccounts = null;
async function faGetGLAccounts() {
  if (!_faGLAccounts) _faGLAccounts = await api.ListGLAccounts();
  return (_faGLAccounts || []).filter(a => a.account_type === 'POSTING' && a.is_active);
}

function faGLSelect(name, value, label) {
  // rendered after accounts are loaded; patched by faPopulateGLSelect
  return `<select class="form-select form-select-sm" name="${name}" id="${name}" required>
    <option value="">Loading accounts…</option>
  </select>`;
}

async function faPopulateGLSelect(id, selectedID) {
  const accounts = await faGetGLAccounts();
  const el = document.getElementById(id);
  if (!el) return;
  el.innerHTML = `<option value="">— Select account —</option>` +
    accounts.map(a => `<option value="${a.id}" ${a.id == selectedID ? 'selected' : ''}>${a.code}  ${a.name}</option>`).join('');
}

// ── Module ────────────────────────────────────────────────────────────────────

Modules.FixedAssets = {
  async load(sub, id, action) {
    const active = sub || 'list';
    showLoading();

    if (active === 'list' && action === 'new') { await faLoadNewForm(); return; }
    if (active === 'list' && id)               { await faLoadDetail(id); return; }
    if (active === 'run-depreciation')          { await faLoadRunDepreciation(); return; }

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'list' ? 'active' : ''}" href="#"
            onclick="navigate('#/fixed-assets/list');return false;">
            <i class="bi bi-building me-1"></i>Assets
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'run-depreciation' ? 'active' : ''}" href="#"
            onclick="navigate('#/fixed-assets/run-depreciation');return false;">
            <i class="bi bi-calendar-check me-1"></i>Run Depreciation
          </a>
        </li>
      </ul>`;

    await faLoadList(tabBar);
  },

  reset() {
    _faGLAccounts = null;
  }
};

// ── Asset List ────────────────────────────────────────────────────────────────

async function faLoadList(tabBar) {
  const assets = await api.ListFixedAssets();
  if (!assets) { showView(`<div class="alert alert-warning">Failed to load assets.</div>`); return; }

  const rows = assets.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No fixed assets recorded.</td></tr>`
    : assets.map(fa => {
        const bv = (fa.acquisition_cost - fa.accum_depreciation).toFixed(2);
        const pct = faPctDepreciated(fa);
        return `
          <tr style="cursor:pointer" onclick="navigate('#/fixed-assets/list/${fa.id}')">
            <td class="fw-semibold">${fa.asset_code}</td>
            <td>${fa.asset_name}</td>
            <td><span class="badge bg-light text-dark border">${fa.category}</span></td>
            <td class="text-end">${formatCurrency(fa.acquisition_cost)}</td>
            <td class="text-end">${formatCurrency(fa.accum_depreciation)}</td>
            <td class="text-end fw-semibold">${formatCurrency(parseFloat(bv))}</td>
            <td>
              <div class="progress" style="height:6px;min-width:60px">
                <div class="progress-bar ${pct >= 100 ? 'bg-secondary' : 'bg-warning'}" style="width:${pct}%"></div>
              </div>
              <small class="text-muted">${pct}%</small>
            </td>
            <td>${faStatusBadge(fa.status)}</td>
          </tr>`;
      }).join('');

  showView(`<div class="container-fluid p-4">${tabBar}
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small">${assets.length} asset(s)</span>
      <button class="btn btn-primary btn-sm" onclick="navigate('#/fixed-assets/list/new')">
        <i class="bi bi-plus-lg me-1"></i>Add Asset
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr>
              <th>Code</th><th>Name</th><th>Category</th>
              <th class="text-end">Cost</th>
              <th class="text-end">Accum. Dep.</th>
              <th class="text-end">Book Value</th>
              <th style="min-width:100px">Depreciated</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  </div>`);
}

// ── New Asset Form ────────────────────────────────────────────────────────────

async function faLoadNewForm() {
  const today = new Date().toISOString().slice(0, 10);
  showView(`<div class="container-fluid p-4">
    <div class="d-flex align-items-center mb-4">
      <button class="btn btn-sm btn-outline-secondary me-3" onclick="navigate('#/fixed-assets/list')">
        <i class="bi bi-arrow-left"></i>
      </button>
      <h5 class="mb-0">New Fixed Asset</h5>
    </div>
    <div class="card border-0 shadow-sm p-4" style="max-width:720px">
      <form id="fa-form">
        <div class="row g-3">
          <div class="col-md-4">
            <label class="form-label fw-semibold">Asset Code <span class="text-danger">*</span></label>
            <input type="text" class="form-control form-control-sm" name="asset_code" required placeholder="e.g. FA-001">
          </div>
          <div class="col-md-8">
            <label class="form-label fw-semibold">Asset Name <span class="text-danger">*</span></label>
            <input type="text" class="form-control form-control-sm" name="asset_name" required placeholder="e.g. Rice Huller Machine">
          </div>
          <div class="col-md-6">
            <label class="form-label fw-semibold">Category <span class="text-danger">*</span></label>
            <select class="form-select form-select-sm" name="category" required>
              <option value="">— Select —</option>
              <option>Land</option>
              <option>Building</option>
              <option>Equipment</option>
              <option>Vehicle</option>
              <option>Office Equipment</option>
            </select>
          </div>
          <div class="col-md-6">
            <label class="form-label fw-semibold">Acquisition Date <span class="text-danger">*</span></label>
            <input type="date" class="form-control form-control-sm" name="acquisition_date" value="${today}" required>
          </div>
          <div class="col-md-4">
            <label class="form-label fw-semibold">Acquisition Cost (₱) <span class="text-danger">*</span></label>
            <input type="number" class="form-control form-control-sm" name="acquisition_cost" min="0" step="0.01" required>
          </div>
          <div class="col-md-4">
            <label class="form-label fw-semibold">Residual Value (₱)</label>
            <input type="number" class="form-control form-control-sm" name="residual_value" min="0" step="0.01" value="0">
          </div>
          <div class="col-md-4">
            <label class="form-label fw-semibold">Useful Life (months) <span class="text-danger">*</span></label>
            <input type="number" class="form-control form-control-sm" name="useful_life_months" min="1" required placeholder="e.g. 60">
          </div>
          <div class="col-md-6">
            <label class="form-label fw-semibold">Depreciation Method <span class="text-danger">*</span></label>
            <select class="form-select form-select-sm" name="depreciation_method">
              <option value="STRAIGHT_LINE" selected>Straight-Line</option>
              <option value="DECLINING_BALANCE">Double Declining Balance</option>
            </select>
          </div>
          <div class="col-12"><hr class="my-1"><p class="text-muted small mb-2">GL Account Mapping</p></div>
          <div class="col-md-4">
            <label class="form-label fw-semibold">Asset Account <span class="text-danger">*</span></label>
            <select class="form-select form-select-sm" id="gl_asset_account_id" name="gl_asset_account_id" required>
              <option value="">Loading…</option>
            </select>
          </div>
          <div class="col-md-4">
            <label class="form-label fw-semibold">Accum. Depreciation Account <span class="text-danger">*</span></label>
            <select class="form-select form-select-sm" id="gl_accum_dep_account_id" name="gl_accum_dep_account_id" required>
              <option value="">Loading…</option>
            </select>
          </div>
          <div class="col-md-4">
            <label class="form-label fw-semibold">Dep. Expense Account <span class="text-danger">*</span></label>
            <select class="form-select form-select-sm" id="gl_dep_exp_account_id" name="gl_dep_exp_account_id" required>
              <option value="">Loading…</option>
            </select>
          </div>
          <div class="col-12">
            <label class="form-label fw-semibold">Notes</label>
            <textarea class="form-control form-control-sm" name="notes" rows="2"></textarea>
          </div>
        </div>
        <div class="mt-4 d-flex gap-2">
          <button type="submit" class="btn btn-primary btn-sm">Save Asset</button>
          <button type="button" class="btn btn-outline-secondary btn-sm" onclick="navigate('#/fixed-assets/list')">Cancel</button>
        </div>
      </form>
    </div>
  </div>`);

  // Populate GL account selects with smart defaults
  const accounts = await faGetGLAccounts();
  const defAsset   = accounts.find(a => a.code === '1-2130');
  const defAccum   = accounts.find(a => a.code === '1-2190');
  const defExp     = accounts.find(a => a.code === '5-7000');
  await Promise.all([
    faPopulateGLSelect('gl_asset_account_id',    defAsset?.id),
    faPopulateGLSelect('gl_accum_dep_account_id', defAccum?.id),
    faPopulateGLSelect('gl_dep_exp_account_id',   defExp?.id),
  ]);

  document.getElementById('fa-form').addEventListener('submit', async e => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const payload = {
      asset_code:            fd.get('asset_code'),
      asset_name:            fd.get('asset_name'),
      category:              fd.get('category'),
      description:           '',
      acquisition_date:      fd.get('acquisition_date'),
      acquisition_cost:      parseFloat(fd.get('acquisition_cost')) || 0,
      residual_value:        parseFloat(fd.get('residual_value')) || 0,
      useful_life_months:    parseInt(fd.get('useful_life_months'), 10) || 0,
      depreciation_method:   fd.get('depreciation_method'),
      gl_asset_account_id:   parseInt(fd.get('gl_asset_account_id'), 10) || 0,
      gl_accum_dep_account_id: parseInt(fd.get('gl_accum_dep_account_id'), 10) || 0,
      gl_dep_exp_account_id: parseInt(fd.get('gl_dep_exp_account_id'), 10) || 0,
      notes:                 fd.get('notes'),
    };
    const fa = await api.CreateFixedAsset(payload);
    if (fa) navigate(`#/fixed-assets/list/${fa.id}`);
  });
}

// ── Asset Detail ──────────────────────────────────────────────────────────────

async function faLoadDetail(id) {
  const [fa, schedule] = await Promise.all([
    api.GetFixedAsset(id),
    api.GetDepreciationSchedule(id),
  ]);
  if (!fa) { showView(`<div class="alert alert-warning">Asset not found.</div>`); return; }

  const bv = fa.acquisition_cost - fa.accum_depreciation;
  const pct = faPctDepreciated(fa);
  const scheduleRows = (schedule || []).map((row, i) => `
    <tr class="${row.is_posted ? 'table-success' : ''}">
      <td>${i + 1}</td>
      <td>${row.period}</td>
      <td class="text-end">${formatCurrency(row.monthly_amount)}</td>
      <td class="text-end">${formatCurrency(row.accum_dep)}</td>
      <td class="text-end">${formatCurrency(row.book_value)}</td>
      <td class="text-center">
        ${row.is_posted
          ? `<span class="badge bg-success"><i class="bi bi-check-lg"></i> Posted</span>`
          : `<span class="badge bg-light text-muted border">Pending</span>`}
      </td>
    </tr>`).join('');

  const isActive = fa.status === 'ACTIVE';
  const disposeBtn = isActive
    ? `<button class="btn btn-sm btn-outline-danger" onclick="faShowDisposeModal(${fa.id})">
         <i class="bi bi-trash me-1"></i>Dispose Asset
       </button>`
    : '';

  showView(`<div class="container-fluid p-4">
    <div class="d-flex align-items-center justify-content-between mb-4">
      <div class="d-flex align-items-center">
        <button class="btn btn-sm btn-outline-secondary me-3" onclick="navigate('#/fixed-assets/list')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div>
          <h5 class="mb-0">${fa.asset_code} — ${fa.asset_name}</h5>
          <small class="text-muted">${fa.category} · ${faDepMethodLabel(fa.depreciation_method)} · ${faUsefulLife(fa.useful_life_months)}</small>
        </div>
      </div>
      <div class="d-flex gap-2 align-items-center">
        ${faStatusBadge(fa.status)}
        ${disposeBtn}
      </div>
    </div>

    <!-- Summary cards -->
    <div class="row g-3 mb-4">
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center p-3">
          <div class="text-muted small">Acquisition Cost</div>
          <div class="fw-bold fs-5">${formatCurrency(fa.acquisition_cost)}</div>
          <div class="text-muted small">${formatDate(fa.acquisition_date)}</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center p-3">
          <div class="text-muted small">Accum. Depreciation</div>
          <div class="fw-bold fs-5 text-danger">${formatCurrency(fa.accum_depreciation)}</div>
          <div class="text-muted small">${pct}% depreciated</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center p-3">
          <div class="text-muted small">Book Value</div>
          <div class="fw-bold fs-5 text-success">${formatCurrency(bv)}</div>
          <div class="text-muted small">Residual: ${formatCurrency(fa.residual_value)}</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center p-3">
          <div class="text-muted small">Progress</div>
          <div class="progress mt-2 mb-1" style="height:10px">
            <div class="progress-bar ${pct >= 100 ? 'bg-secondary' : 'bg-warning'}" style="width:${pct}%"></div>
          </div>
          <div class="text-muted small">${(schedule || []).filter(r => r.is_posted).length} / ${(schedule || []).length} periods posted</div>
        </div>
      </div>
    </div>

    <!-- GL Accounts -->
    <div class="card border-0 shadow-sm mb-4 p-3">
      <h6 class="mb-3">GL Account Mapping</h6>
      <div class="row g-2 small">
        <div class="col-md-4"><span class="text-muted">Asset Account:</span> ${fa.asset_account ? fa.asset_account.code + '  ' + fa.asset_account.name : '—'}</div>
        <div class="col-md-4"><span class="text-muted">Accum. Dep. Account:</span> ${fa.accum_dep_account ? fa.accum_dep_account.code + '  ' + fa.accum_dep_account.name : '—'}</div>
        <div class="col-md-4"><span class="text-muted">Dep. Expense Account:</span> ${fa.dep_exp_account ? fa.dep_exp_account.code + '  ' + fa.dep_exp_account.name : '—'}</div>
      </div>
    </div>

    <!-- Depreciation Schedule -->
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-white d-flex align-items-center justify-content-between">
        <h6 class="mb-0">Depreciation Schedule</h6>
        ${isActive ? `<button class="btn btn-sm btn-outline-primary" onclick="navigate('#/fixed-assets/run-depreciation')">
          <i class="bi bi-play-circle me-1"></i>Run Depreciation
        </button>` : ''}
      </div>
      <div class="table-responsive" style="max-height:400px;overflow-y:auto">
        <table class="table table-sm table-hover align-middle mb-0">
          <thead class="table-light sticky-top">
            <tr>
              <th>#</th><th>Period</th>
              <th class="text-end">Monthly Dep.</th>
              <th class="text-end">Accum. Dep.</th>
              <th class="text-end">Book Value</th>
              <th class="text-center">Status</th>
            </tr>
          </thead>
          <tbody>${scheduleRows || '<tr><td colspan="6" class="text-center text-muted py-3">No schedule available.</td></tr>'}</tbody>
        </table>
      </div>
    </div>

    <!-- Disposal info (if disposed) -->
    ${fa.status === 'DISPOSED' ? `
    <div class="card border-0 shadow-sm mt-4 p-3 border-start border-4 border-danger">
      <h6 class="text-danger mb-2"><i class="bi bi-trash me-1"></i>Asset Disposed</h6>
      <div class="row g-2 small">
        <div class="col-md-4"><span class="text-muted">Disposal Date:</span> ${formatDate(fa.disposal_date)}</div>
        <div class="col-md-4"><span class="text-muted">Proceeds:</span> ${formatCurrency(fa.disposal_proceeds || 0)}</div>
        <div class="col-md-4"><span class="text-muted">Book Value at Disposal:</span> ${formatCurrency(fa.acquisition_cost - fa.accum_depreciation)}</div>
      </div>
    </div>` : ''}

    <!-- Dispose Modal -->
    <div class="modal fade" id="disposeModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Dispose Asset</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="dispose-form">
              <div class="mb-3">
                <label class="form-label fw-semibold">Disposal Date <span class="text-danger">*</span></label>
                <input type="date" class="form-control form-control-sm" name="disposal_date" value="${new Date().toISOString().slice(0,10)}" required>
              </div>
              <div class="mb-3">
                <label class="form-label fw-semibold">Proceeds Received (₱)</label>
                <input type="number" class="form-control form-control-sm" name="proceeds" min="0" step="0.01" value="0">
              </div>
              <div class="mb-3">
                <label class="form-label fw-semibold">Proceeds GL Account <span class="text-danger">*</span></label>
                <select class="form-select form-select-sm" id="dispose_proceeds_acct" name="proceeds_account_id" required>
                  <option value="">Loading…</option>
                </select>
              </div>
              <div class="mb-3">
                <label class="form-label fw-semibold">Gain / Loss GL Account <span class="text-danger">*</span></label>
                <select class="form-select form-select-sm" id="dispose_gainloss_acct" name="gain_loss_account_id" required>
                  <option value="">Loading…</option>
                </select>
                <div class="form-text">Positive difference posts as a credit (gain); negative as a debit (loss).</div>
              </div>
            </form>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-outline-secondary btn-sm" data-bs-dismiss="modal">Cancel</button>
            <button type="button" class="btn btn-danger btn-sm" onclick="faSubmitDisposal(${fa.id})">Confirm Disposal</button>
          </div>
        </div>
      </div>
    </div>
  </div>`);
}

async function faShowDisposeModal(assetId) {
  const accounts = await faGetGLAccounts();
  const defProceeds  = accounts.find(a => a.code === '1-1120'); // Bank Account
  const defGainLoss  = accounts.find(a => a.code === '7-1000'); // Other Income
  await Promise.all([
    faPopulateGLSelect('dispose_proceeds_acct',  defProceeds?.id),
    faPopulateGLSelect('dispose_gainloss_acct',  defGainLoss?.id),
  ]);
  new bootstrap.Modal(document.getElementById('disposeModal')).show();
}

async function faSubmitDisposal(assetId) {
  const form = document.getElementById('dispose-form');
  const fd   = new FormData(form);
  const payload = {
    asset_id:            assetId,
    disposal_date:       fd.get('disposal_date'),
    proceeds:            parseFloat(fd.get('proceeds')) || 0,
    proceeds_account_id: parseInt(fd.get('proceeds_account_id'), 10) || 0,
    gain_loss_account_id: parseInt(fd.get('gain_loss_account_id'), 10) || 0,
    notes:               '',
  };
  const result = await api.DisposeAsset(payload);
  if (result) {
    bootstrap.Modal.getInstance(document.getElementById('disposeModal'))?.hide();
    navigate(`#/fixed-assets/list/${assetId}`);
  }
}

// ── Run Depreciation ──────────────────────────────────────────────────────────

async function faLoadRunDepreciation() {
  const now   = new Date();
  const month = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;

  showView(`<div class="container-fluid p-4" style="max-width:640px">
    <div class="d-flex align-items-center mb-4">
      <button class="btn btn-sm btn-outline-secondary me-3" onclick="navigate('#/fixed-assets/list')">
        <i class="bi bi-arrow-left"></i>
      </button>
      <h5 class="mb-0">Run Monthly Depreciation</h5>
    </div>
    <div class="card border-0 shadow-sm p-4">
      <p class="text-muted small mb-3">
        Posts a depreciation journal entry for every active asset that has not yet been depreciated for the selected period.
      </p>
      <div class="mb-4">
        <label class="form-label fw-semibold">Period (Month) <span class="text-danger">*</span></label>
        <input type="month" class="form-control form-control-sm" id="dep-period" value="${month}">
      </div>
      <button class="btn btn-primary btn-sm" onclick="faRunDepreciation()">
        <i class="bi bi-play-circle me-1"></i>Run Depreciation
      </button>
    </div>
    <div id="dep-result" class="mt-4"></div>
  </div>`);
}

async function faRunDepreciation() {
  const period = document.getElementById('dep-period').value;
  if (!period) { toast('Select a period first.', 'warning'); return; }

  document.getElementById('dep-result').innerHTML =
    `<div class="text-center py-3"><div class="spinner-border text-warning"></div></div>`;

  const result = await api.RunDepreciation({ period_date: period });
  if (!result) { document.getElementById('dep-result').innerHTML = ''; return; }

  const rows = (result.details || []).map(d => `
    <tr>
      <td class="fw-semibold">${d.asset_code}</td>
      <td>${d.asset_name}</td>
      <td class="text-end">${d.amount ? formatCurrency(d.amount) : '—'}</td>
      <td>
        ${d.status === 'posted'  ? `<span class="badge bg-success">Posted</span>` : ''}
        ${d.status === 'skipped' ? `<span class="badge bg-secondary">Skipped</span>` : ''}
        ${d.status === 'error'   ? `<span class="badge bg-danger">Error</span>` : ''}
        ${d.message ? `<small class="text-muted ms-1">${d.message}</small>` : ''}
      </td>
    </tr>`).join('');

  document.getElementById('dep-result').innerHTML = `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-white">
        <strong>Results — ${result.period}</strong>
        <span class="ms-3 badge bg-success">${result.posted} posted</span>
        <span class="ms-1 badge bg-secondary">${result.skipped} skipped</span>
        <span class="ms-1 fw-semibold">Total: ${formatCurrency(result.total)}</span>
      </div>
      <div class="table-responsive">
        <table class="table table-sm align-middle mb-0">
          <thead class="table-light">
            <tr><th>Code</th><th>Asset</th><th class="text-end">Amount</th><th>Result</th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>`;
}
