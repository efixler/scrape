package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
)

func TestRecordFeedUpdatedTime(t *testing.T) {
	// tests := []struct {
	// 	name string
	// }{
	// 	{name: "basic"},
	// }
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(dummyRSS))
	}))
	t.Cleanup(ts.Close)
	client := ts.Client()
	db := database.New(sqlite.MustNew(sqlite.InMemoryDB()))
	if err := db.Open(context.TODO()); err != nil {
		t.Fatalf("could not open db: %v", err)
	}
	_, err := NewFeedFetcher(
		WithClient(client),
		WithSaveActivity(db),
	)
	if err != nil {
		t.Fatalf("can't create feed fetcher %v", err)
	}

}
