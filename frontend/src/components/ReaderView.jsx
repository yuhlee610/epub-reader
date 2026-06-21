import {useCallback, useEffect, useLayoutEffect, useRef, useState} from 'react';
import ReactMarkdown from 'react-markdown';
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
  Volume2,
  X,
} from 'lucide-react';
import {
  BrowserOpenURL,
  EventsOn,
  Quit,
  WindowMinimise,
  WindowToggleMaximise,
} from '../../wailsjs/runtime/runtime';
import {
  StreamGeminiStudyPrompt,
  TranslateSelectedText,
} from '../../wailsjs/go/main/App';
import {coverTitle} from '../lib/bookFormatters';
import remarkGfm from 'remark-gfm';

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
const STUDY_TOOL_CONFIG = {
  google: {
    label: 'Google Translate',
  },
};
const GEMINI_STREAM_EVENT = 'gemini:study-stream';
const GEMINI_MARKDOWN_COMPONENTS = {
  a({children, href, node: _node, ...props}) {
    function handleClick(event) {
      if (!href) {
        return;
      }
      event.preventDefault();
      BrowserOpenURL(href);
    }

    return (
      <a href={href} onClick={handleClick} rel="noreferrer" target="_blank" {...props}>
        {children}
      </a>
    );
  },
  table({children, node: _node, ...props}) {
    return (
      <div className="reader-gemini-table-wrap">
        <table {...props}>{children}</table>
      </div>
    );
  },
};

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
  const stageRef = useRef(null);
  const lastMeasuredPageCountRef = useRef(null);
  const stablePageMeasureCountRef = useRef(0);
  const positioningTimeoutRef = useRef(null);
  const requestedPageIndexRef = useRef(0);
  const savedProgressRef = useRef('');
  const studyMessageTimeoutRef = useRef(null);
  const [contentWidth, setContentWidth] = useState(1);
  const [pageCount, setPageCount] = useState(1);
  const [pageIndex, setPageIndex] = useState(0);
  const [pageStep, setPageStep] = useState(1);
  const [maxPageOffset, setMaxPageOffset] = useState(0);
  const [isTocOpen, setIsTocOpen] = useState(true);
  const [isPageContentHidden, setIsPageContentHidden] = useState(true);
  const [isPagePositioning, setIsPagePositioning] = useState(true);
  const [measuredChapterHref, setMeasuredChapterHref] = useState('');
  const [selectedStudyText, setSelectedStudyText] = useState('');
  const [selectionToolbarPosition, setSelectionToolbarPosition] = useState(null);
  const [studyActionMessage, setStudyActionMessage] = useState('');
  const [geminiResponse, setGeminiResponse] = useState(null);
  const [geminiError, setGeminiError] = useState('');
  const [activeGeminiPromptID, setActiveGeminiPromptID] = useState('');
  const chapterCount = readerBook?.chapters?.length ?? 0;
  const currentBook = readerBook?.book;

  useLayoutEffect(() => {
    window.clearTimeout(positioningTimeoutRef.current);
    window.clearTimeout(studyMessageTimeoutRef.current);
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
    setMaxPageOffset(0);
    setMeasuredChapterHref('');
    setSelectedStudyText('');
    setSelectionToolbarPosition(null);
    setStudyActionMessage('');
    setGeminiError('');
    setActiveGeminiPromptID('');
  }, [chapter?.href]);

  useEffect(() => () => {
    window.clearTimeout(positioningTimeoutRef.current);
    window.clearTimeout(studyMessageTimeoutRef.current);
  }, []);

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
      const nextColumnWidth = readerColumnWidth(nextContentWidth);
      const nextPageStep = Math.max(1, nextColumnWidth + nextColumnGap);
      const totalWidth = readerVisibleContentWidth(content, page) || Math.max(1, content.scrollWidth);
      const nextMaxPageOffset = Math.max(0, totalWidth - nextContentWidth);
      const nextPageCount = Math.max(
        1,
        Math.ceil(nextMaxPageOffset / nextPageStep) + 1,
      );

      setContentWidth(nextContentWidth);
      setPageStep(nextPageStep);
      setMaxPageOffset(nextMaxPageOffset);
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

  const showStudyMessage = useCallback((message) => {
    window.clearTimeout(studyMessageTimeoutRef.current);
    setStudyActionMessage(message);
    studyMessageTimeoutRef.current = window.setTimeout(() => {
      setStudyActionMessage('');
    }, 3600);
  }, []);

  const updateSelectedStudyText = useCallback(() => {
    const selectionDetails = selectedReaderDetails(contentRef.current, stageRef.current);
    setSelectedStudyText(selectionDetails.text);
    setSelectionToolbarPosition(selectionDetails.position);
    if (selectionDetails.text) {
      setStudyActionMessage('');
    }
  }, []);

  const openGoogleTranslate = useCallback(async (prompt = null) => {
    const selectedText = cleanSelectedText(selectedStudyText);
    const toolConfig = STUDY_TOOL_CONFIG.google;
    if (!selectedText) {
      showStudyMessage('Select text in the page first.');
      return;
    }

    setGeminiResponse({
      providerLabel: toolConfig.label,
      promptName: prompt?.name || 'Translate',
      originalText: selectedText,
      textPreview: selectedTextPreview(selectedText),
      text: '',
      isLoading: true,
      isPlainText: true,
    });
    setGeminiError('');
    setActiveGeminiPromptID(prompt?.id || 'google-translate');

    try {
      const response = await TranslateSelectedText({
        text: selectedText,
        sourceLanguage: 'auto',
        targetLanguage: 'vi',
      });
      setGeminiResponse({
        providerLabel: toolConfig.label,
        promptName: prompt?.name || 'Translate',
        originalText: response.originalText || selectedText,
        textPreview: response.textPreview || selectedTextPreview(selectedText),
        text: response.translatedText || '',
        sourceLanguage: response.sourceLanguage,
        targetLanguage: response.targetLanguage,
        pronunciationIpa: response.pronunciationIpa,
        isLoading: false,
        isPlainText: true,
      });
    } catch (error) {
      setGeminiError(error?.message ?? String(error));
      setGeminiResponse((current) => ({
        ...(current || {}),
        providerLabel: toolConfig.label,
        promptName: prompt?.name || 'Translate',
        originalText: selectedText,
        textPreview: selectedTextPreview(selectedText),
        text: current?.text || '',
        isLoading: false,
        isPlainText: true,
      }));
    } finally {
      setActiveGeminiPromptID('');
    }
  }, [selectedStudyText, showStudyMessage]);

  const runGeminiPrompt = useCallback(async (prompt) => {
    const selectedText = cleanSelectedText(selectedStudyText);
    if (!selectedText) {
      showStudyMessage('Select text in the page first.');
      return;
    }
    if (!currentBook?.id || !prompt?.id) {
      showStudyMessage('No Gemini prompt is configured for this book.');
      return;
    }

    const requestID = geminiRequestID();
    setGeminiResponse({
      requestId: requestID,
      promptName: prompt.name || 'Gemini',
      textPreview: selectedTextPreview(selectedText),
      text: '',
      isLoading: true,
    });
    setGeminiError('');
    setActiveGeminiPromptID(prompt.id);

    const stopListening = EventsOn(GEMINI_STREAM_EVENT, (event) => {
      if (!event || event.requestId !== requestID) {
        return;
      }

      if (event.type === 'chunk') {
        setGeminiResponse((current) => {
          if (!current || current.requestId !== requestID) {
            return current;
          }
          return {
            ...current,
            text: `${current.text || ''}${event.text || ''}`,
            isLoading: true,
          };
        });
        return;
      }

      if (event.type === 'done') {
        setGeminiResponse((current) => {
          if (!current || current.requestId !== requestID) {
            return current;
          }
          return {
            ...current,
            text: event.text ?? current.text ?? '',
            isLoading: false,
          };
        });
        return;
      }

      if (event.type === 'error') {
        setGeminiError(event.error || 'Gemini stream failed.');
        setGeminiResponse((current) => {
          if (!current || current.requestId !== requestID) {
            return current;
          }
          return {
            ...current,
            isLoading: false,
          };
        });
      }
    });

    try {
      const response = await StreamGeminiStudyPrompt({
        requestId: requestID,
        bookId: currentBook.id,
        promptId: prompt.id,
        selectedText,
        chapterTitle: chapter?.title || '',
      });
      setGeminiResponse({
        ...response,
        requestId: requestID,
        isLoading: false,
      });
    } catch (error) {
      setGeminiError(error?.message ?? String(error));
      setGeminiResponse((current) => ({
        ...(current || {}),
        requestId: requestID,
        promptName: prompt.name || 'Gemini',
        textPreview: selectedTextPreview(selectedText),
        text: current?.text || '',
        isLoading: false,
      }));
    } finally {
      stopListening();
      setActiveGeminiPromptID('');
    }
  }, [chapter?.title, currentBook?.id, selectedStudyText, showStudyMessage]);

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
  const pageOffset = Math.min(pageIndex * pageStep, maxPageOffset);
  const appearance = normalizeReaderAppearance(readerBook.book.appearance);
  const studyPrompts = normalizeStudyPrompts(currentBook?.prompt?.prompts);
  const translationPrompts = studyPrompts.filter(isTranslationPrompt);
  const geminiPrompts = studyPrompts.filter((prompt) => !isTranslationPrompt(prompt));
  const googleTranslateActions = translationPrompts.length > 0
    ? translationPrompts
    : [{id: 'google-translate', name: 'Translate', shortLabel: 'TR'}];
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

      <section className="reader-stage" aria-label="Reader" ref={stageRef}>
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
          <div
            className="reader-page-frame"
            onKeyUp={updateSelectedStudyText}
            onMouseUp={updateSelectedStudyText}
            onTouchEnd={updateSelectedStudyText}
            ref={pageRef}
          >
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

        {selectedStudyText && (
          <div
            className="reader-selection-actions"
            role="toolbar"
            aria-label="Selected text study actions"
            style={selectionToolbarStyle(selectionToolbarPosition)}
          >
            <span>{selectedStudyText.length} chars selected</span>
            {googleTranslateActions.map((prompt) => (
              <button
                aria-label="Translate selected text with Google Translate"
                className="reader-study-button reader-gemini-prompt-button"
                disabled={activeGeminiPromptID === prompt.id}
                key={prompt.id}
                onClick={() => openGoogleTranslate(prompt)}
                title="Translate selected text in app"
                type="button"
              >
                <GoogleTranslateIcon />
                <span>{prompt.shortLabel}</span>
              </button>
            ))}
            {geminiPrompts.map((prompt) => (
              <button
                aria-label={`Run ${prompt.name} with Gemini`}
                className="reader-study-button reader-gemini-prompt-button"
                disabled={activeGeminiPromptID === prompt.id}
                key={prompt.id}
                onClick={() => runGeminiPrompt(prompt)}
                title={`Send selected text to Gemini with ${prompt.name}`}
                type="button"
              >
                <GeminiIcon />
                <span>{prompt.shortLabel}</span>
              </button>
            ))}
          </div>
        )}

        {geminiResponse && (
          <GeminiResponseDialog
            error={geminiError}
            response={geminiResponse}
            onClose={() => {
              setGeminiResponse(null);
              setGeminiError('');
            }}
          />
        )}

        {studyActionMessage && (
          <div className="reader-study-toast" role="status">
            {studyActionMessage}
          </div>
        )}

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

