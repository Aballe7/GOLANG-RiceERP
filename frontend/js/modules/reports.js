// reports.js — Modules.Reports

Modules.Reports = {
  async load(sub) {
    const active = sub || 'pl';
    showLoading();

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'pl' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/pl');return false;">
            <i class="bi bi-bar-chart-line me-1"></i>P&amp;L
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'balance-sheet' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/balance-sheet');return false;">
            <i class="bi bi-table me-1"></i>Balance Sheet
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'cash-flow' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/cash-flow');return false;">
            <i class="bi bi-water me-1"></i>Cash Flow
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'trial-balance' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/trial-balance');return false;">
            <i class="bi bi-journal-text me-1"></i>Trial Balance
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ar-aging' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/ar-aging');return false;">
            <i class="bi bi-calendar-range me-1"></i>AR Aging
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ap-aging' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/ap-aging');return false;">
            <i class="bi bi-calendar2-range me-1"></i>AP Aging
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'analytics' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/analytics');return false;">
            <i class="bi bi-graph-up-arrow me-1"></i>Analytics
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'inv-valuation' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/inv-valuation');return false;">
            <i class="bi bi-boxes me-1"></i>Inv. Valuation
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'inv-movement' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/inv-movement');return false;">
            <i class="bi bi-arrow-left-right me-1"></i>Movement
          </a>
        </li>
      </ul>
    `;

    if (active === 'pl') loadPlReport(tabBar);
    else if (active === 'balance-sheet') loadBalanceSheet(tabBar);
    else if (active === 'cash-flow') loadCashFlow(tabBar);
    else if (active === 'trial-balance') loadTrialBalance(tabBar);
    else if (active === 'ar-aging') loadArAging(tabBar);
    else if (active === 'ap-aging') loadApAging(tabBar);
    else if (active === 'analytics') loadAnalytics(tabBar);
    else if (active === 'inv-valuation') loadInvValuation(tabBar);
    else if (active === 'inv-movement') loadInvMovement(tabBar);
    else loadPlReport(tabBar);
  }
};

// ── P&L Report ────────────────────────────────────────────────────────────────

function loadPlReport(tabBar) {
  const today = new Date();
  const firstOfMonth = new Date(today.getFullYear(), today.getMonth(), 1).toISOString().slice(0, 10);
  const todayStr = today.toISOString().slice(0, 10);

  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body">
        <div class="row g-3 align-items-end">
          <div class="col-md-4">
            <label class="form-label">From Date</label>
            <input type="date" class="form-control" id="plFrom" value="${firstOfMonth}">
          </div>
          <div class="col-md-4">
            <label class="form-label">To Date</label>
            <input type="date" class="form-control" id="plTo" value="${todayStr}">
          </div>
          <div class="col-md-4">
            <button class="btn btn-primary w-100" onclick="fetchPlReport()">
              <i class="bi bi-search me-1"></i>Generate
            </button>
          </div>
        </div>
      </div>
    </div>
    <div id="plResult"></div>
  `));
}

async function fetchPlReport() {
  const from = document.getElementById('plFrom').value;
  const to = document.getElementById('plTo').value;
  if (!from || !to) { toast('Please select a date range.', 'warning'); return; }

  const resultEl = document.getElementById('plResult');
  resultEl.innerHTML = `<div class="text-center py-4"><div class="spinner-border text-primary"></div></div>`;

  const data = await api.GetPnL(from, to);
  if (!data) {
    resultEl.innerHTML = `<div class="alert alert-warning">Failed to generate P&L report.</div>`;
    return;
  }

  const revenues = data.revenue || [];
  const expenses = data.expenses || [];

  const revRows = revenues.length === 0
    ? `<tr><td colspan="2" class="text-muted text-center">No revenue records.</td></tr>`
    : revenues.map(r => `
        <tr>
          <td class="ps-4">${r.account_name || '—'}</td>
          <td class="text-end">${formatCurrency(r.amount ?? 0)}</td>
        </tr>
      `).join('');

  const expRows = expenses.length === 0
    ? `<tr><td colspan="2" class="text-muted text-center">No expense records.</td></tr>`
    : expenses.map(e => `
        <tr>
          <td class="ps-4">${e.account_name || '—'}</td>
          <td class="text-end">${formatCurrency(e.amount ?? 0)}</td>
        </tr>
      `).join('');

  resultEl.innerHTML = `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-transparent d-flex justify-content-between align-items-center">
        <div>
          <strong>Profit &amp; Loss Statement</strong>
          <span class="text-muted small ms-2">${formatDate(from)} – ${formatDate(to)}</span>
        </div>
        <button class="btn btn-sm btn-outline-success" onclick="exportPnLExcel()">
          <i class="bi bi-file-earmark-spreadsheet me-1"></i>Export Excel
        </button>
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <tbody>
            <tr class="table-light"><td colspan="2" class="fw-bold">Revenue</td></tr>
            ${revRows}
            <tr class="fw-semibold border-top">
              <td>Total Revenue</td>
              <td class="text-end text-success">${formatCurrency(data.total_revenue ?? 0)}</td>
            </tr>
            <tr class="table-light"><td colspan="2" class="fw-bold">Expenses</td></tr>
            ${expRows}
            <tr class="fw-semibold border-top">
              <td>Total Expenses</td>
              <td class="text-end text-danger">${formatCurrency(data.total_expense ?? 0)}</td>
            </tr>
            <tr class="fw-bold fs-6 ${(data.net_income ?? 0) >= 0 ? 'table-success' : 'table-danger'}">
              <td>Net Income</td>
              <td class="text-end">${formatCurrency(data.net_income ?? 0)}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  `;
}

async function exportPnLExcel() {
  const from = document.getElementById('plFrom').value;
  const to = document.getElementById('plTo').value;
  if (!from || !to) { toast('Please select a date range first.', 'warning'); return; }
  await api.ExportPnLExcel(from, to);
}

// ── Balance Sheet ─────────────────────────────────────────────────────────────

function loadBalanceSheet(tabBar) {
  const todayStr = new Date().toISOString().slice(0, 10);
  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body">
        <div class="row g-3 align-items-end">
          <div class="col-md-4">
            <label class="form-label">As of Date</label>
            <input type="date" class="form-control" id="bsDate" value="${todayStr}">
          </div>
          <div class="col-md-4">
            <button class="btn btn-primary w-100" onclick="fetchBalanceSheet()">
              <i class="bi bi-search me-1"></i>Generate
            </button>
          </div>
        </div>
      </div>
    </div>
    <div id="bsResult"></div>
  `));
}

async function fetchBalanceSheet() {
  const asOf = document.getElementById('bsDate').value;
  if (!asOf) { toast('Please select a date.', 'warning'); return; }

  const resultEl = document.getElementById('bsResult');
  resultEl.innerHTML = `<div class="text-center py-4"><div class="spinner-border text-primary"></div></div>`;

  const data = await api.GetBalanceSheet(asOf);
  if (!data) {
    resultEl.innerHTML = `<div class="alert alert-warning">Failed to generate Balance Sheet.</div>`;
    return;
  }

  function bsSection(sec, colorClass) {
    const accounts = (sec && sec.accounts) || [];
    const rows = accounts.map(i => `
      <tr>
        <td class="ps-4">${i.name || '—'} <span class="text-muted small">${i.code || ''}</span></td>
        <td class="text-end">${formatCurrency(i.amount ?? 0)}</td>
      </tr>
    `).join('');
    return `
      <tr class="table-light"><td colspan="2" class="fw-bold">${sec ? sec.name : ''}</td></tr>
      ${rows || `<tr><td colspan="2" class="text-muted text-center ps-4">No records.</td></tr>`}
      <tr class="fw-semibold border-top">
        <td>Total ${sec ? sec.name : ''}</td>
        <td class="text-end ${colorClass}">${formatCurrency(sec ? (sec.total ?? 0) : 0)}</td>
      </tr>
    `;
  }

  const isBalanced = Math.abs(data.check ?? 0) < 0.01;

  resultEl.innerHTML = `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-transparent d-flex justify-content-between align-items-center">
        <strong>Balance Sheet</strong>
        <div class="d-flex align-items-center gap-2">
          <span class="text-muted small">As of ${formatDate(asOf)}</span>
          <button class="btn btn-sm btn-outline-success" onclick="exportBalanceSheetExcel()">
            <i class="bi bi-file-earmark-spreadsheet me-1"></i>Export Excel
          </button>
        </div>
      </div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <tbody>
            ${bsSection(data.assets, 'text-success')}
            ${bsSection(data.liabilities, 'text-danger')}
            ${bsSection(data.equity, 'text-primary')}
            <tr class="fw-bold ${isBalanced ? 'table-success' : 'table-danger'}">
              <td>Balance Check (Assets = Liab + Equity)</td>
              <td class="text-end">${isBalanced ? 'Balanced ✓' : `Off by ${formatCurrency(data.check ?? 0)}`}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  `;
}

async function exportBalanceSheetExcel() {
  const asOf = document.getElementById('bsDate').value;
  if (!asOf) { toast('Please select a date first.', 'warning'); return; }
  await api.ExportBalanceSheetExcel(asOf);
}

// ── Trial Balance ─────────────────────────────────────────────────────────────

function loadTrialBalance(tabBar) {
  const today = new Date();
  const firstOfMonth = new Date(today.getFullYear(), today.getMonth(), 1).toISOString().slice(0, 10);
  const todayStr = today.toISOString().slice(0, 10);

  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body d-flex flex-wrap gap-3 align-items-end">
        <div>
          <label class="form-label small fw-semibold mb-1">From</label>
          <input type="date" id="tbStart" class="form-control form-control-sm" value="${firstOfMonth}">
        </div>
        <div>
          <label class="form-label small fw-semibold mb-1">To</label>
          <input type="date" id="tbEnd" class="form-control form-control-sm" value="${todayStr}">
        </div>
        <button class="btn btn-primary btn-sm" onclick="generateTrialBalance()">
          <i class="bi bi-play-fill me-1"></i>Generate
        </button>
      </div>
    </div>
    <div id="tbResult"></div>
  `));
}

