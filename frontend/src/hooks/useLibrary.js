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

  async function saveBookPrompt(book, customPrompt) {
    if (!book?.id) {
      return;
    }

    setIsSavingPrompt(true);
    clearNotice();

    try {
      const trimmedPrompt = String(customPrompt || '').trim();
      const updatedBook = await SaveBookPrompt(book.id, {
        customPrompt: trimmedPrompt,
      });

      setBooks((currentBooks) => currentBooks.map((currentBook) => (
        currentBook.id === updatedBook.id ? updatedBook : currentBook
      )));
      setActiveBook((currentBook) => (
        currentBook?.id === updatedBook.id ? updatedBook : currentBook
      ));
      setMessage(trimmedPrompt ? 'Study prompt saved.' : 'Study prompt reset to default.');
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
    saveBookPrompt,
    loadBooks,
    setActiveBook,
    setQuery,
    visibleBooks,
  };
}
