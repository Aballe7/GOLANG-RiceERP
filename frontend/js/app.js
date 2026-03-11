/**
 * app.js — SPA router, auth guard, startup lifecycle.
 *
 * Hash-based routing: #/module/view/id
 * Navigation guard: every route change checks if user is logged in
 *                   and has permission for that module.
 */

let currentUser = null;

// ─── Module → route prefix mapping ───────────────────────────────────────────
const MODULE_ROUTES = {
  'Flocks':     ['flocks'],
  'Operations': ['operations', 'record-log'],
  'Inventory':  ['inventory'],
  'Purchasing': ['purchasing', 'suppliers', 'delivery-receipts', 'ap-invoices', 'ap-payments'],
  'Sales':      ['sales', 'customers', 'price-groups', 'delivery-orders', 'ar-invoices', 'collections'],
  'Reports':    ['reports'],
};

const ADMIN_ROUTES = ['accounting', 'admin', 'audit-log', 'backup'];

// ─── Startup ─────────────────────────────────────────────────────────────────
window.addEventListener('DOMContentLoaded', () => {
  const runtime = window.runtime || window.wailsjs?.runtime;

  if (runtime) {
    runtime.EventsOn('startup:ready', onStartupReady);
    runtime.EventsOn('startup:error', onStartupError);
    runtime.EventsOn('startup:db_error', onStartupError);
  } else {
    // Dev mode: skip startup overlay
    setTimeout(onStartupReady, 300);
  }
});

function onStartupReady() {
  document.getElementById('startup-overlay').classList.add('d-none');
  // Check if already logged in (shouldn't be on fresh start, but just in case)
  showLoginPage();
}

function onStartupError(msg) {
  document.getElementById('startup-spinner').classList.add('d-none');
  const errEl = document.getElementById('startup-error');
  errEl.textContent = msg;
  errEl.classList.remove('d-none');
  document.getElementById('startup-status').textContent = 'Startup failed';
}

// ─── Login ────────────────────────────────────────────────────────────────────
function showLoginPage() {
  document.getElementById('login-page').classList.remove('d-none');
  document.getElementById('app-shell').classList.add('d-none');
  document.getElementById('login-username').focus();
}

document.getElementById('login-form')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = document.getElementById('login-username').value.trim();
  const password = document.getElementById('login-password').value;
  const errEl = document.getElementById('login-error');
  errEl.classList.add('d-none');

  const user = await api.Login({ username, password });
  if (!user) {
    errEl.textContent = 'Invalid username or password.';
    errEl.classList.remove('d-none');
    return;
  }
  currentUser = user;
  onLogin();
});

function onLogin() {
  document.getElementById('login-page').classList.add('d-none');
  document.getElementById('app-shell').classList.remove('d-none');
  document.getElementById('nav-username').textContent = currentUser.full_name || currentUser.username;

  // Show/hide admin nav items
  const isAdmin = currentUser.role === 'Admin';
  document.querySelectorAll('.nav-admin').forEach(el => {
    el.classList.toggle('d-none', !isAdmin);
  });

  // Show/hide module nav items based on permissions
  document.querySelectorAll('.nav-module').forEach(el => {
    const mod = el.dataset.module;
    el.classList.toggle('d-none', !canAccess(mod));
  });

  // Route to dashboard
  navigate('#/dashboard');
}

document.getElementById('logout-btn')?.addEventListener('click', async (e) => {
  e.preventDefault();
  await api.Logout();
  currentUser = null;
  showLoginPage();
});

// ─── Permission helpers ───────────────────────────────────────────────────────
function canAccess(module) {
  if (!currentUser) return false;
  if (currentUser.role === 'Admin') return true;
  const perms = currentUser.permissions;
  if (!perms || perms.length === 0) return true; // no restrictions
  return perms.includes(module);
}

function requireAuth() {
  if (!currentUser) { showLoginPage(); return false; }
  return true;
}

function requireModule(module) {
  if (!requireAuth()) return false;
  if (!canAccess(module)) {
    showView(`<div class="alert alert-danger"><i class="bi bi-lock me-2"></i>
      You don't have permission to access <strong>${module}</strong>.</div>`);
    return false;
  }
  return true;
}

// ─── Router ───────────────────────────────────────────────────────────────────
function navigate(hash) {
  window.location.hash = hash;
}

window.addEventListener('hashchange', handleRoute);

