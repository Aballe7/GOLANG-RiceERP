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
          <a class="nav-link ${active === 'ar-aging' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/ar-aging');return false;">
            <i class="bi bi-calendar-range me-1"></i>AR Aging
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'analytics' ? 'active' : ''}" href="#"
            onclick="navigate('#/reports/analytics');return false;">
            <i class="bi bi-graph-up-arrow me-1"></i>Analytics
          </a>
        </li>
      </ul>
    `;

    if (active === 'pl') loadPlReport(tabBar);
    else if (active === 'balance-sheet') loadBalanceSheet(tabBar);
    else if (active === 'ar-aging') loadArAging(tabBar);
    else if (active === 'analytics') loadAnalytics(tabBar);
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
      <div class="card-header bg-transparent">
        <strong>Profit &amp; Loss Statement</strong>
        <span class="text-muted small ms-2">${formatDate(from)} – ${formatDate(to)}</span>
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
        <span class="text-muted small">As of ${formatDate(asOf)}</span>
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

// ── AR Aging ──────────────────────────────────────────────────────────────────

async function loadArAging(tabBar) {
  const rows = await api.GetARAgingReport();
  if (!rows) {
    showView(wrapReports(tabBar, `<div class="alert alert-warning">Failed to load AR Aging report.</div>`));
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
