package resource

// TODO: Move FetchClient to a more sensible package

import (
	"errors"
	"fmt"
)

type ClientIdentifier int

const (
	Unspecified ClientIdentifier = iota
	DefaultClient
	HeadlessChromium
)

var fetchClientNames = map[ClientIdentifier]string{
	Unspecified:      "unspecified",
	DefaultClient:    "direct",
	HeadlessChromium: "chromium-headless",
}

var ErrNoSuchFetchMethod = errors.New("no such fetch client identifier")

func (f ClientIdentifier) String() string {
	if val, ok := fetchClientNames[f]; ok {
		return val
	} else {
		return "Unknown"
	}
}

func (f *ClientIdentifier) UnmarshalText(data []byte) error {
	for k, v := range fetchClientNames {
		if v == string(data) {
			*f = k
			return nil
		}
	}
	return errors.Join(
		fmt.Errorf("invalid name %q", string(data)),
		ErrNoSuchFetchMethod,
	)
}

func (f ClientIdentifier) MarshalText() ([]byte, error) {
	if val, ok := fetchClientNames[f]; ok {
		return []byte(val), nil
	} else {
		return []byte(fetchClientNames[Unspecified]),
			errors.Join(
				fmt.Errorf("invalid name %q", int(f)),
				ErrNoSuchFetchMethod,
			)
	}
}
