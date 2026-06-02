import {useEffect, useRef, useState} from 'react';
import {
  GetReaderBook,
  SaveReadingProgress,
} from '../../wailsjs/go/main/App';

const READER_ERROR_TIMEOUT_MS = 6000;

// useReader owns EPUB loading, chapter navigation, and progress persistence.
export function useReader() {
  const [readerBook, setReaderBook] = useState(null);
  const [currentChapterIndex, setCurrentChapterIndex] = useState(0);
  const [currentPageIndex, setCurrentPageIndex] = useState(0);
  const [isReaderLoading, setIsReaderLoading] = useState(false);
  const [readerError, setReaderError] = useState('');
  const saveQueueRef = useRef(Promise.resolve());

  async function openReader(book) {
    setIsReaderLoading(true);
    setReaderError('');

    try {
      const nextReaderBook = await GetReaderBook(book.id);
      const nextChapterIndex = nextReaderBook.currentChapterIndex ?? 0;

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
    setCurrentChapterIndex(0);
    setCurrentPageIndex(0);
    setReaderError('');
  }

  async function goToChapter(index, pageIndex = 0, percent = null) {
    if (!readerBook) {
      return;
    }

    const nextIndex = clampChapterIndex(index, readerBook.chapters.length);

    setCurrentChapterIndex(nextIndex);
    setCurrentPageIndex(pageIndex);

    await persistProgress(nextIndex, pageIndex, percent);
  }

  async function saveCurrentPage(pageIndex, percent = null) {
    if (!readerBook) {
      return;
    }

    setCurrentPageIndex(pageIndex);
    await persistProgress(currentChapterIndex, pageIndex, percent);
  }

  async function persistProgress(chapterIndex, pageIndex, percent) {
    const chapter = readerBook.chapters[chapterIndex];

    saveQueueRef.current = saveQueueRef.current.catch(() => undefined).then(async () => {
      const nextBook = await SaveReadingProgress(readerBook.book.id, {
        chapterHref: chapter.href,
        chapterIndex,
        location: pageLocation(pageIndex, percent),
      });
      setReaderBook((current) => current && {
        ...current,
        book: nextBook,
        currentChapterIndex: chapterIndex,
      });
    });

    try {
      await saveQueueRef.current;
    } catch (err) {
      setReaderError(readerErrorMessage(err));
    }
  }

  function goToNextChapter() {
    goToChapter(currentChapterIndex + 1);
  }

  function goToPreviousChapter() {
    goToChapter(currentChapterIndex - 1);
  }

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
    goToChapter,
    goToNextChapter,
    goToPreviousChapter,
    isReaderLoading,
    openReader,
    readerBook,
    readerError,
    saveCurrentPage,
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
