// backup.js — Modules.Backup

Modules.Backup = {
  async load() {
    showLoading();

    // Get backup folder information (used in the UI note)
    let backupDir = '~/EggLayerERP/backups';
    try {
      const dirResp = await window.go.app.App.GetBackupDir();
      if (dirResp && dirResp.ok && dirResp.data) {
        backupDir = dirResp.data;
      }
    } catch (e) {
      // ignore; we'll display the default path
    }

    // Use the raw binding so we can display error messages from the backend.
    const resp = await window.go.app.App.GetBackupHistory();
    if (!resp || !resp.ok) {
      const msg = resp?.message || 'Failed to load backup history.';
      showView(`<div class="alert alert-warning m-4">${_esc(msg)}</div>`);
      return;
    }

    const history = resp.data || [];

    const rows = history.length === 0
      ? `<tr><td colspan="4" class="text-center text-muted py-5">No backups found. Create your first backup.</td></tr>`
      : history.map(b => {
          const ts = b.created_at ? new Date(b.created_at) : null;
          const dateStr = ts ? ts.toLocaleDateString('en-CA') : '—';
          const timeStr = ts ? ts.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '';
          return `
          <tr>
            <td class="fw-semibold">
              <i class="bi bi-file-earmark-text me-2 text-secondary"></i>${_esc(b.file_name || '—')}
            </td>
            <td>${formatBackupSize(b.file_size_bytes)}</td>
            <td class="text-nowrap small">${dateStr} <span class="text-muted">${timeStr}</span></td>
            <td class="text-end">
              <button class="btn btn-sm btn-outline-secondary me-1"
                title="Download" onclick="downloadBackup(${JSON.stringify(b.file_name)})">
                <i class="bi bi-download"></i>
              </button>
              <button class="btn btn-sm btn-outline-danger"
                title="Delete" onclick="confirmDeleteBackup(${JSON.stringify(b.file_name)})">
                <i class="bi bi-trash"></i>
              </button>
            </td>
          </tr>
        `;}).join('');

    showView(`
      <div class="container-fluid p-4">
        <div class="d-flex align-items-center justify-content-between mb-4">
          <div>
            <h4 class="fw-bold mb-0"><i class="bi bi-cloud-upload me-2"></i>Backup</h4>
            <p class="text-muted small mb-0 mt-1">Backups are saved to <code id="backupDirPath">${_esc(backupDir)}</code>. Use <i class="bi bi-download"></i> to export a copy elsewhere.</p>
          </div>
          <div class="d-flex gap-2 flex-wrap align-items-start">
            <div>
              <button class="btn btn-primary" id="createBackupBtn" onclick="createBackup()">
                <i class="bi bi-plus-circle me-2"></i>Create Backup
              </button>
              <button class="btn btn-outline-primary" id="createBackupSaveBtn" onclick="createBackupAndSave()">
                <i class="bi bi-folder2-open me-2"></i>Backup & Save...
              </button>
              <div id="backupInlineError" class="text-danger small mt-2 p-2 bg-danger-subtle rounded" style="display:none;white-space:pre-wrap;max-width:600px;"></div>
            </div>
            <button class="btn btn-outline-secondary btn-sm align-self-center" onclick="runBackupDiagnostics()" title="Run diagnostics and show detailed output">
              <i class="bi bi-bug me-1"></i>Diagnostics
            </button>
          </div>
        </div>

        <!-- Storage Info -->
        <div class="row g-3 mb-4">
          <div class="col-sm-4">
            <div class="card border-0 shadow-sm">
              <div class="card-body text-center">
                <div class="fs-4 fw-bold text-primary">${history.length}</div>
                <div class="text-muted small">Total Backups</div>
              </div>
            </div>
          </div>
          <div class="col-sm-4">
            <div class="card border-0 shadow-sm">
              <div class="card-body text-center">
                <div class="fs-4 fw-bold text-success">
                  ${formatBackupSize(history.reduce((s, b) => s + (b.file_size_bytes ?? 0), 0))}
                </div>
                <div class="text-muted small">Total Size</div>
              </div>
            </div>
          </div>
          <div class="col-sm-4">
            <div class="card border-0 shadow-sm">
              <div class="card-body text-center">
                <div class="fs-5 fw-bold text-secondary">
                  ${history.length > 0 ? new Date(history[0].created_at).toLocaleDateString('en-CA') : '—'}
                </div>
                <div class="text-muted small">Latest Backup</div>
              </div>
            </div>
          </div>
        </div>

        <!-- Backup History Table -->
        <div class="card border-0 shadow-sm">
          <div class="card-header bg-transparent fw-semibold d-flex align-items-center justify-content-between">
            Backup History
            <button class="btn btn-outline-secondary btn-sm" onclick="Modules.Backup.load()">
              <i class="bi bi-arrow-clockwise me-1"></i>Refresh
            </button>
          </div>
          <div class="table-responsive">
            <table class="table table-hover align-middle mb-0">
              <thead class="table-light">
                <tr>
                  <th>File Name</th>
                  <th>Size</th>
                  <th>Date Created</th>
                  <th class="text-end">Actions</th>
                </tr>
              </thead>
              <tbody>${rows}</tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Error Detail Modal -->
      <div class="modal fade" id="backupErrorModal" tabindex="-1">
        <div class="modal-dialog modal-lg">
          <div class="modal-content border-0 shadow">
            <div class="modal-header border-0 bg-danger text-white">
              <h6 class="modal-title"><i class="bi bi-exclamation-circle me-2"></i>Backup Error</h6>
              <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
              <pre id="backupErrorText" class="bg-light rounded p-3 small text-danger mb-0" style="white-space:pre-wrap;word-break:break-all;max-height:400px;overflow:auto;"></pre>
            </div>
            <div class="modal-footer border-0">
              <button class="btn btn-sm btn-secondary" data-bs-dismiss="modal">Close</button>
            </div>
          </div>
        </div>
      </div>

      <!-- Confirm Delete Modal -->
      <div class="modal fade" id="deleteBackupModal" tabindex="-1">
        <div class="modal-dialog modal-sm">
          <div class="modal-content border-0 shadow">
            <div class="modal-header border-0">
              <h6 class="modal-title text-danger">
                <i class="bi bi-exclamation-triangle me-2"></i>Delete Backup?
              </h6>
              <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
              <p class="mb-1 small">Are you sure you want to delete:</p>
              <p class="fw-semibold small" id="deleteBackupName"></p>
              <p class="text-danger small mb-0">This action cannot be undone.</p>
            </div>
            <div class="modal-footer border-0">
              <button class="btn btn-sm btn-secondary" data-bs-dismiss="modal">Cancel</button>
              <button class="btn btn-sm btn-danger" id="confirmDeleteBtn">Delete</button>
            </div>
          </div>
        </div>
      </div>
    `);
  }
};

