// operations.js — Modules.Operations

const OPS_EGG_SIZES = ['Jumbo', 'XL', 'Large', 'Medium', 'Small', 'Peewee'];

Modules.Operations = {
  async load(id, sub) {
    if (sub === 'history' && id) {
      await loadHouseHistory(id, '#/operations');
    } else if (sub === 'edit-log-layer' && id) {
      await loadEditLog('layer', id);
    } else if (sub === 'edit-log-grower' && id) {
      await loadEditLog('grower', id);
    } else if ((sub === 'record-log' || sub === 'log') && id) {
      await loadRecordLog(id);
    } else if (sub === 'vaccine') {
      await loadVaccineSchedule();
    } else {
      await loadOperationsHome();
    }
  }
};

function opsTabBar(activeTab) {
  return `
    <ul class="nav nav-tabs mb-4">
      <li class="nav-item">
        <a class="nav-link ${activeTab === 'daily' ? 'active' : ''}" href="#"
          onclick="navigate('#/operations');return false;">
          <i class="bi bi-clipboard2-pulse me-1"></i>Daily Log
        </a>
      </li>
      <li class="nav-item">
        <a class="nav-link ${activeTab === 'vaccine' ? 'active' : ''}" href="#"
          onclick="navigate('#/operations/vaccine');return false;">
          <i class="bi bi-syringe me-1"></i>Vaccine Schedule
        </a>
      </li>
    </ul>
  `;
}

// ── Operations Home (Daily Log tab) ───────────────────────────────────────────

async function loadOperationsHome() {
  showLoading();
  const data = await api.GetFlocks();
  if (!data) {
    showView(`<div class="alert alert-warning m-4">Failed to load flock data.</div>`);
    return;
  }

  const active = data.active || [];
  const tabBar = opsTabBar('daily');

  if (active.length === 0) {
    showView(`
      <div class="container-fluid p-4">
        <h4 class="fw-bold mb-4"><i class="bi bi-clipboard2-pulse me-2"></i>Daily Operations</h4>
        ${tabBar}
        <div class="alert alert-info">
          <i class="bi bi-info-circle me-2"></i>No active flocks found.
          <a href="#" onclick="navigate('#/flocks/add');return false;">Add a flock</a> to get started.
        </div>
      </div>
    `);
    return;
  }

  const cards = active.map(fd => {
    const f = fd.flock;
    const logged = fd.logged_today === true;
    return `
      <div class="col-md-6 col-xl-4">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body d-flex flex-column">
            <div class="d-flex align-items-start justify-content-between mb-2">
              <div>
                <h6 class="fw-bold mb-0">${f.name}</h6>
                <span class="text-muted small">House ${f.house_number} · ${f.house_type}</span>
              </div>
              <span class="badge bg-${f.house_type === 'Layer' ? 'success' : 'info'}">
                ${f.house_type}
              </span>
            </div>
            <div class="text-muted small mb-3">
              ${formatNumber(f.current_count ?? 0)} birds · ${f.breed || 'Unknown breed'}
            </div>
            ${logged
              ? `<span class="badge bg-success-subtle text-success border border-success-subtle mt-auto py-2">
                  <i class="bi bi-check-circle me-1"></i>Logged Today
                </span>`
              : `<button class="btn btn-primary mt-auto" onclick="navigate('#/operations/${f.id}/record-log')">
                  <i class="bi bi-pencil-square me-1"></i>Log Today
                </button>`
            }
          </div>
        </div>
      </div>
    `;
  }).join('');

  showView(`
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4">
        <i class="bi bi-clipboard2-pulse me-2"></i>Daily Operations
      </h4>
      ${tabBar}
      <p class="text-muted mb-4">Record daily logs for each active house.</p>
      <div class="row g-3">
        ${cards}
      </div>
    </div>
  `);
}

// ── Record Daily Log ──────────────────────────────────────────────────────────

