package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	nurl "net/url"
	"strings"

	"github.com/efixler/scrape/resource"
)

var (
	ErrUnsupportedContentType = &UnsupportedContentTypeError{
		ErrHTTPError{http.StatusUnsupportedMediaType, "Unsupported content type"},
		"",
	}
)

type Factory func() (URLData, error)

type URLData interface {
	Open(context.Context) error
	// Store(*StoredUrlData) (uint64, error)
	Fetch(*nurl.URL) (*resource.WebPage, error)
	Close() error
}
type ErrHTTPError struct {
	StatusCode int
	Message    string
}

func NewErrHTTPError(statusCode int, body io.Reader) ErrHTTPError {
	rval := ErrHTTPError{
		StatusCode: statusCode,
	}

	if message, err := io.ReadAll(body); err != nil {
		rval.Message = http.StatusText(statusCode)
	} else {
		rval.Message = strings.TrimSpace(string(message))
	}
	return rval
}

func (e ErrHTTPError) Error() string {
	return fmt.Sprintf("HTTP fetch error: %s", e.Message)
}

func (e ErrHTTPError) String() string {
	return e.Error()
}

type UnsupportedContentTypeError struct {
	ErrHTTPError
	ContentType string
}

// Makes Is(err, ErrUnsupportedContentType) return true for any instance of UnsupportedContentTypeError
func (e UnsupportedContentTypeError) Is(target error) bool {
	_, ok := target.(*UnsupportedContentTypeError)
	return ok
}

func NewUnsupportedContentTypeError(contentType string) *UnsupportedContentTypeError {
	rval := *ErrUnsupportedContentType
	rval.ContentType = contentType
	rval.Message = fmt.Sprintf("%s %s", rval.Message, contentType)
	return &rval
}
