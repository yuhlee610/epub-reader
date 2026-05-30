package library

import "errors"

const (
	appDirName      = "epub-reader"
	booksDirName    = "books"
	libraryFileName = "library.json"
	libraryVersion  = 1
)

var (
	ErrBookNotFound       = errors.New("book not found")
	ErrInvalidBook        = errors.New("invalid book metadata")
	ErrPathOutsideLibrary = errors.New("path is outside managed book storage")
)
