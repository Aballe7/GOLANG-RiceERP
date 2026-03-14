// flocks.js — Modules.Flocks

const FLOCK_BREEDS = ['Lohmann Brown', 'Hy-Line Brown', 'ISA Brown', 'Dekalb White', 'Hisex Brown', 'Other'];
const KNOWN_BREEDS = ['ISA Brown', 'Lohmann Brown', 'Lohmann White', 'Hyline Brown', 'Hyline White', 'Novogen Brown', 'Dekalb White'];

Modules.Flocks = {
  async load(id, sub) {
    if (!id && !sub) {
      await loadFlockList();
    } else if (sub === 'add') {
      await loadAddFlock();
    } else if (id && sub === 'retire') {
      await loadRetireFlock(id);
    } else if (id && sub === 'transfer') {
      await loadTransferFlock(id);
    } else if (id && sub === 'history') {
      await loadHouseHistory(id, '#/flocks');
    } else if (id && sub === 'body-weight') {
      await loadBodyWeight(id);
    } else {
      await loadFlockList();
    }
  }
};

// ── Flock List ────────────────────────────────────────────────────────────────

async function loadFlockList() {
  _flockListData = [];
  showLoading();
  const data = await api.GetFlocks();
  if (!data) {
    showView(`<div class="alert alert-warning m-4">Failed to load flocks.</div>`);
    return;
  }

  const active  = data.active  || [];
  const retired = data.retired || [];

  const activeRows = active.length === 0
    ? `<tr><td colspan="10" class="text-center text-muted py-5">
         <i class="bi bi-clipboard d-block mb-2" style="font-size:2rem;opacity:0.3;"></i>
         No flocks registered yet.
         <a href="#" onclick="navigate('#/flocks/add');return false;">Register your first batch.</a>
       </td></tr>`
    : active.map(fd => activeFlockRow(fd)).join('');

  const retiredRows = retired.length === 0 ? '' : retired.map(fd => retiredFlockRow(fd)).join('');
  const retiredTotal = retired.reduce((sum, fd) => {
    const f = fd.flock;
    return sum + ((f.retirement_price_per_bird || 0) * (f.birds_retired || 0));
  }, 0);

  const retiredSection = retired.length === 0 ? '' : `
    <div class="mt-4">
      <h6 class="fw-bold text-muted text-uppercase mb-2" style="font-size:11px;letter-spacing:1px;">
        <i class="bi bi-archive me-1"></i>Retired Flocks
      </h6>
      <div class="card shadow-sm border-0">
        <div class="card-body p-0">
          <div class="table-responsive">
            <table class="table table-hover align-middle mb-0 small">
              <thead class="table-light">
                <tr>
                  <th class="ps-3">House</th>
                  <th>Batch / Breed</th>
                  <th class="text-center">Initial</th>
                  <th class="text-center">Retired Birds</th>
                  <th>Retirement Date</th>
                  <th>Disposition</th>
                  <th class="text-end">Proceeds</th>
                  <th>Buyer</th>
                  <th>Notes</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>${retiredRows}</tbody>
              <tfoot class="table-light small text-muted">
                <tr>
                  <td colspan="6" class="ps-3 fw-bold">TOTAL</td>
                  <td class="text-end fw-bold text-dark">
                    ${retiredTotal > 0 ? '₱' + formatNumber(retiredTotal, 2) : '—'}
                  </td>
                  <td colspan="3"></td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </div>
    </div>
  `;

  showView(`
    <div class="container-fluid py-4 px-4">

      <!-- HEADER -->
      <div class="d-flex justify-content-between align-items-center mb-4">
        <div>
          <h4 class="fw-bold mb-0"><i class="bi bi-list-ul me-2 text-primary"></i>Master Flock List</h4>
          <small class="text-muted">As of ${new Date().toLocaleDateString('en-US', {month:'long',day:'numeric',year:'numeric'})}</small>
        </div>
        <button class="btn btn-primary btn-sm" onclick="navigate('#/flocks/add')">
          <i class="bi bi-plus-lg me-1"></i> Register New Batch
        </button>
      </div>

      <!-- LEGEND -->
      <div class="d-flex flex-wrap gap-2 mb-3 small">
        <span class="text-muted fw-bold me-1">Production Stage:</span>
        <span class="badge bg-info">Brooding</span><span class="text-muted">0–7 wks</span>
        <span class="badge bg-warning text-dark ms-2">Growing</span><span class="text-muted">8–17 wks</span>
        <span class="badge bg-primary ms-2">Pre-Lay</span><span class="text-muted">18–20 wks</span>
        <span class="badge bg-primary ms-2">Ramp-Up</span><span class="text-muted">21–24 wks (Layer)</span>
        <span class="badge bg-success ms-2">Peak Production</span><span class="text-muted">25–60 wks</span>
        <span class="badge bg-warning text-dark ms-2">Late Lay</span><span class="text-muted">61–80 wks</span>
        <span class="badge bg-danger ms-2">End of Cycle</span><span class="text-muted">&gt;80 wks</span>
      </div>

      <!-- FLOCK TABLE -->
      <div class="card shadow-sm border-0">
        <div class="card-body p-0">
          <div class="table-responsive">
            <table class="table table-hover align-middle mb-0">
              <thead class="table-light">
                <tr>
                  <th class="ps-3">House</th>
                  <th>Batch / Breed</th>
                  <th>Birds</th>
                  <th>Hatch Date</th>
                  <th>Age</th>
                  <th>In Lay</th>
                  <th>Stage</th>
                  <th class="text-center">Today's Log</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>${activeRows}</tbody>
            </table>
          </div>
        </div>
      </div>

      ${retiredSection}

      <!-- INDUSTRY REFERENCE -->
      <div class="card border-0 shadow-sm mt-3">
        <div class="card-body py-3">
          <p class="fw-bold small text-uppercase text-muted mb-2">Industry Reference — Age &amp; Production Timeline</p>
          <div class="row g-2 small">
            <div class="col-md-4">
              <strong>Grower Stage</strong><br>
              <span class="text-muted">Wks 0–7:</span> Brooding — chicks need heat<br>
              <span class="text-muted">Wks 8–17:</span> Growing — skeletal &amp; muscle development<br>
              <span class="text-muted">Wks 18–20:</span> Pre-Lay — reproductive tract developing
            </div>
            <div class="col-md-4">
              <strong>Layer Stage</strong><br>
              <span class="text-muted">Wks 21–24:</span> Ramp-Up — first eggs, small size<br>
              <span class="text-muted">Wks 25–60:</span> Peak Production — target 80%+ HDP<br>
              <span class="text-muted">Wks 61–80:</span> Late Lay — gradual decline
            </div>
            <div class="col-md-4">
              <strong>End of Cycle</strong><br>
              <span class="text-muted">Wks 80+:</span> Below viable production threshold<br>
              <span class="text-muted">Typical cull:</span> 78–80 wks (one-cycle system)<br>
              <span class="text-muted">With molt:</span> Up to 102–106 wks possible
            </div>
          </div>
        </div>
      </div>

    </div>

    <!-- EDIT FLOCK MODAL -->
    <div class="modal fade" id="editFlockModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title fw-bold">
              <i class="bi bi-gear me-2 text-info"></i>Edit Flock Details
            </h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <form id="editFlockForm">
            <div class="modal-body">
              <div class="row g-3">
                <div class="col-12">
                  <label class="form-label small fw-bold">Batch Name <span class="text-danger">*</span></label>
                  <input type="text" name="name" id="ef_name" class="form-control" required>
                </div>
                <div class="col-md-6">
                  <label class="form-label small fw-bold">Building / Block</label>
                  <input type="text" name="building_name" id="ef_building" class="form-control" placeholder="e.g. Block A">
                </div>
                <div class="col-md-6">
                  <label class="form-label small fw-bold">House Number <span class="text-danger">*</span></label>
                  <input type="text" name="house_number" id="ef_house" class="form-control" required>
                </div>
                <div class="col-md-6">
                  <label class="form-label small fw-bold">Breed</label>
                  <select id="ef_breed_sel" class="form-select" onchange="efToggleBreed(this)">
                    <option value="ISA Brown">ISA Brown</option>
                    <option value="Lohmann Brown">Lohmann Brown</option>
                    <option value="Lohmann White">Lohmann White</option>
                    <option value="Hyline Brown">Hyline Brown</option>
                    <option value="Hyline White">Hyline White</option>
                    <option value="Novogen Brown">Novogen Brown</option>
                    <option value="Dekalb White">Dekalb White</option>
                    <option value="__other__">Other (specify below)</option>
                  </select>
                  <input type="text" id="ef_breed_custom" class="form-control mt-2 d-none"
                         placeholder="Enter breed name e.g. Hisex Brown">
                </div>
                <div class="col-md-6">
                  <label class="form-label small fw-bold">Hatch Date</label>
                  <input type="date" name="hatch_date" id="ef_hatch" class="form-control">
                </div>
                <div class="col-md-6">
                  <label class="form-label small fw-bold">Current Bird Count</label>
                  <input type="number" name="current_count" id="ef_count" class="form-control" min="0">
                  <div class="form-text text-warning">
                    <i class="bi bi-exclamation-triangle me-1"></i>
                    Only adjust for manual corrections. Use the daily log for normal mortality.
                  </div>
                </div>
              </div>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
              <button type="submit" class="btn btn-info fw-bold text-white px-4">
                <i class="bi bi-floppy me-1"></i> Save Changes
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  `);

  // Edit flock modal submit
  document.getElementById('editFlockForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const flockId = parseInt(document.getElementById('editFlockModal').dataset.flockId, 10);
    const breedSel = document.getElementById('ef_breed_sel');
    const breedCustom = document.getElementById('ef_breed_custom');
    const breed = breedSel.value === '__other__' ? breedCustom.value.trim() : breedSel.value;
    const payload = {
      id:            flockId,
      name:          document.getElementById('ef_name').value.trim(),
      building_name: document.getElementById('ef_building').value.trim(),
      house_number:  document.getElementById('ef_house').value.trim(),
      breed:         breed,
      hatch_date:    document.getElementById('ef_hatch').value,
      current_count: parseInt(document.getElementById('ef_count').value, 10) || 0,
    };
    const btn = e.target.querySelector('[type=submit]');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';
    const result = await api.UpdateFlock(payload);
    if (result) {
      toast('Flock updated successfully.', 'success');
      bootstrap.Modal.getInstance(document.getElementById('editFlockModal')).hide();
      loadFlockList();
    } else {
      toast('Failed to update flock.', 'danger');
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-floppy me-1"></i> Save Changes';
    }
  });
}

