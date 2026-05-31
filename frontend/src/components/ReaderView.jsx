import {
  Bookmark,
  ChevronLeft,
  ChevronRight,
  Edit3,
  Info,
  List,
  Menu,
  PanelLeftClose,
  Pin,
  Search,
  X,
} from 'lucide-react';
import {coverTitle} from '../lib/bookFormatters';

export function ReaderView({
  chapter,
  chapterIndex,
  isLoading,
  onClose,
  onNext,
  onPrevious,
  onSelectChapter,
  readerBook,
}) {
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

  const chapterCount = readerBook.chapters.length;
  const progressLabel = `${chapterIndex + 1} / ${chapterCount}`;

  return (
    <main className="reader-shell">
      <ReaderSidebar
        activeChapterIndex={chapterIndex}
        book={readerBook.book}
        chapters={readerBook.chapters}
        onClose={onClose}
        onSelectChapter={onSelectChapter}
      />

      <section className="reader-stage" aria-label="Reader">
        <div className="reader-topline">
          <span>{chapter.title}</span>
        </div>

        <article className="reader-page" aria-label={chapter.title}>
          <div
            className="reader-chapter-content"
            dangerouslySetInnerHTML={{__html: chapter.bodyHtml}}
          />
        </article>

        <div className="reader-bottom-bar" aria-label="Reader navigation">
          <div className="reader-nav-buttons">
            <button
              aria-label="Previous chapter"
              className="reader-nav-button"
              disabled={chapterIndex === 0}
              onClick={onPrevious}
              type="button"
            >
              <ChevronLeft aria-hidden="true" size={18} strokeWidth={2} />
            </button>
            <button
              aria-label="Next chapter"
              className="reader-nav-button"
              disabled={chapterIndex >= chapterCount - 1}
              onClick={onNext}
              type="button"
            >
              <ChevronRight aria-hidden="true" size={18} strokeWidth={2} />
            </button>
          </div>
          <span>{progressLabel}</span>
        </div>
      </section>
    </main>
  );
}

function ReaderSidebar({
  activeChapterIndex,
  book,
  chapters,
  onClose,
  onSelectChapter,
}) {
  return (
    <aside className="reader-sidebar" aria-label="Reader table of contents">
      <div className="reader-sidebar-toolbar">
        <button
          aria-label="Back to library"
          className="reader-icon-button"
          onClick={onClose}
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