async function generateTrialBalance() {
  const start = document.getElementById('tbStart')?.value;
  const end   = document.getElementById('tbEnd')?.value;
  if (!start || !end) { toast('Please select a date range.', 'warning'); return; }
  if (start > end)    { toast('Start date must be before end date.', 'warning'); return; }

  const report = await api.GetTrialBalance(start, end);
  if (!report) return;

  const sectionOrder = ['ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE'];
  const sectionLabel = {
    ASSET: 'Assets', LIABILITY: 'Liabilities', EQUITY: 'Equity',
    REVENUE: 'Revenue', EXPENSE: 'Expenses',
  };

  const grouped = {};
  sectionOrder.forEach(s => { grouped[s] = []; });
  (report.rows || []).forEach(r => {
    if (grouped[r.section] !== undefined) grouped[r.section].push(r);
  });

  let bodyHtml = '';
  sectionOrder.forEach(section => {
    const rows = grouped[section];
    if (!rows.length) return;

    let secDebit = 0, secCredit = 0;
    const rowsHtml = rows.map(r => {
      secDebit  += r.debit_balance  ?? 0;
      secCredit += r.credit_balance ?? 0;
      return `
        <tr>
          <td class="text-muted ps-3" style="width:110px">${r.account_code}</td>
          <td>${r.account_name}</td>
          <td class="text-end font-monospace">${r.debit_balance  > 0 ? formatCurrency(r.debit_balance)  : ''}</td>
          <td class="text-end font-monospace">${r.credit_balance > 0 ? formatCurrency(r.credit_balance) : ''}</td>
        </tr>`;
    }).join('');

    bodyHtml += `
      <tr class="table-light">
        <td colspan="2" class="fw-semibold text-uppercase small">${sectionLabel[section]}</td>
        <td></td><td></td>
      </tr>
      ${rowsHtml}
      <tr class="border-top fw-semibold">
        <td colspan="2" class="text-end small text-muted">Subtotal — ${sectionLabel[section]}</td>
        <td class="text-end font-monospace">${secDebit  > 0 ? formatCurrency(secDebit)  : ''}</td>
        <td class="text-end font-monospace">${secCredit > 0 ? formatCurrency(secCredit) : ''}</td>
      </tr>
      <tr><td colspan="4" class="py-0"></td></tr>`;
  });

  const balanced = Math.abs(report.total_debit - report.total_credit) < 0.01;
  const checkBadge = balanced
    ? `<span class="badge bg-success ms-2"><i class="bi bi-check-circle me-1"></i>Balanced</span>`
    : `<span class="badge bg-danger ms-2"><i class="bi bi-exclamation-triangle me-1"></i>Out of balance by ${formatCurrency(Math.abs(report.total_debit - report.total_credit))}</span>`;

  document.getElementById('tbResult').innerHTML = `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-transparent fw-semibold d-flex justify-content-between align-items-center flex-wrap gap-2">
        <span>
          Trial Balance
          ${checkBadge}
        </span>
        <span class="text-muted small">${formatDate(start)} – ${formatDate(end)}</span>
      </div>
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0 small">
          <thead class="table-dark">
            <tr>
              <th style="width:110px">Account Code</th>
              <th>Account Name</th>
              <th class="text-end">Debit</th>
              <th class="text-end">Credit</th>
            </tr>
          </thead>
          <tbody>${bodyHtml}</tbody>
          <tfoot class="table-dark fw-bold">
            <tr>
              <td colspan="2">Grand Total</td>
              <td class="text-end font-monospace">${formatCurrency(report.total_debit)}</td>
              <td class="text-end font-monospace">${formatCurrency(report.total_credit)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
      <div class="card-footer text-muted small">
        Generated ${report.generated_at} &nbsp;·&nbsp; PFRS-aligned · Period Trial Balance
      </div>
    </div>`;
}

// ── AR Aging ──────────────────────────────────────────────────────────────────

async function loadArAging(tabBar) {
  const rows = await api.GetARAgingReport();
  if (!rows) {
    showView(wrapReports(tabBar, `<div class="alert alert-warning">Failed to load AR Aging report.</div>`));
    return;
  }
  if (!Array.isArray(rows)) {
    showView(wrapReports(tabBar, `<div class="text-center py-5 text-muted">No outstanding receivables.</div>`));
    return;
  }

  const totals = { current: 0, d30: 0, d60: 0, d90: 0, over90: 0, balance: 0 };

  const tableRows = rows.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No outstanding receivables.</td></tr>`
    : rows.map(r => {
        totals.current += r.current ?? 0;
        totals.d30    += r.days_30 ?? 0;
        totals.d60    += r.days_60 ?? 0;
        totals.d90    += r.days_90 ?? 0;
        totals.over90 += r.over_90 ?? 0;
        totals.balance += r.balance ?? 0;
        return `
          <tr>
            <td class="fw-semibold">${r.customer_name || '—'}</td>
            <td>${r.invoice_number || '—'}</td>
            <td>${formatDate(r.invoice_date)}</td>
            <td>${formatDate(r.due_date)}</td>
            <td class="text-end">${formatCurrency(r.current ?? 0)}</td>
            <td class="text-end">${formatCurrency(r.days_30 ?? 0)}</td>
            <td class="text-end">${formatCurrency(r.days_60 ?? 0)}</td>
            <td class="text-end">${formatCurrency(r.days_90 ?? 0)}</td>
            <td class="text-end text-danger fw-semibold">${formatCurrency(r.over_90 ?? 0)}</td>
            <td class="text-end fw-semibold">${formatCurrency(r.balance ?? 0)}</td>
          </tr>
        `;
      }).join('');

  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-transparent fw-semibold d-flex justify-content-between align-items-center">
        <span>AR Aging Report</span>
        <span class="text-muted small">As of ${formatDate(new Date().toISOString().slice(0,10))}</span>
      </div>
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0 small">
          <thead class="table-light">
            <tr>
              <th>Customer</th>
              <th>Invoice #</th>
              <th>Invoice Date</th>
              <th>Due Date</th>
              <th class="text-end">Current</th>
              <th class="text-end">1-30 Days</th>
              <th class="text-end">31-60 Days</th>
              <th class="text-end">61-90 Days</th>
              <th class="text-end">Over 90</th>
              <th class="text-end">Balance</th>
            </tr>
          </thead>
          <tbody>${tableRows}</tbody>
          <tfoot class="table-light fw-bold">
            <tr>
              <td colspan="4">Total</td>
              <td class="text-end">${formatCurrency(totals.current)}</td>
              <td class="text-end">${formatCurrency(totals.d30)}</td>
              <td class="text-end">${formatCurrency(totals.d60)}</td>
              <td class="text-end">${formatCurrency(totals.d90)}</td>
              <td class="text-end text-danger">${formatCurrency(totals.over90)}</td>
              <td class="text-end">${formatCurrency(totals.balance)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  `));
}

