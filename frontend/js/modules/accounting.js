// accounting.js — Modules.Accounting

Modules.Accounting = {
  async load(sub, id) {
    const active = sub || 'accounts';
    showLoading();

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'accounts' ? 'active' : ''}" href="#"
            onclick="navigate('#/accounting/accounts');return false;">
            <i class="bi bi-diagram-3 me-1"></i>GL Accounts
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'determination' ? 'active' : ''}" href="#"
            onclick="navigate('#/accounting/determination');return false;">
            <i class="bi bi-sliders me-1"></i>Account Determination
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'payment-methods' ? 'active' : ''}" href="#"
            onclick="navigate('#/accounting/payment-methods');return false;">
            <i class="bi bi-credit-card me-1"></i>Payment Methods
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'journal-entries' ? 'active' : ''}" href="#"
            onclick="navigate('#/accounting/journal-entries');return false;">
            <i class="bi bi-journal-text me-1"></i>Journal Entries
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'new-je' ? 'active' : ''}" href="#"
            onclick="navigate('#/accounting/new-je');return false;">
            <i class="bi bi-plus-square me-1"></i>New JE
          </a>
        </li>
      </ul>
    `;

    if (active === 'accounts') await loadGLAccounts(tabBar);
    else if (active === 'determination') await loadAccountDetermination(tabBar);
    else if (active === 'payment-methods') await loadPaymentMethods(tabBar);
    else if (active === 'journal-entries') {
      if (id) await loadJournalEntryDetail(id, tabBar);
      else await loadJournalEntriesList(tabBar);
    }
    else if (active === 'new-je') await loadNewJournalEntry(tabBar);
    else await loadGLAccounts(tabBar);
  },

  reset() {
    _glAccounts = []; _detRules = []; _pmList = []; _jeLineCount = 0;
    _jeAccountOptions = null;
  }
};

// ── GL Accounts ───────────────────────────────────────────────────────────────

let _glAccounts = [];

async function loadGLAccounts(tabBar) {
  const accounts = await api.GetGLAccounts();
  _glAccounts = accounts || [];

  function buildTree(parentId, depth) {
    return _glAccounts
      .filter(a => (a.parent_id ?? null) === parentId)
      .map(a => {
        const indent = '&nbsp;'.repeat(depth * 4);
        const children = buildTree(a.id, depth + 1);
        return `
          <tr>
            <td>${indent}<strong class="${depth === 0 ? '' : 'fw-normal'}">${a.code}</strong></td>
            <td>${indent}${a.name}</td>
            <td>${a.account_type || '—'}</td>
            <td>${a.normal_balance || '—'}</td>
            <td class="text-end">
              <button class="btn btn-sm btn-outline-secondary" onclick="openAccountModal(${a.id})">
                <i class="bi bi-pencil"></i>
              </button>
            </td>
          </tr>
          ${children}
        `;
      }).join('');
  }

  const treeRows = _glAccounts.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No accounts found.</td></tr>`
    : buildTree(null, 0);

  showView(wrapAccounting(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="openAccountModal(null)">
        <i class="bi bi-plus-lg me-1"></i>Add Account
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-sm table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Code</th><th>Name</th><th>Type</th><th>Normal Balance</th><th></th></tr>
          </thead>
          <tbody>${treeRows}</tbody>
        </table>
      </div>
    </div>

    <!-- Account Modal -->
    <div class="modal fade" id="accountModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="accountModalTitle">Account</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <input type="hidden" id="accountId">
            <div class="row g-3">
              <div class="col-md-6">
                <label class="form-label">Account Code <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="accountCode" required placeholder="e.g. 1000">
              </div>
              <div class="col-md-6">
                <label class="form-label">Account Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="accountName" required>
              </div>
              <div class="col-md-6">
                <label class="form-label">Account Type</label>
                <select class="form-select" id="accountType">
                  <option value="">Select…</option>
                  <option>Asset</option><option>Liability</option><option>Equity</option>
                  <option>Revenue</option><option>Expense</option>
                </select>
              </div>
              <div class="col-md-6">
                <label class="form-label">Normal Balance</label>
                <select class="form-select" id="accountNormalBalance">
                  <option value="">Select…</option>
                  <option>Debit</option><option>Credit</option>
                </select>
              </div>
              <div class="col-12">
                <label class="form-label">Parent Account</label>
                <select class="form-select" id="accountParent">
                  <option value="">— None (top level) —</option>
                  ${_glAccounts.map(a => `<option value="${a.id}">${a.code} — ${a.name}</option>`).join('')}
                </select>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitAccountForm()">Save</button>
          </div>
        </div>
      </div>
    </div>
  `));
}

function openAccountModal(id) {
  const isEdit = id !== null;
  document.getElementById('accountModalTitle').textContent = isEdit ? 'Edit Account' : 'Add Account';
  document.getElementById('accountId').value = id || '';
  if (isEdit) {
    const a = _glAccounts.find(x => x.id === id);
    if (a) {
      document.getElementById('accountCode').value = a.code || '';
      document.getElementById('accountName').value = a.name || '';
      document.getElementById('accountType').value = a.account_type || '';
      document.getElementById('accountNormalBalance').value = a.normal_balance || '';
      document.getElementById('accountParent').value = a.parent_id ?? '';
    }
  } else {
    ['accountCode','accountName'].forEach(id => document.getElementById(id).value = '');
    document.getElementById('accountType').value = '';
    document.getElementById('accountNormalBalance').value = '';
    document.getElementById('accountParent').value = '';
  }
  new bootstrap.Modal(document.getElementById('accountModal')).show();
}

async function submitAccountForm() {
  const id = document.getElementById('accountId').value;
  const parentVal = document.getElementById('accountParent').value;
  const payload = {
    code:           document.getElementById('accountCode').value.trim(),
    name:           document.getElementById('accountName').value.trim(),
    account_type:   document.getElementById('accountType').value,
    normal_balance: document.getElementById('accountNormalBalance').value,
    parent_id:      parentVal ? parseInt(parentVal, 10) : null,
  };
  if (!payload.code || !payload.name) {
    toast('Code and name are required.', 'warning'); return;
  }

  const result = id
    ? await api.UpdateGLAccount(parseInt(id, 10), payload)
    : await api.CreateGLAccount(payload);

  if (result) {
    toast(id ? 'Account updated.' : 'Account created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('accountModal')).hide();
    navigate('#/accounting/accounts');
  } else {
    toast('Failed to save account.', 'danger');
  }
}

// ── Account Determination ─────────────────────────────────────────────────────

let _detRules = [];

// Posting events mirrored from constants.go
const DET_POSTING_EVENTS = {
  SALES: [
    ['AR_RECEIVABLE',      'AR Invoice — Accounts Receivable (Dr)'],
    ['SALES_REVENUE',      'AR Invoice — Sales Revenue (Cr)'],
    ['COGS',               'AR Invoice — Cost of Goods Sold (Dr)'],
    ['INVENTORY_SOLD',     'AR Invoice — Inventory Reduction (Cr)'],
    ['OUTPUT_VAT',         'AR Invoice — Output VAT Payable (Cr)'],
    ['COLLECTION_CLEARING','Collection — AR Clearing / Receivable (Cr)'],
    ['SALES_DISCOUNT',     'Collection — Sales Discount (Dr)'],
  ],
  PURCHASING: [
    ['AP_PAYABLE',              'AP Invoice — Accounts Payable (Cr)'],
    ['GRNI',                    'DR Confirm — Goods Received Not Yet Invoiced (Cr)'],
    ['INVENTORY_RECEIVED',      'DR Confirm — Inventory / Expense (Dr)'],
    ['INPUT_VAT',               'AP Invoice — Input VAT Creditable (Dr)'],
    ['AP_CLEARING',             'AP Payment — AP Clearing / Payable (Dr)'],
    ['PURCHASE_PRICE_VARIANCE', 'AP Invoice — Price Variance (Dr/Cr)'],
  ],
  INVENTORY: [
    ['FEED_CONSUMED',    'Daily Log — Feed Consumed Expense (Dr)'],
    ['INVENTORY_FEED',   'Daily Log — Inventory Feeds Reduction (Cr)'],
    ['MORTALITY_LOSS',   'Daily Log — Mortality Loss Expense (Dr)'],
    ['INVENTORY_BIRDS',  'Daily Log — Inventory Birds Reduction (Cr)'],
    ['STOCK_ADJUSTMENT', 'Manual Stock Adjustment'],
  ],
  BANKING: [
    ['CASH_INFLOW',  'Collection — Cash / Bank Account (Dr)'],
    ['CASH_OUTFLOW', 'AP Payment — Cash / Bank Account (Cr)'],
  ],
  PRODUCTION: [
    ['MILLING_WIP',    'Milling — Work In Progress (WIP) account'],
    ['MILLING_INPUT',  'Milling Start — Input Inventory (Paddy) Credit'],
    ['MILLING_OUTPUT', 'Milling Complete — Output Inventory (Products) Debit'],
  ],
};

const DET_MODULE_STYLE = {
  SALES:      { bg: '#fefce8', icon: 'bi-cart text-warning' },
  PURCHASING: { bg: '#eff6ff', icon: 'bi-truck text-primary' },
  INVENTORY:  { bg: '#f0fdf4', icon: 'bi-boxes text-success' },
  BANKING:    { bg: '#fdf4ff', icon: 'bi-bank' },
  PRODUCTION: { bg: '#fff7ed', icon: 'bi-gear-wide-connected text-warning' },
};

async function loadAccountDetermination(tabBar) {
  const [rules, accounts] = await Promise.all([
    api.ListAccountDeterminations(),
    api.ListGLAccounts(),
  ]);
  _detRules   = rules || [];
  _glAccounts = accounts || [];

  // GL account options — POSTING type only
  const postingAccounts = _glAccounts.filter(a => a.account_type === 'POSTING' && a.is_active);
  const glOptions = postingAccounts.map(a =>
    `<option value="${a.id}">${a.code} — ${a.name}</option>`
  ).join('');

  // Datalist for item categories from existing rules
  const categories = [...new Set(_detRules.map(r => r.item_category).filter(Boolean))];
  const catOptions = categories.map(c => `<option value="${c}">`).join('');

  // Module selector
  const modOptions = Object.keys(DET_POSTING_EVENTS).map(m =>
    `<option value="${m}">${m}</option>`
  ).join('');

  // Module-grouped rule cards
  const moduleCards = Object.entries(DET_POSTING_EVENTS).map(([mod, events]) => {
    const modRules = _detRules.filter(r => r.module === mod);
    const style = DET_MODULE_STYLE[mod] || { bg: '#f8f9fa', icon: 'bi-circle' };

    const rows = events.map(([eventCode, eventLabel]) => {
      const eventRules = modRules.filter(r => r.posting_event === eventCode);
      if (eventRules.length > 0) {
        return eventRules.map(rule => {
          const cat = rule.item_category
            ? `<span class="badge bg-light text-dark border">${rule.item_category}</span>`
            : `<span class="text-muted fst-italic" style="font-size:11px">All categories</span>`;
          const glAcc = rule.gl_account
            ? `<span class="font-monospace text-primary" style="font-size:11px">${rule.gl_account.code}</span>
               <span class="text-muted ms-1" style="font-size:11px">${rule.gl_account.name}</span>`
            : '—';
          return `
            <tr>
              <td class="ps-4 font-monospace fw-semibold text-primary" style="font-size:11px">${rule.posting_event}</td>
              <td class="text-muted" style="font-size:11px">${eventLabel}</td>
              <td>${cat}</td>
              <td class="fw-semibold">${glAcc}</td>
              <td class="text-muted" style="font-size:11px">${rule.notes || '—'}</td>
              <td class="text-end pe-3" style="white-space:nowrap">
                <button class="btn btn-outline-primary py-0 px-2 me-1" style="font-size:10px"
                        onclick="detEditRule(${rule.id})">Edit</button>
                <button class="btn btn-outline-danger py-0 px-2" style="font-size:10px"
                        onclick="detRemoveRule(${rule.id})">Remove</button>
              </td>
            </tr>`;
        }).join('');
      }
      return `
        <tr class="table-warning">
          <td class="ps-4 font-monospace text-muted" style="font-size:11px">${eventCode}</td>
          <td class="text-muted" style="font-size:11px">${eventLabel}</td>
          <td colspan="4">
            <span class="text-danger small">
              <i class="bi bi-exclamation-triangle me-1"></i>Not configured
            </span>
          </td>
        </tr>`;
    }).join('');

    return `
      <div class="card border-0 shadow-sm mb-3">
        <div class="card-header border-bottom py-3 d-flex align-items-center justify-content-between"
             style="background:${style.bg}">
          <span class="fw-bold"><i class="bi ${style.icon} me-2"></i>${mod}</span>
          <span class="badge bg-secondary">${modRules.length} rules</span>
        </div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0 small">
            <thead class="table-light">
              <tr>
                <th class="ps-4">Posting Event</th>
                <th>Description</th>
                <th>Category</th>
                <th>GL Account</th>
                <th>Notes</th>
                <th></th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>`;
  }).join('');

  const unconfiguredCount = Object.values(DET_POSTING_EVENTS)
    .flatMap(events => events)
    .filter(([code]) => !_detRules.some(r => r.posting_event === code))
    .length;

  const setupBanner = unconfiguredCount > 0 ? `
    <div class="alert alert-warning border-0 shadow-sm d-flex align-items-start gap-3 mb-4">
      <i class="bi bi-exclamation-triangle-fill fs-5 mt-1 flex-shrink-0 text-warning"></i>
      <div>
        <div class="fw-bold">Account Determination Setup Required</div>
        <div class="small mt-1">
          <strong>${unconfiguredCount} posting event${unconfiguredCount !== 1 ? 's' : ''}</strong> still need a GL account mapping
          (highlighted in yellow below). Journal entries <strong>cannot be posted</strong> until all required events are configured.
          <br>For each event: select the module, choose the posting event, then assign a GL account.
        </div>
      </div>
    </div>` : `
    <div class="alert alert-success border-0 shadow-sm d-flex align-items-center gap-3 mb-4">
      <i class="bi bi-check-circle-fill fs-5 text-success flex-shrink-0"></i>
      <div class="fw-semibold">All posting events are configured. Journal entries will post correctly.</div>
    </div>`;

  showView(wrapAccounting(tabBar, `
    ${setupBanner}
    <!-- Add / Update Rule -->
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-header bg-white border-bottom py-3">
        <span class="fw-bold"><i class="bi bi-plus-circle me-2 text-success"></i>Add / Update Rule</span>
      </div>
      <div class="card-body">
        <div class="row g-3 align-items-end">
          <div class="col-md-2">
            <label class="form-label small fw-bold">Module <span class="text-danger">*</span></label>
            <select id="detAddModule" class="form-select" required onchange="detFilterEvents(this.value)">
              <option value="">— Select —</option>
              ${modOptions}
            </select>
          </div>
          <div class="col-md-3">
            <label class="form-label small fw-bold">Posting Event <span class="text-danger">*</span></label>
            <select id="detAddEvent" class="form-select" required>
              <option value="">— Select module first —</option>
            </select>
          </div>
          <div class="col-md-2">
            <label class="form-label small fw-bold">
              Item Category <span class="text-muted fw-normal">(blank = all)</span>
            </label>
            <input type="text" id="detAddCategory" class="form-control"
                   list="detCategoryList" placeholder="e.g. Feeds">
            <datalist id="detCategoryList">${catOptions}</datalist>
          </div>
          <div class="col-md-3">
            <label class="form-label small fw-bold">GL Account <span class="text-danger">*</span></label>
            <select id="detAddAccount" class="form-select" required>
              <option value="">— Select GL Account —</option>
              ${glOptions}
            </select>
          </div>
          <div class="col-md-1">
            <label class="form-label small fw-bold">Notes</label>
            <input type="text" id="detAddNotes" class="form-control" placeholder="Optional">
          </div>
          <div class="col-md-1">
            <button class="btn btn-success w-100 fw-bold" onclick="detSaveRule()">Save</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Module Cards -->
    ${moduleCards}
  `));
}

function detFilterEvents(mod) {
  const sel = document.getElementById('detAddEvent');
  sel.innerHTML = '<option value="">— Select event —</option>';
  if (!mod || !DET_POSTING_EVENTS[mod]) return;
  DET_POSTING_EVENTS[mod].forEach(([code, label]) => {
    const opt = document.createElement('option');
    opt.value = code;
    opt.textContent = code + ' — ' + label;
    sel.appendChild(opt);
  });
}

async function detSaveRule() {
  const module       = document.getElementById('detAddModule').value;
  const postingEvent = document.getElementById('detAddEvent').value;
  const categoryRaw  = document.getElementById('detAddCategory').value.trim();
  const glAccountId  = parseInt(document.getElementById('detAddAccount').value, 10);
  const notes        = document.getElementById('detAddNotes').value.trim();

  if (!module)       { toast('Please select a module.', 'warning'); return; }
  if (!postingEvent) { toast('Please select a posting event.', 'warning'); return; }
  if (!glAccountId)  { toast('Please select a GL account.', 'warning'); return; }

  const result = await api.UpsertAccountDetermination({
    module,
    posting_event:  postingEvent,
    item_category:  categoryRaw || null,
    gl_account_id:  glAccountId,
    notes,
  });
  if (result) {
    toast('Rule saved.', 'success');
    navigate('#/accounting/determination');
  } else {
    toast('Failed to save rule.', 'danger');
  }
}

function detEditRule(id) {
  const rule = _detRules.find(r => r.id === id);
  if (!rule) return;

  // Pre-fill module and trigger event dropdown population
  const modSel = document.getElementById('detAddModule');
  modSel.value = rule.module;
  detFilterEvents(rule.module);

  // Set event (detFilterEvents is synchronous so options are ready)
  document.getElementById('detAddEvent').value    = rule.posting_event;
  document.getElementById('detAddCategory').value = rule.item_category || '';
  document.getElementById('detAddNotes').value    = rule.notes || '';

  // Set GL account
  const acctSel = document.getElementById('detAddAccount');
  if (rule.gl_account) acctSel.value = String(rule.gl_account.id);

  // Update form header to signal edit mode
  const header = document.querySelector('#detAddModule')?.closest('.card')?.querySelector('.card-header span');
  if (header) header.innerHTML = '<i class="bi bi-pencil me-2 text-primary"></i>Edit Rule';

  // Scroll the form into view
  modSel.closest('.card').scrollIntoView({ behavior: 'smooth', block: 'start' });
}

async function detRemoveRule(id) {
  if (!confirm('Remove this account determination rule?')) return;
  const result = await api.DeleteAccountDetermination(id);
  if (result) {
    toast('Rule removed.', 'success');
    navigate('#/accounting/determination');
  } else {
    toast('Failed to remove rule.', 'danger');
  }
}

// ── Payment Methods ───────────────────────────────────────────────────────────

let _pmList = [];

async function loadPaymentMethods(tabBar) {
  const [methods, accounts] = await Promise.all([
    api.ListPaymentMethodAccounts(),
    api.ListGLAccounts(),
  ]);
  _pmList     = methods  || [];
  _glAccounts = accounts || [];

  const postingAccounts = _glAccounts.filter(a => a.account_type === 'POSTING' && a.is_active);
  const glOptions = postingAccounts.map(a =>
    `<option value="${a.id}">${a.code} — ${a.name}</option>`
  ).join('');

  const rows = _pmList.length === 0
    ? `<tr><td colspan="5" class="text-center text-muted py-4">No payment method accounts configured.</td></tr>`
    : _pmList.map(m => `
        <tr>
          <td class="fw-semibold">${m.payment_method || '—'}</td>
          <td>${m.bank_name || '—'}</td>
          <td><span class="badge bg-secondary">${m.direction || '—'}</span></td>
          <td>${m.gl_account ? m.gl_account.code + ' — ' + m.gl_account.name : '—'}</td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary" onclick="openPmModal(${m.id})">
              <i class="bi bi-pencil"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapAccounting(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="openPmModal(null)">
        <i class="bi bi-plus-lg me-1"></i>Add Mapping
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Payment Method</th><th>Bank Name</th><th>Direction</th><th>GL Account</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>

    <!-- PM Modal -->
    <div class="modal fade" id="pmModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="pmModalTitle">Payment Method Account</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <div class="mb-3">
              <label class="form-label">Payment Method <span class="text-danger">*</span></label>
              <select class="form-select" id="pmMethod" required>
                <option value="">Select…</option>
                <option value="Cash">Cash</option>
                <option value="Bank Transfer">Bank Transfer</option>
                <option value="Check">Check</option>
                <option value="PDC">PDC (Post-Dated Check)</option>
                <option value="GCash">GCash</option>
                <option value="Maya">Maya</option>
              </select>
            </div>
            <div class="mb-3">
              <label class="form-label">Bank Name</label>
              <input type="text" class="form-control" id="pmBankName" placeholder="e.g. BDO, BPI">
            </div>
            <div class="mb-3">
              <label class="form-label">Direction <span class="text-danger">*</span></label>
              <select class="form-select" id="pmDirection" required>
                <option value="BOTH">BOTH (Inflow &amp; Outflow)</option>
                <option value="INFLOW">INFLOW (Collections)</option>
                <option value="OUTFLOW">OUTFLOW (Payments)</option>
              </select>
            </div>
            <div class="mb-3">
              <label class="form-label">GL Account <span class="text-danger">*</span></label>
              <select class="form-select" id="pmAccount" required>
                <option value="">Select account…</option>
                ${glOptions}
              </select>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitPmForm()">Save</button>
          </div>
        </div>
      </div>
    </div>
  `));
}

function openPmModal(id) {
  document.getElementById('pmModalTitle').textContent = id ? 'Edit Mapping' : 'Add Mapping';
  if (id) {
    const m = _pmList.find(x => x.id === id);
    if (m) {
      document.getElementById('pmMethod').value    = m.payment_method || '';
      document.getElementById('pmBankName').value  = m.bank_name || '';
      document.getElementById('pmDirection').value = m.direction || 'BOTH';
      document.getElementById('pmAccount').value   = m.gl_account_id ?? '';
    }
  } else {
    document.getElementById('pmMethod').value    = '';
    document.getElementById('pmBankName').value  = '';
    document.getElementById('pmDirection').value = 'BOTH';
    document.getElementById('pmAccount').value   = '';
  }
  new bootstrap.Modal(document.getElementById('pmModal')).show();
}

async function submitPmForm() {
  const method    = document.getElementById('pmMethod').value;
  const bankName  = document.getElementById('pmBankName').value.trim();
  const direction = document.getElementById('pmDirection').value;
  const accVal    = document.getElementById('pmAccount').value;

  if (!method)  { toast('Payment method is required.', 'warning'); return; }
  if (!accVal)  { toast('GL account is required.', 'warning'); return; }

  const result = await api.UpsertPaymentMethodAccount({
    payment_method: method,
    bank_name:      bankName,
    gl_account_id:  parseInt(accVal, 10),
    direction,
  });

  if (result) {
    toast('Payment method account saved.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('pmModal')).hide();
    navigate('#/accounting/payment-methods');
  } else {
    toast('Failed to save.', 'danger');
  }
}

// ── Journal Entries List ──────────────────────────────────────────────────────

async function loadJournalEntriesList(tabBar) {
  const entries = await api.ListJournalEntries('', 200);
  if (!entries) {
    showView(wrapAccounting(tabBar, `<div class="alert alert-warning">Failed to load journal entries.</div>`));
    return;
  }

  const rows = entries.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No journal entries found.</td></tr>`
    : entries.map(e => `
        <tr style="cursor:pointer" onclick="navigate('#/accounting/journal-entries/${e.id}')">
          <td>${formatDate(e.date)}</td>
          <td class="fw-semibold">${e.entry_number || '—'}</td>
          <td>${e.module || 'Manual'}</td>
          <td>${e.narration || '—'}</td>
          <td><span class="badge bg-${e.status === 'POSTED' ? 'success' : e.status === 'REVERSED' ? 'secondary' : 'warning'}">${e.status === 'POSTED' ? 'Posted' : e.status === 'REVERSED' ? 'Reversed' : (e.status || 'Draft')}</span></td>
          <td class="text-end"><i class="bi bi-chevron-right text-muted"></i></td>
        </tr>
      `).join('');

  showView(wrapAccounting(tabBar, `
    <div class="d-flex justify-content-end mb-3">
      <button class="btn btn-primary btn-sm" onclick="navigate('#/accounting/new-je')">
        <i class="bi bi-plus-lg me-1"></i>New Manual JE
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Date</th><th>Entry #</th><th>Module</th><th>Narration</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `));
}

// ── Journal Entry Detail ──────────────────────────────────────────────────────

async function loadJournalEntryDetail(id, tabBar) {
  const je = await api.GetJournalEntry(id);
  if (!je) {
    showView(wrapAccounting(tabBar, `<div class="alert alert-warning">Journal entry not found.</div>`));
    return;
  }

  const lines = je.lines || [];
  let totalDebit = 0, totalCredit = 0;
  const lineRows = lines.map(l => {
    totalDebit += l.debit ?? 0;
    totalCredit += l.credit ?? 0;
    return `
      <tr>
        <td>${l.gl_account?.code || '—'}</td>
        <td>${l.gl_account?.name || '—'}</td>
        <td class="text-end">${l.debit ? formatCurrency(l.debit) : ''}</td>
        <td class="text-end">${l.credit ? formatCurrency(l.credit) : ''}</td>
        <td>${l.description || ''}</td>
      </tr>
    `;
  }).join('');

  showView(wrapAccounting(tabBar, `
    <div class="d-flex align-items-center gap-2 mb-4">
      <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/accounting/journal-entries')">
        <i class="bi bi-arrow-left"></i>
      </button>
      <h5 class="mb-0">Journal Entry — ${je.entry_number || ''}</h5>
      <span class="badge bg-${je.status === 'POSTED' ? 'success' : je.status === 'REVERSED' ? 'secondary' : 'warning'} ms-2">${je.status === 'POSTED' ? 'Posted' : je.status === 'REVERSED' ? 'Reversed' : (je.status || 'Draft')}</span>
    </div>
    <div class="row g-3 mb-4">
      <div class="col-sm-4"><strong>Date:</strong> ${formatDate(je.date)}</div>
      <div class="col-sm-4"><strong>Module:</strong> ${je.module || 'Manual'}</div>
      <div class="col-sm-12"><strong>Narration:</strong> ${je.narration || '—'}</div>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th>Code</th><th>Account</th><th class="text-end">Debit</th><th class="text-end">Credit</th><th>Remarks</th></tr>
          </thead>
          <tbody>${lineRows}</tbody>
          <tfoot class="table-light fw-bold">
            <tr>
              <td colspan="2">Total</td>
              <td class="text-end">${formatCurrency(totalDebit)}</td>
              <td class="text-end">${formatCurrency(totalCredit)}</td>
              <td class="${Math.abs(totalDebit - totalCredit) > 0.01 ? 'text-danger' : 'text-success'}">
                ${Math.abs(totalDebit - totalCredit) > 0.01 ? 'Unbalanced!' : 'Balanced'}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  `));
}

// ── New Manual Journal Entry ───────────────────────────────────────────────────

let _jeLineCount = 0;
let _jeAccountOptions = null;

async function loadNewJournalEntry(tabBar) {
  const accounts = await api.GetGLAccounts();
  _glAccounts = accounts || [];
  _jeLineCount = 0;

  const accountOptions = _glAccounts
    .map(a => `<option value="${a.id}">${a.code} — ${a.name}</option>`)
    .join('');

  showView(wrapAccounting(tabBar, `
    <h5 class="fw-bold mb-4">New Manual Journal Entry</h5>
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body">
        <div class="row g-3 mb-4">
          <div class="col-md-4">
            <label class="form-label">Entry Date <span class="text-danger">*</span></label>
            <input type="date" class="form-control" id="jeDate" required value="${new Date().toISOString().slice(0,10)}">
          </div>
          <div class="col-md-8">
            <label class="form-label">Narration <span class="text-danger">*</span></label>
            <input type="text" class="form-control" id="jeNarration" required placeholder="Description of this journal entry">
          </div>
        </div>

        <div class="table-responsive mb-3">
          <table class="table table-bordered align-middle" id="jeTable">
            <thead class="table-light">
              <tr>
                <th style="min-width:260px">Account</th>
                <th style="min-width:140px">Debit</th>
                <th style="min-width:140px">Credit</th>
                <th>Remarks</th>
                <th style="width:40px"></th>
              </tr>
            </thead>
            <tbody id="jeLines"></tbody>
            <tfoot>
              <tr class="table-light fw-bold">
                <td>Total</td>
                <td><span id="jeTotalDebit">0.00</span></td>
                <td><span id="jeTotalCredit">0.00</span></td>
                <td colspan="2" id="jeBalance" class="text-success">Balanced</td>
              </tr>
            </tfoot>
          </table>
        </div>

        <button class="btn btn-outline-secondary btn-sm mb-3" onclick="addJeLine()">
          <i class="bi bi-plus-lg me-1"></i>Add Line
        </button>

        <div class="d-flex gap-2 mt-2">
          <button class="btn btn-primary" onclick="submitJe()">
            <i class="bi bi-check-lg me-1"></i>Post Journal Entry
          </button>
          <button class="btn btn-outline-secondary" onclick="navigate('#/accounting/journal-entries')">Cancel</button>
        </div>
      </div>
    </div>
  `));

  // Store options globally for line rows
  _jeAccountOptions = accountOptions;

  // Add two blank lines to start
  addJeLine();
  addJeLine();
}

function addJeLine() {
  _jeLineCount++;
  const idx = _jeLineCount;
  const tbody = document.getElementById('jeLines');
  const tr = document.createElement('tr');
  tr.id = `jeLine${idx}`;
  tr.innerHTML = `
    <td>
      <select class="form-select form-select-sm" id="jeAcc${idx}">
        <option value="">Select account…</option>
        ${_jeAccountOptions || ''}
      </select>
    </td>
    <td>
      <input type="number" class="form-control form-control-sm" id="jeDr${idx}"
        min="0" step="0.01" value="" placeholder="0.00" oninput="updateJeTotals()">
    </td>
    <td>
      <input type="number" class="form-control form-control-sm" id="jeCr${idx}"
        min="0" step="0.01" value="" placeholder="0.00" oninput="updateJeTotals()">
    </td>
    <td>
      <input type="text" class="form-control form-control-sm" id="jeRem${idx}" placeholder="Optional">
    </td>
    <td>
      <button class="btn btn-sm btn-outline-danger" onclick="removeJeLine(${idx})" title="Remove">
        <i class="bi bi-x"></i>
      </button>
    </td>
  `;
  tbody.appendChild(tr);
}

function removeJeLine(idx) {
  const row = document.getElementById(`jeLine${idx}`);
  if (row) row.remove();
  updateJeTotals();
}

function updateJeTotals() {
  let totalDr = 0, totalCr = 0;
  document.querySelectorAll('#jeLines tr').forEach(tr => {
    const idx = tr.id.replace('jeLine', '');
    const dr = parseFloat(document.getElementById(`jeDr${idx}`)?.value || 0) || 0;
    const cr = parseFloat(document.getElementById(`jeCr${idx}`)?.value || 0) || 0;
    totalDr += dr;
    totalCr += cr;
  });
  document.getElementById('jeTotalDebit').textContent = totalDr.toFixed(2);
  document.getElementById('jeTotalCredit').textContent = totalCr.toFixed(2);
  const balEl = document.getElementById('jeBalance');
  if (Math.abs(totalDr - totalCr) < 0.01) {
    balEl.textContent = 'Balanced';
    balEl.className = 'text-success';
  } else {
    balEl.textContent = `Out of balance by ${(totalDr - totalCr).toFixed(2)}`;
    balEl.className = 'text-danger fw-bold';
  }
}

async function submitJe() {
  const entryDate = document.getElementById('jeDate').value;
  const narration = document.getElementById('jeNarration').value.trim();
  if (!entryDate || !narration) { toast('Date and narration are required.', 'warning'); return; }

  const lines = [];
  let totalDr = 0, totalCr = 0;
  document.querySelectorAll('#jeLines tr').forEach(tr => {
    const idx = tr.id.replace('jeLine', '');
    const accId = document.getElementById(`jeAcc${idx}`)?.value;
    const dr = parseFloat(document.getElementById(`jeDr${idx}`)?.value || 0) || 0;
    const cr = parseFloat(document.getElementById(`jeCr${idx}`)?.value || 0) || 0;
    const rem = document.getElementById(`jeRem${idx}`)?.value.trim() || '';
    if (accId && (dr > 0 || cr > 0)) {
      lines.push({ gl_account_id: parseInt(accId, 10), debit: dr, credit: cr, description: rem });
      totalDr += dr;
      totalCr += cr;
    }
  });

  if (lines.length < 2) { toast('At least two lines with amounts are required.', 'warning'); return; }
  if (Math.abs(totalDr - totalCr) > 0.01) { toast('Journal entry is not balanced.', 'danger'); return; }

  const payload = { date: entryDate, narration, lines };
  const result = await api.PostManualJournalEntry(payload);
  if (result) {
    toast('Journal entry posted.', 'success');
    navigate('#/accounting/journal-entries');
  } else {
    toast('Failed to post journal entry.', 'danger');
  }
}

// ── Layout Helper ─────────────────────────────────────────────────────────────

function wrapAccounting(tabBar, content) {
  return `
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-journal-bookmark me-2"></i>Accounting</h4>
      ${tabBar}
      ${content}
    </div>
  `;
}
