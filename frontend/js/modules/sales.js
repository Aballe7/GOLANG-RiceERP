// sales.js — Modules.Sales

Modules.Sales = {
  async load(sub, id, action) {
    const active = sub || 'orders';
    showLoading();

    // Detail / form views (no tab bar needed)
    if (active === 'orders' && action === 'new')           { await loadNewSOForm(); return; }
    if (active === 'orders' && action === 'edit' && id)    { await loadEditSOForm(id); return; }
    if (active === 'orders' && id)                         { await loadSODetail(id); return; }
    if (active === 'delivery-orders' && action === 'new')  { await loadNewDOForm(); return; }
    if (active === 'delivery-orders' && id)                { await loadDODetail(id); return; }
    if (active === 'ar-invoices' && action === 'new')        { await loadNewARInvoiceForm(); return; }
    if (active === 'ar-invoices' && action === 'standalone') { await loadNewStandaloneARForm(); return; }
    if (active === 'ar-invoices' && id)                      { await loadARInvoiceDetail(id); return; }
    if (active === 'collections' && action === 'new')      { await loadNewCollectionForm(); return; }
    if (active === 'collections' && id)                    { await loadCollectionDetail(id); return; }
    if (active === 'price-groups' && id)                   { await loadPriceGroupDetail(id); return; }

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'orders' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/orders');return false;">
            <i class="bi bi-bag me-1"></i>Sales Orders
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'delivery-orders' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/delivery-orders');return false;">
            <i class="bi bi-truck me-1"></i>Delivery Orders
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'ar-invoices' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/ar-invoices');return false;">
            <i class="bi bi-file-earmark-text me-1"></i>AR Invoices
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'collections' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/collections');return false;">
            <i class="bi bi-cash-coin me-1"></i>Collections
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'customers' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/customers');return false;">
            <i class="bi bi-people me-1"></i>Customers
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'price-groups' ? 'active' : ''}" href="#"
            onclick="navigate('#/sales/price-groups');return false;">
            <i class="bi bi-tags me-1"></i>Price Groups
          </a>
        </li>
      </ul>
    `;

    if (active === 'orders') await loadSalesOrdersList(tabBar);
    else if (active === 'delivery-orders') await loadDeliveryOrdersList(tabBar);
    else if (active === 'ar-invoices') await loadArInvoicesList(tabBar);
    else if (active === 'collections') await loadCollectionsList(tabBar);
    else if (active === 'customers') await this.loadCustomers(id, tabBar);
    else if (active === 'price-groups') await this.loadPriceGroups(id, tabBar);
    else await loadSalesOrdersList(tabBar);
  },

  async loadCustomers(id, tabBar) {
    await loadCustomersView(id, tabBar);
  },

  async loadPriceGroups(id, tabBar) {
    await loadPriceGroupsView(id, tabBar);
  },

  reset() {
    _resetSalesOrders();
    _resetSalesDO();
    _resetSalesAR();
    _resetSalesCustomers();
    _resetSalesCollections();
    _resetSalesPriceGroups();
  }
};