// ── AP Aging ──────────────────────────────────────────────────────────────────

async function loadApAging(tabBar) {
  const rows = await api.GetAPAgingReport();
  if (!rows) {
    showView(wrapReports(tabBar, `<div class="alert alert-warning">Failed to load AP Aging report.</div>`));
    return;
  }
  if (!Array.isArray(rows)) {
    showView(wrapReports(tabBar, `<div class="text-center py-5 text-muted">No outstanding payables.</div>`));
    return;
  }

  const totals = { current: 0, d30: 0, d60: 0, d90: 0, over90: 0, balance: 0 };

  const tableRows = rows.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No outstanding payables.</td></tr>`
    : rows.map(r => {
        totals.current += r.current ?? 0;
        totals.d30    += r.days_30 ?? 0;
        totals.d60    += r.days_60 ?? 0;
        totals.d90    += r.days_90 ?? 0;
        totals.over90 += r.over_90 ?? 0;
        totals.balance += r.balance ?? 0;
        return `
          <tr>
            <td class="fw-semibold">${r.supplier_name || '—'}</td>
            <td>${r.invoice_number || '—'}</td>
            <td>${formatDate(r.invoice_date)}</td>
            <td>${formatDate(r.due_date)}</td>
            <td class="text-end">${formatCurrency(r.current ?? 0)}</td>
            <td class="text-end">${formatCurrency(r.days_30 ?? 0)}</td>
            <td class="text-end">${formatCurrency(r.days_60 ?? 0)}</td>
            <td class="text-end">${formatCurrency(r.days_90 ?? 0)}</td>
            <td class="text-end text-danger fw-semibold">${formatCurrency(r.over_90 ?? 0)}</td>
            <td class="text-end fw-semibold">${formatCurrency(r.balance ?? 0)}</td>
          </tr>
        `;
      }).join('');

  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-transparent fw-semibold d-flex justify-content-between align-items-center">
        <span>AP Aging Report</span>
        <span class="text-muted small">As of ${formatDate(new Date().toISOString().slice(0,10))}</span>
      </div>
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0 small">
          <thead class="table-light">
            <tr>
              <th>Supplier</th>
              <th>Invoice #</th>
              <th>Invoice Date</th>
              <th>Due Date</th>
              <th class="text-end">Current</th>
              <th class="text-end">1-30 Days</th>
              <th class="text-end">31-60 Days</th>
              <th class="text-end">61-90 Days</th>
              <th class="text-end">Over 90</th>
              <th class="text-end">Balance</th>
            </tr>
          </thead>
          <tbody>${tableRows}</tbody>
          <tfoot class="table-light fw-bold">
            <tr>
              <td colspan="4">Total</td>
              <td class="text-end">${formatCurrency(totals.current)}</td>
              <td class="text-end">${formatCurrency(totals.d30)}</td>
              <td class="text-end">${formatCurrency(totals.d60)}</td>
              <td class="text-end">${formatCurrency(totals.d90)}</td>
              <td class="text-end text-danger">${formatCurrency(totals.over90)}</td>
              <td class="text-end">${formatCurrency(totals.balance)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  `));
}

// ── Analytics ─────────────────────────────────────────────────────────────────

let _analyticsCharts = [];

function _kpiCard(label, value, badge, badgeColor, progress, benchmark) {
  const bar = progress != null
    ? `<div class="progress mt-2" style="height:4px;"><div class="progress-bar bg-${badgeColor}" style="width:${Math.min(progress, 100)}%"></div></div>`
    : '';
  return `
    <div class="col-6 col-md-4 col-xl">
      <div class="card border-0 shadow-sm h-100">
        <div class="card-body py-3 px-3">
          <div class="d-flex justify-content-between align-items-start mb-2">
            <small class="text-muted fw-bold text-uppercase" style="font-size:11px;">${label}</small>
            <span class="badge bg-${badgeColor}">${badge}</span>
          </div>
          <h3 class="fw-bold text-${badgeColor} mb-0">${value}</h3>
          ${bar}
          <small class="text-muted" style="font-size:10px;">${benchmark}</small>
        </div>
      </div>
    </div>`;
}

function _analyticsDateRange() {
  const preset = (document.getElementById('analyticsPreset') || {}).value || 'this-year';
  const now = new Date();
  const today = now.toISOString().slice(0, 10);
  const y = now.getFullYear();
  const m = now.getMonth();

  if (preset === 'this-month') {
    const from = `${y}-${String(m + 1).padStart(2, '0')}-01`;
    return { from, to: today };
  }
  if (preset === 'last-month') {
    const lastDay = new Date(y, m, 0);
    const firstDay = new Date(y, m - 1, 1);
    return { from: firstDay.toISOString().slice(0, 10), to: lastDay.toISOString().slice(0, 10) };
  }
  if (preset === 'last-30') {
    return { from: new Date(now - 30 * 86400000).toISOString().slice(0, 10), to: today };
  }
  if (preset === 'last-90') {
    return { from: new Date(now - 90 * 86400000).toISOString().slice(0, 10), to: today };
  }
  if (preset === 'all-time') {
    return { from: '', to: today };
  }
  // year value (e.g. "2026")
  const yr = parseInt(preset, 10);
  if (!isNaN(yr)) {
    return { from: `${yr}-01-01`, to: yr === y ? today : `${yr}-12-31` };
  }
  return { from: `${y}-01-01`, to: today };
}

function loadAnalytics(tabBar) {
  const currentYear = new Date().getFullYear();
  const yearOptions = [currentYear, currentYear - 1, currentYear - 2]
    .map(y => `<option value="${y}">Year ${y}</option>`)
    .join('');

  showView(wrapReports(tabBar, `
    <div class="d-flex align-items-center gap-2 mb-4 flex-wrap">
      <label class="form-label mb-0 fw-semibold me-1">Period:</label>
      <select class="form-select form-select-sm" style="width:auto" id="analyticsPreset" onchange="fetchAnalytics()">
        <optgroup label="Quick Ranges">
          <option value="this-month">This Month</option>
          <option value="last-month">Last Month</option>
          <option value="last-30">Last 30 Days</option>
          <option value="last-90">Last 90 Days</option>
          <option value="all-time">All Time</option>
        </optgroup>
        <optgroup label="By Year">
          <option value="${currentYear}" selected>Year ${currentYear}</option>
          ${yearOptions.replace(`<option value="${currentYear}">Year ${currentYear}</option>`, '')}
        </optgroup>
      </select>
    </div>
    <div id="analyticsResult">
      <div class="text-center py-5"><div class="spinner-border text-primary"></div></div>
    </div>
  `));

  fetchAnalytics();
}

