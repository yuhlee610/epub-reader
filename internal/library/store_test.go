package library

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// newTestStore creates a store with deterministic IDs and timestamps for tests.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	store.idGenerator = func() (string, error) { return "book-1", nil }
	store.timeProvider = func() time.Time { return time.Date(2026, 5, 29, 1, 2, 3, 0, time.UTC) }

	return store
}

// TestNewStoreInitializesLayout verifies a fresh store creates the expected
// managed storage layout and starts with no books.
func TestNewStoreInitializesLayout(t *testing.T) {
	store := newTestStore(t)
	info := store.Info()

	if info.RootDir == "" {
		t.Fatal("RootDir is empty")
	}
	if info.BooksDir != filepath.Join(info.RootDir, booksDirName) {
		t.Fatalf("BooksDir = %q, want managed books dir under root", info.BooksDir)
	}
	if info.LibraryPath != filepath.Join(info.RootDir, libraryFileName) {
		t.Fatalf("LibraryPath = %q, want library file under root", info.LibraryPath)
	}

	books, err := store.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 0 {
		t.Fatalf("ListBooks() length = %d, want 0", len(books))
	}
}

// TestDefaultRootDirUsesAppDirectoryName verifies the default root is stable
// and app-scoped on the current OS.
func TestDefaultRootDirUsesAppDirectoryName(t *testing.T) {
	root, err := DefaultRootDir()
	if err != nil {
		t.Fatalf("DefaultRootDir() error = %v", err)
	}
	if filepath.Base(root) != appDirName {
		t.Fatalf("DefaultRootDir() = %q, want path ending in %q", root, appDirName)
	}
	t.Logf("default storage root: %s", root)
}

// TestSaveBookCreatesAndPersistsMetadata verifies create, reload, and metadata
// persistence for progress and prompt fields.
func TestSaveBookCreatesAndPersistsMetadata(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		Author:           "Will Wight",
		OriginalFileName: "cradle.epub",
		Progress: ReadingProgress{
			ChapterHref:  "chapter-1.xhtml",
			ChapterIndex: 1,
			Location:     "p10",
		},
		Prompt: PromptConfig{
			CustomPrompt: "Translate with context and explain grammar.",
		},
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}
	if book.ID != "book-1" {
		t.Fatalf("book.ID = %q, want generated id", book.ID)
	}
	if book.ImportedAt.IsZero() {
		t.Fatal("ImportedAt was not set")
	}
	if filepath.Dir(filepath.Dir(book.FilePath)) != store.Info().BooksDir {
		t.Fatalf("FilePath = %q, want inside managed books dir", book.FilePath)
	}

	reloaded, err := NewStore(store.Info().RootDir)
	if err != nil {
		t.Fatalf("reload NewStore() error = %v", err)
	}
	got, err := reloaded.GetBook("book-1")
	if err != nil {
		t.Fatalf("GetBook() error = %v", err)
	}
	if got.Title != "Cradle" || len(got.Prompt.Prompts) == 0 || got.Progress.ChapterHref == "" {
		t.Fatalf("persisted book missing expected metadata: %#v", got)
	}
}

// TestSaveBookUpdatePreservesImportedAtWhenOmitted verifies partial updates do
// not accidentally rewrite import-time fields.
func TestSaveBookUpdatePreservesImportedAtWhenOmitted(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
		Prompt: PromptConfig{
			CustomPrompt: "Translate with sacred arts context.",
		},
	})
	if err != nil {
		t.Fatalf("initial SaveBook() error = %v", err)
	}

	updated, err := store.SaveBook(BookMetadata{
		ID:    book.ID,
		Title: "Cradle",
	})
	if err != nil {
		t.Fatalf("update SaveBook() error = %v", err)
	}
	if !updated.ImportedAt.Equal(book.ImportedAt) {
		t.Fatalf("ImportedAt = %s, want preserved %s", updated.ImportedAt, book.ImportedAt)
	}
	if updated.FilePath != book.FilePath {
		t.Fatalf("FilePath = %q, want preserved %q", updated.FilePath, book.FilePath)
	}
	if !reflect.DeepEqual(updated.Prompt, book.Prompt) {
		t.Fatalf("Prompt = %#v, want preserved %#v", updated.Prompt, book.Prompt)
	}
}

