package library

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type packageManifestItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

type packageSpineItem struct {
	IDRef string `xml:"idref,attr"`
}

type readerPackageXML struct {
	Manifest []packageManifestItem `xml:"manifest>item"`
	Spine    []packageSpineItem    `xml:"spine>itemref"`
}

// LoadReaderBook opens the managed EPUB and returns readable chapter content.
func (s *Store) LoadReaderBook(id string) (ReaderBook, error) {
	book, err := s.GetBook(id)
	if err != nil {
		return ReaderBook{}, err
	}

	chapters, err := readEPUBChapters(book.FilePath, book.Title)
	if err != nil {
		return ReaderBook{}, err
	}
	if len(chapters) == 0 {
		return ReaderBook{}, fmt.Errorf("%w: no readable chapters", ErrInvalidEPUB)
	}

	return ReaderBook{
		Book:                book,
		Chapters:            chapters,
		CurrentChapterIndex: currentChapterIndex(book.Progress, chapters),
	}, nil
}

// readEPUBChapters follows the EPUB package spine and extracts readable XHTML.
func readEPUBChapters(epubPath, bookTitle string) ([]ReaderChapter, error) {
	reader, err := zip.OpenReader(epubPath)
	if err != nil {
		return nil, fmt.Errorf("%w: open zip container: %v", ErrInvalidEPUB, err)
	}
	defer reader.Close()

	if err := validateMimetype(&reader.Reader); err != nil {
		return nil, err
	}

	packagePath, err := packageDocumentPath(&reader.Reader)
	if err != nil {
		return nil, err
	}

	packageData, err := readZipFile(&reader.Reader, packagePath)
	if err != nil {
		return nil, fmt.Errorf("%w: missing package document", ErrInvalidEPUB)
	}

	var pkg readerPackageXML
	if err := xml.Unmarshal(packageData, &pkg); err != nil {
		return nil, fmt.Errorf("%w: decode package document: %v", ErrInvalidEPUB, err)
	}

	manifestByID := mapManifestByID(pkg.Manifest)
	chapters := make([]ReaderChapter, 0, len(pkg.Spine))
	baseDir := path.Dir(packagePath)
	if baseDir == "." {
		baseDir = ""
	}

	for _, itemRef := range pkg.Spine {
		item, ok := manifestByID[itemRef.IDRef]
		if !ok || !isReadableSpineItem(item) {
			continue
		}

		chapterPath, err := resolveEPUBPath(baseDir, item.Href)
		if err != nil {
			return nil, err
		}
		data, err := readZipFile(&reader.Reader, chapterPath)
		if err != nil {
			return nil, fmt.Errorf("%w: missing chapter %s", ErrInvalidEPUB, chapterPath)
		}

		bodyHTML, title, err := sanitizeChapterXHTML(data)
		if err != nil {
			return nil, fmt.Errorf("%w: parse chapter %s: %v", ErrInvalidEPUB, chapterPath, err)
		}
		if strings.TrimSpace(bodyHTML) == "" {
			continue
		}
		title = chapterDisplayTitle(title, bookTitle, chapterPath, len(chapters))

		chapters = append(chapters, ReaderChapter{
			Index:    len(chapters),
			Href:     chapterPath,
			Title:    title,
			BodyHTML: bodyHTML,
		})
	}

	return chapters, nil
}

// packageDocumentPath reads META-INF/container.xml and returns the OPF path.
func packageDocumentPath(reader *zip.Reader) (string, error) {
	containerData, err := readZipFile(reader, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("%w: missing META-INF/container.xml", ErrInvalidEPUB)
	}

	var container containerXML
	if err := xml.Unmarshal(containerData, &container); err != nil {
		return "", fmt.Errorf("%w: decode container.xml: %v", ErrInvalidEPUB, err)
	}
	if len(container.Rootfiles) == 0 || strings.TrimSpace(container.Rootfiles[0].FullPath) == "" {
		return "", fmt.Errorf("%w: container.xml has no rootfile", ErrInvalidEPUB)
	}

	return filepath.ToSlash(strings.TrimSpace(container.Rootfiles[0].FullPath)), nil
}

// mapManifestByID builds the lookup used by spine itemrefs.
func mapManifestByID(items []packageManifestItem) map[string]packageManifestItem {
	manifestByID := make(map[string]packageManifestItem, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		manifestByID[id] = item
	}

	return manifestByID
}

// isReadableSpineItem keeps the MVP reader focused on XHTML/HTML text content.
func isReadableSpineItem(item packageManifestItem) bool {
	mediaType := strings.ToLower(strings.TrimSpace(item.MediaType))
	return mediaType == "application/xhtml+xml" || mediaType == "text/html"
}

// resolveEPUBPath joins package-relative hrefs without allowing ZIP traversal.
func resolveEPUBPath(baseDir, href string) (string, error) {
	href = strings.TrimSpace(strings.Split(href, "#")[0])
	if href == "" {
		return "", fmt.Errorf("%w: empty chapter href", ErrInvalidEPUB)
	}
	if decoded, err := url.PathUnescape(href); err == nil {
		href = decoded
	}

	resolved := path.Clean(path.Join(baseDir, filepath.ToSlash(href)))
	if resolved == "." || strings.HasPrefix(resolved, "../") || resolved == ".." {
		return "", fmt.Errorf("%w: unsafe chapter href %q", ErrInvalidEPUB, href)
	}

	return resolved, nil
}

