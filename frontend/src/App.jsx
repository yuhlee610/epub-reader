import {useEffect, useState} from 'react';
import {CircleEllipsis, Maximize2, Menu, Minus, Plus, ScanLine, Search, X} from 'lucide-react';
import './App.css';
import {ImportEPUB, ListBooks} from "../wailsjs/go/main/App";
import {Quit, WindowMinimise, WindowToggleMaximise} from "../wailsjs/runtime/runtime";

// formatImportedAt renders backend timestamps with the user's locale settings.
function formatImportedAt(value) {
    if (!value) {
        return 'Unknown import date';
    }

    return new Intl.DateTimeFormat(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short',
    }).format(new Date(value));
}

// progressLabel formats the currently available persisted reading progress.
function progressLabel(book) {
    if (!book?.progress?.chapterIndex) {
        return '0%';
    }

    return `${Math.max(0, book.progress.chapterIndex)}%`;
}

// coverTitle keeps generated placeholder covers readable for books without art.
function coverTitle(title) {
    const words = (title || 'Untitled Book').split(/\s+/).filter(Boolean);
    return words.slice(0, 4).join(' ');
}

// formatImportError turns backend import failures into UI copy a reader can act on.
function formatImportError(err) {
    const message = err?.message ?? String(err);
    const lowerMessage = message.toLowerCase();

    if (lowerMessage.includes('invalid epub file')) {
        return 'That file does not look like a valid EPUB. Choose a .epub file and try again.';
    }
    if (lowerMessage.includes('copy epub file')) {
        return 'The book could not be copied into the local library. Check file access and try again.';
    }
    if (lowerMessage.includes('open epub picker')) {
        return 'The file picker could not be opened. Try importing again.';
    }

    return message;
}

// App renders the MVP library screen and bridges import actions to Wails.
function App() {
    const [books, setBooks] = useState([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isImporting, setIsImporting] = useState(false);
    const [query, setQuery] = useState('');
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');

    // loadBooks keeps the UI in sync with the persisted local metadata file.
    async function loadBooks() {
        setIsLoading(true);
        setError('');

        try {
            const nextBooks = await ListBooks();
            setBooks(nextBooks ?? []);
        } catch (err) {
            setError(err?.message ?? String(err));
        } finally {
            setIsLoading(false);
        }
    }

    // importBook opens the native picker and refreshes the library after success.
    async function importBook() {
        setIsImporting(true);
        setMessage('');
        setError('');

        try {
            const result = await ImportEPUB();
            if (result?.canceled) {
                setMessage('Import canceled.');
                return;
            }

            const title = result?.book?.title || 'Untitled book';
            setMessage(result?.duplicate ? `${title} is already in your library.` : `${title} imported.`);
            await loadBooks();
        } catch (err) {
            setError(formatImportError(err));
        } finally {
            setIsImporting(false);
        }
    }

    useEffect(() => {
        loadBooks();
    }, []);

    const visibleBooks = books.filter((book) => {
        const text = `${book.title ?? ''} ${book.author ?? ''} ${book.originalFileName ?? ''}`.toLowerCase();
        return text.includes(query.trim().toLowerCase());
    });

    return (
        <main className="app-shell">
            <header className="library-toolbar" aria-label="Library controls">
                <div className="search-shell">
                    <Search aria-hidden="true" className="search-icon" size={19} strokeWidth={2.75} />
                    <input
                        aria-label="Search books"
                        className="search-input"
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder="Search Books..."
                        type="search"
                        value={query}
                    />
                </div>

                <div className="window-actions" aria-label="Window controls">
                    <button className="window-tool" type="button" onClick={WindowMinimise} aria-label="Minimize">
                        <Minus aria-hidden="true" size={16} strokeWidth={2} />
                    </button>
                    <button className="window-tool" type="button" onClick={WindowToggleMaximise} aria-label="Maximize">
                        <Maximize2 aria-hidden="true" size={15} strokeWidth={2} />
                    </button>
                    <button className="window-tool" type="button" onClick={Quit} aria-label="Close">
                        <X aria-hidden="true" size={16} strokeWidth={2.2} />
                    </button>
                </div>
            </header>

            {message && <div className="status-message" role="status">{message}</div>}
            {error && <div className="error-message" role="alert">{error}</div>}

            <section className="library-content" aria-label="Imported books">
                {isLoading ? (
                    <div className="loading-row" role="status">Loading library...</div>
                ) : (
                    <div className="book-grid">
                        <ul className="book-list" aria-label="Books">
                            {visibleBooks.map((book) => (
                                <li className="book-card" key={book.id}>
                                    <button className="book-button" type="button" aria-label={`Open ${book.title || 'Untitled book'}`}>
                                        <span className="book-cover" aria-hidden="true">
                                            <span className="cover-name">{coverTitle(book.title)}</span>
                                            <span className="cover-author">{book.author || 'Unknown author'}</span>
                                        </span>
                                        <span className="book-title">{book.title || 'Untitled book'}</span>
                                    </button>
                                    <span className="book-details">
                                        <span>{progressLabel(book)}</span>
                                        <span className="cloud-mark" aria-label={`Imported ${formatImportedAt(book.importedAt)}`} title={formatImportedAt(book.importedAt)} />
                                    </span>
                                </li>
                            ))}
                        </ul>

                        <button className="add-book-tile" type="button" onClick={importBook} disabled={isImporting} aria-label="Import EPUB">
                            <Plus aria-hidden="true" size={46} strokeWidth={1.25} />
                        </button>

                        {books.length > 0 && visibleBooks.length === 0 && (
                            <div className="empty-search" role="status">
                                No books match this search.
                            </div>
                        )}
                    </div>
                )}
            </section>
        </main>
    )
}

export default App
