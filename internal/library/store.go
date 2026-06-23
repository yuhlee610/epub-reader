package library

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	defaultReaderBackgroundColor = "#ffffff"
	defaultReaderFontSize        = 18
	minReaderFontSize            = 16
	maxReaderFontSize            = 24
)

var allowedReaderBackgroundColors = map[string]struct{}{
	"#ffffff": {},
	"#f8f1e7": {},
	"#efe4ce": {},
	"#eef4ee": {},
	"#f1f1f1": {},
}

var promptIDCleanup = regexp.MustCompile(`[^a-z0-9-]+`)
var noteIDCleanup = regexp.MustCompile(`[^a-z0-9-]+`)

// Store persists local book metadata for the desktop app.
type Store struct {
	mu           sync.Mutex
	rootDir      string
	booksDir     string
	libraryPath  string
	idGenerator  func() (string, error)
	timeProvider func() time.Time
}

// NewStore creates a Store rooted at rootDir. When rootDir is empty, the
// platform user config directory is used.
func NewStore(rootDir string) (*Store, error) {
	if rootDir == "" {
		var err error
		rootDir, err = DefaultRootDir()
		if err != nil {
			return nil, err
		}
	}

	rootDir = filepath.Clean(rootDir)
	store := &Store{
		rootDir:      rootDir,
		booksDir:     filepath.Join(rootDir, booksDirName),
		libraryPath:  filepath.Join(rootDir, libraryFileName),
		idGenerator:  newBookID,
		timeProvider: func() time.Time { return time.Now().UTC() },
	}

	if err := store.ensureLayout(); err != nil {
		return nil, err
	}

	return store, nil
}

// Info returns the managed storage paths used by the store.
func (s *Store) Info() StorageInfo {
	return StorageInfo{
		RootDir:     s.rootDir,
		BooksDir:    s.booksDir,
		LibraryPath: s.libraryPath,
	}
}

// ListBooks returns all persisted book metadata sorted by import date.
func (s *Store) ListBooks() ([]BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return nil, err
	}

	// Clone before sorting so callers never observe mutations to the persisted
	// in-memory slice returned from readLibrary.
	books := slices.Clone(file.Books)
	slices.SortFunc(books, func(a, b BookMetadata) int {
		return a.ImportedAt.Compare(b.ImportedAt)
	})

	return books, nil
}

// GetBook returns one persisted book by ID.
func (s *Store) GetBook(id string) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	for _, book := range file.Books {
		if book.ID == id {
			return book, nil
		}
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// SaveBook creates or updates one book metadata record.
func (s *Store) SaveBook(book BookMetadata) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	// Locate an existing record before normalization so updates can preserve
	// fields that callers commonly omit, such as importedAt or filePath.
	var existing *BookMetadata
	for i := range file.Books {
		if file.Books[i].ID == strings.TrimSpace(book.ID) {
			existing = &file.Books[i]
			break
		}
	}

	prepared, err := s.prepareBook(book, existing)
	if err != nil {
		return BookMetadata{}, err
	}

	// Replace by stable ID if the book already exists; otherwise append a new
	// record while preserving the existing file order.
	replaced := false
	for i := range file.Books {
		if file.Books[i].ID == prepared.ID {
			file.Books[i] = prepared
			replaced = true
			break
		}
	}
	if !replaced {
		file.Books = append(file.Books, prepared)
	}

	if err := s.writeLibrary(file); err != nil {
		return BookMetadata{}, err
	}

	return prepared, nil
}

// DeleteBook removes a book metadata record without deleting the managed EPUB.
func (s *Store) DeleteBook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return err
	}

	// Reuse the backing array while filtering the deleted record. This keeps the
	// operation allocation-light without exposing the internal slice to callers.
	next := file.Books[:0]
	deleted := false
	for _, book := range file.Books {
		if book.ID == id {
			deleted = true
			continue
		}
		next = append(next, book)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrBookNotFound, id)
	}

	file.Books = next
	return s.writeLibrary(file)
}

