package headless

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	hproxy "github.com/efixler/headless/request"
)

type option func(*roundTripper) error

type roundTripper struct {
	client          *http.Client
	headlessAddress string
}

var (
	ErrEmptyHeadlessAddress = errors.New("headless address is empty")
)

func NewRoundTripper(opts ...option) (*roundTripper, error) {
	rt := &roundTripper{client: &http.Client{Timeout: 30 * time.Second}}
	for _, opt := range opts {
		if err := opt(rt); err != nil {
			return nil, err
		}
	}
	return rt, nil
}

func (h *roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	hp := &hproxy.Payload{
		URL: r.URL.String(),
		Headers: map[string]string{
			"User-Agent": r.UserAgent(),
		},
	}
	body, err := json.Marshal(hp)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, h.headlessAddress, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.client.Do(req)
}

func Address(headlessAddress string) option {
	return func(h *roundTripper) error {
		if headlessAddress == "" {
			return ErrEmptyHeadlessAddress
		}
		h.headlessAddress = headlessAddress
		return nil
	}
}
