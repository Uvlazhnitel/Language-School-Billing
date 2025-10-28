export namespace attendance {
	
	export class Row {
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

