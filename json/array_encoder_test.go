package json

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestArrayEncoderStrings(t *testing.T) {
	var buf bytes.Buffer
	ae := NewArrayEncoder[string](&buf, false)
	ae.Encode("foo")
	ae.Encode("bar")
	ae.Finish()
	got := buf.String()
	want := "[\n\"foo\",\n\"bar\"\n]\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type testStruct struct {
	Foo     string        `json:"foo"`
	Bar     int           `json:"bar"`
	BatTime time.Duration `json:"bat_time"`
}

func TestArrayEncoderStruct(t *testing.T) {
	type data struct {
		name    string
		payload []testStruct
	}
	tests := []data{
		{"single zeroes", []testStruct{{}}},
		{"single non-zeroes", []testStruct{{"foo", 1, time.Hour}}},
		{"multiple zeroes", []testStruct{{}, {}, {}}},
		{"multiple non-zeroes", []testStruct{{"foo", 1, time.Hour}, {"bar", 2, time.Minute}, {"baz", 3, time.Second}}},
	}
	var buf bytes.Buffer
	ae := NewArrayEncoder[testStruct](&buf, false)
	for _, indent := range [][]string{{"", ""}, {"", "  "}} {
		ae.SetIndent(indent[0], indent[1])
		for _, test := range tests {
			ae.Reset()
			buf.Reset()
			for _, v := range test.payload {
				ae.Encode(v)
			}
			ae.Finish()
			umarshaled := []testStruct{}
			err := json.Unmarshal(buf.Bytes(), &umarshaled)
			if err != nil {
				t.Errorf("test %q unmarshaling results: %v", test.name, err)
			}
			if len(umarshaled) != len(test.payload) {
				t.Errorf("test %q got %d results, want %d", test.name, len(umarshaled), len(test.payload))
			}
			for i, v := range umarshaled {
				if v.Bar != test.payload[i].Bar {
					t.Errorf("test %q got %#v, want %#v", test.name, v, test.payload[i])
				}
				if v.Foo != test.payload[i].Foo {
					t.Errorf("test %q got %#v, want %#v", test.name, v, test.payload[i])
				}
				if v.BatTime != test.payload[i].BatTime {
					t.Errorf("test %q got %#v, want %#v", test.name, v, test.payload[i])
				}
			}
		}
	}
}
