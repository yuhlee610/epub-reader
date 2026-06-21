import {useEffect, useState} from 'react';
import {ArrowDown, ArrowUp, BookOpen, Plus, Trash2, X} from 'lucide-react';
import {
  coverTitle,
  fileNameLabel,
  formatImportedAt,
  progressLabel,
} from '../lib/bookFormatters';
import {DEFAULT_STUDY_INSTRUCTIONS} from '../lib/studyPrompt';

const EMPTY_PROMPT = {
  id: '',
  name: 'New prompt',
  shortLabel: 'AI',
  instruction: '',
  sortOrder: 0,
  isDefault: false,
};

export function BookPanel({
  book,
  isRemoving,
  isSavingPrompt,
  onClose,
  onOpen,
  onRemove,
  onSavePrompt,
}) {
  const [prompts, setPrompts] = useState([]);
  const [activePromptID, setActivePromptID] = useState('');
  const savedPrompts = normalizeBookPrompts(book);

  useEffect(() => {
    setPrompts(savedPrompts);
    setActivePromptID((currentID) => (
      savedPrompts.some((prompt) => prompt.id === currentID)
        ? currentID
        : savedPrompts[0]?.id || ''
    ));
  }, [book?.id, JSON.stringify(savedPrompts)]);

  if (!book) {
    return null;
  }

  const title = book.title || 'Untitled book';
  const activePrompt = prompts.find((prompt) => prompt.id === activePromptID) ?? prompts[0] ?? EMPTY_PROMPT;
  const isPromptDirty = JSON.stringify(normalizePromptList(prompts)) !== JSON.stringify(normalizePromptList(savedPrompts));
  const isPromptActionDisabled = isSavingPrompt || !onSavePrompt;

  async function handleSavePrompt() {
    try {
      await onSavePrompt?.(book, normalizePromptList(prompts));
    } catch (_err) {
      // The library hook owns the visible error notification.
    }
  }

  function handleResetPrompt() {
    const nextPrompts = defaultPromptList();
    setPrompts(nextPrompts);
    setActivePromptID(nextPrompts[0]?.id || '');
  }

  function handleAddPrompt() {
    const nextPrompt = {
      ...EMPTY_PROMPT,
      id: nextPromptID(prompts),
      name: 'New prompt',
      shortLabel: 'AI',
      instruction: DEFAULT_STUDY_INSTRUCTIONS,
      sortOrder: prompts.length,
    };
    setPrompts([...prompts, nextPrompt]);
    setActivePromptID(nextPrompt.id);
  }

  function handleDeletePrompt(promptID) {
    if (prompts.length <= 1) {
      return;
    }
    const nextPrompts = prompts.filter((prompt) => prompt.id !== promptID);
    setPrompts(normalizePromptList(nextPrompts));
    if (activePromptID === promptID) {
      setActivePromptID(nextPrompts[0]?.id || '');
    }
  }

  function handleMovePrompt(promptID, direction) {
    const index = prompts.findIndex((prompt) => prompt.id === promptID);
    const nextIndex = index + direction;
    if (index < 0 || nextIndex < 0 || nextIndex >= prompts.length) {
      return;
    }
    const nextPrompts = [...prompts];
    const [movedPrompt] = nextPrompts.splice(index, 1);
    nextPrompts.splice(nextIndex, 0, movedPrompt);
    setPrompts(normalizePromptList(nextPrompts));
  }

  function updateActivePrompt(updates) {
    setPrompts((currentPrompts) => currentPrompts.map((prompt) => (
      prompt.id === activePrompt.id
        ? {...prompt, ...updates}
        : prompt
    )));
  }

  function handlePromptIDChange(value) {
    const nextID = sanitizePromptID(value);
    if (!nextID || prompts.some((prompt) => prompt.id === nextID && prompt.id !== activePrompt.id)) {
      return;
    }
    setPrompts((currentPrompts) => currentPrompts.map((prompt) => (
      prompt.id === activePrompt.id
        ? {...prompt, id: nextID}
        : prompt
    )));
    setActivePromptID(nextID);
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
      <section className="panel-prompt-section" aria-label="Book prompt manager">
        <div className="panel-prompt-heading">
          <h2>Prompt manager</h2>
          <span>{prompts.length} prompts</span>
        </div>
        <div className="panel-prompt-list" role="list" aria-label={`Prompts for ${title}`}>
          {prompts.map((prompt, index) => (
            <div className={prompt.id === activePrompt.id ? 'panel-prompt-row is-active' : 'panel-prompt-row'} key={prompt.id} role="listitem">
              <button
                className="panel-prompt-selector"
                onClick={() => setActivePromptID(prompt.id)}
                type="button"
              >
                <span>{prompt.shortLabel || 'AI'}</span>
                <strong>{prompt.name || 'Untitled prompt'}</strong>
              </button>
              <button
                aria-label={`Move ${prompt.name || 'prompt'} up`}
                className="panel-prompt-icon"
                disabled={index === 0}
                onClick={() => handleMovePrompt(prompt.id, -1)}
                type="button"
              >
                <ArrowUp aria-hidden="true" size={13} strokeWidth={2} />
              </button>
              <button
                aria-label={`Move ${prompt.name || 'prompt'} down`}
                className="panel-prompt-icon"
                disabled={index === prompts.length - 1}
                onClick={() => handleMovePrompt(prompt.id, 1)}
                type="button"
              >
                <ArrowDown aria-hidden="true" size={13} strokeWidth={2} />
              </button>
              <button
                aria-label={`Delete ${prompt.name || 'prompt'}`}
                className="panel-prompt-icon is-danger"
                disabled={prompts.length <= 1}
                onClick={() => handleDeletePrompt(prompt.id)}
                type="button"
              >
                <Trash2 aria-hidden="true" size={13} strokeWidth={2} />
              </button>
            </div>
          ))}
        </div>
        <div className="panel-prompt-form">
          <label>
            Name
            <input
              className="panel-prompt-field"
              onChange={(event) => updateActivePrompt({name: event.target.value})}
              value={activePrompt.name}
            />
          </label>
          <label>
            Toolbar label
            <input
              className="panel-prompt-field"
              maxLength={4}
              onChange={(event) => updateActivePrompt({shortLabel: event.target.value.toUpperCase()})}
              value={activePrompt.shortLabel}
            />
          </label>
          <label>
            Prompt ID
            <input
              className="panel-prompt-field"
              onChange={(event) => handlePromptIDChange(event.target.value)}
              value={activePrompt.id}
            />
          </label>
          <label>
            Gemini instructions
            <textarea
              aria-label={`Gemini instructions for ${activePrompt.name || title}`}
              className="panel-prompt-input"
              value={activePrompt.instruction}
              onChange={(event) => updateActivePrompt({instruction: event.target.value})}
              spellCheck="false"
            />
          </label>
        </div>
        <div className="panel-prompt-actions">
          <button
            className="panel-secondary"
            type="button"
            onClick={handleAddPrompt}
            disabled={isPromptActionDisabled}
          >
            <Plus aria-hidden="true" size={14} strokeWidth={2} />
            Add prompt
          </button>
          <button
            className="panel-secondary"
            type="button"
            onClick={handleResetPrompt}
            disabled={isPromptActionDisabled}
          >
            Restore defaults
          </button>
          <button
            className="panel-primary"
            type="button"
            onClick={handleSavePrompt}
            disabled={isPromptActionDisabled || !isPromptDirty}
          >
            Save prompts
          </button>
        </div>
      </section>
    </aside>
  );
}