function selectedReaderDetails(content, stage) {
  const emptySelection = {
    position: null,
    text: '',
  };

  if (!content || !stage || typeof window.getSelection !== 'function') {
    return emptySelection;
  }

  const selection = window.getSelection();
  if (!selection || selection.isCollapsed || selection.rangeCount === 0) {
    return emptySelection;
  }

  const selectedText = cleanSelectedText(selection.toString());
  if (!selectedText || !selectionBelongsToContent(selection, content)) {
    return emptySelection;
  }

  return {
    position: readerSelectionToolbarPosition(selection, stage),
    text: selectedText,
  };
}

function selectionBelongsToContent(selection, content) {
  const anchorNode = selection.anchorNode;
  const focusNode = selection.focusNode;
  if (anchorNode && focusNode) {
    return nodeBelongsToContent(anchorNode, content) &&
      nodeBelongsToContent(focusNode, content);
  }

  const range = selection.getRangeAt(0);
  return nodeBelongsToContent(range.commonAncestorContainer, content);
}

function nodeBelongsToContent(node, content) {
  if (!node) {
    return false;
  }

  return content === node ||
    content.contains(node.nodeType === Node.ELEMENT_NODE ? node : node.parentNode);
}

function cleanSelectedText(text) {
  return String(text || '').replace(/\s+/g, ' ').trim();
}