// UpdateReadingProgress stores the user's last known chapter location.
func (s *Store) UpdateReadingProgress(id string, progress ReadingProgress) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	for i := range file.Books {
		if file.Books[i].ID != strings.TrimSpace(id) {
			continue
		}

		progress.ChapterHref = strings.TrimSpace(progress.ChapterHref)
		progress.Location = strings.TrimSpace(progress.Location)
		if progress.ChapterIndex < 0 {
			progress.ChapterIndex = 0
		}
		progress.UpdatedAt = s.timeProvider()

		file.Books[i].Progress = progress
		if err := s.writeLibrary(file); err != nil {
			return BookMetadata{}, err
		}

		return file.Books[i], nil
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// UpdateReaderAppearance stores reading comfort preferences for one book.
func (s *Store) UpdateReaderAppearance(id string, appearance ReaderAppearance) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	for i := range file.Books {
		if file.Books[i].ID != strings.TrimSpace(id) {
			continue
		}

		file.Books[i].Appearance = normalizeReaderAppearance(appearance)
		if err := s.writeLibrary(file); err != nil {
			return BookMetadata{}, err
		}

		return file.Books[i], nil
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// UpdatePromptConfig stores translation and study prompt preferences for one
// book. Empty prompt lists reset the book to the default study prompt.
func (s *Store) UpdatePromptConfig(id string, prompt PromptConfig) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	for i := range file.Books {
		if file.Books[i].ID != strings.TrimSpace(id) {
			continue
		}

		file.Books[i].Prompt = normalizePromptConfig(prompt, s.timeProvider())
		if err := s.writeLibrary(file); err != nil {
			return BookMetadata{}, err
		}

		return file.Books[i], nil
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// SaveReaderNote creates or updates one note/highlight for a book.
func (s *Store) SaveReaderNote(id string, note ReaderNote) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	for i := range file.Books {
		if file.Books[i].ID != strings.TrimSpace(id) {
			continue
		}

		prepared := normalizeReaderNote(note, s.timeProvider())
		replaced := false
		for noteIndex := range file.Books[i].Notes {
			if file.Books[i].Notes[noteIndex].ID == prepared.ID {
				if note.CreatedAt.IsZero() {
					prepared.CreatedAt = file.Books[i].Notes[noteIndex].CreatedAt
				}
				file.Books[i].Notes[noteIndex] = prepared
				replaced = true
				break
			}
		}
		if !replaced {
			file.Books[i].Notes = append(file.Books[i].Notes, prepared)
		}
		file.Books[i].Notes = normalizeReaderNotes(file.Books[i].Notes)

		if err := s.writeLibrary(file); err != nil {
			return BookMetadata{}, err
		}

		return file.Books[i], nil
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// DeleteReaderNote removes one note/highlight from a book.
func (s *Store) DeleteReaderNote(id string, noteID string) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	noteID = strings.TrimSpace(noteID)
	for i := range file.Books {
		if file.Books[i].ID != strings.TrimSpace(id) {
			continue
		}

		next := file.Books[i].Notes[:0]
		for _, note := range file.Books[i].Notes {
			if note.ID == noteID {
				continue
			}
			next = append(next, note)
		}
		file.Books[i].Notes = normalizeReaderNotes(next)

		if err := s.writeLibrary(file); err != nil {
			return BookMetadata{}, err
		}

		return file.Books[i], nil
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// prepareBook normalizes and validates metadata before it is written to disk.
func (s *Store) prepareBook(book BookMetadata, existing *BookMetadata) (BookMetadata, error) {
	var err error
	book.ID, err = cleanBookID(book.ID)
	if err != nil {
		// Empty IDs are allowed on create, but malformed explicit IDs are
		// rejected because they become directory names under managed storage.
		if book.ID != "" {
			return BookMetadata{}, err
		}
		id, idErr := s.idGenerator()
		if idErr != nil {
			return BookMetadata{}, fmt.Errorf("generate book id: %w", idErr)
		}
		book.ID, err = cleanBookID(id)
		if err != nil {
			return BookMetadata{}, err
		}
	}
	book.Title = strings.TrimSpace(book.Title)
	book.Author = strings.TrimSpace(book.Author)
	book.OriginalFileName = strings.TrimSpace(book.OriginalFileName)
	book.FilePath = filepath.Clean(strings.TrimSpace(book.FilePath))

	if book.Title == "" {
		return BookMetadata{}, fmt.Errorf("%w: title is required", ErrInvalidBook)
	}
	// Updates may intentionally only send changed fields. Preserve immutable
	// import details when the caller omits them.
	if existing != nil && book.OriginalFileName == "" {
		book.OriginalFileName = existing.OriginalFileName
	}
	if existing != nil && (book.FilePath == "" || book.FilePath == ".") {
		book.FilePath = existing.FilePath
	}
	if book.OriginalFileName == "" {
		book.OriginalFileName = filepath.Base(book.FilePath)
	}
	if book.FilePath == "" || book.FilePath == "." {
		path, err := s.ManagedBookPath(book.ID, book.OriginalFileName)
		if err != nil {
			return BookMetadata{}, err
		}
		book.FilePath = path
	}
	if !strings.EqualFold(filepath.Ext(book.FilePath), ".epub") {
		return BookMetadata{}, fmt.Errorf("%w: managed file path must use .epub extension", ErrInvalidBook)
	}
	// Metadata must point at this book's own managed directory, not merely
	// somewhere under the global books folder.
	if err := s.requireManagedBookPath(book.ID, book.FilePath); err != nil {
		return BookMetadata{}, err
	}
	if existing != nil && book.ImportedAt.IsZero() {
		book.ImportedAt = existing.ImportedAt
	}
	if book.ImportedAt.IsZero() {
		book.ImportedAt = s.timeProvider()
	}
	if existing != nil && isZeroReaderAppearance(book.Appearance) {
		book.Appearance = existing.Appearance
	}
	book.Appearance = normalizeReaderAppearance(book.Appearance)
	if existing != nil && isZeroPromptConfig(book.Prompt) {
		book.Prompt = existing.Prompt
	}
	promptUpdatedAt := book.Prompt.UpdatedAt
	if promptUpdatedAt.IsZero() {
		promptUpdatedAt = s.timeProvider()
	}
	book.Prompt = normalizePromptConfig(book.Prompt, promptUpdatedAt)
	if existing != nil && len(book.Notes) == 0 {
		book.Notes = existing.Notes
	}
	book.Notes = normalizeReaderNotes(book.Notes)

	return book, nil
}

func normalizeReaderNote(note ReaderNote, now time.Time) ReaderNote {
	note.ID = cleanNoteID(note.ID)
	if note.ID == "" {
		note.ID = fmt.Sprintf("note-%d", now.UnixNano())
	}
	note.Text = strings.TrimSpace(note.Text)
	note.NoteText = strings.TrimSpace(note.NoteText)
	note.ChapterHref = strings.TrimSpace(note.ChapterHref)
	note.ChapterTitle = strings.TrimSpace(note.ChapterTitle)
	note.Location = strings.TrimSpace(note.Location)
	note.Color = strings.TrimSpace(note.Color)
	if note.ChapterIndex < 0 {
		note.ChapterIndex = 0
	}
	if note.CreatedAt.IsZero() {
		note.CreatedAt = now
	}
	note.UpdatedAt = now

	return note
}

func normalizeReaderNotes(notes []ReaderNote) []ReaderNote {
	normalized := make([]ReaderNote, 0, len(notes))
	seen := map[string]struct{}{}
	for _, note := range notes {
		note = normalizeReaderNote(note, note.CreatedAt)
		if note.Text == "" || note.ChapterHref == "" || note.ID == "" {
			continue
		}
		if _, exists := seen[note.ID]; exists {
			continue
		}
		seen[note.ID] = struct{}{}
		normalized = append(normalized, note)
	}

	slices.SortFunc(normalized, func(a, b ReaderNote) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})

	return normalized
}

func normalizePromptConfig(prompt PromptConfig, updatedAt time.Time) PromptConfig {
	prompts := normalizeStudyPrompts(prompt.Prompts)
	customPrompt := strings.TrimSpace(prompt.CustomPrompt)
	if len(prompts) == 0 && customPrompt != "" {
		prompts = []StudyPrompt{{
			ID:          "translate",
			Name:        "Translate",
			ShortLabel:  "TR",
			Instruction: customPrompt,
			SortOrder:   0,
			IsDefault:   true,
		}}
	}
	if len(prompts) == 0 {
		prompts = defaultStudyPrompts()
	}

	return PromptConfig{
		Prompts:   prompts,
		UpdatedAt: updatedAt,
	}
}

func normalizeReaderAppearance(appearance ReaderAppearance) ReaderAppearance {
	backgroundColor := strings.ToLower(strings.TrimSpace(appearance.BackgroundColor))
	if _, ok := allowedReaderBackgroundColors[backgroundColor]; !ok {
		backgroundColor = defaultReaderBackgroundColor
	}

	fontSize := appearance.FontSize
	if fontSize == 0 {
		fontSize = defaultReaderFontSize
	}
	if fontSize < minReaderFontSize {
		fontSize = minReaderFontSize
	}
	if fontSize > maxReaderFontSize {
		fontSize = maxReaderFontSize
	}

	return ReaderAppearance{
		BackgroundColor: backgroundColor,
		FontSize:        fontSize,
	}
}

func isZeroReaderAppearance(appearance ReaderAppearance) bool {
	return strings.TrimSpace(appearance.BackgroundColor) == "" && appearance.FontSize == 0
}

func isZeroPromptConfig(prompt PromptConfig) bool {
	return strings.TrimSpace(prompt.CustomPrompt) == "" && len(prompt.Prompts) == 0 && prompt.UpdatedAt.IsZero()
}

func normalizeStudyPrompts(prompts []StudyPrompt) []StudyPrompt {
	normalized := make([]StudyPrompt, 0, len(prompts))
	seenIDs := map[string]struct{}{}
	for _, prompt := range prompts {
		instruction := strings.TrimSpace(prompt.Instruction)
		if instruction == "" {
			continue
		}

		name := strings.TrimSpace(prompt.Name)
		if name == "" {
			name = "Gemini prompt"
		}
		shortLabel := strings.ToUpper(strings.TrimSpace(prompt.ShortLabel))
		if shortLabel == "" {
			shortLabel = promptShortLabel(name)
		}
		if len(shortLabel) > 4 {
			shortLabel = shortLabel[:4]
		}

		id := cleanPromptID(prompt.ID)
		if id == "" {
			id = cleanPromptID(name)
		}
		if id == "" {
			id = "prompt"
		}
		baseID := id
		for suffix := 2; ; suffix++ {
			if _, exists := seenIDs[id]; !exists {
				break
			}
			id = fmt.Sprintf("%s-%d", baseID, suffix)
		}
		seenIDs[id] = struct{}{}

		normalized = append(normalized, StudyPrompt{
			ID:          id,
			Name:        name,
			ShortLabel:  shortLabel,
			Instruction: instruction,
			SortOrder:   prompt.SortOrder,
			IsDefault:   prompt.IsDefault,
		})
	}

	slices.SortFunc(normalized, func(a, b StudyPrompt) int {
		if a.SortOrder != b.SortOrder {
			return a.SortOrder - b.SortOrder
		}
		return strings.Compare(a.ID, b.ID)
	})
	for i := range normalized {
		normalized[i].SortOrder = i
	}

	return normalized
}

func defaultStudyPrompts() []StudyPrompt {
	return []StudyPrompt{
		{
			ID:         "translate",
			Name:       "Translate",
			ShortLabel: "TR",
			Instruction: strings.Join([]string{
				"Translate naturally into Vietnamese with the book context in mind.",
				"List important vocabulary from the selected text with short meanings.",
				"Explain useful grammar patterns briefly.",
				"Add any nuance needed to understand the passage.",
			}, "\n"),
			SortOrder: 0,
			IsDefault: true,
		},
		{
			ID:         "grammar",
			Name:       "Grammar",
			ShortLabel: "GR",
			Instruction: strings.Join([]string{
				"Focus on useful grammar patterns in the selected text.",
				"Explain sentence structure and word usage concisely.",
				"Include a short natural translation only when it helps the explanation.",
			}, "\n"),
			SortOrder: 1,
			IsDefault: true,
		},
	}
}

func cleanPromptID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = promptIDCleanup.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 48 {
		value = strings.Trim(value[:48], "-")
	}
	return value
}

func cleanNoteID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = noteIDCleanup.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 64 {
		value = strings.Trim(value[:64], "-")
	}
	return value
}

func promptShortLabel(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return "AI"
	}
	if len(words) == 1 {
		word := strings.ToUpper(words[0])
		if len(word) <= 2 {
			return word
		}
		return word[:2]
	}

	var label strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		label.WriteByte(strings.ToUpper(word)[0])
		if label.Len() == 3 {
			break
		}
	}
	if label.Len() == 0 {
		return "AI"
	}
	return label.String()
}
