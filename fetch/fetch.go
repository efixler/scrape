// Interfaces and a basic client for URL fetching and parsing.
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
	ErrUnsupportedContentType = UnsupportedContentTypeError{
		HttpError{
			StatusCode: http.StatusUnsupportedMediaType,
			Status:     http.StatusText(http.StatusUnsupportedMediaType),
			Message:    "Unsupported content type",
		},
	}
)

type URLFetcher interface {
	Fetch(*nurl.URL) (*resource.WebPage, error)
}

type BatchURLFetcher interface {
	Batch([]string, BatchOptions) <-chan *resource.WebPage
}

type BatchOptions struct {
	//throttle time.Duration
}

type FeedFetcher interface {
	Fetch(*nurl.URL) (*resource.Feed, error)
	FetchContext(context.Context, *nurl.URL) (*resource.Feed, error)
}

type HttpError struct {
	StatusCode int
	Status     string
	Message    string
}

// TODO: Resolve/make consistent the pointer/value semantics of this and other errors here.
func NewHTTPError(resp *http.Response) HttpError {
	rval := HttpError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}
	defer resp.Body.Close()
	if message, err := io.ReadAll(resp.Body); err == nil {
		rval.Message = strings.TrimSpace(string(message))
	}
	return rval
}

// We consider the Is test true if the target is an HttpError. For further resolution,
// check the StatusCode field of the error.
// TODO: The idea of providing generic and specific HTTPErors is well-intentioned, but the
// Is checks are a bit baroque (because to make sense these need to work bidrectionally).
// Consider simplifying.
func (e HttpError) Is(target error) bool {
	switch target.(type) {
	case *HttpError:
		return true
	case HttpError:
		return true
	case *UnsupportedContentTypeError:
		return e.StatusCode == http.StatusUnsupportedMediaType
	case UnsupportedContentTypeError:
		return e.StatusCode == http.StatusUnsupportedMediaType
	default:
		return false
	}
}

func (e HttpError) Error() string {
	if e.Status == "" {
		e.Status = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("HTTP fetch error (%d): %s - %s", e.StatusCode, e.Status, e.Message)
}

func (e HttpError) String() string {
	return e.Error()
}

type UnsupportedContentTypeError struct {
	HttpError
}

// Makes errors.Is(err, ErrUnsupportedContentType) return true for any instance of UnsupportedContentTypeError
// or for an HttpError with a 415 status code.
func (e UnsupportedContentTypeError) Is(target error) bool {
	switch v := target.(type) {
	case *UnsupportedContentTypeError:
		return true
	case UnsupportedContentTypeError:
		return true
	case *HttpError:
		return v.StatusCode == http.StatusUnsupportedMediaType
	case HttpError:
		return v.StatusCode == http.StatusUnsupportedMediaType
	default:
		return false
	}
}

func NewUnsupportedContentTypeError(contentType string) *UnsupportedContentTypeError {
	rval := ErrUnsupportedContentType
	rval.Message = contentType
	return &rval
}
