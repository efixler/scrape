package resource

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestFetchMethodString(t *testing.T) {
	tests := []struct {
		name string
		f    ClientIdentifier
		want string
	}{
		{
			name: "Client",
			f:    DefaultClient,
			want: "direct",
		},
		{
			name: "Headless",
			f:    HeadlessChromium,
			want: "chromium-headless",
		},
		{
			name: "Unknown",
			f:    3,
			want: "Unknown",
		},
	}
	for _, tt := range tests {
		if got := tt.f.String(); got != tt.want {
			t.Errorf("[%s] FetchMethod.String() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	type container struct {
		F ClientIdentifier `json:"fetch_method"`
	}
	tests := []struct {
		input         string
		expectedValue ClientIdentifier
		expectError   bool
	}{
		{input: "unspecified", expectedValue: Unspecified},
		{input: "direct", expectedValue: DefaultClient},
		{input: "chromium-headless", expectedValue: HeadlessChromium},
		{input: "1", expectError: true},
	}
	c := &container{}
	for _, test := range tests {
		jsonString := fmt.Sprintf(`{"fetch_method":"%s"}`, test.input)
		err := json.Unmarshal([]byte(jsonString), c)
		if (err != nil) != test.expectError {
			t.Errorf("%q expected error %v, got %v", test.input, test.expectError, err)
			continue
		}
		if !test.expectError && (test.expectedValue != c.F) {
			t.Errorf("%q expected %d got %d", test.input, test.expectedValue, c.F)
		}
	}
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		input         int
		expectedValue string
		expectError   bool
	}{
		{input: 0, expectedValue: fetchClientNames[Unspecified]},
		{input: 1, expectedValue: fetchClientNames[DefaultClient]},
		{input: 2, expectedValue: fetchClientNames[HeadlessChromium]},
		{input: -1, expectError: true},
	}
	for _, test := range tests {
		fm := ClientIdentifier(test.input)
		val, err := fm.MarshalText()
		if (err != nil) != test.expectError {
			t.Errorf("%q expected error %v, got %v", test.input, test.expectError, err)
			continue
		}
		if !test.expectError && (test.expectedValue != string(val)) {
			t.Errorf("%q expected %s got %s", test.input, test.expectedValue, val)
		}
	}
}