// TestSaveAndDeleteReaderNotePersistsBookNotes verifies reader notes are
// stored per book, survive reload, and can be removed.
func TestSaveAndDeleteReaderNotePersistsBookNotes(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}

	withNote, err := store.SaveReaderNote(book.ID, ReaderNote{
		ID:           "Note 1",
		Text:         "The Unsouled kept walking.",
		NoteText:     "Remember this character beat.",
		ChapterHref:  "chapter-1.xhtml",
		ChapterIndex: 1,
		ChapterTitle: "Chapter 1",
		Location:     "page:2;percent:12",
		Color:        "#facd4d",
	})
	if err != nil {
		t.Fatalf("SaveReaderNote() error = %v", err)
	}
	if len(withNote.Notes) != 1 {
		t.Fatalf("Notes length = %d, want 1", len(withNote.Notes))
	}
	if withNote.Notes[0].ID != "note-1" {
		t.Fatalf("note ID = %q, want cleaned note-1", withNote.Notes[0].ID)
	}
	if withNote.Notes[0].CreatedAt.IsZero() || withNote.Notes[0].UpdatedAt.IsZero() {
		t.Fatalf("note timestamps were not set: %#v", withNote.Notes[0])
	}

	reloaded, err := NewStore(store.Info().RootDir)
	if err != nil {
		t.Fatalf("reload NewStore() error = %v", err)
	}
	got, err := reloaded.GetBook(book.ID)
	if err != nil {
		t.Fatalf("GetBook() error = %v", err)
	}
	if len(got.Notes) != 1 ||
		got.Notes[0].Text != "The Unsouled kept walking." ||
		got.Notes[0].NoteText != "Remember this character beat." {
		t.Fatalf("persisted notes = %#v, want saved note", got.Notes)
	}

	withoutNote, err := reloaded.DeleteReaderNote(book.ID, "note-1")
	if err != nil {
		t.Fatalf("DeleteReaderNote() error = %v", err)
	}
	if len(withoutNote.Notes) != 0 {
		t.Fatalf("Notes length after delete = %d, want 0", len(withoutNote.Notes))
	}
}

// TestSaveAndDeleteReaderBookmarkPersistsBookBookmarks verifies bookmarks are
// stored per book, deduplicated by location, survive reload, and can be removed.
func TestSaveAndDeleteReaderBookmarkPersistsBookBookmarks(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}

	withBookmark, err := store.SaveReaderBookmark(book.ID, ReaderBookmark{
		ID:              "Bookmark 1",
		Title:           "Chapter 1 - 12%",
		ChapterHref:     "chapter-1.xhtml",
		ChapterIndex:    1,
		ChapterTitle:    "Chapter 1",
		Location:        "page:2;percent:12",
		Snippet:         "The Unsouled kept walking.",
		ProgressPercent: 12,
	})
	if err != nil {
		t.Fatalf("SaveReaderBookmark() error = %v", err)
	}
	if len(withBookmark.Bookmarks) != 1 {
		t.Fatalf("Bookmarks length = %d, want 1", len(withBookmark.Bookmarks))
	}
	if withBookmark.Bookmarks[0].ID != "bookmark-1" {
		t.Fatalf("bookmark ID = %q, want cleaned bookmark-1", withBookmark.Bookmarks[0].ID)
	}
	if withBookmark.Bookmarks[0].CreatedAt.IsZero() || withBookmark.Bookmarks[0].UpdatedAt.IsZero() {
		t.Fatalf("bookmark timestamps were not set: %#v", withBookmark.Bookmarks[0])
	}

	deduped, err := store.SaveReaderBookmark(book.ID, ReaderBookmark{
		ID:              "Bookmark duplicate",
		Title:           "Updated title",
		ChapterHref:     "chapter-1.xhtml",
		ChapterIndex:    1,
		ChapterTitle:    "Chapter 1",
		Location:        "page:2;percent:12",
		Snippet:         "Updated snippet.",
		ProgressPercent: 12,
	})
	if err != nil {
		t.Fatalf("duplicate SaveReaderBookmark() error = %v", err)
	}
	if len(deduped.Bookmarks) != 1 || deduped.Bookmarks[0].Title != "Updated title" {
		t.Fatalf("deduped bookmarks = %#v, want one updated bookmark", deduped.Bookmarks)
	}

	reloaded, err := NewStore(store.Info().RootDir)
	if err != nil {
		t.Fatalf("reload NewStore() error = %v", err)
	}
	got, err := reloaded.GetBook(book.ID)
	if err != nil {
		t.Fatalf("GetBook() error = %v", err)
	}
	if len(got.Bookmarks) != 1 ||
		got.Bookmarks[0].Title != "Updated title" ||
		got.Bookmarks[0].Snippet != "Updated snippet." {
		t.Fatalf("persisted bookmarks = %#v, want saved bookmark", got.Bookmarks)
	}

	withoutBookmark, err := reloaded.DeleteReaderBookmark(book.ID, got.Bookmarks[0].ID)
	if err != nil {
		t.Fatalf("DeleteReaderBookmark() error = %v", err)
	}
	if len(withoutBookmark.Bookmarks) != 0 {
		t.Fatalf("Bookmarks length after delete = %d, want 0", len(withoutBookmark.Bookmarks))
	}
}

