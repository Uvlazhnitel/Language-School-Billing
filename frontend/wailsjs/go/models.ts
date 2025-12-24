export namespace attendance {
	
	export class Row {
	    enrollmentId: number;
	    studentId: number;
	    studentName: string;
	    courseId: number;
	    courseName: string;
	    courseType: string;
	    lessonPrice: number;
	    count: number;
	    locked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Row(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enrollmentId = source["enrollmentId"];
	        this.studentId = source["studentId"];
	        this.studentName = source["studentName"];
	        this.courseId = source["courseId"];
	        this.courseName = source["courseName"];
	        this.courseType = source["courseType"];
	        this.lessonPrice = source["lessonPrice"];
	        this.count = source["count"];
	        this.locked = source["locked"];
	    }
	}

}

export namespace invoice {
	
	export class GenerateResult {
	    created: number;
	    updated: number;
	    skippedHasInvoice: number;
	    skippedNoLines: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = source["created"];
	        this.updated = source["updated"];
	        this.skippedHasInvoice = source["skippedHasInvoice"];
	        this.skippedNoLines = source["skippedNoLines"];
	    }
	}
	export class LineDTO {
	    enrollmentId: number;
	    description: string;
	    qty: number;
	    unitPrice: number;
	    amount: number;
	
	    static createFrom(source: any = {}) {
	        return new LineDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enrollmentId = source["enrollmentId"];
	        this.description = source["description"];
	        this.qty = source["qty"];
	        this.unitPrice = source["unitPrice"];
	        this.amount = source["amount"];
	    }
	}
	export class InvoiceDTO {
	    id: number;
	    studentId: number;
	    studentName: string;
	    year: number;
	    month: number;
	    total: number;
	    status: string;
	    number?: string;
	    lines: LineDTO[];
	
	    static createFrom(source: any = {}) {
	        return new InvoiceDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.studentId = source["studentId"];
	        this.studentName = source["studentName"];
	        this.year = source["year"];
	        this.month = source["month"];
	        this.total = source["total"];
	        this.status = source["status"];
	        this.number = source["number"];
	        this.lines = this.convertValues(source["lines"], LineDTO);
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
	
	export class ListItem {
	    id: number;
	    studentId: number;
	    studentName: string;
	    year: number;
	    month: number;
	    total: number;
	    status: string;
	    linesCount: number;
	    number?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.studentId = source["studentId"];
	        this.studentName = source["studentName"];
	        this.year = source["year"];
	        this.month = source["month"];
	        this.total = source["total"];
	        this.status = source["status"];
	        this.linesCount = source["linesCount"];
	        this.number = source["number"];
	    }
	}

}

export namespace main {
	
	export class CourseDTO {
	    id: number;
	    name: string;
	    type: string;
	    lessonPrice: number;
	    subscriptionPrice: number;
	
	    static createFrom(source: any = {}) {
	        return new CourseDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.lessonPrice = source["lessonPrice"];
	        this.subscriptionPrice = source["subscriptionPrice"];
	    }
	}
	export class EnrollmentDTO {
	    id: number;
	    studentId: number;
	    studentName: string;
	    courseId: number;
	    courseName: string;
	    billingMode: string;
	    discountPct: number;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new EnrollmentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.studentId = source["studentId"];
	        this.studentName = source["studentName"];
	        this.courseId = source["courseId"];
	        this.courseName = source["courseName"];
	        this.billingMode = source["billingMode"];
	        this.discountPct = source["discountPct"];
	        this.note = source["note"];
	    }
	}
	export class IssueAllResult {
	    count: number;
	    pdfPaths: string[];
	
	    static createFrom(source: any = {}) {
	        return new IssueAllResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.pdfPaths = source["pdfPaths"];
	    }
	}
	export class IssueResult {
	    number: string;
	    pdfPath: string;
	
	    static createFrom(source: any = {}) {
	        return new IssueResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.pdfPath = source["pdfPath"];
	    }
	}
	export class StudentDTO {
	    id: number;
	    fullName: string;
	    phone: string;
	    email: string;
	    note: string;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StudentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fullName = source["fullName"];
	        this.phone = source["phone"];
	        this.email = source["email"];
	        this.note = source["note"];
	        this.isActive = source["isActive"];
	    }
	}

}

export namespace payment {
	
	export class BalanceDTO {
	    studentId: number;
	    studentName: string;
	    totalInvoiced: number;
	    totalPaid: number;
	    balance: number;
	    debt: number;
	
	    static createFrom(source: any = {}) {
	        return new BalanceDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.studentId = source["studentId"];
	        this.studentName = source["studentName"];
	        this.totalInvoiced = source["totalInvoiced"];
	        this.totalPaid = source["totalPaid"];
	        this.balance = source["balance"];
	        this.debt = source["debt"];
	    }
	}
	export class DebtorDTO {
	    studentId: number;
	    studentName: string;
	    debt: number;
	    totalInvoiced: number;
	    totalPaid: number;
	
	    static createFrom(source: any = {}) {
	        return new DebtorDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.studentId = source["studentId"];
	        this.studentName = source["studentName"];
	        this.debt = source["debt"];
	        this.totalInvoiced = source["totalInvoiced"];
	        this.totalPaid = source["totalPaid"];
	    }
	}
	export class InvoiceSummaryDTO {
	    invoiceId: number;
	    total: number;
	    paid: number;
	    remaining: number;
	    status: string;
	    number?: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.invoiceId = source["invoiceId"];
	        this.total = source["total"];
	        this.paid = source["paid"];
	        this.remaining = source["remaining"];
	        this.status = source["status"];
	        this.number = source["number"];
	    }
	}
	export class PaymentDTO {
	    id: number;
	    studentId: number;
	    invoiceId?: number;
	    paidAt: string;
	    amount: number;
	    method: string;
	    note: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new PaymentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.studentId = source["studentId"];
	        this.invoiceId = source["invoiceId"];
	        this.paidAt = source["paidAt"];
	        this.amount = source["amount"];
	        this.method = source["method"];
	        this.note = source["note"];
	        this.createdAt = source["createdAt"];
	    }
	}

}

