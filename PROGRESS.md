# EggLayerERP-Go — Session Progress Tracker

Last updated: 2026-03-10 (session 3)

---

## Completed Work

### Accounting JE Wiring ✓
- [x] `purchasing/service.go` — wired 4 JE posting blocks (GRNI, AP Invoice, Cancel AP Invoice, AP Payment)
- [x] `sales/service.go` — wired 3 JE posting blocks (AR Invoice, Cancel AR Invoice, Collection)
- [x] `go build ./...` compiles clean

### Frontend Bootstrap Fixes ✓
- [x] Fixed Wails namespace: `window.go.app.App` (not `window.go.main.App`)
- [x] Switched vendor CSS/JS to CDN (Bootstrap 5.3.3, Bootstrap Icons 1.11.3, Chart.js 4.4.3)
- [x] Removed wailsjs script tags (Wails injects `window.go` automatically)
- [x] Fixed router to correctly parse 3-segment hash routes (`#/route/id/sub`)

### Runtime Bug Fixes ✓
- [x] `app.go`: `ListItemMasters(true)`, `ListARInvoices(customerID, nil, status)`
- [x] `wails.json`: fixed `"cmd /c echo no-install"` for Windows
- [x] `models/settings.go`: removed `default:''` from TEXT column (MySQL error 1101)
- [x] `purchasing.js`: `api.ListDeliveryReceipts(0)` (was called with no args)
- [x] `sales.js`: `api.ListARInvoices(null, '')` (was `ListArInvoices()` — wrong case + no args)
- [x] `purchasing/service.go`: `ListDeliveryReceipts(0)` returns all when purchaseID=0
- [x] `sales/service.go`: `ListDeliveryOrders(0)` returns all when soID=0

### UI Features Added ✓
- [x] Feed dashboard icon changed from `bi-fuel-pump` → `bi-bag` (inventory.js + operations.js)
- [x] New Purchase Order form (`loadNewPurchaseForm`) in purchasing.js
- [x] Delivery receipt creation now supplier‑first; undelivered POs filtered by supplier
- [x] Payment status on purchase orders repurposed as delivery status (Partial/Delivered)
- [x] Purchase detail view (`loadPurchaseDetail`) in purchasing.js
- [x] **Item Categories UI** — full CRUD under Inventory → Item Categories tab
  - Model: `ItemCategory` (`item_categories` table)
  - Service: `ListItemCategories`, `GetItemCategory`, `CreateItemCategory`, `UpdateItemCategory`, `DeleteItemCategory`
  - Handler + app.go wired
  - Frontend: `loadItemCategories()`, `openCategoryModal()`, `submitCategoryForm()`, `deleteCategory()`
- [x] **Operations module rework** — tab-based layout
  - "Daily Log" tab: flock cards with Log Today buttons (unchanged behavior)
  - "Vaccine Schedule" tab: list all scheduled vaccines, add new, mark as done
  - `CreateVaccineSchedule` and `MarkVaccineComplete` wired in app.go
  - `ListVaccineSchedules(0)` returns all schedules when flockID=0

### Flask Template UI Review — All Modules Upgraded ✓ (session 3)

#### Purchasing (`frontend/js/modules/purchasing.js`)
- [x] DR detail view: header, items table, linked AP Invoices, Confirm/Cancel/Create AP Invoice buttons
- [x] AP Invoice new form: copy from PO → filter DRs → items table with qty/price adjustment
- [x] AP Invoice detail: status banner, line items, payment history
- [x] AP Payment new form: supplier select → open/partial invoices (client-side filter) → post
- [x] AP Payment detail: payment header + linked invoice lines
- [x] Fixed field bugs in AP Payments list (`payment_date`→`date`, `supplier_name`→`supplier_name_snapshot`, `amount`→`total_amount`)
- [x] Model: `APInvoices []APInvoice` added to `DeliveryReceipt`; `Payment *APPayment` added to `APPaymentLine`
- [x] Service: `GetDeliveryReceipt` preloads APInvoices; `GetAPInvoice` preloads `PaymentLines.Payment`

