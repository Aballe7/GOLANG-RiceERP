# Purchase-to-Payment (P2P) User-End Testing Guide

> **Scope**: Purchasing module only — PO → DR → AP Invoice → AP Payment
> **Prerequisite**: At least one active supplier exists; GL accounts configured for PURCHASING module (GRNI, INVENTORY_RECEIVED, AP_PAYABLE, AP_CLEARING, Cash/Bank)

---

## Status Reference

| Document | States |
|----------|--------|
| Purchase Header | `Open` → `Closed` (auto, all lines received) / `Cancelled` |
| Purchase Line | `open_qty` decrements at DR **confirmation** |
| Delivery Receipt | `Draft` → `Received` / `Cancelled` |
| DR Line | `open_qty` decrements at AP Invoice creation |
| AP Invoice | `Open` → `Partial` → `Paid` / `Cancelled` |
| AP Payment | active → `Cancelled` |

---

## Scenario 1 — Full Delivery, Full Payment (Happy Path)

**Story**: Supplier delivers exactly what was ordered. One invoice. One payment.

### Step 1 — Create Purchase Order
1. Go to **Purchasing → Purchase Orders → New PO**
2. Fill in:
   - Supplier: *(select any active supplier)*
   - Date: today
   - Add 2 lines:
     - Line 1: Feeds, qty **100**, price **₱50**
     - Line 2: Vitamins, qty **20**, price **₱200**
3. Click **Save**. Note the PO number (e.g. `PO-000-00001`).

**Expected**:
- PO created with status **Open**
- `purchase_line.open_qty` = 100 and 20 respectively
- `purchase.payment_status` = `Undelivered`

---

### Step 2 — Create Delivery Receipt
1. Go to **Purchasing → Delivery Receipts → New DR**
2. Check the checkbox next to `PO-000-00001` and click **Generate Lines**
3. Verify both lines appear (qty received = ordered qty):
   - Feeds: **100**
   - Vitamins: **20**
4. Fill **Received By** and **Date**. Click **Save**.

**Expected**:
- DR created with status **Draft**
- DR line `uom_entry` inherits from PO line

---

### Step 3 — Confirm Delivery Receipt
1. Open the DR just created. Click **Confirm**.

**Expected**:
- DR status → **Received**
- `purchase_line.open_qty` for both lines → **0**
- `purchase_header.status` → **Closed** (all lines received)
- `purchase.payment_status` → **Delivered**
- `dr_line.open_qty` = 100 and 20 (set at confirmation, cleared at invoicing)
- Journal entry posted: Dr Inventory Received / Cr GRNI

---

### Step 4 — Create AP Invoice
1. Go to **Purchasing → AP Invoices → New AP Invoice**
2. Select the supplier and the DR (`DR-...`) as the base document
3. Lines should auto-populate from the DR. Verify:
   - Feeds: qty **100** × ₱50 = **₱5,000**
   - Vitamins: qty **20** × ₱200 = **₱4,000**
   - Total: **₱9,000**
4. Fill **Date**, **Due Date**, **Supplier Invoice Ref**. Click **Save**.

**Expected**:
- AP Invoice created with status **Open**, total **₱9,000**
- `dr_line.open_qty` for both lines → **0**
- `purchase.amount_invoiced` += 9,000
- Journal entry posted: Dr GRNI / Cr AP Payable

---

### Step 5 — Post Payment
1. Go to **Purchasing → AP Payments → New Payment**
2. Select the supplier. The open invoice should appear in the list.
3. Apply **₱9,000** to the invoice. Set Date and Payment Method.
4. Click **Post Payment**.

**Expected**:
- AP Payment created
- AP Invoice status → **Paid** (`amount_paid_stored` = 9,000)
- `purchase.amount_settled` += 9,000
- Journal entry posted: Dr AP Clearing / Cr Cash/Bank

---

### Step 6 — Verify (Optional)
- Open the PO detail → linked DRs and AP Invoices should show
- Open the DR detail → linked AP Invoice should show
- Open the AP Invoice detail → Payment section should show the payment with full amount

---

---

## Scenario 2 — Partial Delivery, Full Payment