async function fetchAnalytics() {
  const { from, to } = _analyticsDateRange();

  _analyticsCharts.forEach(c => { try { c.destroy(); } catch(e) {} });
  _analyticsCharts = [];

  const data = await api.GetAnalytics(from, to);
  if (!data) { toast('Failed to load analytics.', 'danger'); return; }

  const MONTHS = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
  const fmtEgg = v => '₱' + (parseFloat(v) || 0).toFixed(4);

  const hdpColor  = data.hdp >= 80 ? 'success' : (data.hdp >= 60 ? 'warning' : 'danger');
  const fcrColor  = data.fcr <= 1.8 ? 'success' : (data.fcr <= 2.2 ? 'warning' : 'danger');
  const mortColor = data.mortality_rate <= 5 ? 'success' : (data.mortality_rate <= 7 ? 'warning' : 'danger');
  const rejColor  = data.reject_rate <= 3 ? 'success' : (data.reject_rate <= 6 ? 'warning' : 'danger');
  const aephColor = data.avg_eggs_per_hen >= 0.8 ? 'success' : (data.avg_eggs_per_hen >= 0.6 ? 'warning' : 'danger');

  const houseRows = (data.house_stats || []).map(h => {
    const hHdpCls = h.hdp >= 80 ? 'text-success' : (h.hdp >= 60 ? 'text-warning' : 'text-danger');
    const hFcrCls = h.fcr <= 1.8 ? 'text-success' : (h.fcr <= 2.2 ? 'text-warning' : 'text-danger');
    return `<tr>
      <td class="ps-3 fw-bold">${h.name}</td>
      <td>${(h.birds ?? 0).toLocaleString()}</td>
      <td>${h.days ?? 0}</td>
      <td>${(h.good_eggs ?? 0).toLocaleString()}</td>
      <td class="text-danger">${(h.cracked ?? 0).toLocaleString()}</td>
      <td>${(h.feed_kg ?? 0).toFixed(1)}</td>
      <td class="fw-bold ${hHdpCls}">${h.hdp ?? 0}%</td>
      <td class="fw-bold ${hFcrCls}">${h.fcr ?? 0}</td>
    </tr>`;
  }).join('');

  const resultEl = document.getElementById('analyticsResult');
  if (!resultEl) return;

  if (data.days_logged === 0) {
    resultEl.innerHTML = `
      <div class="text-center py-5 text-muted">
        <i class="bi bi-bar-chart d-block mb-3" style="font-size:3rem;opacity:0.3;"></i>
        <h5 class="fw-bold">No production data for this period</h5>
        <p>Start recording daily logs for your layer houses to see analytics here.</p>
      </div>`;
    return;
  }

  resultEl.innerHTML = `
    <div class="row g-3 mb-4">
      ${_kpiCard('Hen-Day Production', data.hdp + '%', data.hdp >= 80 ? 'Excellent' : (data.hdp >= 60 ? 'Good' : 'Low'), hdpColor, data.hdp, 'Benchmark: 60–75% avg · 80%+ excellent')}
      ${_kpiCard('Feed Conv. Ratio', data.fcr + ' kg/doz', data.fcr <= 1.8 ? 'Excellent' : (data.fcr <= 2.2 ? 'Good' : 'High'), fcrColor, null, 'Benchmark: ~2.0 good · &lt;1.8 excellent')}
      ${_kpiCard('Mortality Rate', data.mortality_rate + '%', data.mortality_rate <= 5 ? 'Good' : (data.mortality_rate <= 7 ? 'Watch' : 'High'), mortColor, data.mortality_rate, 'Benchmark: &lt;5% good · &lt;7% acceptable')}
      ${_kpiCard('Reject Rate', data.reject_rate + '%', data.reject_rate <= 3 ? 'Good' : (data.reject_rate <= 6 ? 'Watch' : 'High'), rejColor, data.reject_rate, 'Benchmark: &lt;3% target')}
      ${_kpiCard('Eggs / Hen / Day', data.avg_eggs_per_hen, data.avg_eggs_per_hen >= 0.8 ? 'Excellent' : (data.avg_eggs_per_hen >= 0.6 ? 'Good' : 'Low'), aephColor, data.avg_eggs_per_hen * 100, 'Benchmark: 0.8–0.9 target')}
    </div>

    <div class="row g-3 mb-4">
      <div class="col-lg-8">
        <div class="card border-0 shadow-sm">
          <div class="card-header bg-white border-0 py-3 fw-bold">
            <i class="bi bi-graph-up me-2 text-primary"></i>Daily Egg Production (last 30 days)
          </div>
          <div class="card-body"><canvas id="eggTrendChart" height="100"></canvas></div>
        </div>
      </div>
      <div class="col-lg-4">
        <div class="card border-0 shadow-sm">
          <div class="card-header bg-white border-0 py-3 fw-bold">
            <i class="bi bi-pie-chart me-2 text-warning"></i>Egg Quality Split
          </div>
          <div class="card-body d-flex flex-column align-items-center justify-content-center">
            <canvas id="qualityChart" height="160"></canvas>
            <div class="mt-3 text-center">
              <span class="me-3"><span class="badge bg-success me-1">&nbsp;</span>Good: ${(data.total_good ?? 0).toLocaleString()}</span>
              <span><span class="badge bg-danger me-1">&nbsp;</span>Rejects: ${(data.total_cracked ?? 0).toLocaleString()}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="row g-3 mb-4">
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center py-3">
          <div class="fw-bold text-success" style="font-size:1.4rem;">${(data.total_good ?? 0).toLocaleString()}</div>
          <small class="text-muted">Total Good Eggs</small>
          <div class="text-muted small">${data.days_logged > 0 ? Math.round(data.total_good / data.days_logged).toLocaleString() : '—'} eggs/day avg</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center py-3">
          <div class="fw-bold text-primary" style="font-size:1.4rem;">${(data.total_feed_kg ?? 0).toFixed(1)} kg</div>
          <small class="text-muted">Total Feed Consumed</small>
          <div class="text-muted small">${data.days_logged > 0 ? (data.total_feed_kg / data.days_logged).toFixed(1) : '—'} kg/day avg</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center py-3">
          <div class="fw-bold text-dark" style="font-size:1.4rem;">${data.days_logged ?? 0}</div>
          <small class="text-muted">Days Logged</small>
          <div class="text-muted small">${data.total_good > 0 ? Math.round(data.total_good / 30).toLocaleString() : '—'} trays total</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card border-0 shadow-sm text-center py-3">
          <div class="fw-bold text-warning" style="font-size:1.4rem;">${(data.total_birds ?? 0).toLocaleString()}</div>
          <small class="text-muted">Active Birds</small>
          <div class="text-muted small">${(data.total_mortality ?? 0).toLocaleString()} mortality total</div>
        </div>
      </div>
    </div>

    ${houseRows ? `
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-header bg-white border-0 py-3 fw-bold">
        <i class="bi bi-building me-2"></i>Per-House Breakdown
      </div>
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0 small">
          <thead class="table-light">
            <tr>
              <th class="ps-3">House / Batch</th><th>Birds</th><th>Days</th>
              <th>Good Eggs</th><th>Rejects</th><th>Feed (kg)</th><th>HDP %</th><th>FCR</th>
            </tr>
          </thead>
          <tbody>${houseRows}</tbody>
        </table>
      </div>
    </div>` : ''}

    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body py-3">
        <p class="fw-bold small text-uppercase text-muted mb-2">Industry Benchmark Guide</p>
        <div class="row g-2 small">
          <div class="col-md-4">
            <strong>Hen-Day Production (HDP)</strong><br>
            <span class="text-danger">● Below 60%</span> — Investigate feed, health, lighting<br>
            <span class="text-warning">● 60–79%</span> — Average, room to improve<br>
            <span class="text-success">● 80%+</span> — Excellent performance
          </div>
          <div class="col-md-4">
            <strong>Feed Conversion Ratio (FCR)</strong><br>
            <span class="text-success">● &lt;1.8 kg/dozen</span> — Excellent efficiency<br>
            <span class="text-warning">● 1.8–2.2 kg/dozen</span> — Industry average<br>
            <span class="text-danger">● &gt;2.2 kg/dozen</span> — Review feed program
          </div>
          <div class="col-md-4">
            <strong>Mortality Rate</strong><br>
            <span class="text-success">● &lt;5%</span> — Good biosecurity<br>
            <span class="text-warning">● 5–7%</span> — Monitor closely<br>
            <span class="text-danger">● &gt;7%</span> — Investigate disease/housing
          </div>
        </div>
      </div>
    </div>

    <div class="row g-3 mb-4">
      <div class="col-lg-8">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-header bg-transparent fw-semibold"><i class="bi bi-bar-chart me-2 text-primary"></i>Sales by Month</div>
          <div class="card-body"><canvas id="monthlySalesChart" height="120"></canvas></div>
        </div>
      </div>
      <div class="col-lg-4">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-header bg-transparent fw-semibold"><i class="bi bi-pie-chart me-2"></i>Egg Size Mix</div>
          <div class="card-body d-flex align-items-center justify-content-center">
            <canvas id="eggSizePieChart" height="220"></canvas>
          </div>
        </div>
      </div>
    </div>

    <div class="card border-0 shadow-sm">
      <div class="card-header bg-transparent fw-semibold"><i class="bi bi-trophy me-2 text-warning"></i>Top Customers</div>
      <div class="table-responsive">
        <table class="table align-middle mb-0">
          <thead class="table-light">
            <tr><th>Rank</th><th>Customer</th><th class="text-end">Total Sales</th></tr>
          </thead>
          <tbody id="topCustomersBody"></tbody>
        </table>
      </div>
    </div>

    ${(data.feed_cost_total ?? 0) === 0 ? `
      <div class="alert alert-info d-flex align-items-center gap-2 mt-3">
        <i class="bi bi-info-circle-fill"></i>
        No feed purchase data found for this period.
        Record AP Invoices with category <strong>Feeds</strong> to enable Cost Per Egg analysis.
      </div>` : `
    <div class="mt-4">
      <h6 class="fw-bold text-muted text-uppercase mb-3" style="font-size:11px;letter-spacing:.08em">
        <i class="bi bi-egg-fried me-1"></i>Cost Per Egg Analysis
      </h6>

      <div class="row g-3 mb-3">
        <div class="col-6 col-md-3">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body text-center py-3">
              <div class="small text-muted fw-bold mb-1">COST / EGG</div>
              <div class="fs-4 fw-bold">${fmtEgg(data.cost_per_egg)}</div>
              <span class="badge bg-${(data.cost_per_egg > data.revenue_per_egg) ? 'danger' : 'success'}">₱/egg</span>
            </div>
          </div>
        </div>
        <div class="col-6 col-md-3">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body text-center py-3">
              <div class="small text-muted fw-bold mb-1">COST / DOZEN</div>
              <div class="fs-4 fw-bold">${formatCurrency(data.cost_per_dozen)}</div>
              <span class="badge bg-${(data.cost_per_egg > data.revenue_per_egg) ? 'danger' : 'success'}">₱/doz</span>
            </div>
          </div>
        </div>
        <div class="col-6 col-md-3">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body text-center py-3">
              <div class="small text-muted fw-bold mb-1">REVENUE / EGG</div>
              <div class="fs-4 fw-bold">${fmtEgg(data.revenue_per_egg)}</div>
              <span class="badge bg-success">₱/egg</span>
            </div>
          </div>
        </div>
        <div class="col-6 col-md-3">
          <div class="card border-0 shadow-sm h-100">
            <div class="card-body text-center py-3">
              <div class="small text-muted fw-bold mb-1">GROSS MARGIN / EGG</div>
              <div class="fs-4 fw-bold ${data.gross_margin_per_egg >= 0 ? 'text-success' : 'text-danger'}">
                ${fmtEgg(data.gross_margin_per_egg)}
              </div>
              <span class="badge bg-${data.gross_margin_per_egg >= 0 ? 'success' : 'danger'}">₱/egg</span>
            </div>
          </div>
        </div>
      </div>

      <div class="card border-0 shadow-sm mb-3">
        <div class="card-body">
          <div class="fw-bold small text-muted mb-2">Cost vs Revenue per Egg (Monthly Trend)</div>
          <canvas id="cpeLineChart" height="80"></canvas>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-white border-bottom py-2 fw-bold small">Monthly Breakdown</div>
        <div class="table-responsive">
          <table class="table table-sm align-middle mb-0" style="font-size:12px">
            <thead class="table-light">
              <tr>
                <th>Month</th><th class="text-end">Feed Cost</th>
                <th class="text-end">Kg Bought</th><th class="text-end">Kg Used</th>
                <th class="text-end">Good Eggs</th><th class="text-end">Cost/Egg</th>
                <th class="text-end">Rev/Egg</th><th class="text-end">Margin</th>
              </tr>
            </thead>
            <tbody>
              ${(data.monthly_cpe || []).map(p => `<tr>
                <td>${p.month}</td>
                <td class="text-end">${formatCurrency(p.feed_cost)}</td>
                <td class="text-end">${(p.feed_kg_bought||0).toLocaleString('en-PH',{maximumFractionDigits:1})}</td>
                <td class="text-end">${(p.feed_kg_used||0).toLocaleString('en-PH',{maximumFractionDigits:1})}</td>
                <td class="text-end">${(p.good_eggs||0).toLocaleString()}</td>
                <td class="text-end">${fmtEgg(p.cost_per_egg)}</td>
                <td class="text-end">${fmtEgg(p.revenue_per_egg)}</td>
                <td class="text-end fw-semibold ${p.margin >= 0 ? 'text-success' : 'text-danger'}">
                  ${formatCurrency(p.margin)}
                </td>
              </tr>`).join('')}
            </tbody>
          </table>
        </div>
      </div>
    </div>`}
  `;

  // Egg production trend chart
  const trendCtx = document.getElementById('eggTrendChart')?.getContext('2d');
  if (trendCtx) {
    _analyticsCharts.push(new Chart(trendCtx, {
      type: 'bar',
      data: {
        labels: data.trend_labels || [],
        datasets: [
          { label: 'Good Eggs', data: data.trend_good || [], backgroundColor: 'rgba(25,135,84,0.7)', borderColor: 'rgb(25,135,84)', borderWidth: 1, yAxisID: 'y' },
          { label: 'Feed (kg)', data: data.trend_feed || [], type: 'line', borderColor: 'rgb(255,193,7)', backgroundColor: 'rgba(255,193,7,0.1)', borderWidth: 2, pointRadius: 3, tension: 0.3, yAxisID: 'y1' }
        ]
      },
      options: {
        responsive: true,
        interaction: { mode: 'index', intersect: false },
        scales: {
          y:  { position: 'left',  title: { display: true, text: 'Eggs' } },
          y1: { position: 'right', title: { display: true, text: 'Feed (kg)' }, grid: { drawOnChartArea: false } }
        }
      }
    }));
  }

  // Quality donut
  const qualCtx = document.getElementById('qualityChart')?.getContext('2d');
  if (qualCtx && (data.total_good + data.total_cracked) > 0) {
    _analyticsCharts.push(new Chart(qualCtx, {
      type: 'doughnut',
      data: {
        labels: ['Good Eggs', 'Rejects'],
        datasets: [{ data: [data.total_good, data.total_cracked], backgroundColor: ['rgba(25,135,84,0.8)', 'rgba(220,53,69,0.8)'], borderWidth: 2 }]
      },
      options: { responsive: true, plugins: { legend: { display: false } }, cutout: '65%' }
    }));
  }

  // Monthly Sales bar chart
  const salesCtx = document.getElementById('monthlySalesChart')?.getContext('2d');
  if (salesCtx) {
    const salesByMonth = Array(12).fill(0);
    (data.monthly_sales || []).forEach(m => {
      const idx = parseInt((m.month || '').split('-')[1] || '0', 10) - 1;
      if (idx >= 0 && idx < 12) salesByMonth[idx] = m.total ?? 0;
    });
    _analyticsCharts.push(new Chart(salesCtx, {
      type: 'bar',
      data: { labels: MONTHS, datasets: [{ label: 'Sales', data: salesByMonth, backgroundColor: 'rgba(59,130,246,0.7)', borderRadius: 4 }] },
      options: {
        responsive: true,
        plugins: { legend: { display: false }, tooltip: { callbacks: { label: ctx => formatCurrency(ctx.parsed.y) } } },
        scales: { y: { ticks: { callback: v => formatCurrency(v) } } }
      }
    }));
  }

  // Egg size pie (filter out zeros)
  const pieCtx = document.getElementById('eggSizePieChart')?.getContext('2d');
  if (pieCtx) {
    const eggMix = data.egg_size_mix || {};
    const pieLabels = Object.keys(eggMix).filter(k => eggMix[k] > 0);
    _analyticsCharts.push(new Chart(pieCtx, {
      type: 'pie',
      data: {
        labels: pieLabels,
        datasets: [{ data: pieLabels.map(k => eggMix[k]), backgroundColor: ['#6366f1','#22c55e','#f59e0b','#ef4444','#3b82f6','#a855f7','#ec4899','#14b8a6'] }]
      },
      options: { responsive: true, plugins: { legend: { position: 'bottom' } } }
    }));
  }

  // Top Customers table
  const tbody = document.getElementById('topCustomersBody');
  if (tbody) {
    const tc = data.top_customers || [];
    tbody.innerHTML = tc.length === 0
      ? `<tr><td colspan="3" class="text-center text-muted py-3">No sales data for this period.</td></tr>`
      : tc.map((c, i) => `
          <tr>
            <td><span class="badge bg-secondary">#${i + 1}</span></td>
            <td>${c.name || '—'}</td>
            <td class="text-end fw-semibold">${formatCurrency(c.total ?? 0)}</td>
          </tr>`).join('');
  }

  // CPE line chart
  if ((data.feed_cost_total ?? 0) > 0 && (data.monthly_cpe || []).length > 0) {
    const cpeCtx = document.getElementById('cpeLineChart');
    if (cpeCtx) {
      _analyticsCharts.push(new Chart(cpeCtx, {
        type: 'line',
        data: {
          labels: data.monthly_cpe.map(p => p.month),
          datasets: [
            {
              label: 'Cost/Egg (₱)',
              data: data.monthly_cpe.map(p => p.cost_per_egg),
              borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.08)',
              tension: 0.3, pointRadius: 4, fill: true,
            },
            {
              label: 'Revenue/Egg (₱)',
              data: data.monthly_cpe.map(p => p.revenue_per_egg),
              borderColor: '#22c55e', backgroundColor: 'rgba(34,197,94,0.08)',
              tension: 0.3, pointRadius: 4, fill: true,
            },
          ],
        },
        options: {
          responsive: true,
          interaction: { mode: 'index', intersect: false },
          plugins: { legend: { position: 'top' } },
          scales: {
            y: {
              ticks: { callback: v => '₱' + v.toFixed(4) },
              beginAtZero: true,
            },
          },
        },
      }));
    }
  }
}

