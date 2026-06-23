import {useEffect, useRef, useState} from 'react';
import {
  DeleteReaderBookmark,
  DeleteReaderNote,
  GetReaderBook,
  SaveReaderAppearance,
  SaveReaderBookmark,
  SaveReaderNote,
  SaveReadingProgress,
} from '../../wailsjs/go/main/App';

const READER_ERROR_TIMEOUT_MS = 6000;
const LAST_PAGE_INDEX = Number.MAX_SAFE_INTEGER;

// useReader owns EPUB loading, chapter navigation, and progress persistence.
export function useReader() {
  const [readerBook, setReaderBook] = useState(null);
  const [currentChapterIndex, setCurrentChapterIndex] = useState(0);
  const [currentPageIndex, setCurrentPageIndex] = useState(0);
  const [isReaderLoading, setIsReaderLoading] = useState(false);
  const [readerError, setReaderError] = useState('');
  const readerBookRef = useRef(null);
  const currentChapterIndexRef = useRef(0);
  const saveQueueRef = useRef(Promise.resolve());

  async function openReader(book) {
    setIsReaderLoading(true);
    setReaderError('');

    try {
      const nextReaderBook = await GetReaderBook(book.id);
      const nextChapterIndex = clampChapterIndex(
        nextReaderBook.currentChapterIndex ?? 0,
        nextReaderBook.chapters?.length ?? 0,
      );

      readerBookRef.current = nextReaderBook;
      currentChapterIndexRef.current = nextChapterIndex;
      setReaderBook(nextReaderBook);
      setCurrentChapterIndex(nextChapterIndex);
      setCurrentPageIndex(progressPageIndex(nextReaderBook, nextChapterIndex));
    } catch (err) {
      setReaderError(readerErrorMessage(err));
    } finally {
      setIsReaderLoading(false);
    }
  }

  async function closeReader() {
    await saveQueueRef.current.catch(() => undefined);

    setReaderBook(null);
    readerBookRef.current = null;
    currentChapterIndexRef.current = 0;
    setCurrentChapterIndex(0);
    setCurrentPageIndex(0);
    setReaderError('');
  }

  async function goToChapter(index, pageIndex = 0, percent = null, options = {}) {
    const currentReaderBook = readerBookRef.current;
    if (!currentReaderBook) {
      return;
    }

    const nextIndex = clampChapterIndex(index, currentReaderBook.chapters?.length ?? 0);
    const nextPageIndex = Math.max(0, pageIndex);

    currentChapterIndexRef.current = nextIndex;
    setCurrentChapterIndex(nextIndex);
    setCurrentPageIndex(nextPageIndex);

    if (!options.deferProgress) {
      await persistProgress(currentReaderBook, nextIndex, nextPageIndex, percent);
    }
  }

  async function saveCurrentPage(pageIndex, percent = null, source = {}) {
    const currentReaderBook = readerBookRef.current;
    if (!currentReaderBook) {
      return;
    }

    const sourceChapterIndex = Number.isInteger(source.chapterIndex)
      ? clampChapterIndex(source.chapterIndex, currentReaderBook.chapters?.length ?? 0)
      : currentChapterIndexRef.current;
    const sourceChapter = currentReaderBook.chapters?.[sourceChapterIndex];
    if (
      !sourceChapter ||
      sourceChapterIndex !== currentChapterIndexRef.current ||
      (source.chapterHref && source.chapterHref !== sourceChapter.href)
    ) {
      return;
    }

    const nextPageIndex = Math.max(0, pageIndex);

    await persistProgress(
      currentReaderBook,
      sourceChapterIndex,
      nextPageIndex,
      percent,
    );
  }

  async function persistProgress(currentReaderBook, chapterIndex, pageIndex, percent) {
    if (!currentReaderBook?.book?.id) {
      return;
    }

    const chapter = currentReaderBook.chapters?.[chapterIndex];
    if (!chapter) {
      return;
    }

    const bookID = currentReaderBook.book.id;
    const chapterHref = chapter.href;
    const location = pageLocation(pageIndex, percent);

    saveQueueRef.current = saveQueueRef.current.catch(() => undefined).then(async () => {
      const nextBook = await SaveReadingProgress(bookID, {
        chapterHref,
        chapterIndex,
        location,
      });
      setReaderBook((current) => {
        if (!current || current.book.id !== bookID) {
          return current;
        }

        const nextReaderBook = {
          ...current,
          book: nextBook,
        };
        readerBookRef.current = nextReaderBook;
        return nextReaderBook;
      });
    });

    try {
      await saveQueueRef.current;
    } catch (err) {
      setReaderError(readerErrorMessage(err));
    }
  }

  async function saveReaderAppearance(appearance) {
    const currentReaderBook = readerBookRef.current;
    if (!currentReaderBook?.book?.id) {
      return;
    }

    const bookID = currentReaderBook.book.id;
    const nextAppearance = {
      backgroundColor: appearance?.backgroundColor,
      fontSize: appearance?.fontSize,
    };

    setReaderBook((current) => {
      if (!current || current.book.id !== bookID) {
        return current;
      }

      const nextReaderBook = {
        ...current,
        book: {
          ...current.book,
          appearance: nextAppearance,
        },
      };
      readerBookRef.current = nextReaderBook;
      return nextReaderBook;
    });

    saveQueueRef.current = saveQueueRef.current.catch(() => undefined).then(async () => {
      const nextBook = await SaveReaderAppearance(bookID, nextAppearance);
      setReaderBook((current) => {
        if (!current || current.book.id !== bookID) {
          return current;
        }

        const nextReaderBook = {
          ...current,
          book: nextBook,
        };
        readerBookRef.current = nextReaderBook;
        return nextReaderBook;
      });
    });

    try {
      await saveQueueRef.current;
    } catch (err) {
      setReaderError(readerErrorMessage(err));
    }
  }

  async function saveReaderNote(note) {
    const currentReaderBook = readerBookRef.current;
    if (!currentReaderBook?.book?.id) {
      return null;
    }

    const bookID = currentReaderBook.book.id;
    try {
      const nextBook = await SaveReaderNote(bookID, note);
      updateReaderBook(nextBook);
      return nextBook;
    } catch (err) {
      setReaderError(readerErrorMessage(err));
      return null;
    }
  }

  async function deleteReaderNote(noteID) {
    const currentReaderBook = readerBookRef.current;
    if (!currentReaderBook?.book?.id || !noteID) {
      return null;
    }

    const bookID = currentReaderBook.book.id;
    try {
      const nextBook = await DeleteReaderNote(bookID, noteID);
      updateReaderBook(nextBook);
      return nextBook;
    } catch (err) {
      setReaderError(readerErrorMessage(err));
      return null;
    }
  }

  async function saveReaderBookmark(bookmark) {
    const currentReaderBook = readerBookRef.current;
    if (!currentReaderBook?.book?.id) {
      return null;
    }

    const bookID = currentReaderBook.book.id;
    try {
      const nextBook = await SaveReaderBookmark(bookID, bookmark);
      updateReaderBook(nextBook);
      return nextBook;
    } catch (err) {
      setReaderError(readerErrorMessage(err));
      return null;
    }
  }

  async function deleteReaderBookmark(bookmarkID) {
    const currentReaderBook = readerBookRef.current;
    if (!currentReaderBook?.book?.id || !bookmarkID) {
      return null;
    }

    const bookID = currentReaderBook.book.id;
    try {
      const nextBook = await DeleteReaderBookmark(bookID, bookmarkID);
      updateReaderBook(nextBook);
      return nextBook;
    } catch (err) {
      setReaderError(readerErrorMessage(err));
      return null;
    }
  }

  function updateReaderBook(nextBook) {
    if (!nextBook?.id) {
      return;
    }

    setReaderBook((current) => {
      if (!current || current.book.id !== nextBook.id) {
        return current;
      }

      const nextReaderBook = {
        ...current,
        book: nextBook,
      };
      readerBookRef.current = nextReaderBook;
      return nextReaderBook;
    });
  }

  function goToNextChapter() {
    goToChapter(currentChapterIndexRef.current + 1);
  }

  function goToPreviousChapter() {
    goToChapter(currentChapterIndexRef.current - 1, LAST_PAGE_INDEX, null, {
      deferProgress: true,
    });
  }

  useEffect(() => {
    currentChapterIndexRef.current = currentChapterIndex;
  }, [currentChapterIndex]);

  useEffect(() => {
    if (!readerError) {
      return undefined;
    }

    const timeoutID = window.setTimeout(() => setReaderError(''), READER_ERROR_TIMEOUT_MS);
    return () => window.clearTimeout(timeoutID);
  }, [readerError]);

  return {
    closeReader,
    currentChapterIndex,
    currentChapter: readerBook?.chapters?.[currentChapterIndex] ?? null,
    currentPageIndex,
    deleteReaderBookmark,
    deleteReaderNote,
    goToChapter,
    goToNextChapter,
    goToPreviousChapter,
    isReaderLoading,
    openReader,
    readerBook,
    readerError,
    saveCurrentPage,
    saveReaderBookmark,
    saveReaderAppearance,
    saveReaderNote,
  };
}

function clampChapterIndex(index, chapterCount) {
  if (chapterCount <= 0) {
    return 0;
  }

  return Math.min(Math.max(index, 0), chapterCount - 1);
}

function pageLocation(pageIndex, percent) {
  const location = [`page:${Math.max(0, pageIndex)}`];
  if (Number.isFinite(percent)) {
    location.push(`percent:${Math.min(100, Math.max(0, Math.round(percent)))}`);
  }

  return location.join(';');
}

function progressPageIndex(readerBook, chapterIndex) {
  const progress = readerBook?.book?.progress;
  const chapter = readerBook?.chapters?.[chapterIndex];

  if (!progress || !chapter || progress.chapterHref !== chapter.href) {
    return 0;
  }

  const match = /(?:^|;)page:(\d+)(?:;|$)/.exec(progress.location || '');
  return match ? Number(match[1]) : 0;
}

function readerErrorMessage(err) {
  const message = err?.message ?? String(err);
  const lowerMessage = message.toLowerCase();

  if (lowerMessage.includes('invalid epub file')) {
    return 'This EPUB cannot be rendered. The managed copy may be malformed or unsupported.';
  }

  return message;
}