function efToggleBreed(sel) {
  const custom = document.getElementById('ef_breed_custom');
  const isOther = sel.value === '__other__';
  custom.classList.toggle('d-none', !isOther);
  custom.required = isOther;
  if (!isOther) custom.value = '';
}

function openEditFlockModal(fd) {
  const f = fd.flock;
  const modal = document.getElementById('editFlockModal');
  modal.dataset.flockId = f.id;

  document.getElementById('ef_name').value     = f.name || '';
  document.getElementById('ef_building').value = f.building_name || '';
  document.getElementById('ef_house').value    = f.house_number || '';
  document.getElementById('ef_hatch').value    = f.hatch_date ? f.hatch_date.slice(0,10) : '';
  document.getElementById('ef_count').value    = f.current_count ?? 0;

  const sel = document.getElementById('ef_breed_sel');
  const custom = document.getElementById('ef_breed_custom');
  const breed = f.breed || '';
  if (KNOWN_BREEDS.includes(breed)) {
    sel.value = breed;
    custom.classList.add('d-none');
    custom.required = false;
    custom.value = '';
  } else {
    sel.value = '__other__';
    custom.classList.remove('d-none');
    custom.required = true;
    custom.value = breed;
  }

  new bootstrap.Modal(modal).show();
}

// Cache flock data for modal access
let _flockListData = [];

