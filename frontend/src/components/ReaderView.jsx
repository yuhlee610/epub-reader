import {useCallback, useEffect, useRef, useState} from 'react';
import {
  Bookmark,
  ChevronLeft,
  ChevronRight,
  Edit3,
  Info,
  List,
  Maximize2,
  Menu,
  Minus,
  PanelLeftClose,
  PanelLeftOpen,
  Pin,
  Search,
  X,
} from 'lucide-react';
import {
  Quit,
  WindowMinimise,
  WindowToggleMaximise,
} from '../../wailsjs/runtime/runtime';
import {coverTitle} from '../lib/bookFormatters';

export function ReaderView({
  chapter,
  chapterIndex,
  initialPageIndex,
  isLoading,
  onClose,
  onNext,
  onPageChange,
  onPrevious,
  onSelectChapter,
  readerBook,
}) {
  const contentRef = useRef(null);
  const pageRef = useRef(null);
  const savedProgressRef = useRef('');
  const [contentWidth, setContentWidth] = useState(1);
  const [pageCount, setPageCount] = useState(1);
  const [pageIndex, setPageIndex] = useState(0);
  const [pageStep, setPageStep] = useState(1);
  const [isTocOpen, setIsTocOpen] = useState(true);
  const chapterCount = readerBook?.chapters?.length ?? 0;

  useEffect(() => {
    setPageIndex(Math.max(0, initialPageIndex || 0));
  }, [chapter?.href, initialPageIndex]);

  useEffect(() => {
    const page = pageRef.current;
    const content = contentRef.current;
    if (!page || !content) {
      return undefined;
    }

    let frameID = 0;
    let timeoutID = 0;

    function measurePages() {
      const pageStyle = window.getComputedStyle(page);
      const horizontalPadding =
        parseFloat(pageStyle.paddingLeft) + parseFloat(pageStyle.paddingRight);
      const nextContentWidth = Math.max(1, page.clientWidth - horizontalPadding);
      const nextColumnGap = readerColumnGap(nextContentWidth);
      const nextColumnCount = readerColumnCount(nextContentWidth);
      const nextColumnWidth = readerColumnWidth(nextContentWidth);
      const nextPageStep = Math.max(
        1,
        nextColumnCount * (nextColumnWidth + nextColumnGap),
      );
      const totalWidth = Math.max(1, content.scrollWidth);
      const nextPageCount = Math.max(
        1,
        Math.ceil(totalWidth / nextPageStep),
      );

      setContentWidth(nextContentWidth);
      setPageStep(nextPageStep);
      setPageCount(nextPageCount);
      setPageIndex((current) => {
        const nextIndex = Math.min(current, nextPageCount - 1);
        if (nextIndex !== current) {
          onPageChange(nextIndex);
        }
        return nextIndex;
      });
    }

    function scheduleMeasure() {
      window.cancelAnimationFrame(frameID);
      frameID = window.requestAnimationFrame(measurePages);
    }

    scheduleMeasure();
    timeoutID = window.setTimeout(scheduleMeasure, 50);

    if (typeof ResizeObserver === 'undefined') {
      return () => {
        window.cancelAnimationFrame(frameID);
        window.clearTimeout(timeoutID);
      };
    }

    const observer = new ResizeObserver(scheduleMeasure);
    observer.observe(page);
    observer.observe(content);

    return () => {
      window.cancelAnimationFrame(frameID);
      window.clearTimeout(timeoutID);
      observer.disconnect();
    };
  }, [chapter?.bodyHtml, isTocOpen, onPageChange]);

  useEffect(() => {
    const page = pageRef.current;
    if (!page) {
      return;
    }

    page.scrollTo({
      left: pageIndex * pageStep,
      top: 0,
      behavior: 'smooth',
    });
  }, [chapter?.href, pageIndex, pageStep]);

  const progressPercent = readingProgressPercent(
    chapterIndex,
    chapterCount,
    pageIndex,
    pageCount,
  );

  useEffect(() => {
    if (!readerBook || !chapter) {
      return;
    }

    const progressKey = [
      chapter.href,
      pageIndex,
      pageCount,
      progressPercent,
    ].join(':');
    if (savedProgressRef.current === progressKey) {
      return;
    }

    savedProgressRef.current = progressKey;
    onPageChange(pageIndex, progressPercent);
  }, [
    chapter,
    pageCount,
    pageIndex,
    onPageChange,
    progressPercent,
    readerBook,
  ]);

  const goPrevious = useCallback(() => {
    if (!readerBook) {
      return;
    }

    if (pageIndex > 0) {
      const nextPage = pageIndex - 1;
      setPageIndex(nextPage);
      onPageChange(nextPage, readingProgressPercent(
        chapterIndex,
        chapterCount,
        nextPage,
        pageCount,
      ));
      return;
    }

    onPrevious();
  }, [
    chapterCount,
    chapterIndex,
    onPageChange,
    onPrevious,
    pageCount,
    pageIndex,
    readerBook,
  ]);

  const goNext = useCallback(() => {
    if (!readerBook) {
      return;
    }

    if (pageIndex < pageCount - 1) {
      const nextPage = pageIndex + 1;
      setPageIndex(nextPage);
      onPageChange(nextPage, readingProgressPercent(
        chapterIndex,
        chapterCount,
        nextPage,
        pageCount,
      ));
      return;
    }

    onNext();
  }, [
    chapterCount,
    chapterIndex,
    onNext,
    onPageChange,
    pageCount,
    pageIndex,
    readerBook,
  ]);

  useEffect(() => {
    function handleKeyDown(event) {
      if (shouldIgnoreReaderShortcut(event)) {
        return;
      }

      if (event.key === 'ArrowLeft') {
        event.preventDefault();
        goPrevious();
      } else if (event.key === 'ArrowRight') {
        event.preventDefault();
        goNext();
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [goNext, goPrevious]);

  if (isLoading && !readerBook) {
    return (
      <main className="reader-loading" role="status">
        Opening reader...
      </main>
    );
  }

  if (!readerBook || !chapter) {
    return null;
  }

  const progressLabel = `${progressPercent}%`;
  const pageLabel = `${pageIndex + 1} / ${pageCount}`;
  const columnGap = readerColumnGap(contentWidth);
  const columnWidth = readerColumnWidth(contentWidth);

  return (
    <main className={isTocOpen ? 'reader-shell' : 'reader-shell is-toc-collapsed'}>
      {isTocOpen && (
        <ReaderSidebar
          activeChapterIndex={chapterIndex}
          book={readerBook.book}
          chapters={readerBook.chapters}
          onClose={onClose}
          onSelectChapter={onSelectChapter}
          onToggleToc={() => setIsTocOpen(false)}
        />
      )}

      <section className="reader-stage" aria-label="Reader">
        <div className="reader-topline">
          <button
            aria-label={isTocOpen ? 'Collapse table of contents' : 'Show table of contents'}
            className="reader-stage-back"
            onClick={() => setIsTocOpen((isOpen) => !isOpen)}
            type="button"
          >
            {isTocOpen ? (
              <PanelLeftClose aria-hidden="true" size={17} strokeWidth={2} />
            ) : (
              <PanelLeftOpen aria-hidden="true" size={17} strokeWidth={2} />
            )}
          </button>
          <span>{chapter.title}</span>
          <div className="reader-window-actions" aria-label="Window controls">
            <button
              aria-label="Minimize"
              className="window-tool"
              onClick={WindowMinimise}
              type="button"
            >
              <Minus aria-hidden="true" size={16} strokeWidth={2} />
            </button>
            <button
              aria-label="Maximize"
              className="window-tool"
              onClick={WindowToggleMaximise}
              type="button"
            >
              <Maximize2 aria-hidden="true" size={15} strokeWidth={2} />
            </button>
            <button
              aria-label="Close"
              className="window-tool"
              onClick={Quit}
              type="button"
            >
              <X aria-hidden="true" size={16} strokeWidth={2.2} />
            </button>
          </div>
        </div>

        <article className="reader-page" aria-label={chapter.title} ref={pageRef}>
          <div
            className="reader-chapter-content"
            ref={contentRef}
            style={{
              '--reader-column-gap': `${columnGap}px`,
              '--reader-column-width': `${columnWidth}px`,
            }}
            dangerouslySetInnerHTML={{__html: chapter.bodyHtml}}
          />
        </article>

        <div className="reader-bottom-bar" aria-label="Reader navigation">
          <div className="reader-nav-buttons">
            <button
              aria-label="Previous chapter"
              className="reader-nav-button"
              disabled={chapterIndex === 0 && pageIndex === 0}
              onClick={goPrevious}
              type="button"
            >
              <ChevronLeft aria-hidden="true" size={18} strokeWidth={2} />
            </button>
            <button
              aria-label="Next chapter"
              className="reader-nav-button"
              disabled={chapterIndex >= chapterCount - 1 && pageIndex >= pageCount - 1}
              onClick={goNext}
              type="button"
            >
              <ChevronRight aria-hidden="true" size={18} strokeWidth={2} />
            </button>
          </div>
          <span>{progressLabel}</span>
          <span>{pageLabel}</span>
        </div>
      </section>
    </main>
  );
}

function readerColumnCount(contentWidth) {
  return contentWidth < 680 ? 1 : 2;
}

function readerColumnGap(contentWidth) {
  return contentWidth < 680 ? 0 : 92;
}

function readerColumnWidth(contentWidth) {
  if (readerColumnCount(contentWidth) === 1) {
    return contentWidth;
  }

  return Math.max(280, (contentWidth - readerColumnGap(contentWidth)) / 2);
}

function readingProgressPercent(chapterIndex, chapterCount, pageIndex, pageCount) {
  if (chapterCount <= 0 || pageCount <= 0) {
    return 0;
  }

  const chapterProgress = clampRatio((pageIndex + 1) / pageCount);
  const bookProgress = (chapterIndex + chapterProgress) / chapterCount;
  return Math.min(100, Math.max(0, Math.round(bookProgress * 100)));
}

function clampRatio(value) {
  return Math.min(1, Math.max(0, value));
}

function shouldIgnoreReaderShortcut(event) {
  if (
    event.defaultPrevented ||
    event.altKey ||
    event.ctrlKey ||
    event.metaKey ||
    event.shiftKey
  ) {
    return true;
  }

  const target = event.target;
  if (!target || !(target instanceof HTMLElement)) {
    return false;
  }

  return (
    target.isContentEditable ||
    ['INPUT', 'SELECT', 'TEXTAREA'].includes(target.tagName)
  );
}

function ReaderSidebar({
  activeChapterIndex,
  book,
  chapters,
  onClose,
  onSelectChapter,
  onToggleToc,
}) {
  return (
    <aside className="reader-sidebar" aria-label="Reader table of contents">
      <div className="reader-sidebar-toolbar">
        <button
          aria-label="Collapse table of contents"
          className="reader-icon-button"
          onClick={onToggleToc}
          type="button"
        >
          <PanelLeftClose aria-hidden="true" size={18} strokeWidth={2} />
        </button>
        <div className="reader-sidebar-tools">
          <button
            aria-label="Search in book"
            className="reader-icon-button"
            type="button"
          >
            <Search aria-hidden="true" size={17} strokeWidth={2} />
          </button>
          <button
            aria-label="Reader menu"
            className="reader-icon-button"
            type="button"
          >
            <Menu aria-hidden="true" size={18} strokeWidth={2} />
          </button>
          <button
            aria-label="Pin sidebar"
            className="reader-icon-button is-muted"
            type="button"
          >
            <Pin aria-hidden="true" size={16} strokeWidth={2} />
          </button>
          <button
            aria-label="Close reader"
            className="reader-icon-button"
            onClick={onClose}
            type="button"
          >
            <X aria-hidden="true" size={17} strokeWidth={2} />
          </button>
        </div>
      </div>

      <div className="reader-book-summary">
        <div className="reader-cover" aria-hidden="true">
          <span className="cover-name">{coverTitle(book.title)}</span>
          <span className="cover-author">{book.author || 'Unknown author'}</span>
        </div>
        <div className="reader-book-copy">
          <h1>{book.title || 'Untitled book'}</h1>
          <p>{book.author || 'Unknown author'}</p>
        </div>
        <Info
          aria-hidden="true"
          className="reader-info-icon"
          size={16}
          strokeWidth={2}
        />
      </div>

      <ol className="reader-toc">
        {chapters.map((chapter) => (
          <li key={chapter.href}>
            <button
              className={
                chapter.index === activeChapterIndex
                  ? 'reader-toc-item is-active'
                  : 'reader-toc-item'
              }
              onClick={() => onSelectChapter(chapter.index)}
              type="button"
            >
              <span>{chapter.title}</span>
              <span>{chapter.index + 1}</span>
            </button>
          </li>
        ))}
      </ol>

      <div className="reader-sidebar-footer">
        <button
          aria-label="Table of contents"
          className="reader-footer-button is-active"
          type="button"
        >
          <List aria-hidden="true" size={17} strokeWidth={2} />
        </button>
        <button
          aria-label="Notes"
          className="reader-footer-button"
          type="button"
        >
          <Edit3 aria-hidden="true" size={17} strokeWidth={2} />
        </button>
        <button
          aria-label="Bookmarks"
          className="reader-footer-button"
          type="button"
        >
          <Bookmark aria-hidden="true" size={17} strokeWidth={2} />
        </button>
      </div>
    </aside>
  );
}
