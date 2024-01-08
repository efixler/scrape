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

const (
	DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0"
)

var (
	ErrUnsupportedContentType = &UnsupportedContentTypeError{
		ErrHTTPError{
			StatusCode: http.StatusUnsupportedMediaType,
			Status:     http.StatusText(http.StatusUnsupportedMediaType),
			Message:    "Unsupported content type",
		},
	}
)

type Factory func() (URLData, error)

type URLData interface {
	Open(context.Context) error
	Fetch(*nurl.URL) (*resource.WebPage, error)
	Close() error
}

type FeedData interface {
	Open(context.Context) error
	Fetch(*nurl.URL) (*resource.Feed, error)
	Close() error
}

type ErrHTTPError struct {
	StatusCode int
	Status     string
	Message    string
}

// TODO: Change return value to pointer (or at least make it consistent with UnsupportedContentTypeError)
func NewHTTPError(resp *http.Response) ErrHTTPError {
	rval := ErrHTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}
	defer resp.Body.Close()
	if message, err := io.ReadAll(resp.Body); err == nil {
		rval.Message = strings.TrimSpace(string(message))
	}
	return rval
}

// We consider the Is test true if the target is an ErrHTTPError. For further resolution,
// check the StatusCode field of the error.
func (e ErrHTTPError) Is(target error) bool {
	switch target.(type) {
	case *ErrHTTPError:
		return true
	case ErrHTTPError:
		return true
	default:
		return false
	}
}

func (e ErrHTTPError) Error() string {
	return fmt.Sprintf("HTTP fetch error (%d): %s - %s", e.StatusCode, e.Status, e.Message)
}

func (e ErrHTTPError) String() string {
	return e.Error()
}

type UnsupportedContentTypeError struct {
	ErrHTTPError
}

// Makes errors.Is(err, ErrUnsupportedContentType) return true for any instance of UnsupportedContentTypeError
func (e UnsupportedContentTypeError) Is(target error) bool {
	switch target.(type) {
	case *UnsupportedContentTypeError:
		return true
	case UnsupportedContentTypeError:
		return true
	default:
		return false
	}
}

func NewUnsupportedContentTypeError(contentType string) *UnsupportedContentTypeError {
	rval := *ErrUnsupportedContentType
	rval.Message = contentType
	return &rval
}
