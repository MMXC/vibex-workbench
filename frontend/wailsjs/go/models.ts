export namespace designkit {
	
	export class GateCheck {
	    id: string;
	    label: string;
	    passed: boolean;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new GateCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.passed = source["passed"];
	        this.detail = source["detail"];
	    }
	}
	export class GateFailureDetail {
	    codes: string[];
	    checks: GateCheck[];
	    nextAction: string;
	
	    static createFrom(source: any = {}) {
	        return new GateFailureDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.codes = source["codes"];
	        this.checks = this.convertValues(source["checks"], GateCheck);
	        this.nextAction = source["nextAction"];
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
	export class ExtractResult {
	    ok: boolean;
	    writtenPath?: string;
	    specSnippet?: string;
	    sourcePath?: string;
	    error?: string;
	    gateBlocked?: boolean;
	    gateFailure?: GateFailureDetail;
	
	    static createFrom(source: any = {}) {
	        return new ExtractResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.writtenPath = source["writtenPath"];
	        this.specSnippet = source["specSnippet"];
	        this.sourcePath = source["sourcePath"];
	        this.error = source["error"];
	        this.gateBlocked = source["gateBlocked"];
	        this.gateFailure = this.convertValues(source["gateFailure"], GateFailureDetail);
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
	
	
	export class ScaffoldResult {
	    ok: boolean;
	    written?: string[];
	    skipped?: string[];
	    error?: string;
	    gateBlocked?: boolean;
	    gateFailure?: GateFailureDetail;
	
	    static createFrom(source: any = {}) {
	        return new ScaffoldResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.written = source["written"];
	        this.skipped = source["skipped"];
	        this.error = source["error"];
	        this.gateBlocked = source["gateBlocked"];
	        this.gateFailure = this.convertValues(source["gateFailure"], GateFailureDetail);
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
	export class Status {
	    ok: boolean;
	    workspaceRoot?: string;
	    designMdExists: boolean;
	    prototypesDirExists: boolean;
	    designPath: string;
	    prototypesPath: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.workspaceRoot = source["workspaceRoot"];
	        this.designMdExists = source["designMdExists"];
	        this.prototypesDirExists = source["prototypesDirExists"];
	        this.designPath = source["designPath"];
	        this.prototypesPath = source["prototypesPath"];
	        this.error = source["error"];
	    }
	}

}

export namespace main {
	
	export class Signal {
	    path: string;
	    exists: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Signal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.exists = source["exists"];
	        this.reason = source["reason"];
	    }
	}
	export class SpecFile {
	    path: string;
	    level: number;
	    name: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new SpecFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.level = source["level"];
	        this.name = source["name"];
	        this.status = source["status"];
	    }
	}
	export class WorkspaceState {
	    state: string;
	    signals: Signal[];
	    suggestions: string[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.signals = this.convertValues(source["signals"], Signal);
	        this.suggestions = source["suggestions"];
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

