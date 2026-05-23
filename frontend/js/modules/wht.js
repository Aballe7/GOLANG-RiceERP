// wht.js — BIR Expanded Withholding Tax (EWT) Reports module
// Admin-only: manages Form 0619-E, 1601-EQ, and BIR Form 2307 generation.

Modules.WHT = {
  async load(sub, id, action) {
    const active = sub || 'ledger';
    showLoading();

    const year = new Date().getFullYear();

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'ledger' ? 'active' : ''}" href="#"
            onclick="navigate('#/wht/ledger');return false;">
            <i class="bi bi-table me-1"></i>WHT Ledger
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === '0619e' ? 'active' : ''}" href="#"
            onclick="navigate('#/wht/0619e');return false;">
            <i class="bi bi-calendar-month me-1"></i>0619-E Monthly
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === '1601eq' ? 'active' : ''}" href="#"
            onclick="navigate('#/wht/1601eq');return false;">
            <i class="bi bi-calendar-quarter me-1"></i>1601-EQ Quarterly
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'form2307' ? 'active' : ''}" href="#"
            onclick="navigate('#/wht/form2307');return false;">
            <i class="bi bi-file-earmark-person me-1"></i>Form 2307
          </a>
        </li>
      </ul>`;

    if      (active === 'ledger')   await whtLoadLedger(tabBar, year);
    else if (active === '0619e')    await whtLoad0619E(tabBar, year);
    else if (active === '1601eq')   await whtLoad1601EQ(tabBar, year);
    else if (active === 'form2307') await whtLoad2307(tabBar, year);
    else await whtLoadLedger(tabBar, year);
  }
};

// ── Shared helpers ────────────────────────────────────────────────────────────

const WHT_CATEGORY_LABELS = {
  NONE:                  'None',
  AGRICULTURAL_PRODUCER: 'Agricultural Producer',
  PALAY_TRADER:          'Palay Trader',
  GOODS_SUPPLIER:        'Goods Supplier',
  SERVICE_PROVIDER:      'Service Provider',
  PROFESSIONAL:          'Professional',
  RENTAL:                'Rental',
};

const MONTH_NAMES = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
const MONTH_NAMES_FULL = ['January','February','March','April','May','June','July','August','September','October','November','December'];

function whtPageHeader(title, subtitle) {
  return `
    <div class="mb-3">
      <h4 class="fw-bold mb-0">
        <i class="bi bi-receipt me-2 text-warning"></i>${title}
      </h4>
      <p class="text-muted small mb-0">${subtitle}</p>
    </div>`;
}

// ── WHT Ledger ────────────────────────────────────────────────────────────────

async function whtLoadLedger(tabBar, defaultYear) {
  const year = parseInt(document.getElementById('whtLedgerYear')?.value) || defaultYear;
  const list = (await api.ListWHTEntries(year, 0)) || [];

  const yearSel = whtYearSelect('whtLedgerYear', year, 'whtRefreshLedger()');

  const totalWHT   = list.reduce((s, e) => s + (e.wht_amount  || 0), 0);
  const totalGross = list.reduce((s, e) => s + (e.gross_amount || 0), 0);
  const supplierSet = new Set(list.map(e => e.supplier_name));

  const rows = list.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-5">
         <i class="bi bi-inbox fs-2 d-block mb-2 opacity-25"></i>
         No withholding tax entries for ${year}.
       </td></tr>`
    : list.map(e => `
      <tr>
        <td class="fw-semibold">${e.invoice_number}</td>
        <td>${formatDate(e.posting_date)}</td>
        <td>
          <span class="fw-semibold">${_esc(e.supplier_name)}</span><br>
          <small class="text-muted font-monospace">${e.supplier_tin || 'No TIN'}</small>
        </td>
        <td><span class="badge bg-secondary-subtle text-dark border font-monospace">${e.atc_code || '—'}</span></td>
        <td class="text-end">${formatCurrency(e.gross_amount)}</td>
        <td class="text-end text-muted">${formatCurrency(e.vat_exclusive)}</td>
        <td class="text-end fw-bold" style="color:#b45309">${formatCurrency(e.wht_amount)}</td>
        <td class="text-end text-success fw-semibold">${formatCurrency(e.net_payable)}</td>
      </tr>`).join('');

  const content = `
    <!-- ── Banner ── -->
    <div class="rounded-3 mb-4 px-4 py-3 d-flex align-items-center justify-content-between"
         style="background:linear-gradient(135deg,#1e3a5f 0%,#2d5a8e 100%);color:#fff;">
      <div>
        <div class="fw-bold fs-5">
          <i class="bi bi-journal-text me-2"></i>Internal EWT Register
        </div>
        <div class="small opacity-75 mt-1">
          For internal tracking only — not a BIR filing document.
          Use Form 2307 for supplier certificates, 0619-E for monthly remittance.
        </div>
      </div>
      <div class="text-end">
        <div class="opacity-75 small">Tax Year</div>
        <div class="fw-bold fs-4">${year}</div>
      </div>
    </div>

    <!-- ── Stat cards ── -->
    <div class="row g-3 mb-4">
      <div class="col-sm-4">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body text-center py-3">
            <div class="text-muted small text-uppercase fw-bold mb-1">Total WHT Withheld</div>
            <div class="fw-bold fs-4" style="color:#b45309">${formatCurrency(totalWHT)}</div>
            <div class="text-muted small">${year} year-to-date</div>
          </div>
        </div>
      </div>
      <div class="col-sm-4">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body text-center py-3">
            <div class="text-muted small text-uppercase fw-bold mb-1">Gross Invoices</div>
            <div class="fw-bold fs-4 text-secondary">${formatCurrency(totalGross)}</div>
            <div class="text-muted small">Total purchase amount</div>
          </div>
        </div>
      </div>
      <div class="col-sm-4">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body text-center py-3">
            <div class="text-muted small text-uppercase fw-bold mb-1">Payees / Transactions</div>
            <div class="fw-bold fs-4 text-primary">${supplierSet.size} / ${list.length}</div>
            <div class="text-muted small">Unique suppliers / invoices</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Filter row ── -->
    <div class="d-flex align-items-center gap-2 mb-3">
      <span class="text-muted small me-1">Year:</span>
      ${yearSel}
      <button class="btn btn-sm btn-outline-secondary" onclick="whtRefreshLedger()">
        <i class="bi bi-arrow-clockwise me-1"></i>Refresh
      </button>
    </div>

    <!-- ── Table ── -->
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-white border-bottom py-2 px-3 d-flex align-items-center gap-2">
        <i class="bi bi-table text-primary me-1"></i>
        <span class="fw-semibold small">EWT Transaction Register — ${year}</span>
      </div>
      <div class="table-responsive">
        <table class="table table-sm table-hover align-middle mb-0">
          <thead style="background:#f0f4ff">
            <tr>
              <th>Invoice #</th>
              <th>Date</th>
              <th>Supplier / TIN</th>
              <th>ATC</th>
              <th class="text-end">Gross Inv.</th>
              <th class="text-end">VAT-Excl. Base</th>
              <th class="text-end" style="color:#b45309">WHT Amount</th>
              <th class="text-end text-success">Net Payable</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
          ${list.length > 0 ? `
          <tfoot style="background:#f0f4ff;font-weight:600">
            <tr>
              <td colspan="4" class="text-end pe-3 text-muted small">TOTAL ${year}</td>
              <td class="text-end">${formatCurrency(totalGross)}</td>
              <td></td>
              <td class="text-end" style="color:#b45309">${formatCurrency(totalWHT)}</td>
              <td></td>
            </tr>
          </tfoot>` : ''}
        </table>
      </div>
    </div>`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:1100px">
      ${whtPageHeader('Withholding Tax (EWT)', 'BIR Expanded Withholding Tax — NIRC Sec. 57-58, RR 2-98')}
      ${tabBar}
      ${content}
    </div>`);
}

async function whtRefreshLedger() {
  Modules.WHT.load('ledger');
}

// ── Form 0619-E ───────────────────────────────────────────────────────────────

async function whtLoad0619E(tabBar, defaultYear) {
  const now = new Date();
  const year  = parseInt(document.getElementById('wht0619Year')?.value)  || defaultYear;
  const month = parseInt(document.getElementById('wht0619Month')?.value) || now.getMonth() + 1;

  const data = await api.GetForm0619E(year, month);

  const yearSel  = whtYearSelect('wht0619Year', year, 'whtRefresh0619E()');
  const monthSel = whtMonthSelect('wht0619Month', month, 'whtRefresh0619E()');

  if (!data) {
    showView(`<div class="container-fluid px-4 py-4" style="max-width:1100px">
      ${whtPageHeader('Withholding Tax (EWT)', 'BIR Expanded Withholding Tax — NIRC Sec. 57-58, RR 2-98')}
      ${tabBar}
      <div class="alert alert-warning">Failed to load 0619-E data.</div>
    </div>`);
    return;
  }

  const atcRows = (data.atc_summary || []).map(a => `
    <tr>
      <td class="fw-semibold font-monospace">${a.atc_code}</td>
      <td>${WHT_CATEGORY_LABELS[a.wht_category] || a.wht_category}</td>
      <td class="text-end">${formatCurrency(a.total_income)}</td>
      <td class="text-end fw-bold" style="color:#b45309">${formatCurrency(a.total_wht)}</td>
    </tr>`).join('');

  const entryRows = (data.entries || []).map(e => `
    <tr>
      <td>${formatDate(e.posting_date)}</td>
      <td class="fw-semibold">${e.invoice_number}</td>
      <td>${_esc(e.supplier_name)}</td>
      <td><span class="badge bg-secondary-subtle text-dark border font-monospace">${e.atc_code}</span></td>
      <td class="text-end">${formatCurrency(e.vat_exclusive)}</td>
      <td class="text-end fw-bold" style="color:#b45309">${formatCurrency(e.wht_amount)}</td>
    </tr>`).join('');

  const monthName = MONTH_NAMES_FULL[month - 1];
  const hasData   = (data.atc_summary || []).length > 0;

  const content = `
    <div class="d-flex align-items-center gap-2 mb-4 flex-wrap">
      <span class="text-muted small">Period:</span>
      ${yearSel} ${monthSel}
      <button class="btn btn-sm btn-outline-secondary" onclick="whtRefresh0619E()">
        <i class="bi bi-arrow-clockwise me-1"></i>Refresh
      </button>
    </div>

    <!-- ── Remittance callout ── -->
    <div class="rounded-3 mb-4 px-4 py-3" style="background:linear-gradient(135deg,#7c3aed 0%,#a855f7 100%);color:#fff;">
      <div class="row align-items-center">
        <div class="col">
          <div class="fw-bold fs-5">
            <i class="bi bi-send me-2"></i>BIR Form 0619-E — Monthly EWT Remittance
          </div>
          <div class="small opacity-75 mt-1">
            ${monthName} ${year} &ensp;•&ensp; Due: <strong>${data.due_date || '—'}</strong>
            &ensp;•&ensp; File via eFPS or Authorized Agent Bank
          </div>
        </div>
        <div class="col-auto text-end">
          <div class="small opacity-75">Total to Remit</div>
          <div class="fw-bold" style="font-size:2rem">${formatCurrency(data.total_wht)}</div>
        </div>
      </div>
    </div>

    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-2 px-3">
        <span class="fw-semibold small">
          <i class="bi bi-bar-chart-steps me-1 text-purple"></i>
          Summary by ATC Code — ${monthName} ${year}
        </span>
      </div>
      <div class="card-body p-0">
        ${!hasData ? `<div class="text-center py-5 text-muted">
          <i class="bi bi-inbox fs-2 d-block mb-2 opacity-25"></i>
          No WHT transactions for ${monthName} ${year}.
        </div>` : `
        <div class="table-responsive">
          <table class="table table-sm align-middle mb-0">
            <thead style="background:#f5f0ff">
              <tr>
                <th>ATC Code</th>
                <th>WHT Category</th>
                <th class="text-end">Total Income Payment</th>
                <th class="text-end" style="color:#b45309">Total WHT</th>
              </tr>
            </thead>
            <tbody>${atcRows}</tbody>
            <tfoot style="background:#f5f0ff;font-weight:600">
              <tr>
                <td colspan="3" class="text-end pe-3">Total to Remit — ${monthName} ${year}</td>
                <td class="text-end" style="color:#b45309;font-size:1.1rem">${formatCurrency(data.total_wht)}</td>
              </tr>
            </tfoot>
          </table>
        </div>`}
      </div>
    </div>

    ${entryRows ? `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-white border-bottom py-2 px-3">
        <span class="fw-semibold small">Individual Transactions — ${monthName} ${year}</span>
      </div>
      <div class="table-responsive">
        <table class="table table-sm align-middle mb-0">
          <thead class="table-light">
            <tr>
              <th>Date</th><th>Invoice #</th><th>Supplier</th><th>ATC</th>
              <th class="text-end">VAT-Excl. Base</th>
              <th class="text-end" style="color:#b45309">WHT</th>
            </tr>
          </thead>
          <tbody>${entryRows}</tbody>
        </table>
      </div>
    </div>` : ''}`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:1100px">
      ${whtPageHeader('Withholding Tax (EWT)', 'BIR Expanded Withholding Tax — NIRC Sec. 57-58, RR 2-98')}
      ${tabBar}
      ${content}
    </div>`);
}

