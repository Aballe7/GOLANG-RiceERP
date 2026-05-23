// production.js — Modules.Production

Modules.Production = {
  async load(sub, id, action) {
    const active = sub || 'milling-orders';
    showLoading();

    // Detail / form views (no tab bar needed)
    if (active === 'milling-orders' && action === 'new') { await loadNewMillingOrderForm(); return; }
    if (active === 'milling-orders' && id)               { await loadMillingOrderDetail(id); return; }
    if (active === 'work-orders'    && action === 'new') { await loadNewWOForm(); return; }
    if (active === 'work-orders'    && id)               { await loadWODetail(id); return; }

    const tabBar = `
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link ${active === 'milling-orders' ? 'active' : ''}" href="#"
            onclick="navigate('#/production/milling-orders');return false;">
            <i class="bi bi-gear-wide-connected me-1"></i>Milling Orders
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'bom' ? 'active' : ''}" href="#"
            onclick="navigate('#/production/bom');return false;">
            <i class="bi bi-diagram-3 me-1"></i>Bill of Materials
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link ${active === 'work-orders' ? 'active' : ''}" href="#"
            onclick="navigate('#/production/work-orders');return false;">
            <i class="bi bi-clipboard2-check me-1"></i>Work Orders
          </a>
        </li>
      </ul>
    `;

    if (active === 'milling-orders') await loadMillingOrdersList(tabBar);
    else if (active === 'bom')        await loadBOMList(tabBar);
    else if (active === 'work-orders') await loadWOList(tabBar);
    else await loadMillingOrdersList(tabBar);
  },

  reset() {
    _resetProductionMilling();
    _resetProductionWorkorders();
    _resetProductionBOM();
  }
};
