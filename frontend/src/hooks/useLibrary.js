import {useEffect, useMemo, useState} from 'react';
import {
  DeleteBookMetadata,
  GetBook,
  ImportEPUB,
  ListBooks,
  SaveBookPrompt,
} from '../../wailsjs/go/main/App';
import {formatImportError} from '../lib/bookFormatters';

const SUCCESS_MESSAGE_TIMEOUT_MS = 2800;
const ERROR_MESSAGE_TIMEOUT_MS = 6000;

// useLibrary owns the local-library workflow and keeps App focused on layout.
export function useLibrary() {
  const [books, setBooks] = useState([]);
  const [activeBook, setActiveBook] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isImporting, setIsImporting] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [isSavingPrompt, setIsSavingPrompt] = useState(false);
  const [query, setQuery] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  async function loadBooks() {
    setIsLoading(true);
    setError('');

    try {
      const nextBooks = await ListBooks();
      const safeBooks = nextBooks ?? [];

      setBooks(safeBooks);
      setActiveBook((current) => {
        if (!current) {
          return null;
        }

        return safeBooks.find((book) => book.id === current.id) ?? null;
      });
    } catch (err) {
      setError(err?.message ?? String(err));
    } finally {
      setIsLoading(false);
    }
  }

  async function importBook() {
    setIsImporting(true);
    clearNotice();

    try {
      const result = await ImportEPUB();

      if (result?.canceled) {
        setMessage('Import canceled.');
        return;
      }

      const title = result?.book?.title || 'Untitled book';
      const nextMessage = result?.duplicate
        ? `${title} is already in your library.`
        : `${title} imported.`;

      setMessage(nextMessage);
      await loadBooks();
    } catch (err) {
      setError(formatImportError(err));
    } finally {
      setIsImporting(false);
    }
  }

  async function openBook(book) {
    clearNotice();

    try {
      const freshBook = await GetBook(book.id);
      setActiveBook(freshBook);
    } catch (err) {
      setError(err?.message ?? String(err));
      await loadBooks();
    }
  }

  async function removeBook(book, event) {
    event?.stopPropagation();
    clearNotice();

    const title = book.title || 'Untitled book';
    const confirmed = window.confirm(
      `Remove "${title}" from your library? The EPUB file copy will stay in app storage.`,
    );

    if (!confirmed) {
      return;
    }

    setIsRemoving(true);

    try {
      await DeleteBookMetadata(book.id);
      setActiveBook((current) => (current?.id === book.id ? null : current));
      setMessage(`${title} removed from the library.`);
      await loadBooks();
    } catch (err) {
      setError(err?.message ?? String(err));
    } finally {
      setIsRemoving(false);
    }
  }

  async function saveBookPrompts(book, prompts) {
    if (!book?.id) {
      return;
    }

    setIsSavingPrompt(true);
    clearNotice();

    try {
      const updatedBook = await SaveBookPrompt(book.id, {
        prompts: normalizePromptPayload(prompts),
      });

      setBooks((currentBooks) => currentBooks.map((currentBook) => (
        currentBook.id === updatedBook.id ? updatedBook : currentBook
      )));
      setActiveBook((currentBook) => (
        currentBook?.id === updatedBook.id ? updatedBook : currentBook
      ));
      setMessage('Study prompts saved.');
    } catch (err) {
      setError(err?.message ?? String(err));
      throw err;
    } finally {
      setIsSavingPrompt(false);
    }
  }

  function clearNotice() {
    setMessage('');
    setError('');
  }

  useEffect(() => {
    loadBooks();
  }, []);

  useEffect(() => {
    if (!message && !error) {
      return undefined;
    }

    const timeoutID = window.setTimeout(
      clearNotice,
      error ? ERROR_MESSAGE_TIMEOUT_MS : SUCCESS_MESSAGE_TIMEOUT_MS,
    );

    return () => window.clearTimeout(timeoutID);
  }, [message, error]);

  const visibleBooks = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();

    return books.filter((book) => {
      const text = [
        book.title ?? '',
        book.author ?? '',
        book.originalFileName ?? '',
      ].join(' ').toLowerCase();

      return text.includes(normalizedQuery);
    });
  }, [books, query]);

  function normalizePromptPayload(prompts) {
    return (prompts ?? []).map((prompt, index) => ({
      id: String(prompt?.id || '').trim(),
      name: String(prompt?.name || '').trim(),
      shortLabel: String(prompt?.shortLabel || '').trim(),
      instruction: String(prompt?.instruction || '').trim(),
      sortOrder: Number.isFinite(Number(prompt?.sortOrder)) ? Number(prompt.sortOrder) : index,
      isDefault: Boolean(prompt?.isDefault),
    }));
  }

  return {
    activeBook,
    books,
    error,
    importBook,
    isImporting,
    isLoading,
    isRemoving,
    isSavingPrompt,
    message,
    openBook,
    query,
    removeBook,
    saveBookPrompt: saveBookPrompts,
    saveBookPrompts,
    loadBooks,
    setActiveBook,
    setQuery,
    visibleBooks,
  };
}