function readerSelectionToolbarPosition(selection, stage) {
  const range = selection.getRangeAt(0);
  const selectionRect = readerSelectionRect(range);
  if (!selectionRect) {
    return null;
  }

  const stageRect = stage.getBoundingClientRect();
  const toolbarHalfWidth = Math.min(232, Math.max(142, (stageRect.width - 32) / 2));
  const left = clampNumber(
    selectionRect.left + (selectionRect.width / 2) - stageRect.left,
    toolbarHalfWidth + 12,
    stageRect.width - toolbarHalfWidth - 12,
  );
  const hasRoomAbove = selectionRect.top - stageRect.top >= 48;
  const top = hasRoomAbove
    ? selectionRect.top - stageRect.top - 10
    : selectionRect.bottom - stageRect.top + 10;

  return {
    left: Math.round(left),
    placement: hasRoomAbove ? 'above' : 'below',
    top: Math.round(clampNumber(top, 52, stageRect.height - 58)),
  };
}

function readerSelectionRect(range) {
  const rects = Array.from(range.getClientRects()).filter((rect) => (
    rect.width > 0 && rect.height > 0
  ));
  if (rects.length === 0) {
    const rect = range.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0 ? rect : null;
  }

  const firstRect = rects[0];
  return rects.reduce((bounds, rect) => ({
    bottom: Math.max(bounds.bottom, rect.bottom),
    height: Math.max(bounds.bottom, rect.bottom) - Math.min(bounds.top, rect.top),
    left: Math.min(bounds.left, rect.left),
    right: Math.max(bounds.right, rect.right),
    top: Math.min(bounds.top, rect.top),
    width: Math.max(bounds.right, rect.right) - Math.min(bounds.left, rect.left),
  }), {
    bottom: firstRect.bottom,
    height: firstRect.height,
    left: firstRect.left,
    right: firstRect.right,
    top: firstRect.top,
    width: firstRect.width,
  });
}