function activeFlockRow(fd) {
  const f = fd.flock;
  const isLayer = f.house_type === 'Layer';
  const logged = fd.logged_today === true;
  const idx = _flockListData.length;
  _flockListData.push(fd);

  const houseCell = `
    <td class="ps-3">
      <span class="badge ${isLayer ? 'bg-primary' : 'bg-warning text-dark'} me-1">
        ${f.house_type ? f.house_type[0] : '?'}
      </span>
      <strong>H-${f.house_number || '?'}</strong>
      ${f.building_name ? `<br><small class="text-muted">${f.building_name}</small>` : ''}
    </td>`;

  const batchCell = `
    <td>
      <span class="fw-bold">${f.name || '—'}</span>
      ${f.breed ? `<br><small class="text-muted">${f.breed}</small>` : ''}
    </td>`;

  const lost = (f.initial_count || 0) - (f.current_count || 0);
  const birdsCell = `
    <td>
      <span class="fw-bold">${formatNumber(f.current_count ?? 0)}</span>
      ${lost > 0 ? `<br><small class="text-danger"><i class="bi bi-arrow-down"></i> ${formatNumber(lost)} lost</small>` : ''}
    </td>`;

  const hatchCell = `<td class="text-nowrap">${f.hatch_date ? formatDate(f.hatch_date) : '<span class="text-muted">—</span>'}</td>`;

  const ageCell = fd.age_weeks != null
    ? `<td class="text-nowrap"><span class="fw-bold">${fd.age_weeks} wks</span><br><small class="text-muted">${fd.age_days} days</small></td>`
    : `<td class="text-nowrap"><span class="text-muted">—</span></td>`;

  let inLayCell;
  if (fd.weeks_in_lay != null) {
    inLayCell = `<td class="text-nowrap"><span class="fw-bold text-success">${fd.weeks_in_lay} wks</span><br><small class="text-muted">${fd.weeks_in_lay * 7} days</small></td>`;
  } else if (isLayer) {
    inLayCell = `<td class="text-nowrap"><span class="text-muted small">Not yet</span></td>`;
  } else {
    inLayCell = `<td class="text-nowrap"><span class="text-muted">—</span></td>`;
  }

  const stageBadgeText = fd.stage || '—';
  const stageColor = fd.stage_color || 'secondary';
  const stageDark = ['warning','info'].includes(stageColor) ? 'text-dark' : '';
  let stageNote = '';
  if (stageBadgeText === 'Peak Production') stageNote = `<br><small class="text-muted" style="font-size:10px;">Best period</small>`;
  else if (stageBadgeText === 'End of Cycle') stageNote = `<br><small class="text-danger" style="font-size:10px;">Consider replacement</small>`;
  else if (stageBadgeText === 'Late Lay') stageNote = `<br><small class="text-muted" style="font-size:10px;">Declining output</small>`;
  const stageCell = `<td><span class="badge bg-${stageColor} ${stageDark}">${stageBadgeText}</span>${stageNote}</td>`;

  const logCell = logged
    ? `<td class="text-center"><span class="badge bg-success px-3 py-2"><i class="bi bi-check-lg me-1"></i>Done</span></td>`
    : `<td class="text-center">
        <button class="badge bg-danger px-3 py-2 border-0" onclick="navigate('#/operations/${f.id}/record-log')">
          <i class="bi bi-exclamation-circle me-1"></i>Pending
        </button>
       </td>`;

  const actions = `
    <td class="text-nowrap">
      <button class="btn btn-sm btn-outline-info me-1" onclick="_flockListData[${idx}] && openEditFlockModal(_flockListData[${idx}])" title="Edit flock details">
        <i class="bi bi-gear"></i>
      </button>
      <button class="btn btn-sm btn-outline-secondary me-1" onclick="navigate('#/flocks/${f.id}/history')" title="View history">
        <i class="bi bi-clock-history"></i>
      </button>
      <button class="btn btn-sm ${isLayer ? 'btn-outline-primary' : 'btn-outline-warning'} me-1" onclick="navigate('#/operations/${f.id}/record-log')" title="Record log">
        <i class="bi bi-pencil-square"></i>
      </button>
      ${!isLayer ? `
      <button class="btn btn-sm btn-outline-success me-1" onclick="navigate('#/flocks/${f.id}/body-weight')" title="Body weight log">
        <i class="bi bi-clipboard2-pulse"></i>
      </button>
      <button class="btn btn-sm btn-outline-success ms-1" onclick="navigate('#/flocks/${f.id}/transfer')" title="Transfer to Layer">
        <i class="bi bi-arrow-right-circle"></i>
      </button>` : ''}
      ${isLayer ? `
      <button class="btn btn-sm btn-outline-danger ms-1" onclick="navigate('#/flocks/${f.id}/retire')" title="Retire this flock">
        <i class="bi bi-box-arrow-right"></i>
      </button>` : ''}
    </td>`;

  const rowClass = !logged && isLayer ? 'table-warning' : '';
  return `<tr class="${rowClass}">${houseCell}${batchCell}${birdsCell}${hatchCell}${ageCell}${inLayCell}${stageCell}${logCell}${actions}</tr>`;
}

