package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	nurl "net/url"
)

type middleware func(http.HandlerFunc) http.HandlerFunc

type payloadKey struct{}

func Chain(h http.HandlerFunc, m ...middleware) http.HandlerFunc {
	if len(m) == 0 {
		return h
	}
	handler := h
	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}
	return handler
}

func MaxBytes(n int64) middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next(w, r)
		}
	}
}

// //cType := strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])
func ParseSingle() middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			url := r.FormValue("url")
			if url == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("No URL provided"))
				return
			}
			netUrl, err := nurl.ParseRequestURI(url)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("Invalid URL provided: %q, %s", url, err)))
				return
			}
			slog.Debug("ParseSingle", "url", netUrl, "params", netUrl.Query(), "encoding", r.Header.Get("Content-Type"))
			pp := r.FormValue("pp") == "1"
			v := &singleRequest{
				URL:         netUrl,
				PrettyPrint: pp,
			}
			r = r.WithContext(context.WithValue(r.Context(), payloadKey{}, v))
			next(w, r)
		}
	}
}

func DecodeJSONBody[T any]() middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			v := new(T)
			err := decoder.Decode(v)
			if !assertDecode(err, w) {
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), payloadKey{}, v))
			next(w, r)
		}
	}
}

func assertDecode(err error, w http.ResponseWriter) bool {
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
