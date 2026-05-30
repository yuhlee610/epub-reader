package main

import (
	"context"
	"errors"
	"fmt"

	"epub-reader/internal/library"
)

// App owns the Wails-facing application state.
type App struct {
	ctx      context.Context
	store    *library.Store
	storeErr error
}

// NewApp creates a new App application struct
func NewApp() *App {
	store, err := library.NewStore("")
	if err != nil {
		return &App{storeErr: err}
	}

	return &App{store: store}
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
