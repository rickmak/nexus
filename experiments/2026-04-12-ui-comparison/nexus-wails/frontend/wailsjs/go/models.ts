export namespace main {
	
	export class Workspace {
	    id: string;
	    name: string;
	    branch: string;
	    status: string;
	    ports: number[];
	    snapshotCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Workspace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.branch = source["branch"];
	        this.status = source["status"];
	        this.ports = source["ports"];
	        this.snapshotCount = source["snapshotCount"];
	    }
	}
	export class RepoGroup {
	    name: string;
	    workspaces: Workspace[];
	
	    static createFrom(source: any = {}) {
	        return new RepoGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.workspaces = this.convertValues(source["workspaces"], Workspace);
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

