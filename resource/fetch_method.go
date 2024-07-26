package resource

import (
	"errors"
	"fmt"
)

type FetchClient int

const (
	Unspecified FetchClient = iota
	DefaultClient
	HeadlessChromium
)

var fetchMethods = map[FetchClient]string{
	Unspecified:      "unspecified",
	DefaultClient:    "direct",
	HeadlessChromium: "chromium-headless",
}

var ErrNoSuchFetchMethod = errors.New("no such FetchMethod")

func (f FetchClient) String() string {
	if val, ok := fetchMethods[f]; ok {
		return val
	} else {
		return "Unknown"
	}
}

func (f *FetchClient) UnmarshalText(data []byte) error {
	for k, v := range fetchMethods {
		if v == string(data) {
			*f = k
			return nil
		}
	}
	return errors.Join(
		fmt.Errorf("invalid FetchMethod %q", string(data)),
		ErrNoSuchFetchMethod,
	)
}

func (f FetchClient) MarshalText() ([]byte, error) {
	if val, ok := fetchMethods[f]; ok {
		return []byte(val), nil
	} else {
		return []byte(fetchMethods[Unspecified]),
			errors.Join(
				fmt.Errorf("invalid FetchMethod %q", int(f)),
				ErrNoSuchFetchMethod,
			)
	}
}