#### Sales (`frontend/js/modules/sales.js`)
- [x] Fixed `load(sub, id)` → `load(sub, id, action)` — missing action param caused route failures
- [x] Fixed field bugs: SO list (`invoice_number`, `date`, `customer_name_snapshot`, `grand_total`, `payment_status`); DO list (`delivery_number`, `sales_order.customer_name_snapshot`); AR list (`date`, `customer_name_snapshot`, status `Open`); Collections list (`date`, `customer_name_snapshot`, `total_amount`)
- [x] SO detail view: header, items, delivery orders, AR invoices, Void button
- [x] DO detail view: header, items, linked AR invoices, Confirm button
- [x] New DO form: copy from SO → adjustable qty items
- [x] AR Invoice new form: copy from SO or DO → items
- [x] AR Invoice detail: status, header, items, collection history with balance
- [x] New Collection form: customer select → open AR invoices → fill balances → post
- [x] Collection detail: payment info + applied invoice lines
- [x] Model: `Collection *Collection` added to `CollectionLine`
- [x] Service: `GetARInvoice` preloads `CollectionLines.Collection`; `GetDeliveryOrder` preloads ARInvoices; `ListDeliveryOrders` preloads SalesOrder

#### Dashboard (`frontend/js/modules/dashboard.js` + `app/handlers/dashboard.go`)
- [x] KPI strip: HDP%, FCR, Mortality Rate, Reject Rate — computed from last 30 days, color-coded
- [x] Layer + Grower house tiles grid — clickable, opens Bootstrap offcanvas panel
- [x] Offcanvas house panel: bird count, quick-action buttons (Record Log → navigate to ops, View History)
- [x] Population summary card (total, layer/grower breakdown)
- [x] Quick Actions card (navigate to other modules)
- [x] Recent Sales card (last 5 orders with status badges)
- [x] Alerts section: unlogged houses + low feed stock
- [x] Customize modal: localStorage widget show/hide toggles
- [x] Backend `DashboardData` extended: `FlockTiles`, `LayerBirds`, `GrowerBirds`, `RecentOrders`, `HDP`, `FCR`, `MortalityRate`, `RejectRate`

#### Operations (`frontend/js/modules/operations.js` + `app/app.go`)
- [x] House history view (`#/operations/history/{flockId}`): log table with date, eggs, mortality, feed, vaccine, Edit buttons
- [x] Edit log form layer (`#/operations/edit-log-layer/{logId}`): full egg harvest + feed + health form
- [x] Edit log form grower (`#/operations/edit-log-grower/{logId}`): feed + health + remarks form
- [x] Correction warning banner; mortality delta adjusts flock bird count automatically
- [x] New bindings in `app.go`: `GetDailyLog`, `GetGrowerLog`, `UpdateDailyLog`, `UpdateGrowerLog`
- [x] Dashboard offcanvas "Record Daily Log" / "View History" buttons navigate to ops routes

---

## Posting Events Reference
| Event              | Module     | DR                  | CR                   |
|--------------------|------------|---------------------|----------------------|
| Confirm DR         | PURCHASING | INVENTORY_RECEIVED  | GRNI                 |
| AP Invoice (w/ DR) | PURCHASING | GRNI                | AP_PAYABLE           |
| AP Invoice (no DR) | PURCHASING | INVENTORY_RECEIVED  | AP_PAYABLE           |
| Cancel AP Invoice  | PURCHASING | (auto-reversed)     | (auto-reversed)      |
| AP Payment         | PURCHASING | AP_CLEARING         | Cash/Bank (OUTFLOW)  |
| AR Invoice         | SALES      | AR_RECEIVABLE       | SALES_REVENUE        |
| Cancel AR Invoice  | SALES      | (auto-reversed)     | (auto-reversed)      |
| Collection         | SALES      | Cash/Bank (INFLOW)  | COLLECTION_CLEARING  |

---

### Price Groups & Customer Price List ✓ (session 3 cont.)
- [x] Price Groups list: card layout (name, description, Active badge, tray price badges, customer count, Edit button)
- [x] "New Price Group" modal → creates group then navigates to detail to set prices
- [x] Price Group detail (`#/sales/price-groups/{id}`):
  - Left: editable name/description + price table (all egg sizes × Tray/Piece) with `÷30 Piece` and `Clear` helpers → `UpsertPriceGroupItem` per row on save
  - Right: assigned customers list with remove button (`UpdateCustomer` sets `price_group_id: null`) + assign dropdown (`UpdateCustomer` sets `price_group_id: pgId`)
  - Delete group button (soft-deactivates)
- [x] Customer table: fixed all field names (`contact_number`, `customer_type`, `price_group.name`); Price Group badge + tag icon links to group detail
- [x] Customer modal: fixed to correct model fields (`contact_number`, `customer_type`, `delivery_address`, `notes`); Price Group now a `<select>` dropdown (not free text); `price_group_id` sent as FK
- [x] Service: `ListPriceGroups` and `GetPriceGroup` now preload `Customers`

