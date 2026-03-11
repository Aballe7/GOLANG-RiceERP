// admin.js — Modules.Admin

const ALL_MODULES = [
  'Dashboard', 'Flocks', 'Operations', 'Inventory',
  'Purchasing', 'Sales', 'Accounting', 'Reports', 'Admin', 'Backup'
];

Modules.Admin = {
  async load(sub, id) {
    const active = sub || 'users';
    showLoading();

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'users' ? 'active' : ''}" href="#"
            onclick="navigate('#/admin/users');return false;">
            <i class="bi bi-people me-1"></i>Users
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'settings' ? 'active' : ''}" href="#"
            onclick="navigate('#/admin/settings');return false;">
            <i class="bi bi-gear me-1"></i>Farm Settings
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'audit' ? 'active' : ''}" href="#"
            onclick="navigate('#/admin/audit');return false;">
            <i class="bi bi-clock-history me-1"></i>Audit Log
          </a>
        </li>
      </ul>
    `;

    if (active === 'users') await loadUsersView(id, tabBar);
    else if (active === 'settings') await loadFarmSettings(tabBar);
    else if (active === 'audit') await this.loadAuditLog(tabBar);
    else await loadUsersView(id, tabBar);
  },

  async loadAuditLog(tabBar) {
    await loadAuditLogView(tabBar);
  }
};

// ── Users ─────────────────────────────────────────────────────────────────────

let _userList = [];

async function loadUsersView(editId, tabBar) {
  const users = await api.ListUsers();
  _userList = users || [];

  const rows = _userList.length === 0
    ? `<tr><td colspan="6" class="text-center text-muted py-4">No users found.</td></tr>`
    : _userList.map(u => `
        <tr>
          <td class="fw-semibold">${u.username || '—'}</td>
          <td>${u.full_name || '—'}</td>
          <td>${u.email || '—'}</td>
          <td>
            <span class="badge bg-${roleBadgeColor(u.role)}">${u.role || 'User'}</span>
          </td>
          <td>
            <span class="badge bg-${u.is_active ? 'success' : 'secondary'}">
              ${u.is_active ? 'Active' : 'Inactive'}
            </span>
          </td>
          <td class="text-end">
            <button class="btn btn-sm btn-outline-secondary me-1" onclick="openUserModal(${u.id})">
              <i class="bi bi-pencil"></i>
            </button>
            <button class="btn btn-sm btn-outline-info" onclick="openPermissionsModal(${u.id})">
              <i class="bi bi-shield-lock"></i>
            </button>
          </td>
        </tr>
      `).join('');

  showView(wrapAdmin(tabBar, `
    <div class="d-flex align-items-center justify-content-between mb-3">
      <span class="text-muted small">${_userList.length} user(s)</span>
      <button class="btn btn-primary btn-sm" onclick="openUserModal(null)">
        <i class="bi bi-person-plus me-1"></i>Add User
      </button>
    </div>
    <div class="card border-0 shadow-sm">
      <div class="table-responsive">
        <table class="table table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Username</th><th>Full Name</th><th>Email</th><th>Role</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>

    <!-- Add/Edit User Modal -->
    <div class="modal fade" id="userModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="userModalTitle">User</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <input type="hidden" id="userId">
            <div class="row g-3">
              <div class="col-md-6">
                <label class="form-label">Username <span class="text-danger">*</span></label>
                <input type="text" class="form-control" id="userUsername" required autocomplete="off">
              </div>
              <div class="col-md-6">
                <label class="form-label">Full Name</label>
                <input type="text" class="form-control" id="userFullName">
              </div>
              <div class="col-md-6">
                <label class="form-label">Email</label>
                <input type="email" class="form-control" id="userEmail">
              </div>
              <div class="col-md-6">
                <label class="form-label">Role</label>
                <select class="form-select" id="userRole">
                  <option value="Admin">Admin</option>
                  <option value="Manager">Manager</option>
                  <option value="Operator">Operator</option>
                  <option value="Viewer">Viewer</option>
                </select>
              </div>
              <div class="col-md-6" id="userPasswordGroup">
                <label class="form-label">Password <span class="text-danger" id="pwRequired">*</span></label>
                <input type="password" class="form-control" id="userPassword" autocomplete="new-password">
                <div class="form-text" id="pwHint"></div>
              </div>
              <div class="col-md-6">
                <label class="form-label">Status</label>
                <select class="form-select" id="userActive">
                  <option value="true">Active</option>
                  <option value="false">Inactive</option>
                </select>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitUserForm()">Save</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Permissions Modal -->
    <div class="modal fade" id="permissionsModal" tabindex="-1">
      <div class="modal-dialog">
        <div class="modal-content border-0 shadow">
          <div class="modal-header">
            <h5 class="modal-title" id="permModalTitle">Edit Permissions</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body">
            <input type="hidden" id="permUserId">
            <p class="text-muted small mb-3">Select modules this user can access:</p>
            <div id="permCheckboxes" class="row g-2">
              ${ALL_MODULES.map(m => `
                <div class="col-6">
                  <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="perm_${m}" value="${m}">
                    <label class="form-check-label" for="perm_${m}">${m}</label>
                  </div>
                </div>
              `).join('')}
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
            <button class="btn btn-primary" onclick="submitPermissions()">Save Permissions</button>
          </div>
        </div>
      </div>
    </div>
  `));
}

function roleBadgeColor(role) {
  const map = { Admin: 'danger', Manager: 'primary', Operator: 'success', Viewer: 'secondary' };
  return map[role] || 'secondary';
}

function openUserModal(id) {
  const isEdit = id !== null;
  document.getElementById('userModalTitle').textContent = isEdit ? 'Edit User' : 'Add User';
  document.getElementById('userId').value = id || '';
  document.getElementById('pwRequired').style.display = isEdit ? 'none' : '';
  document.getElementById('pwHint').textContent = isEdit ? 'Leave blank to keep current password.' : '';
  document.getElementById('userPassword').required = !isEdit;

  if (isEdit) {
    const u = _userList.find(x => x.id === id);
    if (u) {
      document.getElementById('userUsername').value = u.username || '';
      document.getElementById('userFullName').value = u.full_name || '';
      document.getElementById('userEmail').value = u.email || '';
      document.getElementById('userRole').value = u.role || 'Operator';
      document.getElementById('userActive').value = String(u.is_active !== false);
      document.getElementById('userPassword').value = '';
    }
  } else {
    ['userUsername','userFullName','userEmail','userPassword'].forEach(id => document.getElementById(id).value = '');
    document.getElementById('userRole').value = 'Operator';
    document.getElementById('userActive').value = 'true';
  }
  new bootstrap.Modal(document.getElementById('userModal')).show();
}

async function submitUserForm() {
  const id = document.getElementById('userId').value;
  const password = document.getElementById('userPassword').value;
  const payload = {
    username:  document.getElementById('userUsername').value.trim(),
    full_name: document.getElementById('userFullName').value.trim(),
    email:     document.getElementById('userEmail').value.trim(),
    role:      document.getElementById('userRole').value,
    is_active: document.getElementById('userActive').value === 'true',
  };
  if (!payload.username) { toast('Username is required.', 'warning'); return; }
  if (!id && !password) { toast('Password is required for new users.', 'warning'); return; }
  if (password) payload.password = password;

  const result = id
    ? await api.UpdateUser({ ...payload, id: parseInt(id, 10) })
    : await api.CreateUser(payload);

  if (result) {
    toast(id ? 'User updated.' : 'User created.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('userModal')).hide();
    navigate('#/admin/users');
  } else {
    toast('Failed to save user.', 'danger');
  }
}

async function openPermissionsModal(userId) {
  const u = _userList.find(x => x.id === userId);
  document.getElementById('permModalTitle').textContent = `Permissions — ${u?.username || ''}`;
  document.getElementById('permUserId').value = userId;

  // Reset checkboxes
  ALL_MODULES.forEach(m => {
    const cb = document.getElementById(`perm_${m}`);
    if (cb) cb.checked = false;
  });

  // Load existing permissions
  const perms = await api.GetUserPermissions(userId);
  if (perms && perms.modules) {
    perms.modules.forEach(m => {
      const cb = document.getElementById(`perm_${m}`);
      if (cb) cb.checked = true;
    });
  }

  new bootstrap.Modal(document.getElementById('permissionsModal')).show();
}

async function submitPermissions() {
  const userId = document.getElementById('permUserId').value;
  const selected = ALL_MODULES.filter(m => document.getElementById(`perm_${m}`)?.checked);
  const result = await api.SetUserPermissions({ user_id: parseInt(userId, 10), modules: selected });
  if (result) {
    toast('Permissions updated.', 'success');
    bootstrap.Modal.getInstance(document.getElementById('permissionsModal')).hide();
  } else {
    toast('Failed to update permissions.', 'danger');
  }
}

// ── Farm Settings ─────────────────────────────────────────────────────────────

async function loadFarmSettings(tabBar) {
  const settings = await api.GetFarmSettings();

  showView(wrapAdmin(tabBar, `
    <div class="card border-0 shadow-sm" style="max-width:620px">
      <div class="card-header bg-transparent fw-semibold">
        <i class="bi bi-house-gear me-2"></i>Farm Settings
      </div>
      <div class="card-body">
        <form id="farmSettingsForm">
          <div class="mb-3">
            <label class="form-label">Farm Name <span class="text-danger">*</span></label>
            <input type="text" class="form-control" id="farmName" required
              value="${settings?.farm_name || ''}" placeholder="My Egg Farm">
          </div>
          <div class="mb-3">
            <label class="form-label">Farm Address</label>
            <textarea class="form-control" id="farmAddress" rows="2"
              placeholder="Full address">${settings?.farm_address || ''}</textarea>
          </div>
          <div class="mb-3">
            <label class="form-label">Contact Number</label>
            <input type="text" class="form-control" id="farmContact"
              value="${settings?.farm_contact || ''}" placeholder="+63 XXX XXX XXXX">
          </div>
          <div class="mb-3">
            <label class="form-label">Email</label>
            <input type="email" class="form-control" id="farmEmail"
              value="${settings?.farm_email || ''}">
          </div>
          <div class="mb-3">
            <label class="form-label">Currency Symbol</label>
            <input type="text" class="form-control" id="farmCurrency" style="max-width:100px"
              value="${settings?.currency_symbol || '₱'}">
          </div>
          <button type="submit" class="btn btn-primary px-4">
            <i class="bi bi-save me-1"></i>Save Settings
          </button>
        </form>
      </div>
    </div>
  `));

  document.getElementById('farmSettingsForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const payload = {
      farm_name:       document.getElementById('farmName').value.trim(),
      farm_address:    document.getElementById('farmAddress').value.trim(),
      farm_contact:    document.getElementById('farmContact').value.trim(),
      farm_email:      document.getElementById('farmEmail').value.trim(),
      currency_symbol: document.getElementById('farmCurrency').value.trim() || '₱',
    };
    if (!payload.farm_name) { toast('Farm name is required.', 'warning'); return; }
    const btn = e.target.querySelector('[type=submit]');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Saving…';
    const result = await api.SaveFarmSettings(payload);
    if (result) {
      toast('Settings saved.', 'success');
    } else {
      toast('Failed to save settings.', 'danger');
    }
    btn.disabled = false;
    btn.innerHTML = '<i class="bi bi-save me-1"></i>Save Settings';
  });
}

// ── Audit Log ─────────────────────────────────────────────────────────────────

async function loadAuditLogView(tabBar) {
  const logs = await api.GetAuditLogs(500);
  if (!logs) {
    showView(wrapAdmin(tabBar, `<div class="alert alert-warning">Failed to load audit log.</div>`));
    return;
  }

  const renderTable = (data) => {
    if (data.length === 0)
      return `<tr><td colspan="6" class="text-center text-muted py-4">No audit entries found.</td></tr>`;
    return data.map(l => {
      const ts = l.timestamp ? new Date(l.timestamp) : null;
      const dateStr = ts ? ts.toLocaleDateString('en-CA') : '—';
      const timeStr = ts ? ts.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '';
      return `
        <tr>
          <td class="text-nowrap small text-muted">${dateStr} <span class="text-secondary">${timeStr}</span></td>
          <td>${_esc(l.username || '—')}</td>
          <td><span class="badge bg-${auditActionColor(l.action)}">${_esc(l.action || '—')}</span></td>
          <td class="small">${_esc(l.module || '—')}</td>
          <td class="small">${_esc(l.entity_ref || '—')}</td>
          <td class="small text-muted">${_esc(l.description || '—')}</td>
        </tr>`;
    }).join('');
  };

  // unique values for filter dropdowns
  const users   = [...new Set(logs.map(l => l.username).filter(Boolean))].sort();
  const actions = [...new Set(logs.map(l => l.action).filter(Boolean))].sort();
  const modules = [...new Set(logs.map(l => l.module).filter(Boolean))].sort();

  showView(wrapAdmin(tabBar, `
    <div class="card border-0 shadow-sm">
      <div class="card-header bg-transparent">
        <div class="d-flex flex-wrap align-items-center gap-2">
          <span class="fw-semibold me-auto">Audit Log</span>
          <input id="al-search" type="search" class="form-control form-control-sm" style="width:180px" placeholder="Search description…">
          <select id="al-user" class="form-select form-select-sm" style="width:130px">
            <option value="">All Users</option>
            ${users.map(u => `<option>${_esc(u)}</option>`).join('')}
          </select>
          <select id="al-action" class="form-select form-select-sm" style="width:120px">
            <option value="">All Actions</option>
            ${actions.map(a => `<option>${_esc(a)}</option>`).join('')}
          </select>
          <select id="al-module" class="form-select form-select-sm" style="width:130px">
            <option value="">All Modules</option>
            ${modules.map(m => `<option>${_esc(m)}</option>`).join('')}
          </select>
          <span id="al-count" class="text-muted small">${logs.length} entries</span>
          <button class="btn btn-outline-secondary btn-sm" onclick="navigate('#/admin/audit')">
            <i class="bi bi-arrow-clockwise me-1"></i>Refresh
          </button>
        </div>
      </div>
      <div class="table-responsive">
        <table class="table table-sm table-hover align-middle mb-0">
          <thead class="table-light">
            <tr><th>Timestamp</th><th>User</th><th>Action</th><th>Module</th><th>Entity</th><th>Description</th></tr>
          </thead>
          <tbody id="al-tbody">${renderTable(logs)}</tbody>
        </table>
      </div>
    </div>
  `));

  // live filtering
  const filter = () => {
    const q      = document.getElementById('al-search').value.toLowerCase();
    const user   = document.getElementById('al-user').value;
    const action = document.getElementById('al-action').value;
    const mod    = document.getElementById('al-module').value;
    const filtered = logs.filter(l =>
      (!q      || (l.description || '').toLowerCase().includes(q) || (l.entity_ref || '').toLowerCase().includes(q)) &&
      (!user   || l.username === user) &&
      (!action || l.action   === action) &&
      (!mod    || l.module   === mod)
    );
    document.getElementById('al-tbody').innerHTML = renderTable(filtered);
    document.getElementById('al-count').textContent = `${filtered.length} / ${logs.length} entries`;
  };
  ['al-search','al-user','al-action','al-module'].forEach(id =>
    document.getElementById(id).addEventListener('input', filter)
  );
}

function auditActionColor(action) {
  const map = {
    CREATE: 'success', UPDATE: 'primary', DELETE: 'danger',
    VOID: 'danger', CANCEL: 'warning', CONFIRM: 'info',
    LOGIN: 'info', LOGOUT: 'secondary', ERROR: 'warning'
  };
  return map[(action || '').toUpperCase()] || 'secondary';
}

// ── Layout Helper ─────────────────────────────────────────────────────────────

function wrapAdmin(tabBar, content) {
  return `
    <div class="container-fluid p-4">
      <h4 class="fw-bold mb-4"><i class="bi bi-shield-gear me-2"></i>Administration</h4>
      ${tabBar}
      ${content}
    </div>
  `;
}
