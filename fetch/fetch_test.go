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
		{HttpError{StatusCode: 404}, true},
		{HttpError{StatusCode: 500}, true},
		// {HttpError{StatusCode: 200}, false},
		// {HttpError{StatusCode: 0}, false},
		{&HttpError{StatusCode: 404}, true},
		{&HttpError{StatusCode: 500}, true},
		// {&HttpError{StatusCode: 200}, false},
		// {&HttpError{StatusCode: 0}, false},
		{fmt.Errorf("error"), false},
		{errors.Join(fmt.Errorf("error"), &HttpError{StatusCode: 500}), true},
	}
	referenceError := HttpError{StatusCode: 404, Status: "404 Not Found"}
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
		{HttpError{StatusCode: 415}, true},
		{HttpError{StatusCode: 500}, false},
		{&ErrUnsupportedContentType, true},
		{ErrUnsupportedContentType, true},
		{&HttpError{StatusCode: 415}, true},
		{&HttpError{StatusCode: 500}, false},
		{fmt.Errorf("error"), false},
		{errors.Join(fmt.Errorf("error"), &HttpError{StatusCode: 415}), true},
	}
	referenceError := ErrUnsupportedContentType
	for _, test := range tests {
		if errors.Is(test.err, referenceError) != test.expected {
			t.Errorf("Expected %t for %v, got %t", test.expected, test.err, !test.expected)
		}
	}
}