async function loadRecordLog(flockID) {
  showLoading();
  const [flock, feedStocks] = await Promise.all([
    api.GetFlock(flockID),
    api.ListFeedStocks()
  ]);

  if (!flock) {
    showView(`<div class="alert alert-warning m-4">Flock not found.</div>`);
    return;
  }

  const isLayer = flock.house_type === 'Layer';
  const feedOptions = (feedStocks || []).map(s =>
    `<option value="${s.id}">${s.feed_name} (${formatNumber(s.remaining_kg ?? 0)} kg left)</option>`
  ).join('');

  const eggFields = OPS_EGG_SIZES.map(size => `
    <div class="col-6 col-md-4">
      <label class="form-label small text-muted text-uppercase fw-semibold">${size}</label>
      <input type="number" class="form-control" name="egg_${size.toLowerCase()}" min="0" value="0" placeholder="0">
    </div>
  `).join('');

  const layerForm = `
    <!-- Egg Collection -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-transparent fw-semibold">
        <i class="bi bi-egg-fill me-2 text-warning"></i>Egg Collection
      </div>
      <div class="card-body">
        <div class="row g-3">${eggFields}</div>
        <hr>
        <div class="row g-3">
          <div class="col-md-6">
            <label class="form-label">Cracked / Broken Eggs</label>
            <input type="number" class="form-control" name="cracked_eggs" min="0" value="0">
          </div>
          <div class="col-md-6">
            <label class="form-label">Dirty Eggs</label>
            <input type="number" class="form-control" name="dirty_eggs" min="0" value="0">
          </div>
        </div>
      </div>
    </div>
  `;

  const commonFields = `
    <!-- Feed Consumed -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-transparent fw-semibold">
        <i class="bi bi-bag me-2 text-secondary"></i>Feed Consumed
      </div>
      <div class="card-body">
        <div class="row g-3">
          <div class="col-md-6">
            <label class="form-label">Feed Stock</label>
            <select class="form-select" name="feed_stock_id">
              <option value="">— Select feed —</option>
              ${feedOptions || '<option disabled>No feed stocks available</option>'}
            </select>
          </div>
          <div class="col-md-6">
            <label class="form-label">Amount Consumed (kg)</label>
            <input type="number" class="form-control" name="feed_consumed_kg" min="0" step="0.1" value="0">
          </div>
        </div>
      </div>
    </div>

    <!-- Mortality -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-transparent fw-semibold">
        <i class="bi bi-heartbreak me-2 text-danger"></i>Mortality
      </div>
      <div class="card-body">
        <div class="row g-3">
          <div class="col-md-4">
            <label class="form-label">Dead Count</label>
            <input type="number" class="form-control" name="mortality_dead" min="0" value="0">
          </div>
          <div class="col-md-4">
            <label class="form-label">Culled Count</label>
            <input type="number" class="form-control" name="mortality_culled" min="0" value="0">
          </div>
          <div class="col-md-4">
            <label class="form-label">Cause of Death</label>
            <input type="text" class="form-control" name="mortality_cause" placeholder="Optional">
          </div>
        </div>
      </div>
    </div>

    <!-- Health -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-transparent fw-semibold">
        <i class="bi bi-shield-plus me-2 text-info"></i>Health & Vaccination
      </div>
      <div class="card-body">
        <div class="row g-3">
          <div class="col-md-6">
            <label class="form-label">Vaccine / Treatment</label>
            <input type="text" class="form-control" name="vaccine" placeholder="e.g. Newcastle Disease">
          </div>
          <div class="col-md-6">
            <label class="form-label">Medication</label>
            <input type="text" class="form-control" name="medication" placeholder="Optional">
          </div>
          <div class="col-12">
            <label class="form-label">Remarks</label>
            <textarea class="form-control" name="remarks" rows="2" placeholder="Any observations…"></textarea>
          </div>
        </div>
      </div>
    </div>
  `;

  showView(`
    <div class="container p-4" style="max-width:860px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/operations')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div>
          <h4 class="fw-bold mb-0">Record Daily Log</h4>
          <div class="text-muted small">${flock.name} · House ${flock.house_number} · ${flock.house_type}</div>
        </div>
      </div>

      <form id="recordLogForm">
        <div class="mb-3">
          <label class="form-label fw-semibold">Log Date <span class="text-danger">*</span></label>
          <input type="date" class="form-control" name="log_date" required
            value="${new Date().toISOString().slice(0,10)}" style="max-width:220px">
        </div>

        ${isLayer ? layerForm : ''}
        ${commonFields}

        <div class="d-flex gap-2 mt-2">
          <button type="submit" class="btn btn-primary px-4">
            <i class="bi bi-check-lg me-1"></i>Submit Log
          </button>
          <button type="button" class="btn btn-outline-secondary" onclick="navigate('#/operations')">Cancel</button>
        </div>
      </form>
    </div>
  `);

  document.getElementById('recordLogForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const payload = { flock_id: flockID };
    for (const [k, v] of fd.entries()) {
      const num = Number(v);
      payload[k] = isNaN(num) || v === '' ? v : num;
    }
    if (isLayer) {
      payload.egg_sizes = {};
      OPS_EGG_SIZES.forEach(s => {
        const key = `egg_${s.toLowerCase()}`;
        payload.egg_sizes[s] = payload[key] ?? 0;
        delete payload[key];
      });
    }

    const btn = e.target.querySelector('[type=submit]');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';

    const result = isLayer
      ? await api.RecordDailyLog(payload)
      : await api.RecordGrowerLog(payload);

    if (result) {
      toast('Daily log saved successfully.', 'success');
      navigate('#/operations');
    } else {
      toast('Failed to save log.', 'danger');
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-check-lg me-1"></i>Submit Log';
    }
  });
}