// ── Diagnostics ───────────────────────────────────────────────────────────────

async function runBackupDiagnostics() {
  const errDiv = document.getElementById('backupInlineError');
  if (errDiv) {
    errDiv.textContent = 'Running diagnostics…';
    errDiv.style.display = 'block';
  }
  try {
    const resp = await window.go.app.App.BackupDiagnostics();
    const output = resp?.data || resp?.message || JSON.stringify(resp);
    if (errDiv) {
      errDiv.textContent = output;
      errDiv.style.display = 'block';
      errDiv.className = 'small mt-2 p-2 bg-light border rounded text-dark';
    }
  } catch (ex) {
    if (errDiv) {
      errDiv.textContent = 'Diagnostics threw: ' + (ex?.message || String(ex));
      errDiv.style.display = 'block';
    }
  }
}

// ── Error display ─────────────────────────────────────────────────────────────

function showBackupError(msg) {
  const el = document.getElementById('backupErrorText');
  if (el) el.textContent = msg;
  const modal = document.getElementById('backupErrorModal');
  if (modal) new bootstrap.Modal(modal).show();
  else toast('Error: ' + msg.substring(0, 120), 'danger'); // fallback
}

// ── Create Backup ─────────────────────────────────────────────────────────────

async function createBackup() {
  const btn = document.getElementById('createBackupBtn');
  const errDiv = document.getElementById('backupInlineError');
  if (errDiv) { errDiv.style.display = 'none'; errDiv.textContent = ''; }

  if (btn) {
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Creating…';
  }

  const reset = () => {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = '<i class="bi bi-plus-circle me-2"></i>Create Backup';
    }
  };

  let resp;
  try {
    resp = await window.go.app.App.CreateBackup();
  } catch (ex) {
    reset();
    const msg = 'Wails binding threw an exception:\n' + (ex?.message || String(ex));
    console.error('[backup] exception:', ex);
    _showBackupInlineError(msg);
    showBackupError(msg);
    return;
  }

  if (!resp || !resp.ok) {
    reset();
    const msg = resp?.message || ('Unexpected null response — resp: ' + JSON.stringify(resp));
    console.error('[backup] failed:', msg);
    _showBackupInlineError(msg);
    showBackupError(msg);
    return;
  }

  reset();
  const fname = resp.data?.file_name || 'done';
  const fpath = resp.data?.path || '';
  toast(`Backup saved: ${fname}${fpath ? ' → ' + fpath : ''}`, 'success');
  await Modules.Backup.load();
}

