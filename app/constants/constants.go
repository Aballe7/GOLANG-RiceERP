package constants

// EGG_SIZES lists all egg size SKUs used throughout the system.
var EGGSizes = []string{
	"Pewee", "Pullet", "Small", "Medium", "Large",
	"Extra Large", "Jumbo", "Double Yolk", "Cracked/Dirty",
}

const TraySize = 30 // pieces per tray

var AllModules = []string{
	"Flocks", "Operations", "Inventory", "Purchasing", "Production", "Sales", "Reports",
}

// ModuleMeta holds display metadata for each module.
type ModuleMetaEntry struct {
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

var ModuleMeta = map[string]ModuleMetaEntry{
	"Flocks":     {Color: "#10b981", Icon: "bi-house-door-fill"},
	"Operations": {Color: "#f59e0b", Icon: "bi-clipboard2-pulse-fill"},
	"Inventory":  {Color: "#06b6d4", Icon: "bi-boxes"},
	"Purchasing": {Color: "#8b5cf6", Icon: "bi-cart-fill"},
	"Sales":      {Color: "#ec4899", Icon: "bi-receipt"},
	"Reports":    {Color: "#f97316", Icon: "bi-bar-chart-fill"},
}

// POSTING_EVENTS — all accounting posting events by module.
type PostingEventEntry struct {
	Event       string `json:"event"`
	Description string `json:"description"`
}

var PostingEvents = map[string][]PostingEventEntry{
	"SALES": {
		{Event: "AR_RECEIVABLE", Description: "AR Invoice — Accounts Receivable (Dr)"},
		{Event: "SALES_REVENUE", Description: "AR Invoice — Sales Revenue (Cr)"},
		{Event: "COGS", Description: "AR Invoice — Cost of Goods Sold (Dr)"},
		{Event: "INVENTORY_SOLD", Description: "AR Invoice — Inventory Reduction (Cr)"},
		{Event: "OUTPUT_VAT", Description: "AR Invoice — Output VAT Payable (Cr)"},
		{Event: "COLLECTION_CLEARING", Description: "Collection — AR Clearing / Receivable (Cr)"},
		{Event: "SALES_DISCOUNT", Description: "Collection — Sales Discount (Dr)"},
	},
	"PURCHASING": {
		{Event: "AP_PAYABLE", Description: "AP Invoice — Accounts Payable (Cr)"},
		{Event: "GRNI", Description: "DR Confirm — Goods Received Not Yet Invoiced (Cr)"},
		{Event: "INVENTORY_RECEIVED", Description: "DR Confirm — Inventory / Expense (Dr)"},
		{Event: "INPUT_VAT", Description: "AP Invoice — Input VAT Creditable (Dr)"},
		{Event: "AP_CLEARING", Description: "AP Payment — AP Clearing / Payable (Dr)"},
		{Event: "PURCHASE_PRICE_VARIANCE", Description: "AP Invoice — Price Variance (Dr/Cr)"},
	},
	"INVENTORY": {
		{Event: "FEED_CONSUMED", Description: "Daily Log — Feed Consumed Expense (Dr)"},
		{Event: "INVENTORY_FEED", Description: "Daily Log — Inventory Feeds Reduction (Cr)"},
		{Event: "MORTALITY_LOSS", Description: "Daily Log — Mortality Loss Expense (Dr)"},
		{Event: "INVENTORY_BIRDS", Description: "Daily Log — Inventory Birds Reduction (Cr)"},
		{Event: "STOCK_ADJUSTMENT", Description: "Manual Stock Adjustment"},
	},
	"BANKING": {
		{Event: "CASH_INFLOW", Description: "Collection — Cash / Bank Account (Dr)"},
		{Event: "CASH_OUTFLOW", Description: "AP Payment — Cash / Bank Account (Cr)"},
	},
	"PRODUCTION": {
		{Event: "MILLING_WIP",    Description: "Milling — Work In Progress (WIP) account"},
		{Event: "MILLING_INPUT",  Description: "Milling Start — Input Inventory (Paddy) Credit"},
		{Event: "MILLING_OUTPUT", Description: "Milling Complete — Output Inventory (Products) Debit"},
	},
}

// Body weight breed standards (grams per week of age)
var ISABrownStandard = map[int]float64{
	4: 280, 5: 370, 6: 460, 7: 530, 8: 610, 9: 690, 10: 790,
	11: 880, 12: 970, 13: 1050, 14: 1120, 15: 1200, 16: 1270,
	17: 1340, 18: 1400, 19: 1460, 20: 1520, 21: 1560, 22: 1590,
}

var LohmannStandard = map[int]float64{
	4: 300, 5: 390, 6: 480, 7: 560, 8: 640, 9: 730, 10: 820,
	11: 910, 12: 1000, 13: 1080, 14: 1150, 15: 1230, 16: 1300,
	17: 1370, 18: 1430, 19: 1490, 20: 1550, 21: 1590, 22: 1620,
}

// BreedStandard returns the body weight standard for a given breed name.
// Returns ISA Brown standard by default.
func BreedStandard(breed string) map[int]float64 {
	switch breed {
	case "Lohmann Brown", "Lohmann":
		return LohmannStandard
	default:
		return ISABrownStandard
	}
}

// LifeStageLabels — layer house stages
const LayStartWeek = 21

// GetFlockStage returns the stage label and Bootstrap color class for a flock.
func GetFlockStage(houseType string, ageWeeks int) (string, string) {
	if houseType == "Grower" {
		switch {
		case ageWeeks < 8:
			return "Brooding", "info"
		case ageWeeks < 18:
			return "Growing", "warning"
		default:
			return "Pre-Lay", "primary"
		}
	}
	// Layer house
	switch {
	case ageWeeks < 25:
		return "Ramp-Up", "primary"
	case ageWeeks <= 60:
		return "Peak Production", "success"
	case ageWeeks <= 80:
		return "Late Lay", "warning"
	default:
		return "End of Cycle", "danger"
	}
}
