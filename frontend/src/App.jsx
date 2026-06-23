import './App.css';
import {LibraryContent} from './components/LibraryContent';
import {LibraryToolbar} from './components/LibraryToolbar';
import {Notification} from './components/Notification';
import {ReaderView} from './components/ReaderView';
import {useLibrary} from './hooks/useLibrary';
import {useReader} from './hooks/useReader';

// App composes the library screen while hooks/components own workflow details.
function App() {
  const library = useLibrary();
  const reader = useReader();

  if (reader.readerBook || reader.isReaderLoading) {
    return (
      <>
        <Notification message="" error={reader.readerError} />
        <ReaderView
          chapter={reader.currentChapter}
          chapterIndex={reader.currentChapterIndex}
          initialPageIndex={reader.currentPageIndex}
          isLoading={reader.isReaderLoading}
          onClose={async () => {
            await reader.closeReader();
            library.loadBooks();
          }}
          onNext={reader.goToNextChapter}
          onAppearanceChange={reader.saveReaderAppearance}
          onDeleteBookmark={reader.deleteReaderBookmark}
          onDeleteNote={reader.deleteReaderNote}
          onPageChange={reader.saveCurrentPage}
          onPrevious={reader.goToPreviousChapter}
          onSaveBookmark={reader.saveReaderBookmark}
          onSaveNote={reader.saveReaderNote}
          onSelectChapter={reader.goToChapter}
          readerBook={reader.readerBook}
        />
      </>
    );
  }

  return (
    <main className="app-shell">
      <LibraryToolbar
        isImporting={library.isImporting}
        onImport={library.importBook}
        onQueryChange={library.setQuery}
        query={library.query}
      />

      <Notification message={library.message} error={library.error || reader.readerError} />

      <LibraryContent
        activeBook={library.activeBook}
        books={library.books}
        isImporting={library.isImporting}
        isLoading={library.isLoading}
        isRemoving={library.isRemoving}
        isSavingPrompt={library.isSavingPrompt}
        onCloseBook={() => library.setActiveBook(null)}
        onImport={library.importBook}
        onOpenBook={library.openBook}
        onReadBook={reader.openReader}
        onRemoveBook={library.removeBook}
        onSavePrompt={library.saveBookPrompt}
        visibleBooks={library.visibleBooks}
      />
    </main>
  );
}

export default App;