// ── Create Backup + Save ─────────────────────────────────────────────────────

async function createBackupAndSave() {
  const btn = document.getElementById('createBackupSaveBtn');
  const createBtn = document.getElementById('createBackupBtn');
  const errDiv = document.getElementById('backupInlineError');
  if (errDiv) { errDiv.style.display = 'none'; errDiv.textContent = ''; }

  const setLoading = (loading) => {
    if (btn) {
      btn.disabled = loading;
      btn.innerHTML = loading
        ? '<span class="spinner-border spinner-border-sm me-2"></span>Backing up…'
        : '<i class="bi bi-folder2-open me-2"></i>Backup & Save...';
    }
    if (createBtn) createBtn.disabled = loading;
  };

  setLoading(true);

  let resp;
  try {
    resp = await window.go.app.App.CreateBackup();
  } catch (ex) {
    setLoading(false);
    const msg = 'Wails binding threw an exception:\n' + (ex?.message || String(ex));
    console.error('[backup] exception:', ex);
    _showBackupInlineError(msg);
    showBackupError(msg);
    return;
  }

  if (!resp || !resp.ok) {
    setLoading(false);
    const msg = resp?.message || ('Unexpected null response — resp: ' + JSON.stringify(resp));
    console.error('[backup] failed:', msg);
    _showBackupInlineError(msg);
    showBackupError(msg);
    return;
  }

  const fname = resp.data?.file_name;
  if (!fname) {
    setLoading(false);
    showBackupError('Unexpected response: missing file name');
    return;
  }

  // Open save dialog and copy file to chosen location.
  toast('Opening save dialog…', 'info');
  let dlResp;
  try {
    dlResp = await window.go.app.App.DownloadBackup(fname);
  } catch (ex) {
    setLoading(false);
    showBackupError('Download failed: ' + (ex?.message || String(ex)));
    return;
  }
  setLoading(false);
  if (!dlResp || !dlResp.ok) {
    const msg = dlResp?.message || '';
    if (msg.toLowerCase().includes('cancel')) {
      toast('Save cancelled.', 'warning');
    } else {
      showBackupError(msg || 'Download failed.');
    }
  }

  await Modules.Backup.load();
}

function _showBackupInlineError(msg) {
  const el = document.getElementById('backupInlineError');
  if (el) {
    el.textContent = msg;
    el.style.display = 'block';
  }
}

// ── Download Backup ───────────────────────────────────────────────────────────

async function downloadBackup(fileName) {
  toast('Opening save dialog…', 'info');
  let resp;
  try {
    resp = await window.go.app.App.DownloadBackup(fileName);
  } catch (ex) {
    showBackupError('Download failed: ' + (ex?.message || String(ex)));
    return;
  }
  if (!resp || !resp.ok) {
    const msg = resp?.message || '';
    if (msg.toLowerCase().includes('cancel')) {
      toast('Save cancelled.', 'warning');
    } else {
      showBackupError(msg || 'Download failed.');
    }
    return;
  }
  toast('Backup saved successfully.', 'success');
}

// ── Delete Backup ─────────────────────────────────────────────────────────────

function confirmDeleteBackup(fileName) {
  const nameEl = document.getElementById('deleteBackupName');
  if (nameEl) nameEl.textContent = fileName;

  const confirmBtn = document.getElementById('confirmDeleteBtn');
  if (confirmBtn) {
    // Remove previous listeners by replacing the element
    const newBtn = confirmBtn.cloneNode(true);
    confirmBtn.parentNode.replaceChild(newBtn, confirmBtn);
    newBtn.addEventListener('click', async () => {
      newBtn.disabled = true;
      newBtn.textContent = 'Deleting…';
      const result = await api.DeleteBackup(fileName);
      bootstrap.Modal.getInstance(document.getElementById('deleteBackupModal')).hide();
      if (result) {
        toast(`Backup deleted.`, 'success');
        await Modules.Backup.load();
      } else {
        toast('Failed to delete backup.', 'danger');
      }
    });
  }
  new bootstrap.Modal(document.getElementById('deleteBackupModal')).show();
}

// ── Size Formatter ────────────────────────────────────────────────────────────

function formatBackupSize(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let i = 0;
  let size = bytes;
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024;
    i++;
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}
