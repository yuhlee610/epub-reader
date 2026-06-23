export namespace library {

	export class ReaderBookmark {
	    id: string;
	    title?: string;
	    chapterHref: string;
	    chapterIndex: number;
	    chapterTitle?: string;
	    location: string;
	    snippet?: string;
	    progressPercent?: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt?: any;

	    static createFrom(source: any = {}) {
	        return new ReaderBookmark(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.chapterHref = source["chapterHref"];
	        this.chapterIndex = source["chapterIndex"];
	        this.chapterTitle = source["chapterTitle"];
	        this.location = source["location"];
	        this.snippet = source["snippet"];
	        this.progressPercent = source["progressPercent"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ReaderNote {
	    id: string;
	    text: string;
	    noteText?: string;
	    chapterHref: string;
	    chapterIndex: number;
	    chapterTitle?: string;
	    location?: string;
	    color?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt?: any;

	    static createFrom(source: any = {}) {
	        return new ReaderNote(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.noteText = source["noteText"];
	        this.chapterHref = source["chapterHref"];
	        this.chapterIndex = source["chapterIndex"];
	        this.chapterTitle = source["chapterTitle"];
	        this.location = source["location"];
	        this.color = source["color"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ReaderAppearance {
	    backgroundColor?: string;
	    fontSize?: number;

	    static createFrom(source: any = {}) {
	        return new ReaderAppearance(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backgroundColor = source["backgroundColor"];
	        this.fontSize = source["fontSize"];
	    }
	}
	export class StudyPrompt {
	    id: string;
	    name: string;
	    shortLabel: string;
	    instruction: string;
	    sortOrder: number;
	    isDefault?: boolean;

	    static createFrom(source: any = {}) {
	        return new StudyPrompt(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.shortLabel = source["shortLabel"];
	        this.instruction = source["instruction"];
	        this.sortOrder = source["sortOrder"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class PromptConfig {
	    customPrompt?: string;
	    prompts?: StudyPrompt[];
	    // Go type: time
	    updatedAt?: any;

	    static createFrom(source: any = {}) {
	        return new PromptConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customPrompt = source["customPrompt"];
	        this.prompts = this.convertValues(source["prompts"], StudyPrompt);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ReadingProgress {
	    chapterHref?: string;
	    chapterIndex: number;
	    location?: string;
	    // Go type: time
	    updatedAt?: any;

	    static createFrom(source: any = {}) {
	        return new ReadingProgress(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterHref = source["chapterHref"];
	        this.chapterIndex = source["chapterIndex"];
	        this.location = source["location"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class BookMetadata {
	    id: string;
	    title: string;
	    author?: string;
	    originalFileName: string;
	    filePath: string;
	    // Go type: time
	    importedAt: any;
	    progress: ReadingProgress;
	    prompt: PromptConfig;
	    appearance: ReaderAppearance;
	    notes?: ReaderNote[];
	    bookmarks?: ReaderBookmark[];

	    static createFrom(source: any = {}) {
	        return new BookMetadata(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.originalFileName = source["originalFileName"];
	        this.filePath = source["filePath"];
	        this.importedAt = this.convertValues(source["importedAt"], null);
	        this.progress = this.convertValues(source["progress"], ReadingProgress);
	        this.prompt = this.convertValues(source["prompt"], PromptConfig);
	        this.appearance = this.convertValues(source["appearance"], ReaderAppearance);
	        this.notes = this.convertValues(source["notes"], ReaderNote);
	        this.bookmarks = this.convertValues(source["bookmarks"], ReaderBookmark);
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


	export class ReaderChapter {
	    index: number;
	    href: string;
	    title: string;
	    bodyHtml: string;

	    static createFrom(source: any = {}) {
	        return new ReaderChapter(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.href = source["href"];
	        this.title = source["title"];
	        this.bodyHtml = source["bodyHtml"];
	    }
	}
	export class ReaderBook {
	    book: BookMetadata;
	    chapters: ReaderChapter[];
	    currentChapterIndex: number;

	    static createFrom(source: any = {}) {
	        return new ReaderBook(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.book = this.convertValues(source["book"], BookMetadata);
	        this.chapters = this.convertValues(source["chapters"], ReaderChapter);
	        this.currentChapterIndex = source["currentChapterIndex"];
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




	export class StorageInfo {
	    rootDir: string;
	    booksDir: string;
	    libraryPath: string;

	    static createFrom(source: any = {}) {
	        return new StorageInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootDir = source["rootDir"];
	        this.booksDir = source["booksDir"];
	        this.libraryPath = source["libraryPath"];
	    }
	}

}

export namespace main {

	export class GeminiStudyRequest {
	    requestId?: string;
	    bookId: string;
	    promptId: string;
	    selectedText: string;
	    chapterTitle?: string;

	    static createFrom(source: any = {}) {
	        return new GeminiStudyRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.bookId = source["bookId"];
	        this.promptId = source["promptId"];
	        this.selectedText = source["selectedText"];
	        this.chapterTitle = source["chapterTitle"];
	    }
	}
	export class GeminiStudyResponse {
	    promptId: string;
	    promptName: string;
	    text: string;
	    usedModel?: string;
	    textPreview: string;

	    static createFrom(source: any = {}) {
	        return new GeminiStudyResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptId = source["promptId"];
	        this.promptName = source["promptName"];
	        this.text = source["text"];
	        this.usedModel = source["usedModel"];
	        this.textPreview = source["textPreview"];
	    }
	}
	export class GoogleTranslateRequest {
	    text: string;
	    sourceLanguage?: string;
	    targetLanguage?: string;

	    static createFrom(source: any = {}) {
	        return new GoogleTranslateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.sourceLanguage = source["sourceLanguage"];
	        this.targetLanguage = source["targetLanguage"];
	    }
	}
	export class GoogleTranslateResponse {
	    originalText: string;
	    translatedText: string;
	    sourceLanguage?: string;
	    targetLanguage: string;
	    pronunciationIpa?: string;
	    textPreview: string;

	    static createFrom(source: any = {}) {
	        return new GoogleTranslateResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalText = source["originalText"];
	        this.translatedText = source["translatedText"];
	        this.sourceLanguage = source["sourceLanguage"];
	        this.targetLanguage = source["targetLanguage"];
	        this.pronunciationIpa = source["pronunciationIpa"];
	        this.textPreview = source["textPreview"];
	    }
	}
	export class ImportEPUBResult {
	    book: library.BookMetadata;
	    duplicate: boolean;
	    canceled: boolean;

	    static createFrom(source: any = {}) {
	        return new ImportEPUBResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.book = this.convertValues(source["book"], library.BookMetadata);
	        this.duplicate = source["duplicate"];
	        this.canceled = source["canceled"];
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
