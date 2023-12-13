package sqlite

import (
	"context"
	"encoding/json"
	nurl "net/url"
	"scrape/store"
	"testing"

	"github.com/markusmobius/go-trafilatura"
)

func TestOpen(t *testing.T) {
	s, err := Open(context.TODO(), "test.db")
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	defer s.Close()
	_, ok := dbs[s.dsn()]
	if !ok {
		t.Errorf("Database reference not stored in dbs map")
	}
}

var mdata = `{
	"Title": "About Martin Fowler",
	"Author": "",
	"URL": "https://martinfowler.com/aboutMe.html",
	"Hostname": "martinfowler.com",
	"Description": "Background to Martin Fowler and martinfowler.com",
	"Sitename": "martinfowler.com",
	"Date": "1999-01-01T00:00:00Z",
	"Categories": null,
	"Tags": null,
	"ID": "",
	"Fingerprint": "",
	"License": "",
	"Language": "en",
	"Image": "https://martinfowler.com/logo-sq.png",
	"PageType": "article"
  }`

func TestStore(t *testing.T) {
	s, err := Open(context.TODO(), "test.db")
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	defer s.Close()
	var meta trafilatura.Metadata
	err = json.Unmarshal([]byte(mdata), &meta)
	if err != nil {
		t.Errorf("Error unmarshaling metadata: %v", err)
	}
	url, err := nurl.Parse("https://martinfowler.com/aboutMe.html")
	if err != nil {
		t.Errorf("Error parsing url: %v", err)
	}
	urlData := store.StoredUrlData{
		Url:         url,
		Metadata:    meta,
		ContentText: "Some dummy content",
	}
	_, err = s.Store(&urlData)
	if err != nil {
		t.Errorf("Error storing data: %v", err)
	}

}