// ── Vaccine Schedule ──────────────────────────────────────────────────────────

let _opsFlockList = [];

async function loadVaccineSchedule() {
  showLoading();
  const [data, schedules] = await Promise.all([
    api.GetFlocks(),
    api.ListVaccineSchedules(0)
  ]);

  _opsFlockList = (data && data.active) ? data.active : [];
  const list = schedules || [];
  const tabBar = opsTabBar('vaccine');

  // Build flock name map for display (active is []FlockData, flock fields are under .flock)
  const flockMap = {};
  _opsFlockList.forEach(fd => { flockMap[fd.flock.id] = fd.flock.name; });

  const today = new Date().toISOString().slice(0, 10);

  const rows = list.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No vaccine schedules found.</td></tr>`
    : list.map(s => {
        const isPast = !s.is_completed && s.scheduled_date < today;
        return `
          <tr class="${isPast ? 'table-warning' : ''}">
            <td>${flockMap[s.flock_id] || 'Flock #' + s.flock_id}</td>
            <td class="fw-semibold">${s.vaccine_name || '—'}</td>
            <td>${formatDate(s.scheduled_date)}</td>
            <td>${s.admin_method || '—'}</td>
            <td>
              ${s.is_completed
                ? `<span class="badge bg-success-subtle text-success border border-success-subtle">Completed</span>`
                : isPast
                  ? `<span class="badge bg-warning-subtle text-warning border border-warning-subtle">Overdue</span>`
                  : `<span class="badge bg-info-subtle text-info border border-info-subtle">Scheduled</span>`
              }
            </td>
            <td class="text-end">
              ${!s.is_completed
                ? `<button class="btn btn-sm btn-success" onclick="markVaccineDone(${s.id})">
                    <i class="bi bi-check-lg me-1"></i>Done
                  </button>`
                : ''
              }
            </td>
          </tr>
        `;
      }).join('');

  const flockOptions = _opsFlockList.map(fd =>
    `<option value="${fd.flock.id}">${fd.flock.name} (House ${fd.flock.house_number})</option>`
  ).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center justify-content-between mb-4">
        <h4 class="fw-bold mb-0"><i class="bi bi-syringe me-2"></i>Daily Operations</h4>
        <button class="btn btn-primary" onclick="openVaccineModal()">
          <i class="bi bi-plus-lg me-1"></i>Schedule Vaccine
        </button>
      </div>
      ${tabBar}

      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Flock</th>
                <th>Vaccine</th>
                <th>Scheduled Date</th>
                <th>Method</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Vaccine Schedule Modal -->
    <div class="modal fade" id="vaccineModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title">Schedule Vaccine</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <form id="vaccineForm">
              <div class="mb-3">
                <label class="form-label">Flock <span class="text-danger">*</span></label>
                <select class="form-select" id="vaccineFlockId" required>
                  <option value="">— Select flock —</option>
                  ${flockOptions || '<option disabled>No active flocks</option>'}
                </select>
              </div>
              <div class="mb-3">
                <label class="form-label">Vaccine Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="vaccineName" required
                  placeholder="e.g. Newcastle Disease, Marek's Disease">
              </div>
              <div class="mb-3">
                <label class="form-label">Scheduled Date <span class="text-danger">*</span></label>
                <input type="date" class="form-control" id="vaccineDate" required
                  value="${today}">
              </div>
              <div class="mb-3">
                <label class="form-label">Administration Method</label>
                <select class="form-select" id="vaccineMethod">
                  <option value="">— Select method —</option>
                  <option value="Drinking Water">Drinking Water</option>
                  <option value="Eye Drop">Eye Drop</option>
                  <option value="Injection">Injection</option>
                  <option value="Spray">Spray</option>
                  <option value="Wing Web">Wing Web</option>
                  <option value="Other">Other</option>
                </select>
              </div>
            </form>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitVaccineForm()">Schedule</button>
          </div>
        </div>
      </div>
    </div>
  `);
}