## Known Remaining Work
- Account Determination must be configured in UI before JE posting works
- Full end-to-end testing: Purchase → DR → AP Invoice → Payment → JE verification
- Full end-to-end testing: Sales Order → Delivery → AR Invoice → Collection → JE verification

---

## Purchasing Process Map (Testing Guide)

### 1. Purchase Order (PO)

- **Create PO header**
  - Screen: `Purchasing → Purchases → New Purchase Order`
  - Backend:
    - Writes `purchase_header` (header) and `purchase_line` (single line for now).
    - Also mirrors to legacy `purchase` row (until fully retired).
  - Key fields to verify:
    - Header: supplier, posting date, PO number, total amount.
    - Line: item description, quantity, unit, unit price, line total.

### 2. Delivery Receipt (DR)

- **Create DR from one or more POs**
  - Screen: `Purchasing → Delivery Receipts → New Delivery Receipt`
  - Flow:
    - Select supplier → system filters undelivered POs for that supplier.
    - Multi-select POs to receive against.
    - Enter DR date, supplier DR ref, notes.
  - Backend:
    - Creates one `delivery_receipt` per selected PO (current implementation).
    - Each DR links back to its PO via `purchase_id` and has `dr_line` rows.
  - Status:
    - New DRs are `Draft`.

- **Confirm DR**
  - Screen: DR detail → `Confirm Receipt`.
  - Backend:
    - Changes DR status to `Received`.
    - Increments `purchase.amount_received`.
    - Sets PO delivery status to `Partial` / `Delivered`.
    - Posts JE: **DR_CONFIRM** (DR → GRNI & Inventory Received).

### 3. AP Invoice

- **Create AP Invoice from PO or DR**
  - Screen: `Purchasing → AP Invoices → New AP Invoice`
  - Source:
    - Choose PO only, or DR (when goods already received).
    - Items grid is prefilled from PO line or DR lines; quantities/prices can be adjusted.
  - Backend:
    - Writes `ap_invoice` header + `ap_invoice_line` rows.
    - Updates `purchase.amount_invoiced` and, if linked to DR, `delivery_receipt.amount_invoiced`.
    - Posts JE: **AP_INVOICE** (GRNI or Inventory Received ↔ AP Payable).

### 4. AP Payment

- **Record payment against one or more AP Invoices**
  - Screen: `Purchasing → AP Payments → New AP Payment`
  - Flow:
    - Select supplier → system lists Open/Partial AP Invoices.
    - Choose invoices and amounts to apply.
  - Backend:
    - Writes `ap_payment` header + `ap_payment_line` rows.
    - Updates `ap_invoice.amount_paid_stored` and status (Open → Partial → Paid).
    - Updates `purchase.amount_settled`.
    - Posts JE: **AP_PAYMENT** (AP Clearing ↔ Cash/Bank OUTFLOW).

### 5. Quick End-to-End Test Path

1. Create PO for a supplier (verify header + line saved).
2. Create DR for that PO and **Confirm** it (verify PO delivery status and GRNI/Inventory JE).
3. Create AP Invoice from that DR (verify totals, status, and AP JE).
4. Create AP Payment for that invoice (verify invoice status, purchase settlement, and payment JE).

---

## Key File Paths
| File | Purpose |
|------|---------|
| `app/app.go` | All Go binding registrations |
| `app/db/migrate.go` | AutoMigrate table list |
| `app/models/inventory.go` | ItemCategory, ItemMaster, Inventory models |
| `app/services/inventory/service.go` | Inventory + ItemCategory + ItemMaster CRUD |
| `app/handlers/inventory.go` | Inventory handler functions |
| `app/services/purchasing/service.go` | P2P service + JE wiring |
| `app/services/sales/service.go` | O2C service + JE wiring |
| `app/services/accounting/engine.go` | PostJournalEntry, ReverseJournalEntry |
| `app/services/accounting/determination.go` | GetAccount, GetPaymentAccount |
| `frontend/js/app.js` | Router + module loader |
| `frontend/js/api.js` | Wails binding wrapper |
| `frontend/js/modules/operations.js` | Daily Log + Vaccine Schedule + House History + Edit Log |
| `frontend/js/modules/inventory.js` | Egg/Supply/Feed/Items/Categories |
| `frontend/js/modules/purchasing.js` | Purchasing module UI |
| `frontend/js/modules/sales.js` | Sales module UI |