function retiredFlockRow(fd) {
  const f = fd.flock;
  const rt = f.retirement_type || 'Sold';
  const rtBadge = rt === 'Sold' ? 'bg-success' : (rt === 'Culled' ? 'bg-danger' : 'bg-warning text-dark');
  const proceeds = (f.retirement_price_per_bird || 0) * (f.birds_retired || 0);

  return `
    <tr class="text-muted">
      <td class="ps-3">
        <span class="badge bg-secondary me-1">${f.house_type ? f.house_type[0] : '?'}</span>
        <strong>H-${f.house_number || '?'}</strong>
        ${f.building_name ? `<br><small>${f.building_name}</small>` : ''}
      </td>
      <td>
        <span class="fw-bold text-muted">${f.name || '—'}</span>
        ${f.breed ? `<br><small>${f.breed}</small>` : ''}
      </td>
      <td class="text-center">${formatNumber(f.initial_count ?? 0)}</td>
      <td class="text-center fw-bold">${formatNumber(f.birds_retired ?? 0)}</td>
      <td class="text-nowrap">${f.retirement_date ? formatDate(f.retirement_date) : '—'}</td>
      <td><span class="badge ${rtBadge}">${rt}</span></td>
      <td class="text-end fw-bold">
        ${proceeds > 0
          ? `₱${formatNumber(proceeds, 2)}<br><small class="text-muted fw-normal">₱${formatNumber(f.retirement_price_per_bird || 0, 2)}/bird</small>`
          : '<span class="text-muted">—</span>'}
      </td>
      <td class="small">${f.retirement_buyer || '—'}</td>
      <td class="small text-muted" style="max-width:180px;">
        <span class="d-inline-block text-truncate" style="max-width:170px;" title="${f.retirement_notes || ''}">
          ${f.retirement_notes || '—'}
        </span>
      </td>
      <td>
        <button class="btn btn-sm btn-outline-secondary py-0 px-2" style="font-size:11px;"
                onclick="navigate('#/flocks/${f.id}/history')" title="View history">
          <i class="bi bi-clock-history"></i>
        </button>
      </td>
    </tr>
  `;
}