// TestSaveBookRejectsPathOutsideManagedStorage verifies metadata cannot point
// at a file outside the app-managed books directory.
func TestSaveBookRejectsPathOutsideManagedStorage(t *testing.T) {
	store := newTestStore(t)

	_, err := store.SaveBook(BookMetadata{
		ID:               "book-1",
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
		FilePath:         filepath.Join(t.TempDir(), "cradle.epub"),
	})
	if !errors.Is(err, ErrPathOutsideLibrary) {
		t.Fatalf("SaveBook() error = %v, want ErrPathOutsideLibrary", err)
	}
}

// TestSaveBookRejectsPathUnderDifferentBookDirectory verifies each book is
// constrained to its own managed folder.
func TestSaveBookRejectsPathUnderDifferentBookDirectory(t *testing.T) {
	store := newTestStore(t)

	_, err := store.SaveBook(BookMetadata{
		ID:               "book-2",
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
		FilePath:         filepath.Join(store.Info().BooksDir, "book-1", "cradle.epub"),
	})
	if !errors.Is(err, ErrPathOutsideLibrary) {
		t.Fatalf("SaveBook() error = %v, want ErrPathOutsideLibrary", err)
	}
}

// TestManagedBookPathSanitizesName verifies generated managed filenames ignore
// source path traversal and normalize to a .epub file.
func TestManagedBookPathSanitizesName(t *testing.T) {
	store := newTestStore(t)

	path, err := store.ManagedBookPath("book-1", `..\bad:name.txt`)
	if err != nil {
		t.Fatalf("ManagedBookPath() error = %v", err)
	}
	if filepath.Dir(filepath.Dir(path)) != store.Info().BooksDir {
		t.Fatalf("path = %q, want inside managed books dir", path)
	}
	if filepath.Ext(path) != ".epub" {
		t.Fatalf("path = %q, want .epub extension", path)
	}
	if filepath.Base(path) != "bad-name.epub" {
		t.Fatalf("path base = %q, want sanitized name", filepath.Base(path))
	}
}

// TestManagedBookPathRejectsPathLikeBookID verifies book IDs cannot smuggle path
// traversal or Windows-invalid filename characters into managed paths.
func TestManagedBookPathRejectsPathLikeBookID(t *testing.T) {
	store := newTestStore(t)

	for _, id := range []string{`..\outside`, `bad:id`, ` trailing.`} {
		_, err := store.ManagedBookPath(id, "cradle.epub")
		if !errors.Is(err, ErrInvalidBook) {
			t.Fatalf("ManagedBookPath(%q) error = %v, want ErrInvalidBook", id, err)
		}
	}
}

// TestSaveBookAcceptsUppercaseEPUBExtension verifies validation accepts common
// uppercase EPUB extensions from imported files.
func TestSaveBookAcceptsUppercaseEPUBExtension(t *testing.T) {
	store := newTestStore(t)
	path := filepath.Join(store.Info().BooksDir, "book-1", "CRADLE.EPUB")

	_, err := store.SaveBook(BookMetadata{
		ID:               "book-1",
		Title:            "Cradle",
		OriginalFileName: "CRADLE.EPUB",
		FilePath:         path,
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}
}

// TestUpdateReaderAppearancePersists verifies reader comfort preferences are
// normalized, saved, and available after reloading the store.
func TestUpdateReaderAppearancePersists(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}

	updated, err := store.UpdateReaderAppearance(book.ID, ReaderAppearance{
		BackgroundColor: "#F8F1E7",
		FontSize:        99,
	})
	if err != nil {
		t.Fatalf("UpdateReaderAppearance() error = %v", err)
	}
	if updated.Appearance.BackgroundColor != "#f8f1e7" {
		t.Fatalf("BackgroundColor = %q, want normalized palette color", updated.Appearance.BackgroundColor)
	}
	if updated.Appearance.FontSize != maxReaderFontSize {
		t.Fatalf("FontSize = %d, want clamped max %d", updated.Appearance.FontSize, maxReaderFontSize)
	}

	reloaded, err := NewStore(store.Info().RootDir)
	if err != nil {
		t.Fatalf("reload NewStore() error = %v", err)
	}
	got, err := reloaded.GetBook(book.ID)
	if err != nil {
		t.Fatalf("GetBook() error = %v", err)
	}
	if got.Appearance != updated.Appearance {
		t.Fatalf("persisted Appearance = %#v, want %#v", got.Appearance, updated.Appearance)
	}
}