**Story**: Supplier ships only part of the order. One partial DR is confirmed. A full AP Invoice is raised for what was received. Fully paid.

### Step 1 — Create PO
Same as Scenario 1 but use:
- Line 1: Egg Trays, qty **500**, price **₱12**
- Line 2: Disinfectant, qty **10**, price **₱350**
- PO total: **₱9,500**

---

### Step 2 — Partial Delivery Receipt
1. New DR → Generate Lines from this PO
2. **Edit received qty**:
   - Egg Trays: change to **200** (of 500 ordered)
   - Disinfectant: **10** (fully received)
3. Save → Confirm

**Expected**:
- DR status → **Received**
- `purchase_line.open_qty`:
  - Egg Trays: 500 − 200 = **300** (line still Open)
  - Disinfectant: 10 − 10 = **0** (line Closed)
- `purchase_header.status` → **Open** (Egg Trays line not fully received)
- `purchase.payment_status` → **Partial**

---

### Step 3 — AP Invoice for Partial Delivery
1. New AP Invoice → link to this DR
2. Lines show what was received:
   - Egg Trays: 200 × ₱12 = **₱2,400**
   - Disinfectant: 10 × ₱350 = **₱3,500**
   - Total: **₱5,900**
3. Save

**Expected**:
- AP Invoice total = **₱5,900**, status **Open**

---

### Step 4 — Full Payment of Partial Invoice
1. New Payment → apply **₱5,900** to the invoice
2. Post Payment

**Expected**:
- AP Invoice status → **Paid**
- PO status still **Open** (remaining 300 Egg Trays not yet delivered)

---

### Step 5 — Second Delivery for Remaining Qty
1. New DR → Generate Lines from the same PO
2. Lines will now show:
   - Egg Trays: suggested qty 300 (remaining open_qty)
   - Disinfectant: 0 (already fully received, line closed)
3. Receive remaining **300** Egg Trays → Save → Confirm

**Expected**:
- `purchase_line.open_qty` for Egg Trays → **0**
- `purchase_header.status` → **Closed**
- `purchase.payment_status` → **Delivered**

---

---

## Scenario 3 — Multiple POs, Single DR, 3 Payments for One AP Invoice

**Story**: Two POs from the same supplier are consolidated into one delivery. One AP Invoice is raised for the combined DR. The invoice is paid in 3 installments.

### Step 1 — Create 2 POs
- **PO-A**: Corn Feed, qty **50**, price **₱80** → total **₱4,000**
- **PO-B**: Soy Meal, qty **30**, price **₱100** → total **₱3,000**
- Combined expected invoice: **₱7,000**

---

### Step 2 — Multi-PO Delivery Receipt
1. New DR
2. Check **both** PO-A and PO-B checkboxes, click **Generate Lines**
3. Verify both PO lines appear (source PO column visible)
4. Set received qty = ordered qty for all lines
5. Save → Confirm

**Expected**:
- Single DR with 2 lines, each tagged with its source PO's `BaseDocEntry`
- `purchase_line.open_qty` → 0 for all lines on both POs
- Both PO headers → **Closed**

---

### Step 3 — AP Invoice from Combined DR
1. New AP Invoice → link to the DR
2. Lines: Corn Feed ₱4,000 + Soy Meal ₱3,000 = **₱7,000**
3. Set Due Date: 30 days from now. Save.

**Expected**:
- AP Invoice status **Open**, total **₱7,000**

---

### Step 4 — First Partial Payment (₱2,000)
1. New Payment → apply **₱2,000** to the invoice
2. Post Payment

**Expected**:
- AP Invoice `amount_paid_stored` = 2,000 → status **Partial**
- Payment 1 created

---

### Step 5 — Second Partial Payment (₱3,000)
1. New Payment → apply **₱3,000** to the same invoice
2. Post Payment

**Expected**:
- AP Invoice `amount_paid_stored` = 5,000 → status still **Partial**
- Payment 2 created

---

### Step 6 — Final Payment (₱2,000)
1. New Payment → apply **₱2,000** to the same invoice
2. Post Payment

**Expected**:
- AP Invoice `amount_paid_stored` = 7,000 → status **Paid**
- AP Invoice detail shows all 3 payment lines in the Payments section
- Total of 3 payments = **₱7,000**