function handleRoute() {
  if (!currentUser) return; // not logged in
  const hash = window.location.hash || '#/dashboard';
  console.log('[route]', hash);
  const parts = hash.replace('#/', '').split('/');
  const route = parts[0];
  const p1 = parts[1];
  const p2 = parts[2];
  const p3 = parts[3];

  // Determine id / sub / action from path segments.
  // Pattern A: #/route/id[/sub]   → p1 is numeric  (e.g. #/flocks/5/history)
  // Pattern B: #/route/sub[/id|action] → p1 is text (e.g. #/purchasing/purchases/new)
  const p1IsNum = p1 && /^\d+$/.test(p1);
  const p2IsNum = p2 && /^\d+$/.test(p2);
  let id, sub, action;
  if (p1IsNum) {
    id  = parseInt(p1, 10);
    sub = p2;
    action = p3;
  } else {
    sub = p1;
    if (p2IsNum) {
      id = parseInt(p2, 10);
      action = p3;
    } else {
      action = p2;
    }
  }

  // Highlight active nav link
  document.querySelectorAll('.nav-link, .dropdown-item').forEach(a => {
    a.classList.toggle('active', a.getAttribute('href') === hash);
  });

  console.log('[route] parsed →', { route, id, sub, action });
  routeSwitch(route, id, sub, action, parts);
}

async function routeSwitch(route, id, sub, action, parts) {
  switch (route) {
    case 'dashboard':    return await loadDashboard();
    case 'flocks':       return await loadFlocks(id, sub);
    case 'operations':   return await loadOperations(id, sub);
    case 'inventory':    return await loadInventory(sub);
    case 'purchasing':   return await loadPurchasing(sub, id, action);
    case 'suppliers':    return await loadSuppliers(id);
    case 'sales':        return await loadSales(sub, id, action);
    case 'customers':    return await loadCustomers(id);
    case 'price-groups': return await loadPriceGroups(id);
    case 'accounting':   return await loadAccounting(sub, id);
    case 'reports':      return await loadReports(sub);
    case 'backup':       return await loadBackup();
    case 'admin':        return await loadAdmin(sub, id);
    case 'audit-log':    return await loadAuditLog();
    default:
      showView('<div class="alert alert-warning">Page not found.</div>');
  }
}

// ─── View loader ─────────────────────────────────────────────────────────────
function showView(html) {
  document.getElementById('view-container').innerHTML = html;
}

function showLoading() {
  showView('<div class="text-center py-5"><div class="spinner-border text-warning"></div></div>');
}

// ─── Module loaders (defined in each module's JS file) ───────────────────────
// These are stubs that will be overridden by module JS files:
async function loadDashboard() { if (!requireAuth()) return; await Modules.Dashboard.load(); }
async function loadFlocks(id, sub) { if (!requireModule('Flocks')) return; await Modules.Flocks.load(id, sub); }
async function loadOperations(id, sub) { if (!requireModule('Operations')) return; await Modules.Operations.load(id, sub); }
async function loadInventory(sub) { if (!requireModule('Inventory')) return; await Modules.Inventory.load(sub); }
async function loadPurchasing(sub, id, action) { if (!requireModule('Purchasing')) return; await Modules.Purchasing.load(sub, id, action); }
async function loadSuppliers(id) { if (!requireModule('Purchasing')) return; await Modules.Purchasing.loadSuppliers(id); }
async function loadSales(sub, id, action) { if (!requireModule('Sales')) return; await Modules.Sales.load(sub, id, action); }
async function loadCustomers(id) { if (!requireModule('Sales')) return; await Modules.Sales.loadCustomers(id); }
async function loadPriceGroups(id) { if (!requireModule('Sales')) return; await Modules.Sales.loadPriceGroups(id); }
async function loadAccounting(sub, id) { if (!currentUser || currentUser.role !== 'Admin') { showView('<div class="alert alert-danger">Admin access required.</div>'); return; } await Modules.Accounting.load(sub, id); }
async function loadReports(sub) { if (!requireModule('Reports')) return; await Modules.Reports.load(sub); }
async function loadBackup() { if (!currentUser || currentUser.role !== 'Admin') { showView('<div class="alert alert-danger">Admin access required.</div>'); return; } await Modules.Backup.load(); }
async function loadAdmin(sub, id) { if (!currentUser || currentUser.role !== 'Admin') { showView('<div class="alert alert-danger">Admin access required.</div>'); return; } await Modules.Admin.load(sub, id); }
async function loadAuditLog() { if (!currentUser || currentUser.role !== 'Admin') return; await Modules.Admin.loadAuditLog(); }

// Module registry — each module file populates this
const Modules = {};