// sanitizeChapterXHTML extracts body content and keeps a small safe HTML subset.
func sanitizeChapterXHTML(data []byte) (string, string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false

	var body strings.Builder
	var title strings.Builder
	var titleDepth int
	var bodyDepth int
	var skipDepth int
	var openTags []string
	var bestTitle string
	var headingTitle string

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", "", err
		}

		switch node := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(node.Name.Local)
			if shouldCaptureTitle(name, bodyDepth, bestTitle, headingTitle) {
				title.Reset()
				titleDepth = 1
			} else if titleDepth > 0 {
				titleDepth++
			}

			if name == "body" && bodyDepth == 0 {
				bodyDepth = 1
				continue
			}
			if bodyDepth == 0 {
				continue
			}
			bodyDepth++

			if skipDepth > 0 || isSkippedReaderTag(name) {
				skipDepth++
				openTags = append(openTags, "")
				continue
			}

			tag := readerTagName(name)
			openTags = append(openTags, tag)
			if tag != "" {
				body.WriteString("<")
				body.WriteString(tag)
				body.WriteString(">")
			}
		case xml.EndElement:
			name := strings.ToLower(node.Name.Local)
			if titleDepth > 0 {
				titleDepth--
				if titleDepth == 0 {
					captured := cleanReaderText(title.String())
					if name == "title" {
						bestTitle = captured
					} else if headingTitle == "" {
						headingTitle = captured
					}
				}
			}

			if bodyDepth == 0 {
				continue
			}
			if name == "body" && bodyDepth == 1 {
				bodyDepth = 0
				continue
			}
			if len(openTags) > 0 {
				tag := openTags[len(openTags)-1]
				openTags = openTags[:len(openTags)-1]
				if skipDepth > 0 {
					skipDepth--
				} else if tag != "" {
					body.WriteString("</")
					body.WriteString(tag)
					body.WriteString(">")
				}
			}
			if bodyDepth > 0 {
				bodyDepth--
			}
		case xml.CharData:
			if titleDepth > 0 {
				title.Write([]byte(node))
			}
			if bodyDepth > 0 && skipDepth == 0 {
				body.WriteString(html.EscapeString(string(node)))
			}
		}
	}

	if headingTitle != "" {
		return body.String(), headingTitle, nil
	}

	return body.String(), bestTitle, nil
}

func shouldCaptureTitle(name string, bodyDepth int, documentTitle, headingTitle string) bool {
	if name == "title" && documentTitle == "" {
		return true
	}

	return bodyDepth > 0 && headingTitle == "" && (name == "h1" || name == "h2")
}

func readerTagName(name string) string {
	switch name {
	case "article", "blockquote", "div", "em", "h1", "h2", "h3", "h4", "h5", "h6",
		"i", "li", "ol", "p", "section", "span", "strong", "ul":
		return name
	case "br", "hr":
		return name
	default:
		return ""
	}
}

func isSkippedReaderTag(name string) bool {
	return name == "script" || name == "style" || name == "svg" || name == "math"
}

func cleanReaderText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func titleFromChapterPath(chapterPath string, index int) string {
	name := strings.TrimSuffix(path.Base(chapterPath), path.Ext(chapterPath))
	lowerName := strings.ToLower(name)
	if lowerName == "cover" || lowerName == "titlepage" || lowerName == "title-page" {
		return "Cover"
	}
	if lowerName == "toc" || lowerName == "contents" || lowerName == "nav" {
		return "Contents"
	}
	if number, ok := trailingNumber(lowerName); ok {
		return fmt.Sprintf("Chapter %d", number)
	}

	name = strings.Trim(strings.NewReplacer("_", " ", "-", " ").Replace(name), " ")
	if name == "" {
		return fmt.Sprintf("Chapter %d", index+1)
	}

	return name
}

func chapterDisplayTitle(title, bookTitle, chapterPath string, index int) string {
	title = cleanReaderText(title)
	if title == "" || sameReaderTitle(title, bookTitle) {
		return titleFromChapterPath(chapterPath, index)
	}

	return title
}

func sameReaderTitle(a, b string) bool {
	shortTitle := strings.ToLower(cleanReaderText(a))
	fullTitle := strings.ToLower(cleanReaderText(b))
	return shortTitle == fullTitle ||
		strings.HasPrefix(fullTitle, shortTitle+" ") ||
		strings.HasPrefix(fullTitle, shortTitle+"(")
}

func trailingNumber(value string) (int, bool) {
	end := len(value)
	start := end
	for start > 0 && value[start-1] >= '0' && value[start-1] <= '9' {
		start--
	}
	if start == end {
		return 0, false
	}
	number, err := strconv.Atoi(value[start:end])
	return number, err == nil
}

func currentChapterIndex(progress ReadingProgress, chapters []ReaderChapter) int {
	for i, chapter := range chapters {
		if progress.ChapterHref != "" && chapter.Href == progress.ChapterHref {
			return i
		}
	}
	if progress.ChapterIndex >= 0 && progress.ChapterIndex < len(chapters) {
		return progress.ChapterIndex
	}

	return 0
}