function selectionToolbarStyle(position) {
  if (!position) {
    return undefined;
  }

  return {
    bottom: 'auto',
    left: `${position.left}px`,
    right: 'auto',
    top: `${position.top}px`,
    transform: position.placement === 'above'
      ? 'translate(-50%, -100%)'
      : 'translate(-50%, 0)',
  };
}

function clampNumber(value, min, max) {
  if (max < min) {
    return (min + max) / 2;
  }

  return Math.min(max, Math.max(min, value));
}

function GoogleTranslateIcon() {
  return (
    <svg aria-hidden="true" className="reader-brand-icon" viewBox="0 0 24 24">
      <rect fill="#4285f4" height="14" rx="2.8" width="14" x="3" y="3" />
      <path d="M8.2 7.3h5.1M10.8 6.1v1.2M12.4 7.3c-.5 1.7-1.8 3.3-3.9 4.6M8.2 9.7c.9 1.1 2 1.9 3.3 2.4" fill="none" stroke="#ffffff" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.4" />
      <rect fill="#ffffff" height="11" rx="2.3" stroke="#d7dce2" strokeWidth="1" width="11" x="10" y="10" />
      <path d="M13 18l2.5-6 2.5 6M13.8 16.3h3.4" fill="none" stroke="#3c4043" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.25" />
      <circle cx="5.8" cy="5.8" fill="#34a853" r="1.4" />
      <circle cx="15.6" cy="4.2" fill="#fbbc04" r="1.1" />
      <circle cx="19.5" cy="10.5" fill="#ea4335" r="1.1" />
    </svg>
  );
}

