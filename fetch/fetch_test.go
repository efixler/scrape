package fetch

import (
	"errors"
	"fmt"
	"testing"
)

func TestHttpErrorIs(t *testing.T) {
	type data struct {
		err      error
		expected bool
	}
	tests := []data{
		{ErrHTTPError{StatusCode: 404}, true},
		{ErrHTTPError{StatusCode: 500}, true},
		// {ErrHTTPError{StatusCode: 200}, false},
		// {ErrHTTPError{StatusCode: 0}, false},
		{&ErrHTTPError{StatusCode: 404}, true},
		{&ErrHTTPError{StatusCode: 500}, true},
		// {&ErrHTTPError{StatusCode: 200}, false},
		// {&ErrHTTPError{StatusCode: 0}, false},
		{fmt.Errorf("error"), false},
		{errors.Join(fmt.Errorf("error"), &ErrHTTPError{StatusCode: 500}), true},
	}
	referenceError := ErrHTTPError{StatusCode: 404, Status: "404 Not Found"}
	for _, test := range tests {
		if errors.Is(test.err, referenceError) != test.expected {
			t.Errorf("Expected %t for %v, got %t", test.expected, test.err, !test.expected)
		}
	}
}

func TestUnsupportedContentTypeErrorIs(t *testing.T) {
	type data struct {
		err      error
		expected bool
	}
	tests := []data{
		{ErrHTTPError{StatusCode: 415}, true},
		{ErrHTTPError{StatusCode: 500}, false},
		{&ErrUnsupportedContentType, true},
		{ErrUnsupportedContentType, true},
		{&ErrHTTPError{StatusCode: 415}, true},
		{&ErrHTTPError{StatusCode: 500}, false},
		{fmt.Errorf("error"), false},
		{errors.Join(fmt.Errorf("error"), &ErrHTTPError{StatusCode: 415}), true},
	}
	referenceError := ErrUnsupportedContentType
	for _, test := range tests {
		if errors.Is(test.err, referenceError) != test.expected {
			t.Errorf("Expected %t for %v, got %t", test.expected, test.err, !test.expected)
		}
	}
}
