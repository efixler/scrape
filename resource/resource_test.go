package resource

import (
	"testing"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

func TestAssertTimes(t *testing.T) {
	r := WebPage{Metadata: trafilatura.Metadata{}, ContentText: ""}
	oldNowf := nowf
	defer func() { nowf = oldNowf }()
	rightNow := time.Now()
	nowf = func() time.Time { return rightNow }

	type data struct {
		Name          string
		FetchTime     *time.Time
		TTL           *time.Duration
		wantFetchTime time.Time
		wantTTL       time.Duration
	}
	tests := []data{
		{"nils -> defaults", nil, nil, nowf(), DefaultTTL},
		{
			"Zeroes",
			&time.Time{},
			func() *time.Duration { var d time.Duration; return &d }(),
			nowf(),
			0,
		},
	}
	for _, test := range tests {
		r.FetchTime = test.FetchTime
		r.TTL = test.TTL
		r.AssertTimes()
		if r.FetchTime.IsZero() {
			t.Errorf("%s FetchTime was zero", test.Name)
		}
		if r.FetchTime.Unix() != test.wantFetchTime.Unix() {
			t.Errorf("%s FetchTime was %v, want %v", test.Name, r.FetchTime, test.wantFetchTime)
		}
		if r.TTL == nil {
			t.Errorf("%s TTL was nil", test.Name)
		}
		if *r.TTL != test.wantTTL {
			t.Errorf("%s TTL was %v, want %v", test.Name, *r.TTL, test.wantTTL)
		}
	}
}
