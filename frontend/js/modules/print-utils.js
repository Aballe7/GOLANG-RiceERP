// print-utils.js — shared print helper for all document types

window._printDoc = function(title, bodyHtml) {
  const w = window.open('', '_blank', 'width=820,height=700');
  w.document.write(`<!DOCTYPE html><html><head>
  <title>${title}</title>
  <style>
    body { font-family: Arial, sans-serif; font-size: 12px; margin: 28px; color: #222; }
    h2 { margin: 0 0 2px; font-size: 17px; }
    .sub { color: #555; font-size: 12px; margin-bottom: 14px; }
    .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4px 24px; margin-bottom: 14px; font-size: 12px; }
    .info-grid span { color: #555; }
    table { width: 100%; border-collapse: collapse; margin-bottom: 10px; font-size: 11.5px; }
    th { background: #f0f0f0; text-align: left; padding: 5px 7px; border-bottom: 2px solid #ccc; }
    td { padding: 4px 7px; border-bottom: 1px solid #e0e0e0; vertical-align: top; }
    .text-end { text-align: right; }
    .totals { text-align: right; font-size: 12px; margin-top: 4px; }
    .totals strong { font-size: 13px; }
    hr { border: none; border-top: 1px solid #ccc; margin: 12px 0; }
    .footer { margin-top: 28px; font-size: 10px; color: #999; }
    .badge { display: inline-block; padding: 1px 7px; border-radius: 8px; font-size: 11px; background: #eee; }
    .section-title { font-size: 12px; font-weight: bold; margin: 14px 0 4px; border-bottom: 1px solid #ddd; padding-bottom: 3px; }
  </style></head><body>
  ${bodyHtml}
  <div class="footer">Printed: ${new Date().toLocaleString()}</div>
  <script>window.onload=function(){window.print();window.onafterprint=function(){window.close();};};<\/script>
  </body></html>`);
  w.document.close();
};

// Format helpers for print (no DOM dependency)
window._pfmt = {
  currency: v => '₱' + (parseFloat(v) || 0).toLocaleString('en-PH', { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
  date: v => {
    if (!v) return '—';
    const d = new Date(v);
    return isNaN(d) ? v : d.toLocaleDateString('en-PH', { year: 'numeric', month: 'short', day: 'numeric' });
  },
  num: v => (parseFloat(v) || 0).toLocaleString('en-PH', { maximumFractionDigits: 3 }),
};
