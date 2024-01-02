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

func (e ErrHTTPError) Body(target error) string {
	return e.Message
}

func (e ErrHTTPError) Error() string {
	return fmt.Sprintf("HTTP fetch error [%d:%s]", e.StatusCode, e.Message)
}

func (e ErrHTTPError) String() string {
	return e.Error()
}

type Factory func() (URLData, error)

type URLData interface {
	Open(context.Context) error
	// Store(*StoredUrlData) (uint64, error)
	Fetch(*nurl.URL) (*resource.WebPage, error)
	Close() error
}
