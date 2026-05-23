export namespace handlers {
	
	export class ChangePasswordRequest {
	    user_id: number;
	    new_password: string;
	
	    static createFrom(source: any = {}) {
	        return new ChangePasswordRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.new_password = source["new_password"];
	    }
	}
	export class ComputeWHTRequest {
	    supplier_id: number;
	    wht_category: string;
	    vat_exclusive_amount: number;
	    gross_amount: number;
	    invoice_date: string;
	
	    static createFrom(source: any = {}) {
	        return new ComputeWHTRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplier_id = source["supplier_id"];
	        this.wht_category = source["wht_category"];
	        this.vat_exclusive_amount = source["vat_exclusive_amount"];
	        this.gross_amount = source["gross_amount"];
	        this.invoice_date = source["invoice_date"];
	    }
	}
	export class CreateAPInvoiceRequest {
	    purchase_header_id: number;
	    dr_id?: number;
	    date: string;
	    terms: string;
	    supplier_name: string;
	    ref_number: string;
	    comments: string;
	    due_date?: string;
	    lines: models.APInvoiceItem[];
	    vat_exclusive_amount: number;
	    wht_rate: number;
	    wht_amount: number;
	    wht_atc_code: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateAPInvoiceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.purchase_header_id = source["purchase_header_id"];
	        this.dr_id = source["dr_id"];
	        this.date = source["date"];
	        this.terms = source["terms"];
	        this.supplier_name = source["supplier_name"];
	        this.ref_number = source["ref_number"];
	        this.comments = source["comments"];
	        this.due_date = source["due_date"];
	        this.lines = this.convertValues(source["lines"], models.APInvoiceItem);
	        this.vat_exclusive_amount = source["vat_exclusive_amount"];
	        this.wht_rate = source["wht_rate"];
	        this.wht_amount = source["wht_amount"];
	        this.wht_atc_code = source["wht_atc_code"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateAPPaymentRequest {
	    date: string;
	    supplier_id?: number;
	    supplier_name: string;
	    payment_method: string;
	    ref_num: string;
	    notes: string;
	    lines: purchasing.APPaymentLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateAPPaymentRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name = source["supplier_name"];
	        this.payment_method = source["payment_method"];
	        this.ref_num = source["ref_num"];
	        this.notes = source["notes"];
	        this.lines = this.convertValues(source["lines"], purchasing.APPaymentLineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateARInvoiceRequest {
	    delivery_order_ids: number[];
	    customer_id?: number;
	    customer_name: string;
	    customer_addr: string;
	    customer_contact: string;
	    date: string;
	    terms: string;
	    notes: string;
	    due_date?: string;
	    items: models.ARInvoiceItem[];
	
	    static createFrom(source: any = {}) {
	        return new CreateARInvoiceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.delivery_order_ids = source["delivery_order_ids"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.customer_addr = source["customer_addr"];
	        this.customer_contact = source["customer_contact"];
	        this.date = source["date"];
	        this.terms = source["terms"];
	        this.notes = source["notes"];
	        this.due_date = source["due_date"];
	        this.items = this.convertValues(source["items"], models.ARInvoiceItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateCollectionRequest {
	    date: string;
	    customer_id?: number;
	    customer_name: string;
	    payment_method: string;
	    ref_num: string;
	    notes: string;
	    lines: sales.CollectionLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateCollectionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.payment_method = source["payment_method"];
	        this.ref_num = source["ref_num"];
	        this.notes = source["notes"];
	        this.lines = this.convertValues(source["lines"], sales.CollectionLineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateDeliveryOrderRequest {
	    sales_order_id: number;
	    date: string;
	    delivered_by: string;
	    notes: string;
	    items: models.DeliveryOrderItem[];
	
	    static createFrom(source: any = {}) {
	        return new CreateDeliveryOrderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sales_order_id = source["sales_order_id"];
	        this.date = source["date"];
	        this.delivered_by = source["delivered_by"];
	        this.notes = source["notes"];
	        this.items = this.convertValues(source["items"], models.DeliveryOrderItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DRLineInput {
	    purchase_header_id: number;
	    purchase_line_id: number;
	    description: string;
	    category: string;
	    unit: string;
	    uom_entry: number;
	    quantity_ordered: number;
	    quantity: number;
	    price: number;
	
	    static createFrom(source: any = {}) {
	        return new DRLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.purchase_header_id = source["purchase_header_id"];
	        this.purchase_line_id = source["purchase_line_id"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.unit = source["unit"];
	        this.uom_entry = source["uom_entry"];
	        this.quantity_ordered = source["quantity_ordered"];
	        this.quantity = source["quantity"];
	        this.price = source["price"];
	    }
	}
	export class CreateDeliveryReceiptRequest {
	    date: string;
	    received_by: string;
	    ref_number: string;
	    comments: string;
	    lines: DRLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateDeliveryReceiptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.received_by = source["received_by"];
	        this.ref_number = source["ref_number"];
	        this.comments = source["comments"];
	        this.lines = this.convertValues(source["lines"], DRLineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateFixedAssetRequest {
	    asset_code: string;
	    asset_name: string;
	    category: string;
	    description: string;
	    acquisition_date: string;
	    acquisition_cost: number;
	    residual_value: number;
	    useful_life_months: number;
	    depreciation_method: string;
	    gl_asset_account_id: number;
	    gl_accum_dep_account_id: number;
	    gl_dep_exp_account_id: number;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateFixedAssetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asset_code = source["asset_code"];
	        this.asset_name = source["asset_name"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.acquisition_date = source["acquisition_date"];
	        this.acquisition_cost = source["acquisition_cost"];
	        this.residual_value = source["residual_value"];
	        this.useful_life_months = source["useful_life_months"];
	        this.depreciation_method = source["depreciation_method"];
	        this.gl_asset_account_id = source["gl_asset_account_id"];
	        this.gl_accum_dep_account_id = source["gl_accum_dep_account_id"];
	        this.gl_dep_exp_account_id = source["gl_dep_exp_account_id"];
	        this.notes = source["notes"];
	    }
	}
	export class CreateGLAccountRequest {
	    code: string;
	    name: string;
	    section: string;
	    account_type: string;
	    normal_balance: string;
	    parent_id?: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateGLAccountRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.section = source["section"];
	        this.account_type = source["account_type"];
	        this.normal_balance = source["normal_balance"];
	        this.parent_id = source["parent_id"];
	        this.description = source["description"];
	    }
	}
	export class CreateSalesOrderRequest {
	    date: string;
	    customer_id?: number;
	    customer_name: string;
	    customer_addr: string;
	    customer_contact: string;
	    payment_method: string;
	    terms: string;
	    due_date?: string;
	    notes: string;
	    items: models.SalesOrderItem[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSalesOrderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.customer_addr = source["customer_addr"];
	        this.customer_contact = source["customer_contact"];
	        this.payment_method = source["payment_method"];
	        this.terms = source["terms"];
	        this.due_date = source["due_date"];
	        this.notes = source["notes"];
	        this.items = this.convertValues(source["items"], models.SalesOrderItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateUserRequest {
	    username: string;
	    full_name: string;
	    role: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.full_name = source["full_name"];
	        this.role = source["role"];
	        this.password = source["password"];
	    }
	}
	
	export class DisposeAssetRequest {
	    asset_id: number;
	    disposal_date: string;
	    proceeds: number;
	    proceeds_account_id: number;
	    gain_loss_account_id: number;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new DisposeAssetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asset_id = source["asset_id"];
	        this.disposal_date = source["disposal_date"];
	        this.proceeds = source["proceeds"];
	        this.proceeds_account_id = source["proceeds_account_id"];
	        this.gain_loss_account_id = source["gain_loss_account_id"];
	        this.notes = source["notes"];
	    }
	}
	export class LoginRequest {
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class ManualJELine {
	    gl_account_id: number;
	    debit: number;
	    credit: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ManualJELine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gl_account_id = source["gl_account_id"];
	        this.debit = source["debit"];
	        this.credit = source["credit"];
	        this.description = source["description"];
	    }
	}
	export class ManualJERequest {
	    date: string;
	    narration: string;
	    lines: ManualJELine[];
	
	    static createFrom(source: any = {}) {
	        return new ManualJERequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.narration = source["narration"];
	        this.lines = this.convertValues(source["lines"], ManualJELine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response {
	    ok: boolean;
	    message: string;
	    data: any;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.data = source["data"];
	    }
	}
	export class RunDepreciationRequest {
	    period_date: string;
	
	    static createFrom(source: any = {}) {
	        return new RunDepreciationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.period_date = source["period_date"];
	    }
	}
	export class SetPermissionsRequest {
	    user_id: number;
	    modules: string[];
	
	    static createFrom(source: any = {}) {
	        return new SetPermissionsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.modules = source["modules"];
	    }
	}
	export class UpdateSalesOrderRequest {
	    version: number;
	    date: string;
	    customer_id?: number;
	    customer_name: string;
	    customer_addr: string;
	    customer_contact: string;
	    payment_method: string;
	    terms: string;
	    due_date?: string;
	    notes: string;
	    items: models.SalesOrderItem[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateSalesOrderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.date = source["date"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.customer_addr = source["customer_addr"];
	        this.customer_contact = source["customer_contact"];
	        this.payment_method = source["payment_method"];
	        this.terms = source["terms"];
	        this.due_date = source["due_date"];
	        this.notes = source["notes"];
	        this.items = this.convertValues(source["items"], models.SalesOrderItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateUserRequest {
	    id: number;
	    full_name: string;
	    role: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.full_name = source["full_name"];
	        this.role = source["role"];
	        this.is_active = source["is_active"];
	    }
	}
	export class UpsertAccountDeterminationRequest {
	    module: string;
	    posting_event: string;
	    item_category?: string;
	    gl_account_id: number;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new UpsertAccountDeterminationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.module = source["module"];
	        this.posting_event = source["posting_event"];
	        this.item_category = source["item_category"];
	        this.gl_account_id = source["gl_account_id"];
	        this.notes = source["notes"];
	    }
	}
	export class UpsertItemPriceRequest {
	    item_code: string;
	    price_list: number;
	    price: number;
	    currency: string;
	
	    static createFrom(source: any = {}) {
	        return new UpsertItemPriceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_code = source["item_code"];
	        this.price_list = source["price_list"];
	        this.price = source["price"];
	        this.currency = source["currency"];
	    }
	}
	export class UpsertItemUoMPriceRequest {
	    item_code: string;
	    uom_entry: number;
	    price: number;
	    factor: number;
	
	    static createFrom(source: any = {}) {
	        return new UpsertItemUoMPriceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_code = source["item_code"];
	        this.uom_entry = source["uom_entry"];
	        this.price = source["price"];
	        this.factor = source["factor"];
	    }
	}
	export class UpsertPaymentMethodAccountRequest {
	    payment_method: string;
	    bank_name: string;
	    gl_account_id: number;
	    direction: string;
	
	    static createFrom(source: any = {}) {
	        return new UpsertPaymentMethodAccountRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.payment_method = source["payment_method"];
	        this.bank_name = source["bank_name"];
	        this.gl_account_id = source["gl_account_id"];
	        this.direction = source["direction"];
	    }
	}
	export class UpsertPriceGroupItemRequest {
	    price_group_id: number;
	    egg_size: string;
	    unit: string;
	    price: number;
	
	    static createFrom(source: any = {}) {
	        return new UpsertPriceGroupItemRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.price_group_id = source["price_group_id"];
	        this.egg_size = source["egg_size"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	    }
	}

}

export namespace inventory {
	
	export class GILineInput {
	    item_id: number;
	    uom_entry: number;
	    quantity: number;
	    price: number;
	    warehouse_code: string;
	    account_code: string;
	    project: string;
	
	    static createFrom(source: any = {}) {
	        return new GILineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_id = source["item_id"];
	        this.uom_entry = source["uom_entry"];
	        this.quantity = source["quantity"];
	        this.price = source["price"];
	        this.warehouse_code = source["warehouse_code"];
	        this.account_code = source["account_code"];
	        this.project = source["project"];
	    }
	}
	export class CreateGoodsIssueRequest {
	    posting_date: string;
	    doc_due_date: string;
	    remarks: string;
	    lines: GILineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateGoodsIssueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.posting_date = source["posting_date"];
	        this.doc_due_date = source["doc_due_date"];
	        this.remarks = source["remarks"];
	        this.lines = this.convertValues(source["lines"], GILineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GRLineInput {
	    item_id: number;
	    uom_entry: number;
	    quantity: number;
	    price: number;
	    warehouse_code: string;
	    account_code: string;
	    project: string;
	
	    static createFrom(source: any = {}) {
	        return new GRLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_id = source["item_id"];
	        this.uom_entry = source["uom_entry"];
	        this.quantity = source["quantity"];
	        this.price = source["price"];
	        this.warehouse_code = source["warehouse_code"];
	        this.account_code = source["account_code"];
	        this.project = source["project"];
	    }
	}
	export class CreateGoodsReceiptRequest {
	    posting_date: string;
	    doc_due_date: string;
	    remarks: string;
	    lines: GRLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateGoodsReceiptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.posting_date = source["posting_date"];
	        this.doc_due_date = source["doc_due_date"];
	        this.remarks = source["remarks"];
	        this.lines = this.convertValues(source["lines"], GRLineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UoMGroupLineInput {
	    uom_entry: number;
	    alt_qty: number;
	    base_qty: number;
	
	    static createFrom(source: any = {}) {
	        return new UoMGroupLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uom_entry = source["uom_entry"];
	        this.alt_qty = source["alt_qty"];
	        this.base_qty = source["base_qty"];
	    }
	}
	export class CreateUoMGroupRequest {
	    ugp_code: string;
	    ugp_name: string;
	    base_uom: number;
	    lines: UoMGroupLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateUoMGroupRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ugp_code = source["ugp_code"];
	        this.ugp_name = source["ugp_name"];
	        this.base_uom = source["base_uom"];
	        this.lines = this.convertValues(source["lines"], UoMGroupLineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace models {
	
	export class APInvoiceItem {
	    id: number;
	    ap_invoice_id: number;
	    line_num: number;
	    item_code: string;
	    description: string;
	    unit: string;
	    category: string;
	    quantity: number;
	    open_qty: number;
	    uom_entry: number;
	    price: number;
	    line_total: number;
	    warehouse_code: string;
	    account_code: string;
	    tax_code: string;
	    project_code: string;
	    cost_center: string;
	    line_status: string;
	    base_doc_entry?: number;
	    base_line_num?: number;
	    dr_line_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new APInvoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ap_invoice_id = source["ap_invoice_id"];
	        this.line_num = source["line_num"];
	        this.item_code = source["item_code"];
	        this.description = source["description"];
	        this.unit = source["unit"];
	        this.category = source["category"];
	        this.quantity = source["quantity"];
	        this.open_qty = source["open_qty"];
	        this.uom_entry = source["uom_entry"];
	        this.price = source["price"];
	        this.line_total = source["line_total"];
	        this.warehouse_code = source["warehouse_code"];
	        this.account_code = source["account_code"];
	        this.tax_code = source["tax_code"];
	        this.project_code = source["project_code"];
	        this.cost_center = source["cost_center"];
	        this.line_status = source["line_status"];
	        this.base_doc_entry = source["base_doc_entry"];
	        this.base_line_num = source["base_line_num"];
	        this.dr_line_id = source["dr_line_id"];
	    }
	}
	export class Collection {
	    id: number;
	    collection_number: string;
	    // Go type: time
	    date: any;
	    customer_id?: number;
	    customer_name_snapshot: string;
	    total_amount: number;
	    payment_method: string;
	    reference_number: string;
	    notes: string;
	    status: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    updated_by_id?: number;
	    customer?: Customer;
	    lines?: CollectionLine[];
	
	    static createFrom(source: any = {}) {
	        return new Collection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.collection_number = source["collection_number"];
	        this.date = this.convertValues(source["date"], null);
	        this.customer_id = source["customer_id"];
	        this.customer_name_snapshot = source["customer_name_snapshot"];
	        this.total_amount = source["total_amount"];
	        this.payment_method = source["payment_method"];
	        this.reference_number = source["reference_number"];
	        this.notes = source["notes"];
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.updated_by_id = source["updated_by_id"];
	        this.customer = this.convertValues(source["customer"], Customer);
	        this.lines = this.convertValues(source["lines"], CollectionLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CollectionLine {
	    id: number;
	    collection_id: number;
	    ar_invoice_id: number;
	    amount_applied: number;
	    collection?: Collection;
	    ar_invoice?: ARInvoice;
	
	    static createFrom(source: any = {}) {
	        return new CollectionLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.collection_id = source["collection_id"];
	        this.ar_invoice_id = source["ar_invoice_id"];
	        this.amount_applied = source["amount_applied"];
	        this.collection = this.convertValues(source["collection"], Collection);
	        this.ar_invoice = this.convertValues(source["ar_invoice"], ARInvoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ARInvoiceItem {
	    id: number;
	    ar_invoice_id: number;
	    delivery_order_id?: number;
	    sku: string;
	    unit: string;
	    uom_entry: number;
	    quantity: number;
	    price_per_unit: number;
	
	    static createFrom(source: any = {}) {
	        return new ARInvoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ar_invoice_id = source["ar_invoice_id"];
	        this.delivery_order_id = source["delivery_order_id"];
	        this.sku = source["sku"];
	        this.unit = source["unit"];
	        this.uom_entry = source["uom_entry"];
	        this.quantity = source["quantity"];
	        this.price_per_unit = source["price_per_unit"];
	    }
	}
	export class DeliveryOrderItem {
	    id: number;
	    delivery_order_id: number;
	    sales_order_item_id?: number;
	    sku: string;
	    unit: string;
	    uom_entry: number;
	    quantity_ordered: number;
	    quantity_delivered: number;
	    price_per_unit: number;
	
	    static createFrom(source: any = {}) {
	        return new DeliveryOrderItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.delivery_order_id = source["delivery_order_id"];
	        this.sales_order_item_id = source["sales_order_item_id"];
	        this.sku = source["sku"];
	        this.unit = source["unit"];
	        this.uom_entry = source["uom_entry"];
	        this.quantity_ordered = source["quantity_ordered"];
	        this.quantity_delivered = source["quantity_delivered"];
	        this.price_per_unit = source["price_per_unit"];
	    }
	}
	export class DeliveryOrder {
	    id: number;
	    delivery_number: string;
	    // Go type: time
	    date: any;
	    sales_order_id: number;
	    status: string;
	    delivered_by: string;
	    delivered_by_id?: number;
	    notes: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    updated_by_id?: number;
	    amount_invoiced: number;
	    sales_order?: SalesOrder;
	    items?: DeliveryOrderItem[];
	    ar_invoices?: ARInvoice[];
	
	    static createFrom(source: any = {}) {
	        return new DeliveryOrder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.delivery_number = source["delivery_number"];
	        this.date = this.convertValues(source["date"], null);
	        this.sales_order_id = source["sales_order_id"];
	        this.status = source["status"];
	        this.delivered_by = source["delivered_by"];
	        this.delivered_by_id = source["delivered_by_id"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.updated_by_id = source["updated_by_id"];
	        this.amount_invoiced = source["amount_invoiced"];
	        this.sales_order = this.convertValues(source["sales_order"], SalesOrder);
	        this.items = this.convertValues(source["items"], DeliveryOrderItem);
	        this.ar_invoices = this.convertValues(source["ar_invoices"], ARInvoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SalesOrderItem {
	    id: number;
	    order_id: number;
	    sku: string;
	    unit: string;
	    uom_entry: number;
	    quantity: number;
	    open_qty: number;
	    price_per_unit: number;
	    line_total: number;
	
	    static createFrom(source: any = {}) {
	        return new SalesOrderItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.order_id = source["order_id"];
	        this.sku = source["sku"];
	        this.unit = source["unit"];
	        this.uom_entry = source["uom_entry"];
	        this.quantity = source["quantity"];
	        this.open_qty = source["open_qty"];
	        this.price_per_unit = source["price_per_unit"];
	        this.line_total = source["line_total"];
	    }
	}
	export class PriceGroupItem {
	    id: number;
	    price_group_id: number;
	    egg_size: string;
	    unit: string;
	    price: number;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new PriceGroupItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.price_group_id = source["price_group_id"];
	        this.egg_size = source["egg_size"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PriceGroup {
	    id: number;
	    name: string;
	    description: string;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    items?: PriceGroupItem[];
	
	    static createFrom(source: any = {}) {
	        return new PriceGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.items = this.convertValues(source["items"], PriceGroupItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Customer {
	    id: number;
	    name: string;
	    address: string;
	    delivery_address: string;
	    contact_number: string;
	    email: string;
	    customer_type: string;
	    notes: string;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    price_group_id?: number;
	    price_group?: PriceGroup;
	
	    static createFrom(source: any = {}) {
	        return new Customer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.delivery_address = source["delivery_address"];
	        this.contact_number = source["contact_number"];
	        this.email = source["email"];
	        this.customer_type = source["customer_type"];
	        this.notes = source["notes"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.price_group_id = source["price_group_id"];
	        this.price_group = this.convertValues(source["price_group"], PriceGroup);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SalesOrder {
	    id: number;
	    sales_order_number: string;
	    doc_status: string;
	    // Go type: time
	    date: any;
	    customer_id?: number;
	    customer_name_snapshot: string;
	    customer_address_snapshot: string;
	    customer_contact_snapshot: string;
	    payment_method: string;
	    payment_status: string;
	    amount_paid: number;
	    // Go type: time
	    due_date?: any;
	    terms: string;
	    grand_total: number;
	    amount_delivered: number;
	    amount_invoiced: number;
	    amount_collected: number;
	    notes: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    updated_by_id?: number;
	    version: number;
	    customer?: Customer;
	    items?: SalesOrderItem[];
	    deliveries?: DeliveryOrder[];
	    ar_invoices?: ARInvoice[];
	
	    static createFrom(source: any = {}) {
	        return new SalesOrder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sales_order_number = source["sales_order_number"];
	        this.doc_status = source["doc_status"];
	        this.date = this.convertValues(source["date"], null);
	        this.customer_id = source["customer_id"];
	        this.customer_name_snapshot = source["customer_name_snapshot"];
	        this.customer_address_snapshot = source["customer_address_snapshot"];
	        this.customer_contact_snapshot = source["customer_contact_snapshot"];
	        this.payment_method = source["payment_method"];
	        this.payment_status = source["payment_status"];
	        this.amount_paid = source["amount_paid"];
	        this.due_date = this.convertValues(source["due_date"], null);
	        this.terms = source["terms"];
	        this.grand_total = source["grand_total"];
	        this.amount_delivered = source["amount_delivered"];
	        this.amount_invoiced = source["amount_invoiced"];
	        this.amount_collected = source["amount_collected"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.updated_by_id = source["updated_by_id"];
	        this.version = source["version"];
	        this.customer = this.convertValues(source["customer"], Customer);
	        this.items = this.convertValues(source["items"], SalesOrderItem);
	        this.deliveries = this.convertValues(source["deliveries"], DeliveryOrder);
	        this.ar_invoices = this.convertValues(source["ar_invoices"], ARInvoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ARInvoice {
	    id: number;
	    invoice_number: string;
	    // Go type: time
	    date: any;
	    // Go type: time
	    due_date?: any;
	    terms: string;
	    sales_order_id?: number;
	    customer_id?: number;
	    customer_name_snapshot: string;
	    customer_address_snapshot: string;
	    customer_contact_snapshot: string;
	    total_amount: number;
	    status: string;
	    notes: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    updated_by_id?: number;
	    amount_collected: number;
	    version: number;
	    sales_order?: SalesOrder;
	    delivery_orders?: DeliveryOrder[];
	    customer?: Customer;
	    items?: ARInvoiceItem[];
	    collection_lines?: CollectionLine[];
	
	    static createFrom(source: any = {}) {
	        return new ARInvoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.invoice_number = source["invoice_number"];
	        this.date = this.convertValues(source["date"], null);
	        this.due_date = this.convertValues(source["due_date"], null);
	        this.terms = source["terms"];
	        this.sales_order_id = source["sales_order_id"];
	        this.customer_id = source["customer_id"];
	        this.customer_name_snapshot = source["customer_name_snapshot"];
	        this.customer_address_snapshot = source["customer_address_snapshot"];
	        this.customer_contact_snapshot = source["customer_contact_snapshot"];
	        this.total_amount = source["total_amount"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.updated_by_id = source["updated_by_id"];
	        this.amount_collected = source["amount_collected"];
	        this.version = source["version"];
	        this.sales_order = this.convertValues(source["sales_order"], SalesOrder);
	        this.delivery_orders = this.convertValues(source["delivery_orders"], DeliveryOrder);
	        this.customer = this.convertValues(source["customer"], Customer);
	        this.items = this.convertValues(source["items"], ARInvoiceItem);
	        this.collection_lines = this.convertValues(source["collection_lines"], CollectionLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	export class UoMMaster {
	    uom_entry: number;
	    uom_code: string;
	    uom_name: string;
	    length: number;
	    l_type: number;
	    width: number;
	    w_type: number;
	    height: number;
	    h_type: number;
	    volume: number;
	    v_type: number;
	    weight: number;
	    wgt_type: number;
	    user_sign: number;
	
	    static createFrom(source: any = {}) {
	        return new UoMMaster(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uom_entry = source["uom_entry"];
	        this.uom_code = source["uom_code"];
	        this.uom_name = source["uom_name"];
	        this.length = source["length"];
	        this.l_type = source["l_type"];
	        this.width = source["width"];
	        this.w_type = source["w_type"];
	        this.height = source["height"];
	        this.h_type = source["h_type"];
	        this.volume = source["volume"];
	        this.v_type = source["v_type"];
	        this.weight = source["weight"];
	        this.wgt_type = source["wgt_type"];
	        this.user_sign = source["user_sign"];
	    }
	}
	export class ItemUoMPrice {
	    item_code: string;
	    uom_entry: number;
	    price: number;
	    factor: number;
	    unit?: UoMMaster;
	
	    static createFrom(source: any = {}) {
	        return new ItemUoMPrice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_code = source["item_code"];
	        this.uom_entry = source["uom_entry"];
	        this.price = source["price"];
	        this.factor = source["factor"];
	        this.unit = this.convertValues(source["unit"], UoMMaster);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OITB {
	    itms_grp_cod: number;
	    itms_grp_nam: string;
	    description: string;
	    costing_meth: string;
	    is_active: boolean;
	    for_sales: boolean;
	    for_purchasing: boolean;
	    for_inventory: boolean;
	    for_production: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OITB(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itms_grp_cod = source["itms_grp_cod"];
	        this.itms_grp_nam = source["itms_grp_nam"];
	        this.description = source["description"];
	        this.costing_meth = source["costing_meth"];
	        this.is_active = source["is_active"];
	        this.for_sales = source["for_sales"];
	        this.for_purchasing = source["for_purchasing"];
	        this.for_inventory = source["for_inventory"];
	        this.for_production = source["for_production"];
	    }
	}
	export class OITM {
	    id: number;
	    item_code: string;
	    item_name: string;
	    frgn_name: string;
	    itms_grp_cod: number;
	    code_bars: string;
	    invnt_item: string;
	    sell_item: string;
	    prchse_item: string;
	    mak_item: string;
	    invntry_uom: string;
	    purchase_unit: string;
	    sales_unit: string;
	    i_uom_entry: number;
	    s_uom_entry: number;
	    p_uom_entry: number;
	    ugp_entry: number;
	    num_in_buy: number;
	    num_in_sale: number;
	    on_hand: number;
	    is_commited: number;
	    on_order: number;
	    min_level: number;
	    max_level: number;
	    lead_time: number;
	    avg_price: number;
	    lst_evl_pric: number;
	    last_pur_prc: number;
	    eval_system: string;
	    pricing_cod: string;
	    dflt_wh: string;
	    card_code: string;
	    supp_cat_num: string;
	    vat_gourp_sa: string;
	    wt_liable: string;
	    man_btch_num: string;
	    man_ser_num: string;
	    valid_for: string;
	    description: string;
	    // Go type: time
	    create_date?: any;
	    // Go type: time
	    update_date?: any;
	    uom_prices?: ItemUoMPrice[];
	    item_group?: OITB;
	    category_name?: string;
	    base_uom?: UoMMaster;
	    sales_uom?: UoMMaster;
	    purch_uom?: UoMMaster;
	
	    static createFrom(source: any = {}) {
	        return new OITM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.item_code = source["item_code"];
	        this.item_name = source["item_name"];
	        this.frgn_name = source["frgn_name"];
	        this.itms_grp_cod = source["itms_grp_cod"];
	        this.code_bars = source["code_bars"];
	        this.invnt_item = source["invnt_item"];
	        this.sell_item = source["sell_item"];
	        this.prchse_item = source["prchse_item"];
	        this.mak_item = source["mak_item"];
	        this.invntry_uom = source["invntry_uom"];
	        this.purchase_unit = source["purchase_unit"];
	        this.sales_unit = source["sales_unit"];
	        this.i_uom_entry = source["i_uom_entry"];
	        this.s_uom_entry = source["s_uom_entry"];
	        this.p_uom_entry = source["p_uom_entry"];
	        this.ugp_entry = source["ugp_entry"];
	        this.num_in_buy = source["num_in_buy"];
	        this.num_in_sale = source["num_in_sale"];
	        this.on_hand = source["on_hand"];
	        this.is_commited = source["is_commited"];
	        this.on_order = source["on_order"];
	        this.min_level = source["min_level"];
	        this.max_level = source["max_level"];
	        this.lead_time = source["lead_time"];
	        this.avg_price = source["avg_price"];
	        this.lst_evl_pric = source["lst_evl_pric"];
	        this.last_pur_prc = source["last_pur_prc"];
	        this.eval_system = source["eval_system"];
	        this.pricing_cod = source["pricing_cod"];
	        this.dflt_wh = source["dflt_wh"];
	        this.card_code = source["card_code"];
	        this.supp_cat_num = source["supp_cat_num"];
	        this.vat_gourp_sa = source["vat_gourp_sa"];
	        this.wt_liable = source["wt_liable"];
	        this.man_btch_num = source["man_btch_num"];
	        this.man_ser_num = source["man_ser_num"];
	        this.valid_for = source["valid_for"];
	        this.description = source["description"];
	        this.create_date = this.convertValues(source["create_date"], null);
	        this.update_date = this.convertValues(source["update_date"], null);
	        this.uom_prices = this.convertValues(source["uom_prices"], ItemUoMPrice);
	        this.item_group = this.convertValues(source["item_group"], OITB);
	        this.category_name = source["category_name"];
	        this.base_uom = this.convertValues(source["base_uom"], UoMMaster);
	        this.sales_uom = this.convertValues(source["sales_uom"], UoMMaster);
	        this.purch_uom = this.convertValues(source["purch_uom"], UoMMaster);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OSPP {
	    card_code: string;
	    item_code: string;
	    price: number;
	    currency: string;
	    discount: number;
	    price_list: number;
	    auto_update: string;
	
	    static createFrom(source: any = {}) {
	        return new OSPP(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.card_code = source["card_code"];
	        this.item_code = source["item_code"];
	        this.price = source["price"];
	        this.currency = source["currency"];
	        this.discount = source["discount"];
	        this.price_list = source["price_list"];
	        this.auto_update = source["auto_update"];
	    }
	}
	export class OWHS {
	    whs_code: string;
	    whs_name: string;
	    location: string;
	    street: string;
	    zip_code: string;
	    city: string;
	    state: string;
	    phone: string;
	    inactive: string;
	
	    static createFrom(source: any = {}) {
	        return new OWHS(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.whs_code = source["whs_code"];
	        this.whs_name = source["whs_name"];
	        this.location = source["location"];
	        this.street = source["street"];
	        this.zip_code = source["zip_code"];
	        this.city = source["city"];
	        this.state = source["state"];
	        this.phone = source["phone"];
	        this.inactive = source["inactive"];
	    }
	}
	
	
	export class ProductTreeLine {
	    id: number;
	    code: string;
	    item_code: string;
	    item_name: string;
	    quantity: number;
	    warehouse: string;
	    issue_method: string;
	    uom_code: string;
	    uom_entry: number;
	    price: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductTreeLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.code = source["code"];
	        this.item_code = source["item_code"];
	        this.item_name = source["item_name"];
	        this.quantity = source["quantity"];
	        this.warehouse = source["warehouse"];
	        this.issue_method = source["issue_method"];
	        this.uom_code = source["uom_code"];
	        this.uom_entry = source["uom_entry"];
	        this.price = source["price"];
	    }
	}
	export class ProductTree {
	    code: string;
	    tree_type: string;
	    warehouse: string;
	    quantity: number;
	    notes: string;
	    // Go type: time
	    updated_at: any;
	    lines?: ProductTreeLine[];
	
	    static createFrom(source: any = {}) {
	        return new ProductTree(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.tree_type = source["tree_type"];
	        this.warehouse = source["warehouse"];
	        this.quantity = source["quantity"];
	        this.notes = source["notes"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.lines = this.convertValues(source["lines"], ProductTreeLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class Supplier {
	    id: number;
	    name: string;
	    supplier_type: string;
	    contact_person: string;
	    contact_number: string;
	    email: string;
	    address: string;
	    tin_number: string;
	    payment_terms: string;
	    bank_details: string;
	    categories: string;
	    notes: string;
	    is_active: boolean;
	    status: string;
	    approved_by: string;
	    approved_by_id?: number;
	    // Go type: time
	    approved_at?: any;
	    // Go type: time
	    created_at: any;
	    wht_category: string;
	    is_vat_registered: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Supplier(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.supplier_type = source["supplier_type"];
	        this.contact_person = source["contact_person"];
	        this.contact_number = source["contact_number"];
	        this.email = source["email"];
	        this.address = source["address"];
	        this.tin_number = source["tin_number"];
	        this.payment_terms = source["payment_terms"];
	        this.bank_details = source["bank_details"];
	        this.categories = source["categories"];
	        this.notes = source["notes"];
	        this.is_active = source["is_active"];
	        this.status = source["status"];
	        this.approved_by = source["approved_by"];
	        this.approved_by_id = source["approved_by_id"];
	        this.approved_at = this.convertValues(source["approved_at"], null);
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.wht_category = source["wht_category"];
	        this.is_vat_registered = source["is_vat_registered"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace production {
	
	export class CompleteMillingLineInput {
	    output_item_id: number;
	    actual_qty: number;
	    expected_qty: number;
	    uom_entry: number;
	
	    static createFrom(source: any = {}) {
	        return new CompleteMillingLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output_item_id = source["output_item_id"];
	        this.actual_qty = source["actual_qty"];
	        this.expected_qty = source["expected_qty"];
	        this.uom_entry = source["uom_entry"];
	    }
	}
	export class CompleteMillingRequest {
	    lines: CompleteMillingLineInput[];
	    completion_note: string;
	    completion_date: string;
	
	    static createFrom(source: any = {}) {
	        return new CompleteMillingRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = this.convertValues(source["lines"], CompleteMillingLineInput);
	        this.completion_note = source["completion_note"];
	        this.completion_date = source["completion_date"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateMillingOrderRequest {
	    posting_date: string;
	    expected_date: string;
	    remarks: string;
	    input_item_id: number;
	    input_qty: number;
	    input_uom_entry: number;
	    warehouse_code: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateMillingOrderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.posting_date = source["posting_date"];
	        this.expected_date = source["expected_date"];
	        this.remarks = source["remarks"];
	        this.input_item_id = source["input_item_id"];
	        this.input_qty = source["input_qty"];
	        this.input_uom_entry = source["input_uom_entry"];
	        this.warehouse_code = source["warehouse_code"];
	    }
	}
	export class CreateWORequest {
	    item_code: string;
	    planned_qty: number;
	    start_date: string;
	    due_date: string;
	    warehouse: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateWORequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_code = source["item_code"];
	        this.planned_qty = source["planned_qty"];
	        this.start_date = source["start_date"];
	        this.due_date = source["due_date"];
	        this.warehouse = source["warehouse"];
	        this.notes = source["notes"];
	    }
	}

}

export namespace purchasing {
	
	export class APPaymentLineInput {
	    ap_invoice_id: number;
	    amount_applied: number;
	
	    static createFrom(source: any = {}) {
	        return new APPaymentLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ap_invoice_id = source["ap_invoice_id"];
	        this.amount_applied = source["amount_applied"];
	    }
	}
	export class POLineInput {
	    item_code: string;
	    category: string;
	    item_name: string;
	    unit: string;
	    i_uom_entry: number;
	    quantity: number;
	    unit_price: number;
	
	    static createFrom(source: any = {}) {
	        return new POLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_code = source["item_code"];
	        this.category = source["category"];
	        this.item_name = source["item_name"];
	        this.unit = source["unit"];
	        this.i_uom_entry = source["i_uom_entry"];
	        this.quantity = source["quantity"];
	        this.unit_price = source["unit_price"];
	    }
	}
	export class CreatePurchaseParams {
	    date: string;
	    supplier_id?: number;
	    supplier: string;
	    payment_method: string;
	    remarks: string;
	    po_number: string;
	    lines: POLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreatePurchaseParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.supplier_id = source["supplier_id"];
	        this.supplier = source["supplier"];
	        this.payment_method = source["payment_method"];
	        this.remarks = source["remarks"];
	        this.po_number = source["po_number"];
	        this.lines = this.convertValues(source["lines"], POLineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace sales {
	
	export class CollectionLineInput {
	    ar_invoice_id: number;
	    amount_applied: number;
	
	    static createFrom(source: any = {}) {
	        return new CollectionLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ar_invoice_id = source["ar_invoice_id"];
	        this.amount_applied = source["amount_applied"];
	    }
	}

}

