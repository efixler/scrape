package healthchecks

import (
	"net/http"
	"strings"
)

func heartbeat(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func Handler(root string) http.Handler {
	root = strings.TrimSuffix(root, "/")
	mux := http.NewServeMux()
	mux.HandleFunc("/heartbeat", heartbeat)
	switch root {
	case "":
		return mux
	default:
		return http.StripPrefix(root, mux)
	}
}
