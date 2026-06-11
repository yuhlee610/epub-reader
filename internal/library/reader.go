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
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type packageSpineItem struct {
	IDRef string `xml:"idref,attr"`
}

type packageSpine struct {
	TOC   string             `xml:"toc,attr"`
	Items []packageSpineItem `xml:"itemref"`
}

type readerPackageXML struct {
	Manifest []packageManifestItem `xml:"manifest>item"`
	Spine    packageSpine          `xml:"spine"`
}

type ncxXML struct {
	NavMap struct {
		Points []ncxNavPoint `xml:"navPoint"`
	} `xml:"navMap"`
}

type ncxNavPoint struct {
	Label struct {
		Text string `xml:"text"`
	} `xml:"navLabel"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Children []ncxNavPoint `xml:"navPoint"`
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
	chapters := make([]ReaderChapter, 0, len(pkg.Spine.Items))
	baseDir := path.Dir(packagePath)
	if baseDir == "." {
		baseDir = ""
	}
	tocTitles := readTOCTitles(&reader.Reader, baseDir, pkg, manifestByID)

	for _, itemRef := range pkg.Spine.Items {
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
		if tocTitle := tocTitles[chapterPath]; tocTitle != "" {
			title = tocTitle
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

// readTOCTitles loads EPUB 3 nav or EPUB 2 NCX labels and indexes them by href.
func readTOCTitles(
	reader *zip.Reader,
	baseDir string,
	pkg readerPackageXML,
	manifestByID map[string]packageManifestItem,
) map[string]string {
	if titles := readNavigationTitles(reader, baseDir, pkg.Manifest); len(titles) > 0 {
		return titles
	}

	return readNCXTitles(reader, baseDir, pkg, manifestByID)
}

// readNavigationTitles extracts labels from an EPUB 3 navigation document.
func readNavigationTitles(
	reader *zip.Reader,
	baseDir string,
	items []packageManifestItem,
) map[string]string {
	for _, item := range items {
		if !manifestItemHasProperty(item, "nav") {
			continue
		}

		navPath, err := resolveEPUBPath(baseDir, item.Href)
		if err != nil {
			continue
		}
		data, err := readZipFile(reader, navPath)
		if err != nil {
			continue
		}

		return parseNavigationTitles(data, path.Dir(navPath))
	}

	return map[string]string{}
}

// parseNavigationTitles returns href labels from the document's TOC nav.
func parseNavigationTitles(data []byte, baseDir string) map[string]string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false

	titles := map[string]string{}
	var navDepth int
	var anchorDepth int
	var anchorHref string
	var anchorLabel strings.Builder

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch node := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(node.Name.Local)
			if navDepth == 0 {
				if name == "nav" && elementHasType(node, "toc") {
					navDepth = 1
				}
				continue
			}

			navDepth++
			if name == "a" && anchorDepth == 0 {
				anchorHref = attrValue(node, "href")
				anchorLabel.Reset()
				anchorDepth = 1
				continue
			}
			if anchorDepth > 0 {
				anchorDepth++
			}
		case xml.EndElement:
			if navDepth == 0 {
				continue
			}
			if anchorDepth > 0 {
				anchorDepth--
				if anchorDepth == 0 {
					addTOCTitle(titles, baseDir, anchorHref, anchorLabel.String())
					anchorHref = ""
				}
			}
			navDepth--
		case xml.CharData:
			if anchorDepth > 0 {
				anchorLabel.Write([]byte(node))
			}
		}
	}

	return titles
}

// readNCXTitles extracts labels from an EPUB 2 NCX table of contents.
func readNCXTitles(
	reader *zip.Reader,
	baseDir string,
	pkg readerPackageXML,
	manifestByID map[string]packageManifestItem,
) map[string]string {
	item, ok := manifestByID[strings.TrimSpace(pkg.Spine.TOC)]
	if !ok {
		for _, candidate := range pkg.Manifest {
			if strings.EqualFold(strings.TrimSpace(candidate.MediaType), "application/x-dtbncx+xml") {
				item = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return map[string]string{}
	}

	ncxPath, err := resolveEPUBPath(baseDir, item.Href)
	if err != nil {
		return map[string]string{}
	}
	data, err := readZipFile(reader, ncxPath)
	if err != nil {
		return map[string]string{}
	}

	var ncx ncxXML
	if err := xml.Unmarshal(data, &ncx); err != nil {
		return map[string]string{}
	}

	titles := map[string]string{}
	addNCXNavPoints(titles, path.Dir(ncxPath), ncx.NavMap.Points)
	return titles
}

// addNCXNavPoints flattens nested NCX points into the MVP's one-level TOC.
func addNCXNavPoints(titles map[string]string, baseDir string, points []ncxNavPoint) {
	for _, point := range points {
		addTOCTitle(titles, baseDir, point.Content.Src, point.Label.Text)
		addNCXNavPoints(titles, baseDir, point.Children)
	}
}

// addTOCTitle stores the first non-empty label for a normalized EPUB href.
func addTOCTitle(titles map[string]string, baseDir, href, label string) {
	label = cleanReaderText(label)
	if label == "" {
		return
	}

	resolved, err := resolveEPUBPath(baseDir, href)
	if err != nil {
		return
	}
	if _, exists := titles[resolved]; !exists {
		titles[resolved] = label
	}
}

func manifestItemHasProperty(item packageManifestItem, property string) bool {
	for _, value := range strings.Fields(strings.ToLower(item.Properties)) {
		if value == property {
			return true
		}
	}

	return false
}

func elementHasType(node xml.StartElement, value string) bool {
	for _, attr := range node.Attr {
		if strings.ToLower(attr.Name.Local) != "type" {
			continue
		}
		for _, token := range strings.Fields(strings.ToLower(attr.Value)) {
			if token == value || strings.TrimPrefix(token, "epub:") == value {
				return true
			}
		}
	}

	return false
}

func attrValue(node xml.StartElement, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Name.Local, name) {
			return attr.Value
		}
	}

	return ""
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
