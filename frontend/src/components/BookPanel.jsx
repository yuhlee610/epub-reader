import {BookOpen, Trash2, X} from 'lucide-react';
import {
  coverTitle,
  fileNameLabel,
  formatImportedAt,
  progressLabel,
} from '../lib/bookFormatters';

export function BookPanel({book, isRemoving, onClose, onOpen, onRemove}) {
  if (!book) {
    return null;
  }

  const title = book.title || 'Untitled book';

  return (
    <aside className="book-panel" aria-label="Selected book">
      <button
        aria-label="Close selected book"
        className="panel-close"
        onClick={onClose}
        type="button"
      >
        <X aria-hidden="true" size={15} strokeWidth={2} />
      </button>
      <div className="panel-cover" aria-hidden="true">
        <span className="cover-name">{coverTitle(book.title)}</span>
        <span className="cover-author">{book.author || 'Unknown author'}</span>
      </div>
      <div className="panel-copy">
        <h1>{title}</h1>
        <p>{book.author || 'Unknown author'}</p>
        <dl>
          <div>
            <dt>Progress</dt>
            <dd>{progressLabel(book)}</dd>
          </div>
          <div>
            <dt>Imported</dt>
            <dd>{formatImportedAt(book.importedAt)}</dd>
          </div>
          <div>
            <dt>File</dt>
            <dd>{fileNameLabel(book)}</dd>
          </div>
        </dl>
      </div>
      <div className="panel-actions">
        <button className="panel-primary" type="button" onClick={() => onOpen(book)}>
          <BookOpen aria-hidden="true" size={15} strokeWidth={2} />
          Open
        </button>
        <button
          className="panel-danger"
          type="button"
          onClick={(event) => onRemove(book, event)}
          disabled={isRemoving}
        >
          <Trash2 aria-hidden="true" size={15} strokeWidth={2} />
          Remove
        </button>
      </div>
    </aside>
  );
}
