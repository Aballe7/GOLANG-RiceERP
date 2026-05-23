// relationship-map.js — SAP B1-style Document Flow (Relationship Map)
//
// Entry point:
//   showRelationshipMap(chain, rootId, currentType, currentId)
//     chain       : 'purchasing' | 'sales'
//     rootId      : PO.id  (purchasing) | SO.id (sales)
//     currentType : 'PO'|'DR'|'AP_INV'|'AP_PAY'|'SO'|'DO'|'AR_INV'|'COL'
//     currentId   : id of the currently-open document (highlighted in map)

window.showRelationshipMap = async function(chain, rootId, currentType, currentId) {
  if (!rootId) {
    toast('Cannot determine source document chain.', 'warning');
    return;
  }

  // Open the modal shell immediately with a spinner — never disturb the page behind it
  document.getElementById('relMapModal')?.remove();
  document.body.insertAdjacentHTML('beforeend', `
    <div class="modal fade" id="relMapModal" tabindex="-1" aria-hidden="true">
      <div class="modal-dialog modal-xl modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header border-0 pb-1">
            <h5 class="modal-title fw-bold">
              <i class="bi bi-diagram-3 me-2 text-info"></i>Document Flow
            </h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
          </div>
          <div class="modal-body pt-1 pb-4" id="relMapBody">
            <div class="text-center py-5">
              <div class="spinner-border text-info" role="status"></div>
              <div class="text-muted small mt-2">Loading document chain…</div>
            </div>
          </div>
        </div>
      </div>
    </div>`);
  new bootstrap.Modal(document.getElementById('relMapModal')).show();

  let stages;
  try {
    if (chain === 'purchasing') {
      const po = await api.GetPurchaseHeader(rootId);
      if (!po) { toast('Could not load document chain.', 'warning'); relMapClose(); return; }
      stages = _buildPurchasingStages(po, currentType, Number(currentId));
    } else {
      const so = await api.GetSalesOrder(rootId);
      if (!so) { toast('Could not load document chain.', 'warning'); relMapClose(); return; }
      stages = _buildSalesStages(so, currentType, Number(currentId));
    }
  } catch (e) {
    toast('Could not load document chain.', 'warning');
    relMapClose();
    return;
  }
  _fillRelMapBody(stages);
};

window.relMapClose = function() {
  const el = document.getElementById('relMapModal');
  if (el) bootstrap.Modal.getInstance(el)?.hide();
};

// ── Stage builders ────────────────────────────────────────────────────────────

function _isCur(type, id, cType, cId) {
  return type === cType && Number(id) === Number(cId);
}

function _buildPurchasingStages(po, cType, cId) {
  // Collect unique AP Payments from all AP Invoice payment lines
  const payMap = new Map();
  (po.ap_invoices || []).forEach(inv => {
    (inv.payment_lines || []).forEach(pl => {
      if (pl.payment && pl.payment.id && !payMap.has(pl.payment.id)) {
        payMap.set(pl.payment.id, pl.payment);
      }
    });
  });

  return [
    {
      label: 'Purchase Order',
      route: '#/purchasing/purchases/',
      docs: [{
        id: po.id, type: 'PO',
        num:    po.po_number || `PO-${String(po.id).padStart(5,'0')}`,
        status: po.status,
        amount: po.doc_total || 0,
        current: _isCur('PO', po.id, cType, cId),
      }],
    },
    {
      label: 'Delivery Receipt',
      route: '#/purchasing/delivery-receipts/',
      docs: (po.delivery_receipts || []).map(dr => ({
        id: dr.id, type: 'DR',
        num:    dr.dr_number || `DR-${String(dr.id).padStart(5,'0')}`,
        status: dr.status,
        amount: dr.doc_total || 0,
        current: _isCur('DR', dr.id, cType, cId),
      })),
    },
    {
      label: 'AP Invoice',
      route: '#/purchasing/ap-invoices/',
      docs: (po.ap_invoices || []).map(inv => ({
        id: inv.id, type: 'AP_INV',
        num:    inv.invoice_number || `AP-${String(inv.id).padStart(5,'0')}`,
        status: inv.status,
        amount: inv.doc_total || 0,
        current: _isCur('AP_INV', inv.id, cType, cId),
      })),
    },
    {
      label: 'AP Payment',
      route: '#/purchasing/ap-payments/',
      docs: [...payMap.values()].map(pay => ({
        id: pay.id, type: 'AP_PAY',
        num:    pay.payment_number || `PAY-${String(pay.id).padStart(5,'0')}`,
        status: pay.status || 'Posted',
        amount: pay.total_amount || 0,
        current: _isCur('AP_PAY', pay.id, cType, cId),
      })),
    },
  ];
}

