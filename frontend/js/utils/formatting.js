// formatting.js — shared formatters used across all modules

const PHP_LOCALE = 'en-PH';
const CURRENCY_SYM = '₱';

/**
 * Format a number as Philippine Peso currency.
 * @param {number} amount
 * @returns {string}
 */
function formatCurrency(amount) {
  if (amount === null || amount === undefined) return CURRENCY_SYM + '0.00';
  return CURRENCY_SYM + Number(amount).toLocaleString(PHP_LOCALE, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

/**
 * Format a stock quantity with 3 decimal places and comma separators.
 * Trailing zeros are trimmed (e.g. 500.000 → "500", 5.500 → "5.5").
 * @param {number} qty
 * @returns {string}
 */
function formatQty(qty) {
  if (qty === null || qty === undefined) return '0';
  const n = Number(qty);
  // Show up to 3 decimal places, remove trailing zeros
  return n.toLocaleString(PHP_LOCALE, {
    minimumFractionDigits: 0,
    maximumFractionDigits: 3,
  });
}

/**
 * Format a number with comma separators and given decimal places.
 * @param {number} n
 * @param {number} [decimals=0]
 */
function formatNumber(n, decimals = 0) {
  if (n === null || n === undefined) return '0';
  return Number(n).toLocaleString(PHP_LOCALE, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

/**
 * Format a date string (ISO 8601 or Date object) to "Jan 09, 2025" style.
 * @param {string|Date} dateStr
 */
function formatDate(dateStr) {
  if (!dateStr) return '—';
  // Go zero time ("0001-01-01…") is not a real date — treat as empty
  if (typeof dateStr === 'string' && dateStr.startsWith('0001-')) return '—';
  // Parse using only the date portion (YYYY-MM-DD) at local noon to avoid
  // timezone roll-over that can shift the displayed day by ±1.
  const datePart = typeof dateStr === 'string' ? dateStr.slice(0, 10) : null;
  const d = datePart && /^\d{4}-\d{2}-\d{2}$/.test(datePart)
    ? new Date(datePart + 'T12:00:00')
    : new Date(dateStr);
  if (isNaN(d)) return '—';
  return d.toLocaleDateString(PHP_LOCALE, { year: 'numeric', month: 'short', day: '2-digit' });
}

/**
 * Format a date to YYYY-MM-DD (for input[type=date] default values).
 * @param {string|Date} [d]
 */
function toInputDate(d) {
  const dt = d ? new Date(d) : new Date();
  return dt.toISOString().slice(0, 10);
}

/**
 * Convert pieces to trays (30 pcs/tray).
 * @param {number} pieces
 */
function piecesToTrays(pieces) {
  return Math.floor(pieces / 30);
}

/**
 * Convert trays to pieces.
 * @param {number} trays
 */
function traysToPieces(trays) {
  return trays * 30;
}
