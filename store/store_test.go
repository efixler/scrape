package store

import (
	nurl "net/url"
	"testing"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

func TestAssertTimes(t *testing.T) {
	u := StoredUrlData{
		Url:         &nurl.URL{},
		Metadata:    trafilatura.Metadata{},
		ContentText: "",
	}
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
		{"nils -> defaults", nil, nil, nowf(), DEFAULT_TTL},
		{
			"Zeroes",
			&time.Time{},
			func() *time.Duration { var d time.Duration; return &d }(),
			nowf(),
			0,
		},
	}
	for _, test := range tests {
		u.FetchTime = test.FetchTime
		u.TTL = test.TTL
		u.AssertTimes()
		if u.FetchTime == nil {
			t.Errorf("%s FetchTime was nil", test.Name)
		}
		if u.FetchTime.IsZero() {
			t.Errorf("%s FetchTime was zero", test.Name)
		}
		if *u.FetchTime != test.wantFetchTime {
			t.Errorf("%s FetchTime was %v, want %v", test.Name, *u.FetchTime, test.wantFetchTime)
		}
		if u.TTL == nil {
			t.Errorf("%s TTL was nil", test.Name)
		}
		if *u.TTL != test.wantTTL {
			t.Errorf("%s TTL was %v, want %v", test.Name, *u.TTL, test.wantTTL)
		}
	}
}