function GeminiIcon() {
  return (
    <svg aria-hidden="true" className="reader-brand-icon" viewBox="0 0 24 24">
      <defs>
        <linearGradient id="reader-gemini-gradient" x1="4" x2="20" y1="20" y2="4" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#4c8bf5" />
          <stop offset="0.48" stopColor="#8e75ff" />
          <stop offset="1" stopColor="#c36bff" />
        </linearGradient>
      </defs>
      <path d="M12 2.8c.7 5.1 4.1 8.5 9.2 9.2-5.1.7-8.5 4.1-9.2 9.2-.7-5.1-4.1-8.5-9.2-9.2 5.1-.7 8.5-4.1 9.2-9.2z" fill="url(#reader-gemini-gradient)" />
    </svg>
  );
}

function GeminiResponseDialog({error, response, onClose}) {
  const responseText = String(response.text || '');
  const providerLabel = response.providerLabel || 'Gemini response';
  const loadingLabel = response.providerLabel === STUDY_TOOL_CONFIG.google.label
    ? 'Translating...'
    : 'Asking Gemini...';
  const streamingLabel = response.providerLabel === STUDY_TOOL_CONFIG.google.label
    ? ''
    : 'Gemini is still writing...';
  function handleBackdropMouseDown(event) {
    if (event.target === event.currentTarget) {
      onClose();
    }
  }

  return (
    <div
      className="reader-gemini-backdrop"
      onMouseDown={handleBackdropMouseDown}
      role="presentation"
    >
      <section
        aria-label={`${providerLabel} response`}
        aria-live="polite"
        aria-modal="true"
        className="reader-gemini-dialog"
        role="dialog"
      >
        <div className="reader-gemini-dialog-header">
          <div>
            <span>{providerLabel}</span>
            <h2>{response.promptName || 'Study prompt'}</h2>
          </div>
          <button
            aria-label={`Close ${providerLabel} response`}
            className="reader-icon-button"
            onClick={onClose}
            type="button"
          >
            <X aria-hidden="true" size={17} strokeWidth={2} />
          </button>
        </div>
        {response.textPreview && (
          <div className="reader-gemini-preview">
            <p>{response.textPreview}</p>
            {response.pronunciationIpa && (
              <span className="reader-ipa">{response.pronunciationIpa}</span>
            )}
            <PronunciationButton
              label="Listen to selected text pronunciation"
              lang={speechLanguage(response.sourceLanguage, 'en-US')}
              text={response.originalText || response.textPreview}
            />
          </div>
        )}
        {response.isLoading && !responseText ? (
          <div className="reader-gemini-state" role="status">
            {loadingLabel}
          </div>
        ) : (
          <div className="reader-gemini-response-shell">
            {responseText && response.isPlainText ? (
              <div className="reader-gemini-response reader-translation-text">
                <p>{responseText}</p>
              </div>
            ) : responseText ? (
              <ReactMarkdown
                className="reader-gemini-response"
                components={GEMINI_MARKDOWN_COMPONENTS}
                remarkPlugins={[remarkGfm]}
                skipHtml
              >
                {responseText}
              </ReactMarkdown>
            ) : null}
            {response.sourceLanguage && response.targetLanguage && (
              <p className="reader-translation-meta">
                {response.sourceLanguage.toUpperCase()} to {response.targetLanguage.toUpperCase()}
              </p>
            )}
            {response.isLoading && streamingLabel && (
              <div className="reader-gemini-state is-streaming" role="status">
                {streamingLabel}
              </div>
            )}
            {error && (
              <div className="reader-gemini-error" role="alert">
                {error}
              </div>
            )}
          </div>
        )}
      </section>
    </div>
  );
}