// TestUpdatePromptConfigPersists verifies per-book prompt lists survive reloads.
func TestUpdatePromptConfigPersists(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}

	updated, err := store.UpdatePromptConfig(book.ID, PromptConfig{
		Prompts: []StudyPrompt{
			{
				ID:          "translate",
				Name:        "Translate",
				ShortLabel:  "tr",
				Instruction: "  Translate naturally and explain advancement terms.  ",
				SortOrder:   2,
			},
			{
				ID:          "grammar",
				Name:        "Grammar",
				ShortLabel:  "grammar",
				Instruction: "Explain grammar.",
				SortOrder:   1,
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdatePromptConfig() error = %v", err)
	}
	if len(updated.Prompt.Prompts) != 2 {
		t.Fatalf("prompt count = %d, want 2", len(updated.Prompt.Prompts))
	}
	if updated.Prompt.Prompts[0].ID != "grammar" || updated.Prompt.Prompts[0].ShortLabel != "GRAM" {
		t.Fatalf("first prompt = %#v, want normalized grammar prompt", updated.Prompt.Prompts[0])
	}
	if updated.Prompt.Prompts[1].Instruction != "Translate naturally and explain advancement terms." {
		t.Fatalf("second prompt instruction = %q, want trimmed instruction", updated.Prompt.Prompts[1].Instruction)
	}
	if !updated.Prompt.UpdatedAt.Equal(store.timeProvider()) {
		t.Fatalf("UpdatedAt = %s, want store timestamp %s", updated.Prompt.UpdatedAt, store.timeProvider())
	}

	reloaded, err := NewStore(store.Info().RootDir)
	if err != nil {
		t.Fatalf("reload NewStore() error = %v", err)
	}
	got, err := reloaded.GetBook(book.ID)
	if err != nil {
		t.Fatalf("GetBook() error = %v", err)
	}
	if !reflect.DeepEqual(got.Prompt, updated.Prompt) {
		t.Fatalf("persisted Prompt = %#v, want %#v", got.Prompt, updated.Prompt)
	}
}

// TestUpdatePromptConfigCreatesDefaults verifies empty prompt settings provide
// usable Gemini actions instead of leaving the selection toolbar empty.
func TestUpdatePromptConfigCreatesDefaults(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}

	reset, err := store.UpdatePromptConfig(book.ID, PromptConfig{})
	if err != nil {
		t.Fatalf("UpdatePromptConfig() error = %v", err)
	}
	if len(reset.Prompt.Prompts) < 2 {
		t.Fatalf("default prompt count = %d, want at least 2", len(reset.Prompt.Prompts))
	}
	if !reset.Prompt.UpdatedAt.Equal(store.timeProvider()) {
		t.Fatalf("UpdatedAt = %s, want store timestamp %s", reset.Prompt.UpdatedAt, store.timeProvider())
	}
}

// TestNormalizePromptConfigMigratesLegacyCustomPrompt verifies existing library
// metadata with a single customPrompt becomes one selectable prompt action.
func TestNormalizePromptConfigMigratesLegacyCustomPrompt(t *testing.T) {
	prompt := normalizePromptConfig(PromptConfig{
		CustomPrompt: "  Explain sacred arts terms.  ",
	}, time.Date(2026, 6, 19, 1, 2, 3, 0, time.UTC))

	if len(prompt.Prompts) != 1 {
		t.Fatalf("prompt count = %d, want 1 migrated prompt", len(prompt.Prompts))
	}
	if prompt.Prompts[0].ID != "translate" {
		t.Fatalf("prompt ID = %q, want translate", prompt.Prompts[0].ID)
	}
	if prompt.Prompts[0].Instruction != "Explain sacred arts terms." {
		t.Fatalf("instruction = %q, want trimmed legacy prompt", prompt.Prompts[0].Instruction)
	}
}
