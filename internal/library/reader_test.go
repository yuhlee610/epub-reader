package library

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadReaderBookReturnsSpineChapters verifies managed EPUB content is
// loaded from the imported copy and ordered by the package spine.
func TestLoadReaderBookReturnsSpineChapters(t *testing.T) {
	store := newTestStore(t)
	source := writeReaderTestEPUB(t, t.TempDir(), "cradle.epub")

	imported, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatalf("remove original fixture: %v", err)
	}

	readerBook, err := store.LoadReaderBook(imported.Book.ID)
	if err != nil {
		t.Fatalf("LoadReaderBook() error = %v", err)
	}
	if readerBook.Book.ID != imported.Book.ID {
		t.Fatalf("Book.ID = %q, want %q", readerBook.Book.ID, imported.Book.ID)
	}
	if len(readerBook.Chapters) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(readerBook.Chapters))
	}
	if readerBook.Chapters[0].Title != "Chapter One" {
		t.Fatalf("first chapter title = %q, want Chapter One", readerBook.Chapters[0].Title)
	}
	if !strings.Contains(readerBook.Chapters[0].BodyHTML, "<p>The first lesson.</p>") {
		t.Fatalf("first chapter HTML = %q, want sanitized paragraph", readerBook.Chapters[0].BodyHTML)
	}
}

// TestLoadReaderBookUsesNavigationTitles verifies EPUB 3 nav labels are used
// for reader TOC entries instead of generic spine document titles.
func TestLoadReaderBookUsesNavigationTitles(t *testing.T) {
	store := newTestStore(t)
	source := writeReaderNavigationTestEPUB(t, t.TempDir(), "nav.epub")

	imported, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}

	readerBook, err := store.LoadReaderBook(imported.Book.ID)
	if err != nil {
		t.Fatalf("LoadReaderBook() error = %v", err)
	}
	if len(readerBook.Chapters) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(readerBook.Chapters))
	}
	if readerBook.Chapters[0].Title != "The Empty Badge" {
		t.Fatalf("first chapter title = %q, want The Empty Badge", readerBook.Chapters[0].Title)
	}
	if readerBook.Chapters[1].Title != "Sacred Valley" {
		t.Fatalf("second chapter title = %q, want Sacred Valley", readerBook.Chapters[1].Title)
	}
}

// TestLoadReaderBookUsesNCXTitles verifies EPUB 2 NCX labels are accepted as
// TOC data when an EPUB 3 nav document is not present.
func TestLoadReaderBookUsesNCXTitles(t *testing.T) {
	store := newTestStore(t)
	source := writeReaderNCXTestEPUB(t, t.TempDir(), "ncx.epub")

	imported, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}

	readerBook, err := store.LoadReaderBook(imported.Book.ID)
	if err != nil {
		t.Fatalf("LoadReaderBook() error = %v", err)
	}
	if len(readerBook.Chapters) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(readerBook.Chapters))
	}
	if readerBook.Chapters[0].Title != "Unsouled" {
		t.Fatalf("first chapter title = %q, want Unsouled", readerBook.Chapters[0].Title)
	}
	if readerBook.Chapters[1].Title != "The Trial" {
		t.Fatalf("second chapter title = %q, want The Trial", readerBook.Chapters[1].Title)
	}
}

// TestLoadReaderBookUsesPersistedProgress verifies reopening starts near the
// last saved chapter.
func TestLoadReaderBookUsesPersistedProgress(t *testing.T) {
	store := newTestStore(t)
	source := writeReaderTestEPUB(t, t.TempDir(), "cradle.epub")
	imported, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}

	_, err = store.UpdateReadingProgress(imported.Book.ID, ReadingProgress{
		ChapterHref:  "OPS/chapter-2.xhtml",
		ChapterIndex: 1,
	})
	if err != nil {
		t.Fatalf("UpdateReadingProgress() error = %v", err)
	}

	readerBook, err := store.LoadReaderBook(imported.Book.ID)
	if err != nil {
		t.Fatalf("LoadReaderBook() error = %v", err)
	}
	if readerBook.CurrentChapterIndex != 1 {
		t.Fatalf("CurrentChapterIndex = %d, want 1", readerBook.CurrentChapterIndex)
	}
}