function PronunciationButton({label, lang, text}) {
  if (!cleanSelectedText(text)) {
    return null;
  }

  return (
    <button
      aria-label={label}
      className="reader-audio-button"
      onClick={() => speakText(text, lang)}
      title={label}
      type="button"
    >
      <Volume2 aria-hidden="true" size={15} strokeWidth={2} />
    </button>
  );
}

function normalizeStudyPrompts(prompts) {
  return (prompts ?? [])
    .map((prompt) => ({
      id: String(prompt?.id || '').trim(),
      name: String(prompt?.name || 'Gemini prompt').trim(),
      shortLabel: String(prompt?.shortLabel || 'AI').trim().toUpperCase().slice(0, 4),
    }))
    .filter((prompt) => prompt.id);
}

function isTranslationPrompt(prompt) {
  const id = String(prompt?.id || '').trim().toLowerCase();
  const name = String(prompt?.name || '').trim().toLowerCase();
  return id === 'translate' || name === 'translate' || name.includes('translation');
}

function selectedTextPreview(value) {
  const text = cleanSelectedText(value);
  return text.length > 180 ? `${text.slice(0, 180).trim()}...` : text;
}

function speakText(value, lang) {
  const text = cleanSelectedText(value);
  const Utterance = globalThis.SpeechSynthesisUtterance;
  if (!text || typeof Utterance !== 'function' || !globalThis.speechSynthesis) {
    return;
  }

  const utterance = new Utterance(text);
  utterance.lang = lang;
  globalThis.speechSynthesis.cancel();
  globalThis.speechSynthesis.speak(utterance);
}

function speechLanguage(language, fallback) {
  const normalized = String(language || '').trim().toLowerCase();
  if (normalized === 'vi') {
    return 'vi-VN';
  }
  if (normalized === 'en') {
    return 'en-US';
  }
  if (/^[a-z]{2}-[a-z]{2}$/i.test(normalized)) {
    return normalized;
  }

  return fallback;
}

function geminiRequestID() {
  if (typeof globalThis.crypto?.randomUUID === 'function') {
    return globalThis.crypto.randomUUID();
  }

  return `gemini-${Date.now()}-${Math.random().toString(36).slice(2)}`;
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

function readerVisibleContentWidth(content, page) {
  if (typeof document.createRange !== 'function') {
    return 0;
  }

  const pageRect = page.getBoundingClientRect();
  const transform = window.getComputedStyle(content).transform;
  const currentOffset = readerTransformOffset(transform);
  const range = document.createRange();
  range.selectNodeContents(content);

  let maxRight = 0;
  for (const rect of range.getClientRects()) {
    if (
      rect.width <= 0 ||
      rect.height <= 0 ||
      rect.bottom <= pageRect.top ||
      rect.top >= pageRect.bottom
    ) {
      continue;
    }

    maxRight = Math.max(maxRight, rect.right + currentOffset - pageRect.left);
  }
  range.detach();

  return Math.max(0, maxRight);
}

function readerTransformOffset(transform) {
  if (!transform || transform === 'none') {
    return 0;
  }

  const match = /matrix\(([^)]+)\)/.exec(transform);
  if (!match) {
    return 0;
  }

  const values = match[1].split(',').map((value) => Number(value.trim()));
  return Math.abs(values[4] || 0);
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