function _buildSalesStages(so, cType, cId) {
  return [
    {
      label: 'Sales Order',
      route: '#/sales/orders/',
      docs: [{
        id: so.id, type: 'SO',
        num:    so.sales_order_number || `SO-${String(so.id).padStart(5,'0')}`,
        status: so.doc_status || so.status || '—',
        amount: so.grand_total || 0,
        current: _isCur('SO', so.id, cType, cId),
      }],
    },
    {
      label: 'Delivery Order',
      route: '#/sales/delivery-orders/',
      docs: (so.deliveries || []).map(d => {
        const total = (d.items || []).reduce(
          (s, i) => s + ((i.quantity_delivered || 0) * (i.price_per_unit || 0)), 0
        );
        return {
          id: d.id, type: 'DO',
          num:    d.delivery_number || `DO-${String(d.id).padStart(5,'0')}`,
          status: d.status,
          amount: total,
          current: _isCur('DO', d.id, cType, cId),
        };
      }),
    },
    {
      label: 'AR Invoice',
      route: '#/sales/ar-invoices/',
      docs: (so.ar_invoices || []).map(inv => ({
        id: inv.id, type: 'AR_INV',
        num:    inv.invoice_number || `AR-${String(inv.id).padStart(5,'0')}`,
        status: inv.status,
        amount: inv.total_amount || 0,
        current: _isCur('AR_INV', inv.id, cType, cId),
      })),
    },
  ];
}

// ── Rendering ─────────────────────────────────────────────────────────────────

function _relBadge(status) {
  const map = {
    Open:      'primary',   Partial:   'warning',  Paid:      'success',
    Posted:    'success',   Draft:     'secondary', Pending:   'warning',
    Approved:  'primary',   Delivered: 'success',   Received:  'success',
    Confirmed: 'primary',   Closed:    'info',      Cancelled: 'dark',
  };
  return `<span class="badge bg-${map[status] || 'secondary'} text-white">${status || '—'}</span>`;
}

function _relNodeHtml(node, route) {
  const borderCls = node.current ? 'border-primary' : 'border-light';
  const bgCls     = node.current ? 'bg-primary bg-opacity-10' : '';
  const cursor    = node.current ? 'default' : 'pointer';
  const click     = node.current ? '' : `onclick="navigate('${route}${node.id}');relMapClose()"`;
  const curLabel  = node.current
    ? `<div class="text-primary mb-1" style="font-size:9px;font-weight:700;letter-spacing:.07em;text-transform:uppercase">● Current</div>`
    : '';
  return `
    <div class="card border-2 shadow-sm ${borderCls} ${bgCls}"
         style="cursor:${cursor};min-width:152px;max-width:172px" ${click}>
      <div class="card-body py-2 px-3">
        ${curLabel}
        <div class="fw-bold small text-truncate" title="${node.num}">${node.num}</div>
        <div class="mt-1">${_relBadge(node.status)}</div>
        <div class="text-muted mt-1" style="font-size:.7rem">${formatCurrency(node.amount)}</div>
      </div>
    </div>`;
}

function _stageColHtml(stage, isLast) {
  const nodesHtml = stage.docs.length === 0
    ? `<div class="card border-light" style="min-width:152px">
         <div class="card-body py-2 px-3 text-center text-muted small">
           <i class="bi bi-dash-circle me-1"></i>None
         </div>
       </div>`
    : stage.docs.map(n => _relNodeHtml(n, stage.route)).join('\n');

  const arrow = isLast ? '' : `
    <div class="d-flex align-items-start px-2" style="padding-top:38px">
      <i class="bi bi-arrow-right fs-4 text-muted"></i>
    </div>`;

  return `
    <div class="d-flex align-items-start">
      <div class="d-flex flex-column gap-2">
        <div class="text-muted fw-bold text-center"
             style="font-size:.65rem;text-transform:uppercase;letter-spacing:.06em;min-width:152px;margin-bottom:2px">
          ${stage.label}
        </div>
        ${nodesHtml}
      </div>
      ${arrow}
    </div>`;
}

function _fillRelMapBody(stages) {
  const body = document.getElementById('relMapBody');
  if (!body) return;
  const stagesHtml = stages
    .map((s, i) => _stageColHtml(s, i === stages.length - 1))
    .join('');
  body.innerHTML = `
    <p class="text-muted small mb-3">
      <i class="bi bi-info-circle me-1"></i>
      Click any document box to navigate to it. The highlighted box is the currently open document.
    </p>
    <div class="d-flex align-items-start flex-nowrap gap-0 overflow-auto pb-2">
      ${stagesHtml}
    </div>`;
}