// TestChapterDisplayTitleFallsBackFromBookTitle verifies TOC labels avoid
// repeating the book title for every spine item.
func TestChapterDisplayTitleFallsBackFromBookTitle(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		bookTitle   string
		chapterPath string
		index       int
		want        string
	}{
		{
			name:        "cover",
			title:       "Soulsmith",
			bookTitle:   "Soulsmith",
			chapterPath: "OPS/cover.xhtml",
			want:        "Cover",
		},
		{
			name:        "contents",
			title:       "Soulsmith",
			bookTitle:   "Soulsmith",
			chapterPath: "OPS/nav.xhtml",
			want:        "Contents",
		},
		{
			name:        "numbered chapter",
			title:       "Soulsmith",
			bookTitle:   "Soulsmith",
			chapterPath: "OPS/chapter-7.xhtml",
			want:        "Chapter 7",
		},
		{
			name:        "real title",
			title:       "The Trial",
			bookTitle:   "Soulsmith",
			chapterPath: "OPS/chapter-7.xhtml",
			want:        "The Trial",
		},
		{
			name:        "short book title",
			title:       "Soulsmith",
			bookTitle:   "Soulsmith (Cradle Book 2)",
			chapterPath: "OPS/chapter-12.xhtml",
			want:        "Chapter 12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chapterDisplayTitle(tt.title, tt.bookTitle, tt.chapterPath, tt.index)
			if got != tt.want {
				t.Fatalf("chapterDisplayTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLoadReaderBookReportsMalformedManagedCopy verifies reader errors are
// surfaced as invalid EPUB failures.
func TestLoadReaderBookReportsMalformedManagedCopy(t *testing.T) {
	store := newTestStore(t)
	path, err := store.ManagedBookPath("book-1", "bad.epub")
	if err != nil {
		t.Fatalf("ManagedBookPath() error = %v", err)
	}

	book, err := store.SaveBook(BookMetadata{
		ID:               "book-1",
		Title:            "Bad Book",
		OriginalFileName: "bad.epub",
		FilePath:         path,
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create managed dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("not an epub"), 0o644); err != nil {
		t.Fatalf("write malformed managed copy: %v", err)
	}

	if _, err := store.LoadReaderBook(book.ID); !errors.Is(err, ErrInvalidEPUB) {
		t.Fatalf("LoadReaderBook() error = %v, want ErrInvalidEPUB", err)
	}
}

func writeReaderTestEPUB(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub fixture: %v", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	mustWriteZipFile(t, archive, "mimetype", "application/epub+zip")
	mustWriteZipFile(t, archive, "META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OPS/package.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	mustWriteZipFile(t, archive, "OPS/package.opf", `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Cradle</dc:title>
    <dc:creator>Will Wight</dc:creator>
  </metadata>
  <manifest>
    <item id="chapter-1" href="chapter-1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter-2" href="chapter-2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter-1"/>
    <itemref idref="chapter-2"/>
  </spine>
</package>`)
	mustWriteZipFile(t, archive, "OPS/chapter-1.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Chapter One</title></head>
  <body><h1>Chapter One</h1><p>The first lesson.</p><script>bad()</script></body>
</html>`)
	mustWriteZipFile(t, archive, "OPS/chapter-2.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Chapter Two</title></head>
  <body><h1>Chapter Two</h1><p>The second lesson.</p></body>
</html>`)
	if err := archive.Close(); err != nil {
		t.Fatalf("close epub fixture: %v", err)
	}

	return path
}

func writeReaderNavigationTestEPUB(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub fixture: %v", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	mustWriteZipFile(t, archive, "mimetype", "application/epub+zip")
	mustWriteZipFile(t, archive, "META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OPS/package.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	mustWriteZipFile(t, archive, "OPS/package.opf", `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Cradle</dc:title>
    <dc:creator>Will Wight</dc:creator>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="chapter-1" href="chapter-1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter-2" href="chapter-2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter-1"/>
    <itemref idref="chapter-2"/>
  </spine>
</package>`)
	mustWriteZipFile(t, archive, "OPS/nav.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
  <body>
    <nav epub:type="toc">
      <ol>
        <li><a href="chapter-1.xhtml#start">The Empty Badge</a></li>
        <li><a href="chapter-2.xhtml">Sacred Valley</a></li>
      </ol>
    </nav>
  </body>
</html>`)
	mustWriteZipFile(t, archive, "OPS/chapter-1.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Cradle</title></head>
  <body><h1>Cradle</h1><p>The first lesson.</p></body>
</html>`)
	mustWriteZipFile(t, archive, "OPS/chapter-2.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Cradle</title></head>
  <body><h1>Cradle</h1><p>The second lesson.</p></body>
</html>`)
	if err := archive.Close(); err != nil {
		t.Fatalf("close epub fixture: %v", err)
	}

	return path
}

func writeReaderNCXTestEPUB(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub fixture: %v", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	mustWriteZipFile(t, archive, "mimetype", "application/epub+zip")
	mustWriteZipFile(t, archive, "META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OPS/package.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	mustWriteZipFile(t, archive, "OPS/package.opf", `<?xml version="1.0"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Cradle</dc:title>
    <dc:creator>Will Wight</dc:creator>
  </metadata>
  <manifest>
    <item id="toc" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="chapter-1" href="chapter-1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter-2" href="chapter-2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="toc">
    <itemref idref="chapter-1"/>
    <itemref idref="chapter-2"/>
  </spine>
</package>`)
	mustWriteZipFile(t, archive, "OPS/toc.ncx", `<?xml version="1.0"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/">
  <navMap>
    <navPoint id="chapter-1" playOrder="1">
      <navLabel><text>Unsouled</text></navLabel>
      <content src="chapter-1.xhtml"/>
    </navPoint>
    <navPoint id="chapter-2" playOrder="2">
      <navLabel><text>The Trial</text></navLabel>
      <content src="chapter-2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)
	mustWriteZipFile(t, archive, "OPS/chapter-1.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Cradle</title></head>
  <body><h1>Cradle</h1><p>The first lesson.</p></body>
</html>`)
	mustWriteZipFile(t, archive, "OPS/chapter-2.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Cradle</title></head>
  <body><h1>Cradle</h1><p>The second lesson.</p></body>
</html>`)
	if err := archive.Close(); err != nil {
		t.Fatalf("close epub fixture: %v", err)
	}

	return path
}