// ── Add Flock ─────────────────────────────────────────────────────────────────

async function loadAddFlock() {
  showLoading();
  const breedOptions = FLOCK_BREEDS.map(b =>
    `<option value="${b}">${b}</option>`
  ).join('');

  showView(`
    <div class="container p-4" style="max-width:680px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/flocks')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">Add New Flock</h4>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-body">
          <form id="addFlockForm">
            <div class="row g-3">
              <div class="col-md-6">
                <label class="form-label">Flock Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="flockName" required placeholder="e.g. Batch A-2024">
              </div>
              <div class="col-md-6">
                <label class="form-label">House Number <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="flockHouseNumber" required placeholder="e.g. H1">
              </div>
              <div class="col-md-6">
                <label class="form-label">House Type <span class="text-danger">*</span></label>
                <select class="form-select" id="flockHouseType" required>
                  <option value="">Select type…</option>
                  <option value="Layer">Layer</option>
                  <option value="Grower">Grower</option>
                </select>
              </div>
              <div class="col-md-6">
                <label class="form-label">Breed <span class="text-danger">*</span></label>
                <select class="form-select" id="flockBreed" required onchange="toggleCustomBreed(this.value)">
                  <option value="">Select breed…</option>
                  ${breedOptions}
                </select>
              </div>
              <div class="col-md-6" id="customBreedGroup" style="display:none">
                <label class="form-label">Custom Breed Name <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="flockCustomBreed" placeholder="Enter breed name">
              </div>
              <div class="col-md-6">
                <label class="form-label">Building Name</label>
                <input type="text" class="form-control" id="flockBuilding" placeholder="e.g. Building 1">
              </div>
              <div class="col-md-6">
                <label class="form-label">Hatch Date <span class="text-danger">*</span></label>
                <input type="date" class="form-control" id="flockHatchDate" required>
              </div>
              <div class="col-md-6">
                <label class="form-label">Initial Bird Count <span class="text-danger">*</span></label>
                <input type="number" class="form-control" id="flockInitialCount" required min="1" placeholder="0">
              </div>
            </div>
            <div class="d-flex gap-2 mt-4">
              <button type="submit" class="btn btn-primary px-4">
                <i class="bi bi-check-lg me-1"></i>Create Flock
              </button>
              <button type="button" class="btn btn-outline-secondary" onclick="navigate('#/flocks')">Cancel</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  `);

  document.getElementById('addFlockForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const breed = document.getElementById('flockBreed').value;
    const customBreed = document.getElementById('flockCustomBreed').value;
    const payload = {
      name:          document.getElementById('flockName').value.trim(),
      house_number:  document.getElementById('flockHouseNumber').value.trim(),
      house_type:    document.getElementById('flockHouseType').value,
      breed:         breed === 'Other' ? customBreed.trim() : breed,
      building_name: document.getElementById('flockBuilding').value.trim(),
      hatch_date:    document.getElementById('flockHatchDate').value,
      initial_count: parseInt(document.getElementById('flockInitialCount').value, 10),
    };
    if (!payload.breed) { toast('Please enter a breed name.', 'warning'); return; }
    const btn = e.target.querySelector('[type=submit]');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';
    const result = await api.CreateFlock(payload);
    if (result) {
      toast('Flock created successfully.', 'success');
      navigate('#/flocks');
    } else {
      toast('Failed to create flock.', 'danger');
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-check-lg me-1"></i>Create Flock';
    }
  });
}

