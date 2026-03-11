export namespace flocks {
	
	export class CreateFlockRequest {
	    name: string;
	    house_number: string;
	    house_type: string;
	    breed: string;
	    building_name: string;
	    hatch_date: string;
	    initial_count: number;
	    created_by_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new CreateFlockRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.house_number = source["house_number"];
	        this.house_type = source["house_type"];
	        this.breed = source["breed"];
	        this.building_name = source["building_name"];
	        this.hatch_date = source["hatch_date"];
	        this.initial_count = source["initial_count"];
	        this.created_by_id = source["created_by_id"];
	    }
	}
	export class RetireFlockRequest {
	    id: number;
	    retirement_date: string;
	    retirement_type: string;
	    birds_retired: number;
	    retirement_price_per_bird: number;
	    retirement_buyer: string;
	    retirement_notes: string;
	    created_by_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new RetireFlockRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.retirement_date = source["retirement_date"];
	        this.retirement_type = source["retirement_type"];
	        this.birds_retired = source["birds_retired"];
	        this.retirement_price_per_bird = source["retirement_price_per_bird"];
	        this.retirement_buyer = source["retirement_buyer"];
	        this.retirement_notes = source["retirement_notes"];
	        this.created_by_id = source["created_by_id"];
	    }
	}
	export class TransferFlockRequest {
	    id: number;
	    new_house_number: string;
	    new_building: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferFlockRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.new_house_number = source["new_house_number"];
	        this.new_building = source["new_building"];
	    }
	}
	export class UpdateFlockRequest {
	    id: number;
	    name: string;
	    building_name: string;
	    house_number: string;
	    breed: string;
	    hatch_date: string;
	    current_count: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateFlockRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.building_name = source["building_name"];
	        this.house_number = source["house_number"];
	        this.breed = source["breed"];
	        this.hatch_date = source["hatch_date"];
	        this.current_count = source["current_count"];
	    }
	}

}

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
	export class CreateAPInvoiceRequest {
	    purchase_id: number;
	    dr_id?: number;
	    date: string;
	    terms: string;
	    supplier_name: string;
	    supplier_inv_ref: string;
	    notes: string;
	    due_date?: string;
	    items: models.APInvoiceItem[];
	
	    static createFrom(source: any = {}) {
	        return new CreateAPInvoiceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.purchase_id = source["purchase_id"];
	        this.dr_id = source["dr_id"];
	        this.date = source["date"];
	        this.terms = source["terms"];
	        this.supplier_name = source["supplier_name"];
	        this.supplier_inv_ref = source["supplier_inv_ref"];
	        this.notes = source["notes"];
	        this.due_date = source["due_date"];
	        this.items = this.convertValues(source["items"], models.APInvoiceItem);
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
	    sales_order_id?: number;
	    delivery_order_id?: number;
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
	        this.sales_order_id = source["sales_order_id"];
	        this.delivery_order_id = source["delivery_order_id"];
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
	export class CreateDeliveryReceiptRequest {
	    purchase_id: number;
	    date: string;
	    received_by: string;
	    supplier_dr_ref: string;
	    notes: string;
	    items: models.DeliveryReceiptItem[];
	
	    static createFrom(source: any = {}) {
	        return new CreateDeliveryReceiptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.purchase_id = source["purchase_id"];
	        this.date = source["date"];
	        this.received_by = source["received_by"];
	        this.supplier_dr_ref = source["supplier_dr_ref"];
	        this.notes = source["notes"];
	        this.items = this.convertValues(source["items"], models.DeliveryReceiptItem);
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

export namespace models {
	
	export class APPayment {
	    id: number;
	    payment_number: string;
	    // Go type: time
	    date: any;
	    supplier_id?: number;
	    supplier_name_snapshot: string;
	    total_amount: number;
	    payment_method: string;
	    reference_number: string;
	    notes: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    supplier?: Supplier;
	    lines?: APPaymentLine[];
	
	    static createFrom(source: any = {}) {
	        return new APPayment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.payment_number = source["payment_number"];
	        this.date = this.convertValues(source["date"], null);
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name_snapshot = source["supplier_name_snapshot"];
	        this.total_amount = source["total_amount"];
	        this.payment_method = source["payment_method"];
	        this.reference_number = source["reference_number"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.supplier = this.convertValues(source["supplier"], Supplier);
	        this.lines = this.convertValues(source["lines"], APPaymentLine);
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
	export class APPaymentLine {
	    id: number;
	    payment_id: number;
	    ap_invoice_id: number;
	    amount_applied: number;
	    payment?: APPayment;
	    ap_invoice?: APInvoice;
	
	    static createFrom(source: any = {}) {
	        return new APPaymentLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.payment_id = source["payment_id"];
	        this.ap_invoice_id = source["ap_invoice_id"];
	        this.amount_applied = source["amount_applied"];
	        this.payment = this.convertValues(source["payment"], APPayment);
	        this.ap_invoice = this.convertValues(source["ap_invoice"], APInvoice);
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
	export class APInvoiceItem {
	    id: number;
	    ap_invoice_id: number;
	    category: string;
	    item_name: string;
	    unit: string;
	    quantity: number;
	    unit_price: number;
	
	    static createFrom(source: any = {}) {
	        return new APInvoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ap_invoice_id = source["ap_invoice_id"];
	        this.category = source["category"];
	        this.item_name = source["item_name"];
	        this.unit = source["unit"];
	        this.quantity = source["quantity"];
	        this.unit_price = source["unit_price"];
	    }
	}
	export class DeliveryReceiptItem {
	    id: number;
	    delivery_receipt_id: number;
	    category: string;
	    item_name: string;
	    unit: string;
	    quantity_ordered: number;
	    quantity_received: number;
	    unit_price: number;
	
	    static createFrom(source: any = {}) {
	        return new DeliveryReceiptItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.delivery_receipt_id = source["delivery_receipt_id"];
	        this.category = source["category"];
	        this.item_name = source["item_name"];
	        this.unit = source["unit"];
	        this.quantity_ordered = source["quantity_ordered"];
	        this.quantity_received = source["quantity_received"];
	        this.unit_price = source["unit_price"];
	    }
	}
	export class DeliveryReceipt {
	    id: number;
	    dr_number: string;
	    // Go type: time
	    date: any;
	    purchase_id: number;
	    status: string;
	    received_by: string;
	    supplier_dr_ref: string;
	    notes: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    amount_invoiced: number;
	    purchase?: Purchase;
	    items?: DeliveryReceiptItem[];
	    ap_invoices?: APInvoice[];
	
	    static createFrom(source: any = {}) {
	        return new DeliveryReceipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.dr_number = source["dr_number"];
	        this.date = this.convertValues(source["date"], null);
	        this.purchase_id = source["purchase_id"];
	        this.status = source["status"];
	        this.received_by = source["received_by"];
	        this.supplier_dr_ref = source["supplier_dr_ref"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.amount_invoiced = source["amount_invoiced"];
	        this.purchase = this.convertValues(source["purchase"], Purchase);
	        this.items = this.convertValues(source["items"], DeliveryReceiptItem);
	        this.ap_invoices = this.convertValues(source["ap_invoices"], APInvoice);
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
	    // Go type: time
	    created_at: any;
	
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
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class Purchase {
	    id: number;
	    // Go type: time
	    date: any;
	    category: string;
	    item_name: string;
	    quantity: number;
	    unit: string;
	    unit_price: number;
	    total_cost: number;
	    supplier: string;
	    supplier_id?: number;
	    po_number: string;
	    invoice_number: string;
	    received_by: string;
	    payment_status: string;
	    amount_paid: number;
	    payment_method: string;
	    remarks: string;
	    created_by_id?: number;
	    amount_received: number;
	    amount_invoiced: number;
	    amount_settled: number;
	    is_active: boolean;
	    supplier_rel?: Supplier;
	    delivery_receipts?: DeliveryReceipt[];
	    ap_invoices?: APInvoice[];
	
	    static createFrom(source: any = {}) {
	        return new Purchase(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date = this.convertValues(source["date"], null);
	        this.category = source["category"];
	        this.item_name = source["item_name"];
	        this.quantity = source["quantity"];
	        this.unit = source["unit"];
	        this.unit_price = source["unit_price"];
	        this.total_cost = source["total_cost"];
	        this.supplier = source["supplier"];
	        this.supplier_id = source["supplier_id"];
	        this.po_number = source["po_number"];
	        this.invoice_number = source["invoice_number"];
	        this.received_by = source["received_by"];
	        this.payment_status = source["payment_status"];
	        this.amount_paid = source["amount_paid"];
	        this.payment_method = source["payment_method"];
	        this.remarks = source["remarks"];
	        this.created_by_id = source["created_by_id"];
	        this.amount_received = source["amount_received"];
	        this.amount_invoiced = source["amount_invoiced"];
	        this.amount_settled = source["amount_settled"];
	        this.is_active = source["is_active"];
	        this.supplier_rel = this.convertValues(source["supplier_rel"], Supplier);
	        this.delivery_receipts = this.convertValues(source["delivery_receipts"], DeliveryReceipt);
	        this.ap_invoices = this.convertValues(source["ap_invoices"], APInvoice);
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
	export class APInvoice {
	    id: number;
	    invoice_number: string;
	    // Go type: time
	    date: any;
	    // Go type: time
	    due_date?: any;
	    terms: string;
	    purchase_id: number;
	    delivery_receipt_id?: number;
	    supplier_name: string;
	    supplier_invoice_ref: string;
	    total_amount: number;
	    status: string;
	    notes: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
	    amount_paid_stored: number;
	    purchase?: Purchase;
	    delivery_receipt?: DeliveryReceipt;
	    items?: APInvoiceItem[];
	    payment_lines?: APPaymentLine[];
	
	    static createFrom(source: any = {}) {
	        return new APInvoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.invoice_number = source["invoice_number"];
	        this.date = this.convertValues(source["date"], null);
	        this.due_date = this.convertValues(source["due_date"], null);
	        this.terms = source["terms"];
	        this.purchase_id = source["purchase_id"];
	        this.delivery_receipt_id = source["delivery_receipt_id"];
	        this.supplier_name = source["supplier_name"];
	        this.supplier_invoice_ref = source["supplier_invoice_ref"];
	        this.total_amount = source["total_amount"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.amount_paid_stored = source["amount_paid_stored"];
	        this.purchase = this.convertValues(source["purchase"], Purchase);
	        this.delivery_receipt = this.convertValues(source["delivery_receipt"], DeliveryReceipt);
	        this.items = this.convertValues(source["items"], APInvoiceItem);
	        this.payment_lines = this.convertValues(source["payment_lines"], APPaymentLine);
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
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
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
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
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
	    sku: string;
	    unit: string;
	    quantity: number;
	    price_per_unit: number;
	
	    static createFrom(source: any = {}) {
	        return new ARInvoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ar_invoice_id = source["ar_invoice_id"];
	        this.sku = source["sku"];
	        this.unit = source["unit"];
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
	    notes: string;
	    // Go type: time
	    created_at: any;
	    created_by_id?: number;
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
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
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
	    quantity: number;
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
	        this.quantity = source["quantity"];
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
	    delivery_order_id?: number;
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
	    amount_collected: number;
	    sales_order?: SalesOrder;
	    delivery_order?: DeliveryOrder;
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
	        this.delivery_order_id = source["delivery_order_id"];
	        this.customer_id = source["customer_id"];
	        this.customer_name_snapshot = source["customer_name_snapshot"];
	        this.customer_address_snapshot = source["customer_address_snapshot"];
	        this.customer_contact_snapshot = source["customer_contact_snapshot"];
	        this.total_amount = source["total_amount"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.created_by_id = source["created_by_id"];
	        this.amount_collected = source["amount_collected"];
	        this.sales_order = this.convertValues(source["sales_order"], SalesOrder);
	        this.delivery_order = this.convertValues(source["delivery_order"], DeliveryOrder);
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
	
	
	
	
	
	
	
	
	export class ItemCategory {
	    id: number;
	    name: string;
	    description: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ItemCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.is_active = source["is_active"];
	    }
	}
	export class ItemMaster {
	    id: number;
	    item_code: string;
	    name: string;
	    category: string;
	    unit: string;
	    unit_price: number;
	    description: string;
	    reorder_level: number;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ItemMaster(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.item_code = source["item_code"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.unit = source["unit"];
	        this.unit_price = source["unit_price"];
	        this.description = source["description"];
	        this.reorder_level = source["reorder_level"];
	        this.is_active = source["is_active"];
	    }
	}
	
	
	
	
	

}

export namespace operations {
	
	export class RecordDailyLogRequest {
	    flock_id: number;
	    date: string;
	    pewee: number;
	    pullet: number;
	    small: number;
	    medium: number;
	    large: number;
	    extra_large: number;
	    jumbo: number;
	    double_yolk: number;
	    cracked_dirty: number;
	    feed_consumed_kg: number;
	    mortality: number;
	    vaccine_name: string;
	    feed_type_id?: number;
	    created_by_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new RecordDailyLogRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.flock_id = source["flock_id"];
	        this.date = source["date"];
	        this.pewee = source["pewee"];
	        this.pullet = source["pullet"];
	        this.small = source["small"];
	        this.medium = source["medium"];
	        this.large = source["large"];
	        this.extra_large = source["extra_large"];
	        this.jumbo = source["jumbo"];
	        this.double_yolk = source["double_yolk"];
	        this.cracked_dirty = source["cracked_dirty"];
	        this.feed_consumed_kg = source["feed_consumed_kg"];
	        this.mortality = source["mortality"];
	        this.vaccine_name = source["vaccine_name"];
	        this.feed_type_id = source["feed_type_id"];
	        this.created_by_id = source["created_by_id"];
	    }
	}
	export class RecordGrowerLogRequest {
	    flock_id: number;
	    date: string;
	    feed_consumed_kg: number;
	    mortality: number;
	    vaccine_name: string;
	    medication: string;
	    remarks: string;
	    feed_type_id?: number;
	    created_by_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new RecordGrowerLogRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.flock_id = source["flock_id"];
	        this.date = source["date"];
	        this.feed_consumed_kg = source["feed_consumed_kg"];
	        this.mortality = source["mortality"];
	        this.vaccine_name = source["vaccine_name"];
	        this.medication = source["medication"];
	        this.remarks = source["remarks"];
	        this.feed_type_id = source["feed_type_id"];
	        this.created_by_id = source["created_by_id"];
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

