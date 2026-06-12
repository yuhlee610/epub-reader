import {useCallback, useEffect, useLayoutEffect, useRef, useState} from 'react';
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

const LAST_PAGE_REQUEST_THRESHOLD = Number.MAX_SAFE_INTEGER / 2;
const DEFAULT_READER_APPEARANCE = {
  backgroundColor: '#ffffff',
  fontSize: 18,
};
const READER_BACKGROUND_OPTIONS = [
  {label: 'White', value: '#ffffff'},
  {label: 'Warm', value: '#f8f1e7'},
  {label: 'Sepia', value: '#efe4ce'},
  {label: 'Sage', value: '#eef4ee'},
  {label: 'Gray', value: '#f1f1f1'},
];
const READER_FONT_SIZE_OPTIONS = [16, 18, 20, 22, 24];

export function ReaderView({
  chapter,
  chapterIndex,
  initialPageIndex,
  isLoading,
  onClose,
  onAppearanceChange,
  onNext,
  onPageChange,
  onPrevious,
  onSelectChapter,
  readerBook,
}) {
  const contentRef = useRef(null);
  const pageRef = useRef(null);
  const lastMeasuredPageCountRef = useRef(null);
  const stablePageMeasureCountRef = useRef(0);
  const positioningTimeoutRef = useRef(null);
  const requestedPageIndexRef = useRef(0);
  const savedProgressRef = useRef('');
  const [contentWidth, setContentWidth] = useState(1);
  const [pageCount, setPageCount] = useState(1);
  const [pageIndex, setPageIndex] = useState(0);
  const [pageStep, setPageStep] = useState(1);
  const [isTocOpen, setIsTocOpen] = useState(true);
  const [isPageContentHidden, setIsPageContentHidden] = useState(true);
  const [isPagePositioning, setIsPagePositioning] = useState(true);
  const [measuredChapterHref, setMeasuredChapterHref] = useState('');
  const chapterCount = readerBook?.chapters?.length ?? 0;

  useLayoutEffect(() => {
    window.clearTimeout(positioningTimeoutRef.current);
    const requestedPageIndex = Math.max(0, initialPageIndex || 0);
    lastMeasuredPageCountRef.current = null;
    stablePageMeasureCountRef.current = 0;
    requestedPageIndexRef.current = requestedPageIndex;
    savedProgressRef.current = '';
    setIsPageContentHidden(true);
    setIsPagePositioning(true);
    setContentWidth(1);
    setPageCount(1);
    setPageIndex(0);
    setPageStep(1);
    setMeasuredChapterHref('');
  }, [chapter?.href]);

  useEffect(() => {
    const page = pageRef.current;
    const content = contentRef.current;
    if (!page || !content) {
      return undefined;
    }

    let frameID = 0;
    const timeoutIDs = [];

    function finishPagePositioning() {
      window.clearTimeout(positioningTimeoutRef.current);
      positioningTimeoutRef.current = window.setTimeout(() => {
        setIsPageContentHidden(false);
        setIsPagePositioning(false);
      }, 80);
    }

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
        if (requestedPageIndexRef.current !== null) {
          const requestedPageIndex = requestedPageIndexRef.current;

          // WebView column layout can briefly report the wrong page count while
          // a new chapter settles, so do not consume an out-of-range request
          // until the measured page count has stabilized.
          const isEndRequest =
            requestedPageIndex >= LAST_PAGE_REQUEST_THRESHOLD;
          const isInRange = requestedPageIndex <= nextPageCount - 1;
          const stableMeasureCount =
            lastMeasuredPageCountRef.current === nextPageCount
              ? stablePageMeasureCountRef.current + 1
              : 1;
          const isStable = stableMeasureCount >= 2;
          const isSettledEndRequest =
            isEndRequest &&
            isStable &&
            (nextPageCount > 1 || stableMeasureCount >= 4);

          stablePageMeasureCountRef.current = stableMeasureCount;
          if (isInRange || (!isEndRequest && isStable) || isSettledEndRequest) {
            requestedPageIndexRef.current = null;
            finishPagePositioning();
          }
          lastMeasuredPageCountRef.current = nextPageCount;
          return Math.min(requestedPageIndex, nextPageCount - 1);
        }

        lastMeasuredPageCountRef.current = nextPageCount;
        finishPagePositioning();
        return Math.min(current, nextPageCount - 1);
      });
      setMeasuredChapterHref(chapter?.href || '');
    }

    function scheduleMeasure() {
      window.cancelAnimationFrame(frameID);
      frameID = window.requestAnimationFrame(measurePages);
    }

    scheduleMeasure();
    timeoutIDs.push(window.setTimeout(scheduleMeasure, 50));
    timeoutIDs.push(window.setTimeout(scheduleMeasure, 150));
    timeoutIDs.push(window.setTimeout(scheduleMeasure, 300));

    if (typeof ResizeObserver === 'undefined') {
      return () => {
        window.cancelAnimationFrame(frameID);
        window.clearTimeout(positioningTimeoutRef.current);
        timeoutIDs.forEach((timeoutID) => window.clearTimeout(timeoutID));
      };
    }

    const observer = new ResizeObserver(scheduleMeasure);
    observer.observe(page);
    observer.observe(content);

    return () => {
      window.cancelAnimationFrame(frameID);
      window.clearTimeout(positioningTimeoutRef.current);
      timeoutIDs.forEach((timeoutID) => window.clearTimeout(timeoutID));
      observer.disconnect();
    };
  }, [chapter?.bodyHtml, chapter?.href, isTocOpen, readerBook?.book?.appearance?.fontSize]);

  const progressPercent = readingProgressPercent(
    chapterIndex,
    chapterCount,
    pageIndex,
    pageCount,
  );
  const isChapterMeasured = !chapter || measuredChapterHref === chapter.href;

  const saveVisiblePage = useCallback((nextPageIndex, nextProgressPercent) => {
    if (!chapter) {
      return;
    }

    onPageChange(nextPageIndex, nextProgressPercent, {
      chapterHref: chapter.href,
      chapterIndex,
    });
  }, [chapter, chapterIndex, onPageChange]);

  useEffect(() => {
    if (!readerBook || !chapter || measuredChapterHref !== chapter.href) {
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
    saveVisiblePage(pageIndex, progressPercent);
  }, [
    chapter,
    measuredChapterHref,
    pageIndex,
    progressPercent,
    readerBook,
    saveVisiblePage,
  ]);

  const goPrevious = useCallback(() => {
    if (!readerBook || !isChapterMeasured) {
      return;
    }

    if (pageIndex > 0) {
      const nextPage = pageIndex - 1;
      requestedPageIndexRef.current = null;
      setIsPageContentHidden(false);
      setIsPagePositioning(false);
      setPageIndex(nextPage);
      saveVisiblePage(nextPage, readingProgressPercent(
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
    isChapterMeasured,
    onPrevious,
    pageCount,
    pageIndex,
    readerBook,
    saveVisiblePage,
  ]);

  const goNext = useCallback(() => {
    if (!readerBook || !isChapterMeasured) {
      return;
    }

    if (pageIndex < pageCount - 1) {
      const nextPage = pageIndex + 1;
      requestedPageIndexRef.current = null;
      setIsPageContentHidden(false);
      setIsPagePositioning(false);
      setPageIndex(nextPage);
      saveVisiblePage(nextPage, readingProgressPercent(
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
    isChapterMeasured,
    onNext,
    pageCount,
    pageIndex,
    readerBook,
    saveVisiblePage,
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
  const pageOffset = pageIndex * pageStep;
  const appearance = normalizeReaderAppearance(readerBook.book.appearance);
  const contentClassName = [
    'reader-chapter-content',
    isPagePositioning ? 'is-positioning' : '',
    isPageContentHidden ? 'is-hidden' : '',
  ].filter(Boolean).join(' ');

  return (
    <main
      className={isTocOpen ? 'reader-shell' : 'reader-shell is-toc-collapsed'}
      style={{
        '--reader-background-color': appearance.backgroundColor,
        '--reader-font-size': `${appearance.fontSize}px`,
      }}
    >
      {isTocOpen && (
        <ReaderSidebar
          activeChapterIndex={chapterIndex}
          appearance={appearance}
          book={readerBook.book}
          chapters={readerBook.chapters}
          onClose={onClose}
          onAppearanceChange={onAppearanceChange}
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

        <article className="reader-page" aria-label={chapter.title}>
          <div className="reader-page-frame" ref={pageRef}>
            <div
              className={contentClassName}
              ref={contentRef}
              style={{
                '--reader-column-gap': `${columnGap}px`,
                '--reader-column-width': `${columnWidth}px`,
                '--reader-page-offset': `${pageOffset}px`,
              }}
              dangerouslySetInnerHTML={{__html: chapter.bodyHtml}}
            />
          </div>
        </article>

        <div className="reader-bottom-bar" aria-label="Reader navigation">
          <div className="reader-nav-buttons">
            <button
              aria-label="Previous chapter"
              className="reader-nav-button"
              disabled={!isChapterMeasured || (chapterIndex === 0 && pageIndex === 0)}
              onClick={goPrevious}
              type="button"
            >
              <ChevronLeft aria-hidden="true" size={18} strokeWidth={2} />
            </button>
            <button
              aria-label="Next chapter"
              className="reader-nav-button"
              disabled={
                !isChapterMeasured ||
                (chapterIndex >= chapterCount - 1 && pageIndex >= pageCount - 1)
              }
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

function normalizeReaderAppearance(appearance) {
  const backgroundColor = String(appearance?.backgroundColor || '').toLowerCase();
  const backgroundOption = READER_BACKGROUND_OPTIONS.find((option) => (
    option.value === backgroundColor
  ));
  const fontSize = Number(appearance?.fontSize);

  return {
    backgroundColor: backgroundOption?.value || DEFAULT_READER_APPEARANCE.backgroundColor,
    fontSize: READER_FONT_SIZE_OPTIONS.includes(fontSize)
      ? fontSize
      : DEFAULT_READER_APPEARANCE.fontSize,
  };
}

function ReaderSidebar({
  activeChapterIndex,
  appearance,
  book,
  chapters,
  onClose,
  onAppearanceChange,
  onSelectChapter,
  onToggleToc,
}) {
  const [isAppearanceOpen, setIsAppearanceOpen] = useState(false);

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
            aria-label="Reader appearance settings"
            className={isAppearanceOpen ? 'reader-icon-button is-active' : 'reader-icon-button'}
            onClick={() => setIsAppearanceOpen((isOpen) => !isOpen)}
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

      <div className="reader-sidebar-header">
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

        {isAppearanceOpen && (
          <div className="reader-appearance-panel" aria-label="Reader appearance settings">
            <div className="reader-settings-row">
              <span>Background</span>
              <div className="reader-color-options">
                {READER_BACKGROUND_OPTIONS.map((option) => (
                  <button
                    aria-label={`Use ${option.label} background`}
                    aria-pressed={appearance.backgroundColor === option.value}
                    className={
                      appearance.backgroundColor === option.value
                        ? 'reader-color-swatch is-active'
                        : 'reader-color-swatch'
                    }
                    key={option.value}
                    onClick={() => onAppearanceChange?.({
                      ...appearance,
                      backgroundColor: option.value,
                    })}
                    style={{'--swatch-color': option.value}}
                    title={option.label}
                    type="button"
                  />
                ))}
              </div>
            </div>

            <div className="reader-settings-row">
              <span>Font size</span>
              <div className="reader-font-size-options">
                {READER_FONT_SIZE_OPTIONS.map((size) => (
                  <button
                    aria-pressed={appearance.fontSize === size}
                    className={
                      appearance.fontSize === size
                        ? 'reader-font-size-option is-active'
                        : 'reader-font-size-option'
                    }
                    key={size}
                    onClick={() => onAppearanceChange?.({
                      ...appearance,
                      fontSize: size,
                    })}
                    type="button"
                  >
                    {size}
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}
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
