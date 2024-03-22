/*
Package json provides an interface to stream an array of JSON objects to an
io.Writer. This is useful for streaming long JSON arrays to an HTTP response.
*/
package json

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

var (
	comma    = []byte(",\n")
	start    = []byte("[\n")
	end      = []byte("\n]\n")
	noindent = []string{"", ""}
)

type Encoder[T any] struct {
	w        io.Writer
	flusher  func()
	prefixer func() error
	indent   []string
	start    []byte
	comma    []byte
}

func NewArrayEncoder[T any](w io.Writer, hotPipe bool) *Encoder[T] {
	ae := &Encoder[T]{
		w:       w,
		flusher: func() {},
		indent:  noindent,
		start:   start,
		comma:   comma,
	}
	if hotPipe {
		flusher, ok := w.(http.Flusher)
		if ok {
			ae.flusher = flusher.Flush
		} else {
			slog.Warn("ArrayEncoder: hot pipe requested, but Writer is not a Flusher")
		}
	}
	ae.Reset()
	return ae
}

func (ae *Encoder[T]) SetIndent(prefix, indent string) {
	if prefix == "" && indent == "" {
		ae.indent = noindent
	} else {
		ae.indent = []string{prefix, indent}
	}
	ae.start = append(start, []byte(ae.indent[1])...)
	ae.comma = append(comma, []byte(ae.indent[1])...)
}

func (ae *Encoder[T]) hasIndent() bool {
	if len(ae.indent) < 2 || ((ae.indent[0] == "") && (ae.indent[1] == "")) {
		return false
	}
	return true
}

// TODO: Better indent implementation. The current implementation
// does work for single depth objects, but it won't work right for
// deeper nesting and it's a little hacky.

// Encode writes a JSON object to the underlying io.Writer. If the
// ArrayEncoder was created with hotPipe set to true, it will also
// flush the writer after writing the object.
// It is assumed that writing is being done in a single goroutine.
// If there's a chance of concurrent writes, lock the function invocation
// with a mutex.
func (ae *Encoder[T]) Encode(v T) error {
	err := ae.prefixer()
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	if !ae.hasIndent() || len(b) <= 2 {
		_, err = ae.w.Write(b)
		return err
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, b, ae.indent[0], strings.Repeat(ae.indent[1], 2))
	if err != nil {
		return err
	}
	bb := buf.Bytes()
	bbLen := len(bb)
	lastChar := string(bb[bbLen-1])
	buf.Truncate(bbLen - 1)
	buf.Write([]byte(ae.indent[1] + lastChar))
	_, err = buf.WriteTo(ae.w)
	return err
}

func (ae *Encoder[T]) Finish() error {
	var buf bytes.Buffer
	buf.Write(end)
	_, err := ae.w.Write(end)
	return err
}

func (ae *Encoder[T]) Reset() {
	ae.prefixer = func() error {
		ae.prefixer = func() error {
			_, err := ae.w.Write(ae.comma)
			return err
		}
		_, err := ae.w.Write(ae.start)
		return err
	}
}
