package library

import "time"

// StorageInfo describes the app-managed local storage layout.
type StorageInfo struct {
	RootDir     string `json:"rootDir"`
	BooksDir    string `json:"booksDir"`
	LibraryPath string `json:"libraryPath"`
}

// BookMetadata is the persisted record for an imported EPUB.
type BookMetadata struct {
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	Author           string           `json:"author,omitempty"`
	OriginalFileName string           `json:"originalFileName"`
	FilePath         string           `json:"filePath"`
	ImportedAt       time.Time        `json:"importedAt"`
	Progress         ReadingProgress  `json:"progress"`
	Prompt           PromptConfig     `json:"prompt"`
	Appearance       ReaderAppearance `json:"appearance"`
	Notes            []ReaderNote     `json:"notes,omitempty"`
}

// ReadingProgress stores the user's last known location in a book.
type ReadingProgress struct {
	ChapterHref  string    `json:"chapterHref,omitempty"`
	ChapterIndex int       `json:"chapterIndex"`
	Location     string    `json:"location,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

// PromptConfig stores per-book translation and study prompt settings.
type PromptConfig struct {
	CustomPrompt string        `json:"customPrompt,omitempty"`
	Prompts      []StudyPrompt `json:"prompts,omitempty"`
	UpdatedAt    time.Time     `json:"updatedAt,omitempty"`
}

// StudyPrompt is one book-specific Gemini prompt action shown in the reader.
type StudyPrompt struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortLabel  string `json:"shortLabel"`
	Instruction string `json:"instruction"`
	SortOrder   int    `json:"sortOrder"`
	IsDefault   bool   `json:"isDefault,omitempty"`
}

// ReaderAppearance stores reading comfort preferences for a book.
type ReaderAppearance struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
	FontSize        int    `json:"fontSize,omitempty"`
}

// ReaderNote stores a selected passage or highlight for one book.
type ReaderNote struct {
	ID           string    `json:"id"`
	Text         string    `json:"text"`
	NoteText     string    `json:"noteText,omitempty"`
	ChapterHref  string    `json:"chapterHref"`
	ChapterIndex int       `json:"chapterIndex"`
	ChapterTitle string    `json:"chapterTitle,omitempty"`
	Location     string    `json:"location,omitempty"`
	Color        string    `json:"color,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

// ReaderBook is the UI-facing payload for opening one imported EPUB.
type ReaderBook struct {
	Book                BookMetadata    `json:"book"`
	Chapters            []ReaderChapter `json:"chapters"`
	CurrentChapterIndex int             `json:"currentChapterIndex"`
}

// ReaderChapter is one readable spine item extracted from an EPUB.
type ReaderChapter struct {
	Index    int    `json:"index"`
	Href     string `json:"href"`
	Title    string `json:"title"`
	BodyHTML string `json:"bodyHtml"`
}
