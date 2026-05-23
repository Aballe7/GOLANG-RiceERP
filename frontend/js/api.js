/**
 * api.js — Thin wrapper around Wails Go bindings.
 * All calls show a toast on error and return response.data on success.
 *
 * Usage:
 *   const data = await api.GetDashboard();  // returns data or null on error
 *   const user = await api.Login({username:'admin', password:'...'});
 */

// The Wails runtime injects window.go.main.App with all exported methods
const _go = window.go?.app?.App;

const api = new Proxy({}, {
  get(_, method) {
    return async (...args) => {
      if (!_go || !_go[method]) {
        toast(`Go binding not found: ${method}`, 'danger');
        return null;
      }
      try {
        const resp = await _go[method](...args);
        if (!resp.ok) {
          console.error(`[api] ${method} failed:`, resp.message, resp);
          toast(resp.message || 'An error occurred', 'danger', 8000);
          return null;
        }
        // Show success toast only for mutating operations
        if (resp.message && !isQueryMethod(method)) {
          toast(resp.message, 'success');
        }
        // null means the Go handler returned nil data (e.g. update/delete with no payload) → treat as success
        return (resp.data !== undefined && resp.data !== null) ? resp.data : true;
      } catch (err) {
        console.error(`[api] ${method} threw:`, err);
        toast(`Error calling ${method}: ${err.message || err}`, 'danger');
        return null;
      }
    };
  }
});

function isQueryMethod(method) {
  const queryPrefixes = ['Get', 'List', 'Search', 'Compute', 'Generate'];
  return queryPrefixes.some(p => method.startsWith(p));
}

/**
 * Show a Bootstrap toast notification.
 * @param {string} message
 * @param {'success'|'danger'|'warning'|'info'} type
 * @param {number} [duration=4000] ms
 */
function toast(message, type = 'info', duration = 4000) {
  const container = document.getElementById('toast-container');
  if (!container) return;

  const id = 'toast-' + Date.now();
  const iconMap = {
    success: 'bi-check-circle-fill',
    danger: 'bi-x-circle-fill',
    warning: 'bi-exclamation-triangle-fill',
    info: 'bi-info-circle-fill',
  };
  const icon = iconMap[type] || 'bi-info-circle-fill';

  const html = `
    <div id="${id}" class="toast align-items-center text-white bg-${type} border-0 show" role="alert">
      <div class="d-flex">
        <div class="toast-body">
          <i class="bi ${icon} me-2"></i>${message}
        </div>
        <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
      </div>
    </div>`;

  container.insertAdjacentHTML('beforeend', html);
  const el = document.getElementById(id);
  setTimeout(() => el?.remove(), duration + 300);
}
