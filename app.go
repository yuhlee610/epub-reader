package main

import (
	"context"
	"errors"
	"fmt"

	"epub-reader/internal/library"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App owns the Wails-facing application state.
type App struct {
	ctx          context.Context
	store        *library.Store
	storeErr     error
	openEPUBFile func(context.Context) (string, error)
}

// ImportEPUBResult describes the UI-facing result of an import attempt.
type ImportEPUBResult struct {
	Book      library.BookMetadata `json:"book"`
	Duplicate bool                 `json:"duplicate"`
	Canceled  bool                 `json:"canceled"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	store, err := library.NewStore("")
	if err != nil {
		return &App{storeErr: err, openEPUBFile: openEPUBFileDialog}
	}

	return &App{store: store, openEPUBFile: openEPUBFileDialog}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetStorageInfo returns the app-managed local storage paths.
func (a *App) GetStorageInfo() (library.StorageInfo, error) {
	store, err := a.libraryStore()
	if err != nil {
		return library.StorageInfo{}, err
	}

	return store.Info(), nil
}

// ImportEPUB opens a native picker and imports the selected EPUB into storage.
func (a *App) ImportEPUB() (ImportEPUBResult, error) {
	store, err := a.libraryStore()
	if err != nil {
		return ImportEPUBResult{}, err
	}
	if a.ctx == nil {
		return ImportEPUBResult{}, errors.New("application is not ready")
	}
	if a.openEPUBFile == nil {
		return ImportEPUBResult{}, errors.New("epub file picker is not configured")
	}

	selection, err := a.openEPUBFile(a.ctx)
	if err != nil {
		return ImportEPUBResult{}, fmt.Errorf("open epub picker: %w", err)
	}
	if selection == "" {
		return ImportEPUBResult{Canceled: true}, nil
	}

	result, err := store.ImportEPUBFromPath(selection)
	if err != nil {
		return ImportEPUBResult{}, err
	}

	return ImportEPUBResult{
		Book:      result.Book,
		Duplicate: result.Duplicate,
	}, nil
}

// openEPUBFileDialog uses the Wails runtime native picker for EPUB files.
func openEPUBFileDialog(ctx context.Context) (string, error) {
	// Wails v2 exposes native file dialogs through the Go runtime:
	// https://pkg.go.dev/github.com/wailsapp/wails/v2@v2.12.0/pkg/runtime#OpenFileDialog
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Import EPUB",
		Filters: []runtime.FileFilter{
			{DisplayName: "EPUB files (*.epub)", Pattern: "*.epub"},
		},
	})
}

// ManagedBookPath returns where an imported EPUB should be copied.
func (a *App) ManagedBookPath(bookID string, originalFileName string) (string, error) {
	store, err := a.libraryStore()
	if err != nil {
		return "", err
	}

	return store.ManagedBookPath(bookID, originalFileName)
}

// ListBooks returns all persisted book metadata.
func (a *App) ListBooks() ([]library.BookMetadata, error) {
	store, err := a.libraryStore()
	if err != nil {
		return nil, err
	}

	return store.ListBooks()
}

// GetBook returns one persisted book by ID.
func (a *App) GetBook(id string) (library.BookMetadata, error) {
	store, err := a.libraryStore()
	if err != nil {
		return library.BookMetadata{}, err
	}

	return store.GetBook(id)
}

// GetReaderBook returns imported EPUB content for the reader view.
func (a *App) GetReaderBook(id string) (library.ReaderBook, error) {
	store, err := a.libraryStore()
	if err != nil {
		return library.ReaderBook{}, err
	}

	return store.LoadReaderBook(id)
}

// SaveReadingProgress persists the reader's last opened chapter.
func (a *App) SaveReadingProgress(id string, progress library.ReadingProgress) (library.BookMetadata, error) {
	store, err := a.libraryStore()
	if err != nil {
		return library.BookMetadata{}, err
	}

	return store.UpdateReadingProgress(id, progress)
}

// SaveBookMetadata creates or updates one book metadata record.
func (a *App) SaveBookMetadata(book library.BookMetadata) (library.BookMetadata, error) {
	store, err := a.libraryStore()
	if err != nil {
		return library.BookMetadata{}, err
	}

	return store.SaveBook(book)
}

// DeleteBookMetadata removes a book metadata record without deleting the EPUB file.
func (a *App) DeleteBookMetadata(id string) error {
	store, err := a.libraryStore()
	if err != nil {
		return err
	}

	return store.DeleteBook(id)
}

// libraryStore returns the initialized library store or the startup error that
// prevented it from being created.
func (a *App) libraryStore() (*library.Store, error) {
	if a.storeErr != nil {
		return nil, fmt.Errorf("initialize library storage: %w", a.storeErr)
	}
	if a.store == nil {
		return nil, errors.New("library storage is not initialized")
	}

	return a.store, nil
}
