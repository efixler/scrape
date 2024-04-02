package ua

import "testing"

func TestUserAgent(t *testing.T) {
	type data struct {
		name     string
		in       string
		expected UserAgent
	}
	tests := []data{
		{"Firefox88", ":firefox:", Firefox88},
		{"Safari537", ":safari:", Safari537},
		{"Custom", "custom", UserAgent("custom")},
	}
	for _, test := range tests {
		var a UserAgent
		err := a.UnmarshalText([]byte(test.in))
		if err != nil {
			t.Errorf("[%s] Error unmarshalling %s: %s", test.name, test.in, err)
		}
		if a != test.expected {
			t.Errorf("[%s] Expected %s, got %s", test.name, test.expected, a)
		}
	}
}