function toggleCustomBreed(val) {
  const grp = document.getElementById('customBreedGroup');
  const inp = document.getElementById('flockCustomBreed');
  if (val === 'Other') {
    grp.style.display = '';
    inp.required = true;
  } else {
    grp.style.display = 'none';
    inp.required = false;
    inp.value = '';
  }
}

// ── Retire Flock ──────────────────────────────────────────────────────────────

async function loadRetireFlock(id) {
  showLoading();
  const flock = await api.GetFlock(id);
  if (!flock) {
    showView(`<div class="alert alert-warning m-4">Flock not found.</div>`);
    return;
  }

  showView(`
    <div class="container p-4" style="max-width:540px">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/flocks')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">Retire Flock</h4>
      </div>

      <div class="card border-0 shadow-sm">
        <div class="card-body">
          <div class="alert alert-warning">
            <i class="bi bi-exclamation-triangle-fill me-2"></i>
            You are retiring <strong>${flock.name}</strong> (House ${flock.house_number}).
            This action cannot be undone.
          </div>
          <form id="retireFlockForm">
            <div class="mb-3">
              <label class="form-label">Retirement Date <span class="text-danger">*</span></label>
              <input type="date" class="form-control" id="retireDate" required value="${new Date().toISOString().slice(0,10)}">
            </div>
            <div class="mb-3">
              <label class="form-label">Reason / Notes</label>
              <textarea class="form-control" id="retireReason" rows="3" placeholder="Optional notes about retirement…"></textarea>
            </div>
            <div class="d-flex gap-2">
              <button type="submit" class="btn btn-danger px-4">
                <i class="bi bi-archive me-1"></i>Retire Flock
              </button>
              <button type="button" class="btn btn-outline-secondary" onclick="navigate('#/flocks')">Cancel</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  `);

  document.getElementById('retireFlockForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const payload = {
      flock_id:    id,
      retire_date: document.getElementById('retireDate').value,
      reason:      document.getElementById('retireReason').value.trim(),
    };
    const btn = e.target.querySelector('[type=submit]');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Retiring…';
    const result = await api.RetireFlock(payload);
    if (result) {
      toast('Flock retired.', 'success');
      navigate('#/flocks');
    } else {
      toast('Failed to retire flock.', 'danger');
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-archive me-1"></i>Retire Flock';
    }
  });
}

// ── Transfer Flock ────────────────────────────────────────────────────────────

