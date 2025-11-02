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

}

