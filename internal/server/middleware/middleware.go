// Middleware definitions and helper functions.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Step func(http.HandlerFunc) http.HandlerFunc

// Prepend the middlewares to the handler in the order they are provided,
// and return the resulting (chained) handler.
func Chain(h http.HandlerFunc, m ...Step) http.HandlerFunc {
	if len(m) == 0 {
		return h
	}
	handler := h
	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}
	return handler
}

// Abort the request if the body is over the specified size in bytes.
func MaxBytes(n int64) Step {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next(w, r)
		}
	}
}

// Decode the JSON body of the request into the provided type and store it in the context.
// If there are JSON errors, the request will be aborted, and an error response will be
// sent.
func DecodeJSONBody[T any](pkey any) Step {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			v := new(T)
			err := decoder.Decode(v)
			if !AssertJSONDecode(err, w) {
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), pkey, v))
			next(w, r)
		}
	}
}

// Check JSON errors and send a correct error response if needed.
// Possible http errors are:
// - 400 Bad Request if the JSON is invalid
// - 413 Request Entity Too Large if the JSON is too large
//
// returns true if the JSON is valid, false otherwise.
func AssertJSONDecode(err error, w http.ResponseWriter) bool {
	if err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalErr *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesError):
			http.Error(w, fmt.Sprintf("Invalid request %s", maxBytesError), http.StatusRequestEntityTooLarge)
			return false
		case errors.Is(err, io.ErrUnexpectedEOF):
			http.Error(w, "Invalid JSON: unexpected end of JSON input", http.StatusBadRequest)
			return false
		case errors.As(err, &syntaxErr):
			http.Error(w, fmt.Sprintf("Invalid JSON: bad syntax at byte offset %d", syntaxErr.Offset), http.StatusBadRequest)
			return false
		case errors.As(err, &unmarshalErr):
			http.Error(w, fmt.Sprintf("Invalid JSON: value %q at offset %d is not of type %s", unmarshalErr.Value, unmarshalErr.Offset, unmarshalErr.Type), http.StatusBadRequest)
			return false
		}
		http.Error(w, fmt.Sprintf("Error decoding request: %s", err), http.StatusBadRequest)
		return false
	}
	return true
}