---

---

## Scenario 4 — Cancel and Reverse (Error Correction Flow)

**Story**: DR was confirmed by mistake. Tester walks through the proper reversal chain.

### Step 1 — Create PO → DR → Confirm
- Create a PO: Bedding Material, qty **40**, price **₱60** = **₱2,400**
- Create and confirm the DR (full qty)

**Expected**: PO closed, DR = Received

---

### Step 2 — Attempt to Cancel DR with Linked AP Invoice
1. Create an AP Invoice from the DR (₱2,400)
2. Now try to **Cancel** the DR

**Expected**:
- Error toast: *"cannot cancel DR-...: 1 AP invoice(s) exist against it — cancel the invoice(s) first"*
- DR remains **Received**

---

### Step 3 — Cancel AP Invoice First
1. Open the AP Invoice. Click **Cancel**.
2. Confirm: no payments on this invoice.

**Expected**:
- AP Invoice status → **Cancelled**
- `purchase.amount_invoiced` restored
- `dr_line.open_qty` restored
- Reversal journal entry posted for AP Invoice JE

---

### Step 4 — Cancel DR
1. Now Cancel the DR.

**Expected**:
- DR status → **Cancelled**
- `purchase_line.open_qty` restored (Bedding Material → 40 again)
- `purchase_header.status` → **Open** (was auto-closed, now reopened)
- `purchase.payment_status` → **Undelivered**
- Reversal journal entry posted for GRNI JE

---

### Step 5 — Re-confirm and Re-invoice
- Create a new DR (corrected qty) → Confirm → Create new AP Invoice → Pay
- Verify the full flow completes cleanly

---

---

## Scenario 5 — Over-Payment Guard

**Story**: AP Invoice for ₱5,000. Tester tries to apply ₱6,000 in a single payment.

> **Note**: The current system does **not** block over-payment at the service layer — `amount_paid_stored` will exceed `doc_total` and status will be `Paid`. This is a known gap. **Verify current behavior** and note whether the UI should add a validation.

### Steps
1. Create PO → DR → Confirm → AP Invoice (₱5,000)
2. New Payment → apply **₱6,000** to the invoice
3. Post Payment

**What to observe**:
- Does the UI warn before posting?
- AP Invoice `amount_paid_stored` = 6,000 (over-paid by ₱1,000)
- Invoice status = **Paid**
- No error thrown — document this as a known gap if confirmed

---

---

## Scenario 6 — Cancel Payment Reversal

**Story**: A payment was posted to the wrong AP Invoice. Tester cancels the payment and re-applies it.

### Steps
1. Complete Scenario 1 steps 1–4 (PO → DR → AP Invoice, status Open, ₱9,000)
2. Post a payment of **₱9,000** → Invoice becomes **Paid**
3. Cancel the Payment

**Expected after cancel**:
- AP Invoice status → **Open** (back to 0 paid)
- `purchase.amount_settled` restored
- Reversal journal entry posted for AP Payment JE
- Payment record status → **Cancelled**

4. Create a new correct payment → Invoice becomes Paid again

---

---

## Quick Checklist

| # | Check | Pass? |
|---|-------|-------|
| 1 | PO created; `purchase_line.open_qty` = ordered qty | ☐ |
| 2 | DR confirmed; `purchase_line.open_qty` decrements | ☐ |
| 3 | PO auto-closes when all lines fully received | ☐ |
| 4 | DR open_qty set at confirmation, decrements at invoice | ☐ |
| 5 | AP Invoice total = sum of line qty × price | ☐ |
| 6 | AP Invoice status: Open → Partial → Paid | ☐ |
| 7 | AP Payment detail shows all applied invoice lines | ☐ |
| 8 | Multi-PO DR: each line tagged to its source PO | ☐ |
| 9 | Cancel DR blocked when AP Invoice exists | ☐ |
| 10 | Cancel AP Invoice restores DR line open_qty | ☐ |
| 11 | Cancel Payment restores invoice amount_paid_stored | ☐ |
| 12 | All journal entries post without GL account errors | ☐ |
