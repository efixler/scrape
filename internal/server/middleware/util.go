package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

func WriteJSONOutput(w http.ResponseWriter, v any, pp bool, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if pp {
		encoder.SetIndent("", "  ")
	}
	encoder.Encode(v)
}

// Anything that's not a GET and not a form is assumed to be JSON
// This is imperfect but it allows for requests that don't send a content-type
// header or inadvertently use text/plain
func IsJSONRequest(r *http.Request) bool {
	if r.Method == http.MethodGet {
		return false
	}
	contentType := strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0]
	switch contentType {
	case "application/x-www-form-urlencoded":
		return false
	case "multipart/form-data":
		return false
	}
	return true
}