// ── Cash Flow Statement (PAS 7) ───────────────────────────────────────────────

function loadCashFlow(tabBar) {
  const today  = new Date();
  const firstOfMonth = new Date(today.getFullYear(), today.getMonth(), 1).toISOString().slice(0, 10);
  const todayStr     = today.toISOString().slice(0, 10);

  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body">
        <div class="row g-3 align-items-end">
          <div class="col-md-4">
            <label class="form-label">From Date</label>
            <input type="date" class="form-control" id="cfFrom" value="${firstOfMonth}">
          </div>
          <div class="col-md-4">
            <label class="form-label">To Date</label>
            <input type="date" class="form-control" id="cfTo" value="${todayStr}">
          </div>
          <div class="col-md-4">
            <button class="btn btn-primary w-100" onclick="fetchCashFlow()">
              <i class="bi bi-search me-1"></i>Generate
            </button>
          </div>
        </div>
      </div>
    </div>
    <div id="cfResult"></div>
  `));
}

async function fetchCashFlow() {
  const from = document.getElementById('cfFrom').value;
  const to   = document.getElementById('cfTo').value;
  if (!from || !to) { toast('Please select a date range.', 'warning'); return; }

  const resultEl = document.getElementById('cfResult');
  resultEl.innerHTML = `<div class="text-center py-4"><div class="spinner-border text-primary"></div></div>`;

  const data = await api.GetCashFlowStatement(from, to);
  if (!data) {
    resultEl.innerHTML = `<div class="alert alert-warning">Failed to generate Cash Flow Statement.</div>`;
    return;
  }

  resultEl.innerHTML = renderCashFlowStatement(data);
}

function renderCashFlowStatement(d) {
  const fmt   = v => formatCurrency(v ?? 0);
  const fmtSgn = v => {
    const n = v ?? 0;
    return `<span class="${n < 0 ? 'text-danger' : (n > 0 ? 'text-success' : 'text-muted')}">${fmt(n)}</span>`;
  };

  function renderSection(sec) {
    if (!sec || !sec.lines || sec.lines.length === 0) {
      return `<tr><td colspan="2" class="text-muted text-center ps-4">No activity.</td></tr>`;
    }
    return sec.lines.map(line => {
      if (line.is_separator) {
        return `<tr><td colspan="2"><hr class="my-1"></td></tr>`;
      }
      const indent  = line.indent || 0;
      const ps      = indent === 0 ? '' : (indent === 1 ? 'ps-4' : 'ps-5');
      const bold    = line.is_total ? 'fw-semibold' : '';
      const amtCell = line.is_total
        ? `<td class="text-end fw-semibold">${fmtSgn(line.amount)}</td>`
        : (line.indent === 0
            ? `<td></td>`
            : `<td class="text-end">${fmtSgn(line.amount)}</td>`);
      return `
        <tr>
          <td class="${ps} ${bold}">${line.description || ''}</td>
          ${amtCell}
        </tr>`;
    }).join('');
  }

  const opNet  = d.operating?.net  ?? 0;
  const invNet = d.investing?.net  ?? 0;
  const finNet = d.financing?.net  ?? 0;
  const netChg = d.net_cash_change ?? 0;
  const openCash  = d.opening_cash  ?? 0;
  const closeCash = d.closing_cash  ?? 0;
  const checkDiff = d.check_diff    ?? 0;

  // cash composition note
  const comp = d.cash_composition || [];
  const compRows = comp.length === 0
    ? `<tr><td colspan="2" class="text-muted">No cash accounts found.</td></tr>`
    : comp.map(c => `
        <tr>
          <td class="ps-3">${c.code} — ${c.name}</td>
          <td class="text-end">${fmt(c.amount)}</td>
        </tr>`).join('');

  // check-diff badge
  const diffBadge = Math.abs(checkDiff) < 0.01
    ? `<span class="badge bg-success ms-2">Balanced ✓</span>`
    : `<span class="badge bg-danger ms-2">Diff: ${fmt(checkDiff)}</span>`;

  return `
    <div class="card border-0 shadow-sm" id="cf-statement">
      <!-- Header -->
      <div class="card-header bg-transparent d-flex justify-content-between align-items-center">
        <div>
          <strong>Statement of Cash Flows</strong>
          <span class="text-muted small ms-2">${formatDate(d.period_start)} – ${formatDate(d.period_end)}</span>
          <span class="badge bg-secondary ms-2">PAS 7 — Indirect Method</span>
          ${diffBadge}
        </div>
        <button class="btn btn-sm btn-outline-secondary d-print-none" onclick="window.print()">
          <i class="bi bi-printer me-1"></i>Print
        </button>
      </div>

      <div class="table-responsive">
        <table class="table align-middle mb-0" style="font-size:.93rem">
          <tbody>

            <!-- ── OPERATING ACTIVITIES ── -->
            <tr class="table-light">
              <td colspan="2" class="fw-bold text-uppercase" style="letter-spacing:.04em">
                <i class="bi bi-gear me-1"></i>Cash Flows from Operating Activities
              </td>
            </tr>
            ${renderSection(d.operating)}
            <tr class="fw-bold border-top border-2" style="background:#f0f4ff">
              <td>Net Cash from Operating Activities</td>
              <td class="text-end">${fmtSgn(opNet)}</td>
            </tr>

            <tr><td colspan="2" style="height:12px"></td></tr>

            <!-- ── INVESTING ACTIVITIES ── -->
            <tr class="table-light">
              <td colspan="2" class="fw-bold text-uppercase" style="letter-spacing:.04em">
                <i class="bi bi-building me-1"></i>Cash Flows from Investing Activities
              </td>
            </tr>
            ${renderSection(d.investing)}
            <tr class="fw-bold border-top border-2" style="background:#f0f4ff">
              <td>Net Cash from Investing Activities</td>
              <td class="text-end">${fmtSgn(invNet)}</td>
            </tr>

            <tr><td colspan="2" style="height:12px"></td></tr>

            <!-- ── FINANCING ACTIVITIES ── -->
            <tr class="table-light">
              <td colspan="2" class="fw-bold text-uppercase" style="letter-spacing:.04em">
                <i class="bi bi-bank me-1"></i>Cash Flows from Financing Activities
              </td>
            </tr>
            ${renderSection(d.financing)}
            <tr class="fw-bold border-top border-2" style="background:#f0f4ff">
              <td>Net Cash from Financing Activities</td>
              <td class="text-end">${fmtSgn(finNet)}</td>
            </tr>

            <tr><td colspan="2" style="height:16px"></td></tr>

            <!-- ── RECONCILIATION ── -->
            <tr class="table-secondary">
              <td colspan="2" class="fw-bold">Cash Reconciliation</td>
            </tr>
            <tr>
              <td class="ps-4">Net Increase / (Decrease) in Cash &amp; Cash Equivalents</td>
              <td class="text-end fw-semibold">${fmtSgn(netChg)}</td>
            </tr>
            <tr>
              <td class="ps-4">Cash &amp; Cash Equivalents — Beginning of Period</td>
              <td class="text-end">${fmt(openCash)}</td>
            </tr>
            <tr class="fw-bold fs-6 ${closeCash >= 0 ? 'table-success' : 'table-danger'}">
              <td>Cash &amp; Cash Equivalents — End of Period</td>
              <td class="text-end">${fmt(closeCash)}</td>
            </tr>

          </tbody>
        </table>
      </div>

      <!-- Note: Cash Composition -->
      <div class="card-body border-top" style="background:#fafbfc">
        <p class="fw-semibold mb-2 small text-uppercase" style="letter-spacing:.04em">
          Note: Composition of Cash &amp; Cash Equivalents — End of Period
        </p>
        <table class="table table-sm mb-0" style="font-size:.88rem">
          <tbody>
            ${compRows}
            <tr class="fw-semibold border-top">
              <td>Total</td>
              <td class="text-end">${fmt(closeCash)}</td>
            </tr>
          </tbody>
        </table>
        <p class="text-muted mt-2 mb-0" style="font-size:.78rem">
          Prepared in accordance with Philippine Accounting Standard 7 (PAS 7) — Statement of Cash Flows,
          using the <strong>indirect method</strong> for operating activities.
          Generated: ${d.generated_at || '—'}
        </p>
      </div>
    </div>
  `;
}

// ── Inventory Valuation Report (Report A) ─────────────────────────────────────

async function loadInvValuation(tabBar) {
  const today = new Date().toISOString().slice(0, 10);

  // api.* returns data directly (null on error) — no .data unwrapping needed
  const [whsList, grpList] = await Promise.all([
    api.ListWarehouses(),
    api.ListItemCategories(),    // bound in app.go as ListItemCategories() — no arg
  ]);

  const whs  = whsList || [];
  const grps = grpList || [];

  const whsOptions = `<option value="">All Warehouses</option>` +
    whs.map(w => `<option value="${w.whs_code}">${w.whs_code} — ${w.whs_name}</option>`).join('');
  const grpOptions = `<option value="0">All Item Groups</option>` +
    grps.map(g => `<option value="${g.itms_grp_cod}">${g.itms_grp_nam}</option>`).join('');

  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body">
        <div class="row g-3 align-items-end">
          <div class="col-md-3">
            <label class="form-label fw-semibold">As of Date</label>
            <input type="date" class="form-control" id="invValAsOf" value="${today}">
          </div>
          <div class="col-md-3">
            <label class="form-label fw-semibold">Warehouse</label>
            <select class="form-select" id="invValWhs">${whsOptions}</select>
          </div>
          <div class="col-md-3">
            <label class="form-label fw-semibold">Item Group</label>
            <select class="form-select" id="invValGrp">${grpOptions}</select>
          </div>
          <div class="col-md-3">
            <button class="btn btn-primary w-100" onclick="fetchInvValuation()">
              <i class="bi bi-search me-1"></i>Generate
            </button>
          </div>
        </div>
      </div>
    </div>
    <div id="invValResult"></div>
  `));

  // Auto-generate for today on first load
  fetchInvValuation();
}

