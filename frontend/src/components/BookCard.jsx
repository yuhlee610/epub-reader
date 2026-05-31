import {Trash2} from 'lucide-react';
import {coverTitle, formatImportedAt, progressLabel} from '../lib/bookFormatters';

export function BookCard({book, isActive, isRemoving, onOpen, onRemove}) {
  const title = book.title || 'Untitled book';
  const className = isActive ? 'book-card is-active' : 'book-card';

  return (
    <li className={className}>
      <button
        aria-label={`Open ${title}`}
        className="book-button"
        onClick={() => onOpen(book)}
        type="button"
      >
        <span className="book-cover" aria-hidden="true">
          <span className="cover-name">{coverTitle(book.title)}</span>
          <span className="cover-author">{book.author || 'Unknown author'}</span>
        </span>
        <span className="book-title">{title}</span>
      </button>
      <button
        aria-label={`Remove ${title} from library`}
        className="book-remove-button"
        disabled={isRemoving}
        onClick={(event) => onRemove(book, event)}
        type="button"
      >
        <Trash2 aria-hidden="true" size={13} strokeWidth={2} />
      </button>
      <span className="book-details">
        <span>{progressLabel(book)}</span>
        <span
          aria-label={`Imported ${formatImportedAt(book.importedAt)}`}
          className="cloud-mark"
          title={formatImportedAt(book.importedAt)}
        />
      </span>
    </li>
  );
}
