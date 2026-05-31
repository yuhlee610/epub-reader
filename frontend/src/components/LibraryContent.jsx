import {Plus} from 'lucide-react';
import {BookCard} from './BookCard';
import {BookPanel} from './BookPanel';

export function LibraryContent({
  activeBook,
  books,
  isImporting,
  isLoading,
  isRemoving,
  onCloseBook,
  onImport,
  onOpenBook,
  onRemoveBook,
  visibleBooks,
}) {
  if (isLoading) {
    return (
      <section className="library-content" aria-label="Imported books">
        <div className="loading-row" role="status">
          Loading library...
        </div>
      </section>
    );
  }

  return (
    <section className="library-content" aria-label="Imported books">
      <div className={activeBook ? 'library-layout has-active-book' : 'library-layout'}>
        <div className="book-grid">
          {books.length === 0 ? (
            <EmptyLibrary isImporting={isImporting} onImport={onImport} />
          ) : (
            <>
              <ul className="book-list" aria-label="Books">
                {visibleBooks.map((book) => (
                  <BookCard
                    book={book}
                    isActive={activeBook?.id === book.id}
                    isRemoving={isRemoving}
                    key={book.id}
                    onOpen={onOpenBook}
                    onRemove={onRemoveBook}
                  />
                ))}
              </ul>

              <AddBookTile isImporting={isImporting} onImport={onImport} />
            </>
          )}

          {books.length > 0 && visibleBooks.length === 0 && (
            <div className="empty-search" role="status">
              No books match this search.
            </div>
          )}
        </div>

        <BookPanel
          book={activeBook}
          isRemoving={isRemoving}
          onClose={onCloseBook}
          onOpen={onOpenBook}
          onRemove={onRemoveBook}
        />
      </div>
    </section>
  );
}

function EmptyLibrary({isImporting, onImport}) {
  return (
    <div className="empty-library" role="status">
      <AddBookTile isImporting={isImporting} onImport={onImport} />
      <span>No books yet</span>
    </div>
  );
}

function AddBookTile({isImporting, onImport}) {
  return (
    <button
      className="add-book-tile"
      type="button"
      onClick={onImport}
      disabled={isImporting}
      aria-label="Import EPUB"
    >
      <Plus aria-hidden="true" size={46} strokeWidth={1.25} />
    </button>
  );
}