async function fetchInvValuation() {
  const asOf    = document.getElementById('invValAsOf')?.value || '';
  const whsCode = document.getElementById('invValWhs')?.value || '';
  const grpCod  = parseInt(document.getElementById('invValGrp')?.value || '0', 10);

  const el = document.getElementById('invValResult');
  if (!el) return;
  el.innerHTML = `<div class="text-center py-5"><div class="spinner-border text-primary"></div><div class="mt-2 text-muted">Generating report…</div></div>`;

  // api.* returns data directly; null means the call failed (toast shown by api proxy)
  const d = await api.GetInventoryValuationReport(asOf, whsCode, grpCod);
  if (!d) {
    el.innerHTML = `<div class="alert alert-warning"><i class="bi bi-exclamation-triangle me-1"></i>Failed to load inventory valuation.</div>`;
    return;
  }

  const groups = d.groups || [];

  if (groups.length === 0) {
    el.innerHTML = `<div class="alert alert-info"><i class="bi bi-info-circle me-1"></i>No inventory records found for the selected criteria.</div>`;
    return;
  }

  const historicalBanner = d.is_historical
    ? `<div class="alert alert-info d-flex align-items-center gap-2 py-2 mb-3" role="alert">
         <i class="bi bi-clock-history fs-5"></i>
         <small><strong>Historical mode:</strong> Values reconstructed from the OIVL ledger as of <strong>${d.as_of}</strong>.
         Verify opening balances if data was imported directly.</small>
       </div>`
    : '';

  const groupRows = groups.map((g, gi) => {
    const collapseId = `invValGrp_${gi}`;
    const itemRows = (g.rows || []).map(r => `
      <tr>
        <td class="ps-4 text-muted small">${r.item_code}</td>
        <td>${r.item_name}</td>
        <td>${r.warehouse || '<span class="text-muted">—</span>'}</td>
        <td class="text-muted small">${r.uom || '—'}</td>
        <td class="text-end">${formatQty(r.qty_on_hand)}</td>
        <td class="text-end">${formatCurrency(r.unit_cost)}</td>
        <td class="text-end fw-semibold">${formatCurrency(r.total_value)}</td>
        <td class="text-end text-muted small">${(r.pct_of_total || 0).toFixed(2)}%</td>
      </tr>
    `).join('');

    return `
      <tr class="table-secondary" style="cursor:pointer;" onclick="document.getElementById('${collapseId}').classList.toggle('d-none')">
        <td colspan="4" class="fw-bold">
          <i class="bi bi-chevron-down me-1 small"></i>${g.category_name}
        </td>
        <td class="text-end fw-bold">${formatQty(g.sub_total_qty)}</td>
        <td></td>
        <td class="text-end fw-bold text-primary">${formatCurrency(g.sub_total_value)}</td>
        <td></td>
      </tr>
      <tbody id="${collapseId}">${itemRows}</tbody>
    `;
  }).join('');

  el.innerHTML = `
    ${historicalBanner}
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-white d-flex justify-content-between align-items-center border-0 py-3">
        <div>
          <span class="fw-bold">Inventory Valuation</span>
          <span class="ms-2 badge bg-secondary">${d.is_historical ? 'Historical' : 'Current'}</span>
          <span class="ms-2 text-muted small">As of ${d.as_of}</span>
        </div>
        <small class="text-muted">Generated ${d.generated_at}</small>
      </div>
      <div class="card-body p-0">
        <div class="table-responsive">
          <table class="table table-hover table-sm mb-0 align-middle">
            <thead class="table-light">
              <tr>
                <th>Item Code</th>
                <th>Item Name</th>
                <th>Warehouse</th>
                <th>UoM</th>
                <th class="text-end">Qty on Hand</th>
                <th class="text-end">Unit Cost</th>
                <th class="text-end">Total Value</th>
                <th class="text-end">% of Total</th>
              </tr>
            </thead>
            <tbody>${groupRows}</tbody>
            <tfoot class="table-dark">
              <tr>
                <td colspan="4" class="fw-bold">GRAND TOTAL</td>
                <td class="text-end fw-bold">${formatQty(d.total_qty)}</td>
                <td></td>
                <td class="text-end fw-bold fs-6">${formatCurrency(d.total_value)}</td>
                <td class="text-end fw-bold">100%</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>
  `;
}

