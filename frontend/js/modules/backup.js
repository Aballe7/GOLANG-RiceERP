// backup.js — Modules.Backup

Modules.Backup = {
  async load() {
    showLoading();
    const history = await api.GetBackupHistory();
    if (!history) {
      showView(`<div class="alert alert-warning m-4">Failed to load backup history.</div>`);
      return;
    }

    const rows = history.length === 0
      ? `<tr><td colspan="4" class="text-center text-muted py-5">No backups found. Create your first backup.</td></tr>`
      : history.map(b => `
          <tr>
            <td class="fw-semibold">
              <i class="bi bi-file-zip me-2 text-secondary"></i>${b.file_name || '—'}
            </td>
            <td>${formatBackupSize(b.file_size_bytes)}</td>
            <td>${formatDate(b.created_at)} ${b.time_str || ''}</td>
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
        `).join('');

    showView(`
      <div class="container-fluid p-4">
        <div class="d-flex align-items-center justify-content-between mb-4">
          <div>
            <h4 class="fw-bold mb-0"><i class="bi bi-cloud-upload me-2"></i>Backup</h4>
            <p class="text-muted small mb-0 mt-1">Manage database backups.</p>
          </div>
          <button class="btn btn-primary" id="createBackupBtn" onclick="createBackup()">
            <i class="bi bi-plus-circle me-2"></i>Create Backup
          </button>
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
                  ${history.length > 0 ? formatDate(history[0].created_at) : '—'}
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

// ── Create Backup ─────────────────────────────────────────────────────────────

async function createBackup() {
  const btn = document.getElementById('createBackupBtn');
  if (!btn) return;
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Creating…';

  const result = await api.CreateBackup();
  if (result) {
    toast(`Backup created: ${result.file_name || 'backup file'}`, 'success');
    await Modules.Backup.load();
  } else {
    toast('Failed to create backup.', 'danger');
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-plus-circle me-2"></i>Create Backup';
  }
}

// ── Download Backup ───────────────────────────────────────────────────────────

async function downloadBackup(fileName) {
  const result = await api.DownloadBackup(fileName);
  if (!result) {
    toast('Failed to download backup.', 'danger');
    return;
  }
  // result is expected to be a file path or blob trigger handled by Go
  toast(`Downloading ${fileName}…`, 'info');
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
