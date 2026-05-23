# CLAUDE.md — RiceMill_Go

## Project
Wails v2 desktop app for rice mill operations. Go backend + Vanilla JS frontend, MySQL database, GORM ORM.

## Architecture
```
main.go              → Wails entry point
app/app.go           → App struct; all Wails bindings live here as methods
app/models/          → GORM structs (SAP B1-aligned naming: OITM, OITB, etc.)
app/services/<mod>/  → Business logic (inventory, production, purchasing, sales, accounting)
app/handlers/        → Thin wrappers; call services, return handlers.Response{Success, Data, Error}
app/db/              → DB connection, AutoMigrateAll(), RunOITMSchemaMigrations(), SeedDefaults()
frontend/js/modules/ → Vanilla JS modules; call Go via window.go.<MethodName>(args)
```

## Key Patterns

**Adding a feature:**
1. Model field → `app/models/<domain>.go`
2. Business logic → `app/services/<domain>/service.go`
3. Handler → `app/handlers/<domain>.go` (return `handlers.Response`)
4. Wails binding → `app/app.go` method on `*App`
5. Frontend → `frontend/js/modules/<domain>.js` using `window.go.<MethodName>()`

**Handler response shape:**
```go
return handlers.Response{Success: true, Data: result}
return handlers.Response{Success: false, Error: err.Error()}
```

**DB migrations:**
- Schema renames/conversions: add to `db.RunOITMSchemaMigrations()` — runs **before** `AutoMigrateAll()`
- New tables/columns: GORM AutoMigrate handles it automatically
- GORM never drops columns; soft-delete via `valid_for = 'N'`

## Database Conventions (SAP B1-aligned)
| Concept | Table | Key fields |
|---|---|---|
| Item Master | `oitm` | `item_code`, `item_name`, `invntry_uom`, `avg_price`, `on_hand`, `valid_for` (Y/N), `mak_item` (Y/N), `itms_grp_cod` |
| Item Category | `oitb` | `itms_grp_cod` (PK int autoincrement), `itms_grp_nam` |
| Warehouse | `owhs` | `whs_code`, `whs_name` |
| Per-warehouse stock | `oitw` | composite PK: `item_code + whs_code` |
| Inventory ledger | `oivl` | audit trail for all stock movements |
| Item prices | `itm1` | composite PK: `item_code + price_list` |
| UoM per item | `itm9` | composite PK: `item_code + uom_entry` |

Bool fields use `char(1)` Y/N (SAP B1 convention), not Go `bool`.

## Commands
```bash
go build ./...                          # compile check
go run ./cmd/seed-master/main.go        # seed categories, UoMs, 50 items, customers, suppliers
go run ./cmd/seed-demo/main.go          # seed 2-month demo transactions (run seed-master first)
wails dev                               # run in dev mode
wails build                             # production build
```

## Config
`config.toml` at project root — DB connection (host, port, user, password, dbname).

## Type Aliases (backward compat)
```go
type ItemMaster = OITM
type ItemCategory = OITB
```
Use the canonical names (`OITM`, `OITB`) in new code.
