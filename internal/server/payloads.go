package server

import (
	"encoding/json"
	"errors"
	nurl "net/url"
)

type BatchRequest struct {
	Urls []string `json:"urls"`
}
type singleURLRequest struct {
	URL         *nurl.URL `json:"url"`
	PrettyPrint bool      `json:"pp,omitempty"`
}

var errNoURL = errors.New("URL is required")

func (sur *singleURLRequest) UnmarshalJSON(b []byte) error {
	type alias singleURLRequest
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

type domainSettingsRequest struct {
	Domain string `json:"domain"`
	Query  string `json:"q,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
