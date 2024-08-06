package api

import (
	"encoding/json"
	"errors"

	nurl "net/url"
)

type payloadKey struct{}

// Defines the input payload for a batch request.
type BatchRequest struct {
	Urls []string `json:"urls"`
}

// Defines the input payload for a single URL request.
// The URL field is required, converted from a string on input,
// and must be an absolute URL.
type SingleURLRequest struct {
	URL         *nurl.URL `json:"url"`
	PrettyPrint bool      `json:"pp,omitempty"`
}

var errNoURL = errors.New("URL is required")

func (sur *SingleURLRequest) UnmarshalJSON(b []byte) error {
	type alias SingleURLRequest
	asur := &struct {
		URL string `json:"url"`
		*alias
	}{
		alias: (*alias)(sur),
	}
	if err := json.Unmarshal(b, asur); err != nil {
		return err
	}
	if asur.URL == "" {
		return errNoURL
	}
	var err error
	if sur.URL, err = nurl.Parse(asur.URL); err != nil {
		return err
	}
	if !sur.URL.IsAbs() {
		return errors.New("URL must be absolute")
	}
	return nil
}
