import {useEffect, useState} from 'react';
import {
  GetReaderBook,
  SaveReadingProgress,
} from '../../wailsjs/go/main/App';

const READER_ERROR_TIMEOUT_MS = 6000;

// useReader owns EPUB loading, chapter navigation, and progress persistence.
export function useReader() {
  const [readerBook, setReaderBook] = useState(null);
  const [currentChapterIndex, setCurrentChapterIndex] = useState(0);
  const [isReaderLoading, setIsReaderLoading] = useState(false);
  const [readerError, setReaderError] = useState('');

  async function openReader(book) {
    setIsReaderLoading(true);
    setReaderError('');

    try {
      const nextReaderBook = await GetReaderBook(book.id);
      setReaderBook(nextReaderBook);
      setCurrentChapterIndex(nextReaderBook.currentChapterIndex ?? 0);
    } catch (err) {
      setReaderError(readerErrorMessage(err));
    } finally {
      setIsReaderLoading(false);
    }
  }

  function closeReader() {
    setReaderBook(null);
    setCurrentChapterIndex(0);
    setReaderError('');
  }

  async function goToChapter(index) {
    if (!readerBook) {
      return;
    }

    const nextIndex = clampChapterIndex(index, readerBook.chapters.length);
    const chapter = readerBook.chapters[nextIndex];

    setCurrentChapterIndex(nextIndex);

    try {
      const nextBook = await SaveReadingProgress(readerBook.book.id, {
        chapterHref: chapter.href,
        chapterIndex: nextIndex,
        location: 'chapter',
      });

      setReaderBook((current) => current && {
        ...current,
        book: nextBook,
        currentChapterIndex: nextIndex,
      });
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
    goToChapter,
    goToNextChapter,
    goToPreviousChapter,
    isReaderLoading,
    openReader,
    readerBook,
    readerError,
  };
}

function clampChapterIndex(index, chapterCount) {
  if (chapterCount <= 0) {
    return 0;
  }

  return Math.min(Math.max(index, 0), chapterCount - 1);
}

function readerErrorMessage(err) {
  const message = err?.message ?? String(err);
  const lowerMessage = message.toLowerCase();

  if (lowerMessage.includes('invalid epub file')) {
    return 'This EPUB cannot be rendered. The managed copy may be malformed or unsupported.';
  }

  return message;
}
