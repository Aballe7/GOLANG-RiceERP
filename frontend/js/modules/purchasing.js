// purchasing.js — Modules.Purchasing

Modules.Purchasing = {
  async load(sub, id, action) {
    const active = sub || 'purchases';
    showLoading();

    // Detail / form views (no tab bar needed)
    if (active === 'purchases' && action === 'new')    { await loadNewPurchaseForm(); return; }
    if (active === 'purchases' && id)                  { await loadPurchaseDetail(id); return; }
    if (active === 'delivery-receipts' && action === 'new') { await loadNewDRForm(); return; }
    if (active === 'delivery-receipts' && id)          { await loadDRDetail(id); return; }
    if (active === 'ap-invoices' && action === 'new')  { await loadNewAPInvoiceForm(); return; }
    if (active === 'ap-invoices' && id)                { await loadAPInvoiceDetail(id); return; }
    if (active === 'ap-payments' && action === 'new')  { await loadNewAPPaymentForm(); return; }
    if (active === 'ap-payments' && id)                { await loadAPPaymentDetail(id); return; }

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'purchases' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/purchases');return false;">
            <i class="bi bi-cart me-1"></i>Purchases
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'delivery-receipts' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/delivery-receipts');return false;">
            <i class="bi bi-truck me-1"></i>Delivery Receipts
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ap-invoices' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/ap-invoices');return false;">
            <i class="bi bi-file-earmark-text me-1"></i>AP Invoices
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ap-payments' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/ap-payments');return false;">
            <i class="bi bi-cash me-1"></i>AP Payments
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'suppliers' ? 'active' : ''}" href="#"
            onclick="navigate('#/purchasing/suppliers');return false;">
            <i class="bi bi-building me-1"></i>Suppliers
          </a>
        </li>
      </ul>
    `;

    if (active === 'purchases')         await loadPurchasesList(tabBar);
    else if (active === 'delivery-receipts') await loadDeliveryReceiptsList(tabBar);
    else if (active === 'ap-invoices')  await loadApInvoicesList(tabBar);
    else if (active === 'ap-payments')  await loadApPaymentsList(tabBar);
    else if (active === 'suppliers')    await this.loadSuppliers(id, tabBar);
    else await loadPurchasesList(tabBar);
  },

  async loadSuppliers(id, tabBar) {
    await loadSuppliersView(id, tabBar);
  },

  reset() {
    _resetPurchasingSuppliers();
    _resetPurchasingPO();
    _resetPurchasingDR();
    _resetPurchasingAPInvoice();
    _resetPurchasingAPPayment();
  }
};
