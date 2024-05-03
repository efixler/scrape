package resource

import (
	"errors"
	"fmt"
)

type FetchMethod int

const (
	Unspecified FetchMethod = iota
	DefaultClient
	HeadlessChrome
)

var fetchMethods = map[FetchMethod]string{
	Unspecified:    "Unspecified",
	DefaultClient:  "DefaultClient",
	HeadlessChrome: "HeadlessChrome",
}

var ErrNoSuchFetchMethod = errors.New("no such FetchMethod")

func (f FetchMethod) String() string {
	if val, ok := fetchMethods[f]; ok {
		return val
	} else {
		return "Unknown"
	}
}

func (f *FetchMethod) UnmarshalText(data []byte) error {
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

func (f FetchMethod) MarshalText() ([]byte, error) {
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
