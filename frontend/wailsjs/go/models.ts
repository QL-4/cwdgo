export namespace main {
	
	export class Folder {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}

}

export namespace openactions {
	
	export class Software {
	    name: string;
	    exe: string;
	    args: string[];
	
	    static createFrom(source: any = {}) {
	        return new Software(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.exe = source["exe"];
	        this.args = source["args"];
	    }
	}

}

export namespace settings {
	
	export class Settings {
	    historyLimit: number;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.historyLimit = source["historyLimit"];
	        this.autoStart = source["autoStart"];
	    }
	}

}