async function whtRefresh0619E() { await Modules.WHT.load('0619e'); }

// ── Form 1601-EQ ──────────────────────────────────────────────────────────────

async function whtLoad1601EQ(tabBar, defaultYear) {
  const now     = new Date();
  const year    = parseInt(document.getElementById('wht1601Year')?.value)    || defaultYear;
  const quarter = parseInt(document.getElementById('wht1601Quarter')?.value) || Math.ceil((now.getMonth() + 1) / 3);

  const data = await api.GetForm1601EQ(year, quarter);

  const yearSel    = whtYearSelect('wht1601Year', year, 'whtRefresh1601EQ()');
  const quarterSel = whtQuarterSelect('wht1601Quarter', quarter, 'whtRefresh1601EQ()');

  if (!data) {
    showView(`<div class="container-fluid px-4 py-4" style="max-width:1100px">
      ${whtPageHeader('Withholding Tax (EWT)', 'BIR Expanded Withholding Tax — NIRC Sec. 57-58, RR 2-98')}
      ${tabBar}
      <div class="alert alert-warning">Failed to load 1601-EQ data.</div>
    </div>`);
    return;
  }

  const atcRows = (data.atc_summary || []).map(a => `
    <tr>
      <td class="fw-semibold font-monospace">${a.atc_code}</td>
      <td>${WHT_CATEGORY_LABELS[a.wht_category] || a.wht_category}</td>
      <td class="text-end">${formatCurrency(a.total_income)}</td>
      <td class="text-end fw-bold" style="color:#b45309">${formatCurrency(a.total_wht)}</td>
    </tr>`).join('');

  const entryRows = (data.entries || []).map(e => `
    <tr>
      <td>${formatDate(e.posting_date)}</td>
      <td class="fw-semibold">${e.invoice_number}</td>
      <td>
        <span class="fw-semibold">${_esc(e.supplier_name)}</span><br>
        <small class="text-muted font-monospace">${e.supplier_tin || 'No TIN'}</small>
      </td>
      <td><span class="badge bg-secondary-subtle text-dark border font-monospace">${e.atc_code}</span></td>
      <td class="text-end">${formatCurrency(e.gross_amount)}</td>
      <td class="text-end text-muted">${formatCurrency(e.vat_exclusive)}</td>
      <td class="text-end fw-bold" style="color:#b45309">${formatCurrency(e.wht_amount)}</td>
    </tr>`).join('');

  const hasData = (data.atc_summary || []).length > 0;

  const content = `
    <div class="d-flex align-items-center gap-2 mb-4 flex-wrap">
      <span class="text-muted small">Period:</span>
      ${yearSel} ${quarterSel}
      <button class="btn btn-sm btn-outline-secondary" onclick="whtRefresh1601EQ()">
        <i class="bi bi-arrow-clockwise me-1"></i>Refresh
      </button>
    </div>

    <!-- ── Remittance callout ── -->
    <div class="rounded-3 mb-4 px-4 py-3" style="background:linear-gradient(135deg,#065f46 0%,#059669 100%);color:#fff;">
      <div class="row align-items-center">
        <div class="col">
          <div class="fw-bold fs-5">
            <i class="bi bi-file-earmark-spreadsheet me-2"></i>BIR Form 1601-EQ — Quarterly EWT Return
          </div>
          <div class="small opacity-75 mt-1">
            Q${quarter} ${year} &ensp;•&ensp; Due: <strong>${data.due_date || '—'}</strong>
            &ensp;•&ensp; File via eFPS &ensp;•&ensp; Attach QAP (Quarterly Alphalist of Payees)
          </div>
        </div>
        <div class="col-auto text-end">
          <div class="small opacity-75">Total to Remit</div>
          <div class="fw-bold" style="font-size:2rem">${formatCurrency(data.total_wht)}</div>
        </div>
      </div>
    </div>

    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-bottom py-2 px-3">
        <span class="fw-semibold small">
          <i class="bi bi-bar-chart-steps me-1"></i>
          Schedule 4 (QAP) — Summary by ATC Code — Q${quarter} ${year}
        </span>
      </div>
      <div class="card-body p-0">
        ${!hasData ? `<div class="text-center py-5 text-muted">
          <i class="bi bi-inbox fs-2 d-block mb-2 opacity-25"></i>
          No WHT transactions for Q${quarter} ${year}.
        </div>` : `
        <div class="table-responsive">
          <table class="table table-sm align-middle mb-0">
            <thead style="background:#ecfdf5">
              <tr>
                <th>ATC</th>
                <th>WHT Category</th>
                <th class="text-end">Total Income Payment</th>
                <th class="text-end" style="color:#b45309">Total WHT</th>
              </tr>
            </thead>
            <tbody>${atcRows}</tbody>
            <tfoot style="background:#ecfdf5;font-weight:600">
              <tr>
                <td colspan="3" class="text-end pe-3">Total EWT to Remit — Q${quarter} ${year}</td>
                <td class="text-end" style="color:#b45309;font-size:1.1rem">${formatCurrency(data.total_wht)}</td>
              </tr>
            </tfoot>
          </table>
        </div>`}
      </div>
    </div>

    ${entryRows ? `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-white border-bottom py-2 px-3">
        <span class="fw-semibold small">Payee Detail — Q${quarter} ${year}</span>
      </div>
      <div class="table-responsive">
        <table class="table table-sm align-middle mb-0">
          <thead class="table-light">
            <tr>
              <th>Date</th><th>Invoice #</th><th>Supplier / TIN</th><th>ATC</th>
              <th class="text-end">Gross Inv.</th>
              <th class="text-end">VAT-Excl.</th>
              <th class="text-end" style="color:#b45309">WHT</th>
            </tr>
          </thead>
          <tbody>${entryRows}</tbody>
        </table>
      </div>
    </div>` : ''}`;

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:1100px">
      ${whtPageHeader('Withholding Tax (EWT)', 'BIR Expanded Withholding Tax — NIRC Sec. 57-58, RR 2-98')}
      ${tabBar}
      ${content}
    </div>`);
}

async function whtRefresh1601EQ() { await Modules.WHT.load('1601eq'); }

// ── Form 2307 ─────────────────────────────────────────────────────────────────

async function whtLoad2307(tabBar, defaultYear) {
  const now      = new Date();
  const year     = parseInt(document.getElementById('wht2307Year')?.value)    || defaultYear;
  const quarter  = parseInt(document.getElementById('wht2307Quarter')?.value) || Math.ceil((now.getMonth() + 1) / 3);

  const suppliers = (await api.ListSuppliersWithWHT(year)) || [];

  const yearSel    = whtYearSelect('wht2307Year', year, 'whtRefresh2307()');
  const quarterSel = whtQuarterSelect('wht2307Quarter', quarter, 'whtRefresh2307()');

  const supOptions = suppliers.length === 0
    ? '<option value="">— No suppliers with WHT in this year —</option>'
    : suppliers.map(s => `<option value="${s.id}">${_esc(s.name)} (${s.tin_number || 'No TIN'})</option>`).join('');

  const selectedSupId = parseInt(document.getElementById('wht2307Supplier')?.value) || (suppliers[0]?.id || 0);
  const company = 'RiceMill Operations'; // TODO: pull from company settings

  let certSection = '';
  if (selectedSupId && suppliers.length > 0) {
    const data = await api.GetForm2307(selectedSupId, year, quarter, company);
    if (data) certSection = whtRenderForm2307(data, quarter, year);
  }

  showView(`
    <div class="container-fluid px-4 py-4" style="max-width:900px">
      ${whtPageHeader('Withholding Tax (EWT)', 'BIR Expanded Withholding Tax — NIRC Sec. 57-58, RR 2-98')}
      ${tabBar}

      <!-- ── Controls ── -->
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-body py-3 px-4">
          <div class="row align-items-center g-2">
            <div class="col-auto">
              <label class="form-label small text-muted mb-1">Year</label>
              ${yearSel}
            </div>
            <div class="col-auto">
              <label class="form-label small text-muted mb-1">Quarter</label>
              ${quarterSel}
            </div>
            <div class="col">
              <label class="form-label small text-muted mb-1">Supplier / Payee</label>
              <select class="form-select form-select-sm" id="wht2307Supplier" onchange="whtRefresh2307()">
                ${supOptions}
              </select>
            </div>
            <div class="col-auto pt-3">
              <button class="btn btn-sm btn-outline-secondary" onclick="whtRefresh2307()">
                <i class="bi bi-arrow-clockwise me-1"></i>Refresh
              </button>
            </div>
            ${certSection ? `
            <div class="col-auto pt-3">
              <button class="btn btn-sm btn-primary" onclick="window.print()">
                <i class="bi bi-printer me-1"></i>Print Certificate
              </button>
            </div>` : ''}
          </div>
        </div>
      </div>

      ${certSection || `
      <div class="alert alert-info d-flex align-items-center gap-3 py-4">
        <i class="bi bi-file-earmark-person fs-2 text-info flex-shrink-0"></i>
        <div>
          <div class="fw-semibold">Select a supplier above to generate Form 2307</div>
          <div class="small text-muted mt-1">
            This certificate is issued quarterly to each supplier from whom tax was withheld.
            It serves as their proof of creditable withholding tax for their income tax return.
          </div>
        </div>
      </div>`}
    </div>`);

  if (selectedSupId) {
    const sel = document.getElementById('wht2307Supplier');
    if (sel) sel.value = selectedSupId;
  }
}

function whtRenderForm2307(data, quarter, year) {
  const qMonths = [[1,2,3],[4,5,6],[7,8,9],[10,11,12]][quarter - 1];

  const totalPmt = data.total_payment || 0;
  const totalWHT = data.total_wht    || 0;

  const monthRows = qMonths.map(m => {
    const ln = (data.lines || []).find(l => l.month === m) || { payment: 0, wht: 0 };
    return `
      <tr>
        <td class="px-3 py-2 border" style="font-size:12px">${MONTH_NAMES_FULL[m - 1]}</td>
        <td class="px-3 py-2 border text-end" style="font-size:12px">${formatCurrency(ln.payment || 0)}</td>
        <td class="px-3 py-2 border text-end" style="font-size:12px">${formatCurrency(ln.wht || 0)}</td>
      </tr>`;
  }).join('');

  return `
    <!-- ── BIR FORM 2307 CERTIFICATE ── -->
    <div id="bir-form-2307" style="font-family:'Times New Roman',serif;background:#fff;border:2px solid #333;max-width:860px;margin:0 auto;">

      <!-- Republic header -->
      <div style="background:#1a3a6e;color:#fff;text-align:center;padding:10px 20px;">
        <div style="font-size:11px;letter-spacing:1px;text-transform:uppercase;opacity:.8">
          Republika ng Pilipinas &bull; Republic of the Philippines
        </div>
        <div style="font-size:12px;letter-spacing:1px;text-transform:uppercase;opacity:.85">
          Kagawaran ng Pananalapi &bull; Department of Finance
        </div>
        <div style="font-size:13px;font-weight:bold;letter-spacing:2px;text-transform:uppercase;margin-top:2px">
          Bureau of Internal Revenue
        </div>
      </div>

      <!-- Form title bar -->
      <div style="display:flex;align-items:center;justify-content:space-between;border-bottom:2px solid #333;padding:8px 16px;background:#f8f8f2;">
        <div>
          <div style="font-size:9px;text-transform:uppercase;letter-spacing:1px;color:#555">BIR Form No.</div>
          <div style="font-size:26px;font-weight:bold;color:#1a3a6e;letter-spacing:2px">2307</div>
        </div>
        <div style="text-align:center;flex:1;padding:0 20px">
          <div style="font-size:14px;font-weight:bold;text-transform:uppercase;letter-spacing:1px">
            Certificate of Creditable Tax Withheld at Source
          </div>
          <div style="font-size:10px;color:#555;margin-top:2px">
            (Expanded Withholding Tax) &bull; Pursuant to NIRC Sec. 58(A), RR No. 2-98
          </div>
        </div>
        <div style="text-align:right">
          <div style="font-size:9px;text-transform:uppercase;letter-spacing:1px;color:#555">For the</div>
          <div style="font-size:15px;font-weight:bold;color:#1a3a6e">Quarter ${quarter}</div>
          <div style="font-size:13px;font-weight:bold;color:#1a3a6e">${year}</div>
        </div>
      </div>

      <!-- Body -->
      <div style="padding:14px 16px;">

        <!-- Payor / Agent section -->
        <div style="border:1px solid #555;margin-bottom:10px;">
          <div style="background:#1a3a6e;color:#fff;font-size:9px;text-transform:uppercase;letter-spacing:1px;padding:3px 8px;font-weight:bold;">
            Part I — Withholding Agent / Payor (Issued By)
          </div>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0;">
            <div style="padding:8px 10px;border-right:1px solid #ccc;">
              <div style="font-size:9px;text-transform:uppercase;color:#777;letter-spacing:.5px">Name of Withholding Agent</div>
              <div style="font-size:13px;font-weight:bold;margin-top:2px">${_esc(data.issued_by || '—')}</div>
            </div>
            <div style="padding:8px 10px;">
              <div style="font-size:9px;text-transform:uppercase;color:#777;letter-spacing:.5px">Date of Issue</div>
              <div style="font-size:13px;font-weight:bold;margin-top:2px">${data.issue_due_date || '—'}</div>
            </div>
          </div>
        </div>

        <!-- Payee section -->
        <div style="border:1px solid #555;margin-bottom:10px;">
          <div style="background:#1a3a6e;color:#fff;font-size:9px;text-transform:uppercase;letter-spacing:1px;padding:3px 8px;font-weight:bold;">
            Part II — Payee / Supplier
          </div>
          <div style="display:grid;grid-template-columns:2fr 1fr 1fr;gap:0;">
            <div style="padding:8px 10px;border-right:1px solid #ccc;">
              <div style="font-size:9px;text-transform:uppercase;color:#777;letter-spacing:.5px">Name of Payee</div>
              <div style="font-size:13px;font-weight:bold;margin-top:2px">${_esc(data.payee_name || '—')}</div>
            </div>
            <div style="padding:8px 10px;border-right:1px solid #ccc;">
              <div style="font-size:9px;text-transform:uppercase;color:#777;letter-spacing:.5px">TIN</div>
              <div style="font-size:13px;font-weight:bold;margin-top:2px;font-family:monospace">${data.payee_tin || 'Not on file'}</div>
            </div>
            <div style="padding:8px 10px;">
              <div style="font-size:9px;text-transform:uppercase;color:#777;letter-spacing:.5px">ATC Code</div>
              <div style="font-size:13px;font-weight:bold;margin-top:2px;font-family:monospace">${data.atc_code || '—'}</div>
            </div>
          </div>
          <div style="border-top:1px solid #ccc;padding:8px 10px;">
            <div style="font-size:9px;text-transform:uppercase;color:#777;letter-spacing:.5px">Nature of Income Payment / WHT Category</div>
            <div style="font-size:12px;margin-top:2px">${WHT_CATEGORY_LABELS[data.wht_category] || data.wht_category || '—'}</div>
          </div>
        </div>

        <!-- Income / Tax table -->
        <div style="border:1px solid #555;margin-bottom:10px;">
          <div style="background:#1a3a6e;color:#fff;font-size:9px;text-transform:uppercase;letter-spacing:1px;padding:3px 8px;font-weight:bold;">
            Part III — Income Payments and Tax Withheld (Q${quarter} ${year})
          </div>
          <table style="width:100%;border-collapse:collapse;font-size:12px;">
            <thead>
              <tr style="background:#e8eef8">
                <th style="padding:6px 12px;border:1px solid #999;text-align:left;font-size:11px">Month</th>
                <th style="padding:6px 12px;border:1px solid #999;text-align:right;font-size:11px">Income Payment (₱)</th>
                <th style="padding:6px 12px;border:1px solid #999;text-align:right;font-size:11px">Tax Withheld (₱)</th>
              </tr>
            </thead>
            <tbody>
              ${monthRows}
              <tr style="background:#f0f4ff;font-weight:bold;">
                <td style="padding:8px 12px;border:2px solid #333;font-size:12px">TOTAL</td>
                <td style="padding:8px 12px;border:2px solid #333;text-align:right;font-size:13px">${formatCurrency(totalPmt)}</td>
                <td style="padding:8px 12px;border:2px solid #333;text-align:right;font-size:13px;color:#b45309">${formatCurrency(totalWHT)}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Certification -->
        <div style="border:1px solid #555;margin-bottom:10px;padding:12px 14px;background:#fffef0;">
          <div style="font-size:10px;font-weight:bold;text-transform:uppercase;color:#1a3a6e;margin-bottom:6px;letter-spacing:.5px">
            Certification
          </div>
          <div style="font-size:11px;text-align:justify;line-height:1.6;color:#222;">
            I/We hereby certify, under the penalties of perjury, that the information herein stated are true,
            correct and complete pursuant to the provisions of the National Internal Revenue Code, as amended,
            and the regulations issued under authority thereof; and that the amount of creditable withholding tax
            shown herein has been deducted from the income payment made to the payee named above and
            duly remitted to the Bureau of Internal Revenue.
          </div>
        </div>

        <!-- Signatures -->
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px;margin-top:10px;">
          <div style="border-top:1px solid #333;padding-top:6px;text-align:center;">
            <div style="height:36px"></div>
            <div style="font-size:10px;color:#555">Signature of Withholding Agent / Authorized Representative</div>
            <div style="font-size:11px;font-weight:bold;margin-top:4px">${_esc(data.issued_by || '')}</div>
            <div style="font-size:10px;color:#777">Date: ${data.issue_due_date || '________________'}</div>
          </div>
          <div style="border-top:1px solid #333;padding-top:6px;text-align:center;">
            <div style="height:36px"></div>
            <div style="font-size:10px;color:#555">Title / Designation</div>
            <div style="font-size:11px;font-weight:bold;margin-top:4px">Authorized Signatory</div>
            <div style="font-size:10px;color:#777">TIN: ________________________</div>
          </div>
        </div>

        <!-- Footer note -->
        <div style="margin-top:14px;padding:8px 10px;background:#f3f4f6;border:1px solid #ddd;border-radius:4px;">
          <div style="font-size:9px;color:#666;line-height:1.5">
            <strong>NOTE:</strong> This Form 2307 is to be issued quarterly. The payee (supplier) shall attach this to
            their quarterly/annual income tax return as proof of creditable withholding tax. The withholding agent
            shall retain a copy of this certificate for at least <strong>5 years</strong> for audit purposes.
            Failure to issue this certificate is subject to penalties under the Tax Code.
          </div>
        </div>

      </div><!-- /body -->
    </div><!-- /form-2307 -->

    ${totalWHT === 0 ? `
    <div class="alert alert-light border mt-3 small">
      <i class="bi bi-info-circle me-1"></i>
      No WHT transactions for this supplier in Q${quarter} ${year}. The certificate will show zero amounts.
    </div>` : `
    <div class="alert alert-warning border-0 mt-3 small d-print-none">
      <i class="bi bi-exclamation-triangle me-1"></i>
      <strong>Action required:</strong> Issue this Form 2307 to <strong>${_esc(data.payee_name)}</strong> by
      <strong>${data.issue_due_date}</strong>. Click <strong>Print Certificate</strong> above to print.
    </div>`}`;
}

async function whtRefresh2307() { await Modules.WHT.load('form2307'); }

// ── Select helpers ────────────────────────────────────────────────────────────

function whtYearSelect(id, selected, onchange) {
  const currentYear = new Date().getFullYear();
  const years = [currentYear, currentYear - 1, currentYear - 2];
  return `<select class="form-select form-select-sm" id="${id}" style="width:90px" onchange="${onchange}">
    ${years.map(y => `<option value="${y}" ${y === selected ? 'selected' : ''}>${y}</option>`).join('')}
  </select>`;
}

function whtMonthSelect(id, selected, onchange) {
  return `<select class="form-select form-select-sm" id="${id}" style="width:110px" onchange="${onchange}">
    ${MONTH_NAMES_FULL.map((m, i) => `<option value="${i + 1}" ${(i + 1) === selected ? 'selected' : ''}>${m}</option>`).join('')}
  </select>`;
}

function whtQuarterSelect(id, selected, onchange) {
  return `<select class="form-select form-select-sm" id="${id}" style="width:80px" onchange="${onchange}">
    <option value="1" ${selected === 1 ? 'selected' : ''}>Q1</option>
    <option value="2" ${selected === 2 ? 'selected' : ''}>Q2</option>
    <option value="3" ${selected === 3 ? 'selected' : ''}>Q3</option>
    <option value="4" ${selected === 4 ? 'selected' : ''}>Q4</option>
  </select>`;
}
