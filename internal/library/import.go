package library

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidEPUB = errors.New("invalid epub file")
	ErrNoFile      = errors.New("no file selected")
)

// ImportResult describes the outcome of importing one EPUB file.
type ImportResult struct {
	Book      BookMetadata `json:"book"`
	Duplicate bool         `json:"duplicate"`
}

// importCandidate carries validated source data across the copy/persist steps.
type importCandidate struct {
	SourcePath       string
	OriginalFileName string
	Metadata         epubMetadata
	Hash             string
	BookID           string
	Title            string
}

// ImportEPUBFromPath validates, copies, and records one EPUB in managed storage.
func (s *Store) ImportEPUBFromPath(sourcePath string) (ImportResult, error) {
	candidate, err := newImportCandidate(sourcePath)
	if err != nil {
		return ImportResult{}, err
	}

	return s.saveImportedEPUB(candidate)
}

// newImportCandidate validates the selected file and derives import metadata.
func newImportCandidate(sourcePath string) (importCandidate, error) {
	sourcePath, err := cleanImportSourcePath(sourcePath)
	if err != nil {
		return importCandidate{}, err
	}

	metadata, err := inspectEPUB(sourcePath)
	if err != nil {
		return importCandidate{}, err
	}

	hash, err := fileSHA256(sourcePath)
	if err != nil {
		return importCandidate{}, fmt.Errorf("hash epub file: %w", err)
	}

	title := metadata.Title
	if title == "" {
		title = titleFromFilename(sourcePath)
	}

	return importCandidate{
		SourcePath:       sourcePath,
		OriginalFileName: filepath.Base(sourcePath),
		Metadata:         metadata,
		Hash:             hash,
		BookID:           "sha256-" + hash[:32],
		Title:            title,
	}, nil
}

// cleanImportSourcePath normalizes the user-selected source path.
func cleanImportSourcePath(sourcePath string) (string, error) {
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "" || sourcePath == "." {
		return "", ErrNoFile
	}
	if !strings.EqualFold(filepath.Ext(sourcePath), ".epub") {
		return "", fmt.Errorf("%w: file must use .epub extension", ErrInvalidEPUB)
	}

	return sourcePath, nil
}

// saveImportedEPUB persists a prepared EPUB import while holding the store lock.
func (s *Store) saveImportedEPUB(candidate importCandidate) (ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return ImportResult{}, err
	}

	if existing, ok := findBookByID(file.Books, candidate.BookID); ok {
		return restoreDuplicateImport(candidate, existing)
	}

	return s.addImportedBook(file, candidate)
}

// restoreDuplicateImport repairs the managed EPUB copy and returns existing metadata.
func restoreDuplicateImport(candidate importCandidate, existing BookMetadata) (ImportResult, error) {
	if err := ensureCopied(candidate.SourcePath, existing.FilePath, candidate.Hash); err != nil {
		return ImportResult{}, err
	}

	return ImportResult{Book: existing, Duplicate: true}, nil
}

// addImportedBook copies a new EPUB and appends its metadata to the library file.
func (s *Store) addImportedBook(file libraryFile, candidate importCandidate) (ImportResult, error) {
	targetPath, err := s.ManagedBookPath(candidate.BookID, candidate.OriginalFileName)
	if err != nil {
		return ImportResult{}, err
	}
	if err := copyFileExclusive(candidate.SourcePath, targetPath); err != nil {
		return ImportResult{}, err
	}

	book, err := s.prepareBook(BookMetadata{
		ID:               candidate.BookID,
		Title:            candidate.Title,
		Author:           candidate.Metadata.Author,
		OriginalFileName: candidate.OriginalFileName,
		FilePath:         targetPath,
	}, nil)
	if err != nil {
		return ImportResult{}, err
	}

	file.Books = append(file.Books, book)
	if err := s.writeLibrary(file); err != nil {
		// If metadata persistence fails, remove the new managed copy so storage
		// does not accumulate files that the library cannot show or recover.
		_ = os.Remove(targetPath)
		return ImportResult{}, err
	}

	return ImportResult{Book: book}, nil
}

// findBookByID returns the first book whose stable ID matches the import ID.
func findBookByID(books []BookMetadata, id string) (BookMetadata, bool) {
	for _, book := range books {
		if book.ID == id {
			return book, true
		}
	}

	return BookMetadata{}, false
}

