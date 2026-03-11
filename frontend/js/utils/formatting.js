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
  const d = new Date(dateStr);
  if (isNaN(d)) return dateStr;
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
