import {Maximize2, Minus, Plus, ScanLine, Search, X} from 'lucide-react';
import {
  Quit,
  WindowMinimise,
  WindowToggleMaximise,
} from '../../wailsjs/runtime/runtime';

export function LibraryToolbar({isImporting, onImport, onQueryChange, query}) {
  return (
    <header className="library-toolbar" aria-label="Library controls">
      <div className="search-shell">
        <Search
          aria-hidden="true"
          className="search-icon"
          size={19}
          strokeWidth={2.75}
        />
        <input
          aria-label="Search books"
          className="search-input"
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Search Books..."
          type="search"
          value={query}
        />
        <button
          aria-label="Import EPUB"
          className="toolbar-icon"
          disabled={isImporting}
          onClick={onImport}
          type="button"
        >
          <Plus aria-hidden="true" size={19} strokeWidth={1.8} />
        </button>
        <button aria-label="Scan library" className="toolbar-icon" disabled type="button">
          <ScanLine aria-hidden="true" size={20} strokeWidth={1.75} />
        </button>
      </div>

      <div className="window-actions" aria-label="Window controls">
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
    </header>
  );
}