async function loadTransferFlock(id) {
  showLoading();
  const [flock, layerHouses] = await Promise.all([api.GetFlock(id), api.GetLayerHouses()]);
  if (!flock) {
    showView(`<div class="alert alert-warning m-4">Flock not found.</div>`);
    return;
  }

  // Compute age in weeks from hatch_date
  let ageStr = '—';
  if (flock.hatch_date) {
    const diffDays = Math.floor((Date.now() - new Date(flock.hatch_date)) / 86400000);
    ageStr = Math.floor(diffDays / 7) + ' weeks';
  }
  const hatchStr = flock.hatch_date
    ? new Date(flock.hatch_date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
    : '—';

  const existingBadges = (layerHouses || [])
    .map(h => `<span class="badge bg-primary me-1">H-${_esc(h)}</span>`)
    .join('');

  const today = new Date().toISOString().slice(0, 10);

  showView(`
    <div class="container py-4" style="max-width:680px">

      <!-- Header -->
      <div class="d-flex align-items-center gap-3 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/flocks')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <div>
          <h4 class="fw-bold mb-0">
            <i class="bi bi-arrow-right-circle me-2 text-warning"></i>Transfer Flock to Layer House
          </h4>
          <small class="text-muted">Grower → Layer transition</small>
        </div>
      </div>

      <!-- Flock Summary Card -->
      <div class="card border-warning border-2 shadow-sm mb-4">
        <div class="card-header bg-warning bg-opacity-10 border-0 py-3">
          <h6 class="fw-bold mb-0"><i class="bi bi-egg me-2 text-warning"></i>Flock Being Transferred</h6>
        </div>
        <div class="card-body">
          <div class="row g-3">
            <div class="col-6 col-md-4">
              <p class="small text-muted mb-0">Batch Name</p>
              <p class="fw-bold mb-0">${_esc(flock.name)}</p>
            </div>
            <div class="col-6 col-md-4">
              <p class="small text-muted mb-0">Current House</p>
              <p class="fw-bold mb-0">
                <span class="badge bg-warning text-dark me-1">Grower</span>H-${_esc(flock.house_number)}
              </p>
            </div>
            <div class="col-6 col-md-4">
              <p class="small text-muted mb-0">Bird Count</p>
              <p class="fw-bold mb-0">${formatNumber(flock.current_count)} birds</p>
            </div>
            <div class="col-6 col-md-4">
              <p class="small text-muted mb-0">Breed</p>
              <p class="fw-bold mb-0">${_esc(flock.breed || '—')}</p>
            </div>
            <div class="col-6 col-md-4">
              <p class="small text-muted mb-0">Hatch Date</p>
              <p class="fw-bold mb-0">${hatchStr}</p>
            </div>
            <div class="col-6 col-md-4">
              <p class="small text-muted mb-0">Age</p>
              <p class="fw-bold mb-0">${ageStr}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Transfer Form -->
      <form id="transferFlockForm">
        <div class="card shadow-sm border-0 mb-3">
          <div class="card-header bg-primary bg-opacity-10 border-0 py-3">
            <h6 class="fw-bold mb-0"><i class="bi bi-house-door me-2 text-primary"></i>Destination Layer House</h6>
          </div>
          <div class="card-body">
            <div class="row g-3">
              <div class="col-md-4">
                <label class="form-label small fw-bold">
                  Layer House Number <span class="text-danger">*</span>
                </label>
                <input type="text" class="form-control" id="transferHouseNumber"
                       placeholder="e.g. 3" required autofocus>
                ${existingBadges ? `<div class="form-text">Existing layer houses: ${existingBadges}</div>` : ''}
              </div>
              <div class="col-md-4">
                <label class="form-label small fw-bold">Building / Section</label>
                <input type="text" class="form-control" id="transferBuilding"
                       placeholder="e.g. Building A"
                       value="${_esc(flock.building_name || '')}">
              </div>
              <div class="col-md-4">
                <label class="form-label small fw-bold">Transfer Date</label>
                <input type="date" class="form-control" id="transferDate" value="${today}">
              </div>
              <div class="col-12">
                <label class="form-label small fw-bold">Notes / Remarks</label>
                <textarea class="form-control" id="transferNotes" rows="2"
                          placeholder="e.g. Birds moved to new layer house, started layer feed…"></textarea>
              </div>
            </div>
          </div>
        </div>

        <!-- What this does -->
        <div class="alert alert-info d-flex gap-2 mb-4">
          <i class="bi bi-info-circle-fill flex-shrink-0 mt-1"></i>
          <div class="small">
            <strong>What this does:</strong>
            <ul class="mb-0 mt-1">
              <li>Changes flock type from <strong>Grower → Layer</strong></li>
              <li>Assigns the flock to the new Layer house number</li>
              <li>Flock will appear on the dashboard under Layer Houses</li>
              <li>Daily logs will now include egg harvest fields</li>
              <li><strong>This action cannot be undone</strong> from the UI</li>
            </ul>
          </div>
        </div>

        <div class="d-grid gap-2">
          <button type="submit" class="btn btn-warning btn-lg fw-bold py-3" id="transferSubmitBtn">
            <i class="bi bi-arrow-right-circle me-2"></i>Confirm Transfer
          </button>
          <button type="button" class="btn btn-outline-secondary" onclick="navigate('#/flocks')">Cancel</button>
        </div>
      </form>
    </div>
  `);

  document.getElementById('transferFlockForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const newHouse = document.getElementById('transferHouseNumber').value.trim();
    if (!confirm(`Confirm transfer of ${flock.name} to Layer House ${newHouse}? This cannot be undone.`)) return;

    const btn = document.getElementById('transferSubmitBtn');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Transferring…';

    const result = await api.TransferFlock({
      id:               id,
      new_house_number: newHouse,
      new_building:     document.getElementById('transferBuilding').value.trim(),
    });

    if (result) {
      navigate('#/flocks');
    } else {
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-arrow-right-circle me-2"></i>Confirm Transfer';
    }
  });
}