// ── Inventory Movement Report (Report B) ──────────────────────────────────────

async function loadInvMovement(tabBar) {
  const today        = new Date().toISOString().slice(0, 10);
  const firstOfMonth = new Date(new Date().getFullYear(), new Date().getMonth(), 1).toISOString().slice(0, 10);

  const transTypes = [
    { v: '',           l: 'All Types' },
    { v: 'GR',         l: 'Goods Receipt (GR)' },
    { v: 'GI',         l: 'Goods Issue (GI)' },
    { v: 'DR',         l: 'Purchase Receipt (DR)' },
    { v: 'DO',         l: 'Sales Delivery (DO)' },
    { v: 'WO_ISSUE',   l: 'WO Component Issue' },
    { v: 'WO_RECEIPT', l: 'WO Completion Receipt' },
    { v: 'GR_CANCEL',  l: 'GR Cancellation' },
    { v: 'GI_CANCEL',  l: 'GI Cancellation' },
    { v: 'DR_CANCEL',  l: 'DR Cancellation' },
    { v: 'DO_CANCEL',  l: 'DO Cancellation' },
    { v: 'WO_CANCEL',  l: 'WO Cancellation' },
  ];
  const ttOptions = transTypes.map(t => `<option value="${t.v}">${t.l}</option>`).join('');

  const whsList   = await api.ListWarehouses() || [];
  const whsOptions = `<option value="">All Warehouses</option>` +
    whsList.map(w => `<option value="${w.whs_code}">${w.whs_code} — ${w.whs_name}</option>`).join('');

  showView(wrapReports(tabBar, `
    <div class="card border-0 shadow-sm mb-4">
      <div class="card-body">
        <div class="row g-3 align-items-end">
          <div class="col-md-2">
            <label class="form-label fw-semibold">From Date</label>
            <input type="date" class="form-control" id="invMovFrom" value="${firstOfMonth}">
          </div>
          <div class="col-md-2">
            <label class="form-label fw-semibold">To Date</label>
            <input type="date" class="form-control" id="invMovTo" value="${today}">
          </div>
          <div class="col-md-2">
            <label class="form-label fw-semibold">Item Code</label>
            <input type="text" class="form-control" id="invMovItem" placeholder="e.g. PALAY">
          </div>
          <div class="col-md-2">
            <label class="form-label fw-semibold">Warehouse</label>
            <select class="form-select" id="invMovWhs">${whsOptions}</select>
          </div>
          <div class="col-md-2">
            <label class="form-label fw-semibold">Transaction Type</label>
            <select class="form-select" id="invMovType">${ttOptions}</select>
          </div>
          <div class="col-md-2">
            <button class="btn btn-primary w-100" onclick="fetchInvMovement()">
              <i class="bi bi-search me-1"></i>Generate
            </button>
          </div>
        </div>
      </div>
    </div>
    <div id="invMovResult"></div>
  `));
}

