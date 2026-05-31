import './App.css';
import {LibraryContent} from './components/LibraryContent';
import {LibraryToolbar} from './components/LibraryToolbar';
import {Notification} from './components/Notification';
import {useLibrary} from './hooks/useLibrary';

// App composes the library screen while hooks/components own workflow details.
function App() {
  const library = useLibrary();

  return (
    <main className="app-shell">
      <LibraryToolbar
        isImporting={library.isImporting}
        onImport={library.importBook}
        onQueryChange={library.setQuery}
        query={library.query}
      />

      <Notification message={library.message} error={library.error} />

      <LibraryContent
        activeBook={library.activeBook}
        books={library.books}
        isImporting={library.isImporting}
        isLoading={library.isLoading}
        isRemoving={library.isRemoving}
        onCloseBook={() => library.setActiveBook(null)}
        onImport={library.importBook}
        onOpenBook={library.openBook}
        onRemoveBook={library.removeBook}
        visibleBooks={library.visibleBooks}
      />
    </main>
  );
}

export default App;
