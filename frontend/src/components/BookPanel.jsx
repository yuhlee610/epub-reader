import {useEffect, useState} from 'react';
import {BookOpen, Trash2, X} from 'lucide-react';
import {
  coverTitle,
  fileNameLabel,
  formatImportedAt,
  progressLabel,
} from '../lib/bookFormatters';
import {DEFAULT_STUDY_INSTRUCTIONS} from '../lib/studyPrompt';

export function BookPanel({
  book,
  isRemoving,
  isSavingPrompt,
  onClose,
  onOpen,
  onRemove,
  onSavePrompt,
}) {
  const [promptValue, setPromptValue] = useState(DEFAULT_STUDY_INSTRUCTIONS);
  const savedPrompt = String(book?.prompt?.customPrompt || '').trim();

  useEffect(() => {
    setPromptValue(savedPrompt || DEFAULT_STUDY_INSTRUCTIONS);
  }, [book?.id, savedPrompt]);

  if (!book) {
    return null;
  }

  const title = book.title || 'Untitled book';
  const activePrompt = savedPrompt || DEFAULT_STUDY_INSTRUCTIONS;
  const hasCustomPrompt = savedPrompt.length > 0;
  const isPromptDirty = promptValue.trim() !== activePrompt.trim();
  const isPromptActionDisabled = isSavingPrompt || !onSavePrompt;

  async function handleSavePrompt() {
    try {
      await onSavePrompt?.(book, promptValue);
    } catch (_err) {
      // The library hook owns the visible error notification.
    }
  }

  async function handleResetPrompt() {
    try {
      await onSavePrompt?.(book, '');
      setPromptValue(DEFAULT_STUDY_INSTRUCTIONS);
    } catch (_err) {
      // The library hook owns the visible error notification.
    }
  }

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
      <section className="panel-prompt-section" aria-label="AI study prompt">
        <div className="panel-prompt-heading">
          <h2>AI study prompt</h2>
          <span>{hasCustomPrompt ? 'Custom' : 'Default'}</span>
        </div>
        <textarea
          aria-label={`AI study prompt for ${title}`}
          className="panel-prompt-input"
          value={promptValue}
          onChange={(event) => setPromptValue(event.target.value)}
          spellCheck="false"
        />
        <div className="panel-prompt-actions">
          <button
            className="panel-primary"
            type="button"
            onClick={handleSavePrompt}
            disabled={isPromptActionDisabled || !isPromptDirty}
          >
            Save prompt
          </button>
          <button
            className="panel-secondary"
            type="button"
            onClick={handleResetPrompt}
            disabled={isPromptActionDisabled || (!hasCustomPrompt && !isPromptDirty)}
          >
            Reset
          </button>
        </div>
      </section>
    </aside>
  );
}