async function fetchInvMovement() {
  const dateFrom  = document.getElementById('invMovFrom')?.value || '';
  const dateTo    = document.getElementById('invMovTo')?.value || '';
  const itemCode  = document.getElementById('invMovItem')?.value.trim() || '';
  const whsCode   = document.getElementById('invMovWhs')?.value || '';
  const transType = document.getElementById('invMovType')?.value || '';

  if (!dateFrom || !dateTo) { toast('Please select a date range.', 'warning'); return; }

  const el = document.getElementById('invMovResult');
  if (!el) return;
  el.innerHTML = `<div class="text-center py-5"><div class="spinner-border text-primary"></div><div class="mt-2 text-muted">Loading movements…</div></div>`;

  // api.* returns data directly; null means the call failed (toast shown by api proxy)
  const d = await api.GetInventoryMovementReport(dateFrom, dateTo, itemCode, whsCode, transType);
  if (!d) {
    el.innerHTML = `<div class="alert alert-warning"><i class="bi bi-exclamation-triangle me-1"></i>Failed to load movement report.</div>`;
    return;
  }

  const rows = d.rows || [];

  const truncBanner = d.truncated
    ? `<div class="alert alert-warning py-2 mb-3 small">
         <i class="bi bi-exclamation-triangle me-1"></i>
         <strong>Result limited to 2,000 rows.</strong> Narrow the date range or add a filter.
       </div>`
    : '';

  // Trans type badge colours
  const ttColor = {
    GR: 'success', DR: 'success', WO_RECEIPT: 'success',
    GI: 'danger',  DO: 'danger',  WO_ISSUE: 'danger',
    GR_CANCEL: 'warning', GI_CANCEL: 'warning',
    DR_CANCEL: 'warning', DO_CANCEL: 'warning', WO_CANCEL: 'warning',
  };

  const tableRows = rows.length === 0
    ? `<tr><td colspan="10" class="text-center text-muted py-4">No movements found for the selected criteria.</td></tr>`
    : rows.map(r => {
        const colour = ttColor[r.trans_type] || 'secondary';
        const inQty  = r.in_qty  > 0 ? `<span class="text-success fw-semibold">+${formatQty(r.in_qty)}</span>`  : '—';
        const outQty = r.out_qty > 0 ? `<span class="text-danger fw-semibold">−${formatQty(r.out_qty)}</span>` : '—';
        return `
          <tr>
            <td class="text-muted small">${r.doc_date}</td>
            <td><span class="badge bg-${colour} text-truncate" style="max-width:130px" title="${r.trans_label}">${r.trans_label}</span></td>
            <td class="text-muted small">${r.doc_num || '—'}</td>
            <td class="small"><span class="text-muted">${r.item_code}</span></td>
            <td class="small">${r.item_name}</td>
            <td class="small text-muted">${r.warehouse || '—'}</td>
            <td class="text-end">${inQty}</td>
            <td class="text-end">${outQty}</td>
            <td class="text-end text-muted small">${formatCurrency(r.price)}</td>
            <td class="text-end small">${formatCurrency(r.value)}</td>
          </tr>
        `;
      }).join('');

  el.innerHTML = `
    ${truncBanner}
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-white d-flex justify-content-between align-items-center border-0 py-3">
        <span class="fw-bold">Inventory Movement — ${dateFrom} to ${dateTo}</span>
        <small class="text-muted">${rows.length.toLocaleString()} row${rows.length !== 1 ? 's' : ''}</small>
      </div>
      <div class="card-body p-0">
        <div class="table-responsive">
          <table class="table table-hover table-sm mb-0 align-middle">
            <thead class="table-light">
              <tr>
                <th>Date</th>
                <th>Type</th>
                <th>Doc #</th>
                <th>Item Code</th>
                <th>Item Name</th>
                <th>Warehouse</th>
                <th class="text-end">In Qty</th>
                <th class="text-end">Out Qty</th>
                <th class="text-end">Unit Cost</th>
                <th class="text-end">Value</th>
              </tr>
            </thead>
            <tbody>${tableRows}</tbody>
            <tfoot class="table-light">
              <tr>
                <td colspan="6" class="fw-bold text-end">Totals</td>
                <td class="text-end fw-bold text-success">+${formatQty(d.total_in_qty)}</td>
                <td class="text-end fw-bold text-danger">−${formatQty(d.total_out_qty)}</td>
                <td></td>
                <td class="text-end fw-bold">${formatCurrency((d.total_in_value || 0) + (d.total_out_value || 0))}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>
  `;
}

// ── Layout Helper ─────────────────────────────────────────────────────────────

function wrapReports(tabBar, content) {
  return `
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-file-earmark-bar-graph me-2"></i>Reports</h4>
      ${tabBar}
      ${content}
    </div>
  `;
}