// ── House History ─────────────────────────────────────────────────────────────
// Note: loadHouseHistory is defined in operations.js (handles {logs, log_type} response).
// Flocks module routes #/flocks/:id/history here via Modules.Flocks.load → loadHouseHistory.

// ── Body Weight ───────────────────────────────────────────────────────────────

async function loadBodyWeight(id) {
  showLoading();
  const bwData = await api.GetBodyWeightData(id);
  if (!bwData) {
    showView(`<div class="alert alert-warning m-4">Failed to load body weight data.</div>`);
    return;
  }

  const logs = bwData.logs || [];
  const rows = logs.length === 0
    ? `<tr><td colspan="4" class="text-center text-muted py-4">No weight records yet.</td></tr>`
    : logs.map(l => `
        <tr>
          <td>${formatDate(l.date)}</td>
          <td>${formatNumber(l.sample_size ?? 0)}</td>
          <td>${l.avg_weight_g != null ? l.avg_weight_g.toFixed(1) + ' g' : '—'}</td>
          <td>${l.notes || '—'}</td>
        </tr>
      `).join('');

  showView(`
    <div class="container-fluid p-4">
      <div class="d-flex align-items-center gap-2 mb-4">
        <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/flocks')">
          <i class="bi bi-arrow-left"></i>
        </button>
        <h4 class="fw-bold mb-0">Body Weight — ${bwData.flock_name || ''}</h4>
      </div>

      ${logs.length > 0 ? `
      <div class="card border-0 shadow-sm mb-4">
        <div class="card-body">
          <canvas id="bwChart" height="100"></canvas>
        </div>
      </div>
      ` : ''}

      <div class="card border-0 shadow-sm">
        <div class="card-header bg-transparent fw-semibold">Weight Log</div>
        <div class="table-responsive">
          <table class="table table-hover align-middle mb-0">
            <thead class="table-light">
              <tr><th>Date</th><th>Sample Size</th><th>Avg Weight</th><th>Notes</th></tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>
    </div>
  `);

  if (logs.length > 0) {
    const labels = logs.map(l => formatDate(l.date));
    const weights = logs.map(l => l.avg_weight_g ?? 0);
    const ctx = document.getElementById('bwChart').getContext('2d');
    new Chart(ctx, {
      type: 'line',
      data: {
        labels,
        datasets: [{
          label: 'Avg Body Weight (g)',
          data: weights,
          borderColor: 'rgba(59,130,246,0.9)',
          backgroundColor: 'rgba(59,130,246,0.1)',
          tension: 0.3,
          fill: true,
        }]
      },
      options: {
        responsive: true,
        plugins: { legend: { display: false } },
        scales: { y: { beginAtZero: false } }
      }
    });
  }
}