function openVaccineModal() {
  document.getElementById('vaccineForm').reset();
  document.getElementById('vaccineDate').value = new Date().toISOString().slice(0, 10);
  new bootstrap.Modal(document.getElementById('vaccineModal')).show();
}

async function submitVaccineForm() {
  const flockId = parseInt(document.getElementById('vaccineFlockId').value, 10);
  const name = document.getElementById('vaccineName').value.trim();
  const date = document.getElementById('vaccineDate').value;
  const method = document.getElementById('vaccineMethod').value;

  if (!flockId) { toast('Please select a flock.', 'warning'); return; }
  if (!name) { toast('Vaccine name is required.', 'warning'); return; }
  if (!date) { toast('Scheduled date is required.', 'warning'); return; }

  const result = await api.CreateVaccineSchedule(flockId, name, date, method);
  if (result) {
    toast('Vaccine scheduled successfully.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('vaccineModal')).hide();
    navigate('#/operations/vaccine');
  } else {
    toast('Failed to schedule vaccine.', 'danger');
  }
}

async function markVaccineDone(id) {
  if (!confirm('Mark this vaccine as completed?')) return;
  const result = await api.MarkVaccineComplete(id);
  if (result) {
    toast('Vaccine marked as completed.', 'success');
    navigate('#/operations/vaccine');
  } else {
    toast('Failed to update vaccine status.', 'danger');
  }
}

// ── House History ──────────────────────────────────────────────────────────────

async function loadHouseHistory(flockId, backRoute = '#/operations') {
  showLoading();
  const [histData, flock] = await Promise.all([
    api.GetHouseHistory(flockId),
    api.GetFlock(flockId),
  ]);
  if (!histData || !flock) {
    showView(`<div class="alert alert-warning m-4">Failed to load history.</div>`);
    return;
  }
  const logType  = histData.log_type; // 'layer' | 'grower'
  const logs     = histData.logs || [];
  const isLayer  = logType === 'layer';

  const rows = logs.length === 0
    ? `<tr><td colspan="8" class="text-center text-muted py-4">No logs recorded yet.</td></tr>`
    : logs.map(l => {
        const total = isLayer
          ? (l.pewee||0)+(l.pullet||0)+(l.small||0)+(l.medium||0)+(l.large||0)+(l.extra_large||0)+(l.jumbo||0)+(l.double_yolk||0)
          : null;
        const editRoute = `#/operations/edit-log-${logType}/${l.id}`;
        return `<tr>
          <td class="text-muted" style="font-size:12px;">${formatDate(l.date)}</td>
          ${isLayer ? `
            <td class="text-end">${formatNumber(total)}</td>
            <td class="text-end text-danger">${l.cracked_dirty||0}</td>
          ` : ''}
          <td class="text-end">${l.feed_consumed_kg > 0 ? l.feed_consumed_kg + ' kg' : '—'}</td>
          <td class="text-end ${(l.mortality||0) > 0 ? 'text-danger fw-bold' : ''}">${l.mortality||0}</td>
          <td class="text-muted" style="font-size:12px;">${l.vaccine_name || '—'}</td>
          ${!isLayer ? `<td class="text-muted" style="font-size:12px;">${l.medication||'—'}</td>` : ''}
          <td class="text-end">
            <button class="btn btn-sm btn-outline-warning py-0" onclick="navigate('${editRoute}')">
              <i class="bi bi-pencil-square me-1"></i>Edit
            </button>
          </td>
        </tr>`;
      }).join('');

  const layerCols = isLayer
    ? `<th class="text-end">Total Eggs</th><th class="text-end text-danger">Rejects</th>`
    : '';
  const growerCols = !isLayer ? `<th>Medication</th>` : '';

  showView(`
    <div class="container-fluid p-4" style="max-width:1000px;">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('${backRoute}')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div>
          <h4 class="fw-bold mb-0">
            <i class="bi bi-clock-history me-2"></i>Log History
          </h4>
          <small class="text-muted">
            ${isLayer ? 'Layer' : 'Grower'} House ${flock.house_number} · ${flock.name}
            · ${formatNumber(flock.current_count)} birds
          </small>
        </div>
        <div class="ms-auto">
          <button class="btn btn-primary btn-sm" onclick="navigate('#/operations/${flockId}/record-log')">
            <i class="bi bi-journal-plus me-1"></i> Log Today
          </button>
        </div>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0 small">
            <thead class="table-light">
              <tr>
                <th>Date</th>
                ${layerCols}
                <th>Feed</th>
                <th class="text-end">Mortality</th>
                <th>Vaccine</th>
                ${growerCols}
                <th></th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);
}

// ── Edit Log ───────────────────────────────────────────────────────────────────

async function loadEditLog(logType, logId) {
  showLoading();
  const isLayer = logType === 'layer';
  const log = isLayer ? await api.GetDailyLog(logId) : await api.GetGrowerLog(logId);
  if (!log) {
    showView(`<div class="alert alert-warning m-4">Log entry not found.</div>`);
    return;
  }
  const flock = await api.GetFlock(log.flock_id);
  if (!flock) {
    showView(`<div class="alert alert-warning m-4">Flock not found.</div>`);
    return;
  }

  const dateStr = log.date ? log.date.slice(0, 10) : new Date().toISOString().slice(0, 10);

  const EGG_SIZES = [
    { field: 'pewee',       label: 'Pewee',       weight: '< 45g' },
    { field: 'pullet',      label: 'Pullet',      weight: '45–50g' },
    { field: 'small',       label: 'Small',       weight: '50–55g' },
    { field: 'medium',      label: 'Medium',      weight: '55–60g' },
    { field: 'large',       label: 'Large',       weight: '60–65g' },
    { field: 'extra_large', label: 'Extra Large', weight: '65–70g' },
    { field: 'jumbo',       label: 'Jumbo',       weight: '> 70g' },
    { field: 'double_yolk', label: 'Double Yolk', weight: 'Special' },
  ];

  const eggSection = isLayer ? `
    <div class="card shadow-sm border-0 mb-3">
      <div class="card-header bg-primary bg-opacity-10 border-0 py-3">
        <h6 class="fw-bold mb-0"><i class="bi bi-egg-fill me-2 text-primary"></i>Egg Harvest</h6>
      </div>
      <div class="card-body">
        <div class="row g-2">
          ${EGG_SIZES.map(s => `
            <div class="col-6 col-md-3">
              <label class="form-label small fw-bold">${s.label} <span class="text-muted fw-normal">(${s.weight})</span></label>
              <input type="number" name="${s.field}" min="0" class="form-control form-control-sm text-center egg-inp"
                     value="${log[s.field] || 0}" oninput="opsEditUpdateTotals()">
            </div>`).join('')}
          <div class="col-12"><hr class="my-1"></div>
          <div class="col-6 col-md-3">
            <label class="form-label small fw-bold text-danger">Cracked / Dirty</label>
            <input type="number" name="cracked_dirty" min="0" class="form-control form-control-sm text-center"
                   value="${log.cracked_dirty || 0}">
          </div>
          <div class="col-6 col-md-3">
            <label class="form-label small fw-bold text-success">Good Total</label>
            <input type="text" id="editGoodTotal" class="form-control form-control-sm text-center fw-bold bg-light" disabled>
          </div>
          <div class="col-6 col-md-3">
            <label class="form-label small fw-bold text-primary">Total Trays</label>
            <input type="text" id="editTotalTrays" class="form-control form-control-sm text-center fw-bold bg-light" disabled>
          </div>
        </div>
      </div>
    </div>
  ` : '';

  const growerExtras = !isLayer ? `
    <div class="col-md-6">
      <label class="form-label small fw-bold">Medication</label>
      <input type="text" name="medication" class="form-control" value="${log.medication || ''}">
    </div>
    <div class="col-12">
      <label class="form-label small fw-bold">Remarks</label>
      <textarea name="remarks" class="form-control" rows="2">${log.remarks || ''}</textarea>
    </div>
  ` : '';

  showView(`
    <div class="container py-4" style="max-width:720px;">
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/operations/history/${flock.id}')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div>
          <h4 class="fw-bold mb-0"><i class="bi bi-pencil-square me-2 text-warning"></i>Edit Log Entry</h4>
          <small class="text-muted">
            ${isLayer ? 'Layer' : 'Grower'} House ${flock.house_number} · ${flock.name}
            · Originally recorded: <strong>${formatDate(dateStr)}</strong>
          </small>
        </div>
      </div>

      <div class="alert alert-warning d-flex gap-2 mb-4 py-2">
        <i class="bi bi-exclamation-triangle-fill flex-shrink-0 mt-1"></i>
        <small>
          You are correcting a past log entry. If you change <strong>Mortality</strong>,
          the flock's current bird count will be adjusted automatically.
        </small>
      </div>

      <form id="editLogForm">
        <div class="card shadow-sm border-0 mb-3">
          <div class="card-body">
            <div class="row g-3">
              <div class="col-md-4">
                <label class="form-label small fw-bold">Log Date</label>
                <input type="date" name="date" class="form-control" value="${dateStr}" required>
              </div>
            </div>
          </div>
        </div>

        ${eggSection}

        <div class="card shadow-sm border-0 mb-3">
          <div class="card-header bg-light border-0 py-3">
            <h6 class="fw-bold mb-0"><i class="bi bi-bag me-2"></i>Feed</h6>
          </div>
          <div class="card-body">
            <div class="row g-3">
              <div class="col-md-6">
                <label class="form-label small fw-bold">Feed Consumed (kg)</label>
                <input type="number" name="feed_consumed_kg" step="0.1" min="0"
                       class="form-control" value="${log.feed_consumed_kg || 0}">
              </div>
            </div>
          </div>
        </div>

        <div class="card shadow-sm border-0 mb-3">
          <div class="card-header bg-light border-0 py-3">
            <h6 class="fw-bold mb-0"><i class="bi bi-heart-pulse me-2"></i>Health</h6>
          </div>
          <div class="card-body">
            <div class="row g-3">
              <div class="col-md-4">
                <label class="form-label small fw-bold">
                  Mortality
                  ${log.mortality ? `<span class="text-muted fw-normal">(was: ${log.mortality})</span>` : ''}
                </label>
                <input type="number" name="mortality" min="0" class="form-control" value="${log.mortality || 0}">
                <div class="form-text text-warning">
                  <i class="bi bi-info-circle"></i> Changing this adjusts the flock's current bird count.
                </div>
              </div>
              <div class="col-md-8">
                <label class="form-label small fw-bold">Vaccine / Medicine Given</label>
                <input type="text" name="vaccine_name" class="form-control"
                       placeholder="e.g. Newcastle, Vitamins" value="${log.vaccine_name || ''}">
              </div>
              ${growerExtras}
            </div>
          </div>
        </div>

        <div class="d-flex gap-2">
          <button type="submit" class="btn btn-warning fw-bold px-4">
            <i class="bi bi-floppy me-2"></i>Save Correction
          </button>
          <button type="button" class="btn btn-outline-secondary px-4"
                  onclick="navigate('#/operations/history/${flock.id}')">Cancel</button>
        </div>
      </form>
    </div>
  `);

  if (isLayer) opsEditUpdateTotals();

  document.getElementById('editLogForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const payload = { flock_id: log.flock_id };
    for (const [k, v] of fd.entries()) {
      const num = Number(v);
      payload[k] = (!isNaN(num) && v !== '') ? num : v;
    }
    const btn = e.target.querySelector('[type=submit]');
    btn.disabled = true; btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';
    const result = isLayer
      ? await api.UpdateDailyLog(logId, payload)
      : await api.UpdateGrowerLog(logId, payload);
    if (result) {
      toast('Log updated successfully.', 'success');
      navigate('#/operations/history/' + flock.id);
    } else {
      toast('Failed to update log.', 'danger');
      btn.disabled = false; btn.innerHTML = '<i class="bi bi-floppy me-2"></i>Save Correction';
    }
  });
}

function opsEditUpdateTotals() {
  const EGG_FIELDS = ['pewee','pullet','small','medium','large','extra_large','jumbo','double_yolk'];
  let good = 0;
  EGG_FIELDS.forEach(f => {
    const el = document.querySelector(`[name="${f}"]`);
    good += parseInt(el?.value || 0, 10);
  });
  const gt = document.getElementById('editGoodTotal');
  const tt = document.getElementById('editTotalTrays');
  if (gt) gt.value = good.toLocaleString();
  if (tt) tt.value = (good / 30).toFixed(1) + ' trays';
}