function normalizeBookPrompts(book) {
  const prompts = normalizePromptList(book?.prompt?.prompts);
  if (prompts.length > 0) {
    return prompts;
  }

  const legacyPrompt = String(book?.prompt?.customPrompt || '').trim();
  if (legacyPrompt) {
    return normalizePromptList([{
      id: 'translate',
      name: 'Translate',
      shortLabel: 'TR',
      instruction: legacyPrompt,
      sortOrder: 0,
      isDefault: true,
    }]);
  }

  return defaultPromptList();
}

function defaultPromptList() {
  return normalizePromptList([
    {
      id: 'translate',
      name: 'Translate',
      shortLabel: 'TR',
      instruction: DEFAULT_STUDY_INSTRUCTIONS,
      sortOrder: 0,
      isDefault: true,
    },
    {
      id: 'grammar',
      name: 'Grammar',
      shortLabel: 'GR',
      instruction: [
        'Focus on useful grammar patterns in the selected text.',
        'Explain sentence structure and word usage concisely.',
        'Include a short natural translation only when it helps the explanation.',
      ].join('\n'),
      sortOrder: 1,
      isDefault: true,
    },
  ]);
}

function normalizePromptList(prompts) {
  return (prompts ?? [])
    .map((prompt, index) => ({
      id: sanitizePromptID(prompt?.id) || `prompt-${index + 1}`,
      name: String(prompt?.name || 'Gemini prompt').trim(),
      shortLabel: String(prompt?.shortLabel || 'AI').trim().toUpperCase().slice(0, 4),
      instruction: String(prompt?.instruction || '').trim(),
      sortOrder: index,
      isDefault: Boolean(prompt?.isDefault),
    }))
    .filter((prompt) => prompt.instruction);
}

function nextPromptID(prompts) {
  let index = prompts.length + 1;
  let id = `prompt-${index}`;
  const existingIDs = new Set(prompts.map((prompt) => prompt.id));
  while (existingIDs.has(id)) {
    index += 1;
    id = `prompt-${index}`;
  }
  return id;
}

function sanitizePromptID(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/_/g, '-')
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 48);
}