type epubMetadata struct {
	Title  string
	Author string
}

type containerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type packageXML struct {
	Title   string `xml:"metadata>title"`
	Creator string `xml:"metadata>creator"`
}

// inspectEPUB verifies the EPUB container shape and extracts basic metadata.
func inspectEPUB(path string) (epubMetadata, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return epubMetadata{}, fmt.Errorf("%w: open zip container: %v", ErrInvalidEPUB, err)
	}
	defer reader.Close()

	if err := validateMimetype(&reader.Reader); err != nil {
		return epubMetadata{}, err
	}

	containerData, err := readZipFile(&reader.Reader, "META-INF/container.xml")
	if err != nil {
		return epubMetadata{}, fmt.Errorf("%w: missing META-INF/container.xml", ErrInvalidEPUB)
	}

	var container containerXML
	if err := xml.Unmarshal(containerData, &container); err != nil {
		return epubMetadata{}, fmt.Errorf("%w: decode container.xml: %v", ErrInvalidEPUB, err)
	}
	if len(container.Rootfiles) == 0 || strings.TrimSpace(container.Rootfiles[0].FullPath) == "" {
		return epubMetadata{}, fmt.Errorf("%w: container.xml has no rootfile", ErrInvalidEPUB)
	}

	packageData, err := readZipFile(&reader.Reader, container.Rootfiles[0].FullPath)
	if err != nil {
		return epubMetadata{}, fmt.Errorf("%w: missing package document", ErrInvalidEPUB)
	}

	var pkg packageXML
	if err := xml.Unmarshal(packageData, &pkg); err != nil {
		return epubMetadata{}, fmt.Errorf("%w: decode package document: %v", ErrInvalidEPUB, err)
	}

	return epubMetadata{
		Title:  strings.TrimSpace(pkg.Title),
		Author: strings.TrimSpace(pkg.Creator),
	}, nil
}

// validateMimetype checks the required EPUB mimetype marker.
func validateMimetype(reader *zip.Reader) error {
	data, err := readZipFile(reader, "mimetype")
	if err != nil {
		return fmt.Errorf("%w: missing mimetype", ErrInvalidEPUB)
	}
	if !bytes.Equal(bytes.TrimSpace(data), []byte("application/epub+zip")) {
		return fmt.Errorf("%w: unexpected mimetype", ErrInvalidEPUB)
	}

	return nil
}

// readZipFile returns the contents of one file inside a ZIP archive.
func readZipFile(reader *zip.Reader, name string) ([]byte, error) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}

		body, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer body.Close()

		return io.ReadAll(body)
	}

	return nil, os.ErrNotExist
}

// fileSHA256 returns the hex SHA-256 digest for a file.
func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ensureCopied restores the managed copy if metadata exists but the file is
// missing or does not match the expected source hash.
func ensureCopied(sourcePath, targetPath, expectedHash string) error {
	if _, err := os.Stat(targetPath); err == nil {
		targetHash, err := fileSHA256(targetPath)
		if err != nil {
			return fmt.Errorf("hash managed epub copy: %w", err)
		}
		if targetHash == expectedHash {
			return nil
		}
		// The metadata ID is content-derived, so a mismatched managed copy is
		// stale or corrupted. Replace it with the selected source file.
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("remove stale managed epub copy: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect managed epub copy: %w", err)
	}

	return copyFileExclusive(sourcePath, targetPath)
}

// copyFileExclusive copies sourcePath to targetPath without overwriting files.
func copyFileExclusive(sourcePath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create managed epub directory: %w", err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source epub: %w", err)
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("create managed epub copy: %w", err)
	}

	copied := false
	defer func() {
		if !copied {
			_ = os.Remove(targetPath)
		}
	}()

	// A failed copy can leave a partial EPUB; remove it so future imports do not
	// mistake the path for a valid managed copy.
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return fmt.Errorf("copy epub into managed storage: %w", err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("close managed epub copy: %w", err)
	}
	copied = true

	return nil
}

// titleFromFilename derives a readable fallback title from the source filename.
func titleFromFilename(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name = strings.TrimSpace(strings.ReplaceAll(name, "_", " "))
	if name == "" {
		return "Untitled Book"
	}

	return name
}
