// Enable persistent per-domain fetch settings for resources.
package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/ua"
)

type stmtKey int

const (
	_ stmtKey = iota
	delete
	fetch
	save
)

var (
	ErrDomainRequired = errors.New("domain is required")
)

type DomainSettings struct {
	Domain      string               `json:"-"`
	Sitename    string               `json:"sitename,omitempty"`
	FetchClient resource.FetchClient `json:"fetch_client,omitempty"`
	UserAgent   ua.UserAgent         `json:"user_agent,omitempty"`
	Headers     map[string]string    `json:"headers,omitempty"`
}

func NewDomainSettings(domain string) *DomainSettings {
	d := &DomainSettings{
		Domain: domain,
	}
	return d
}

type DomainSettingsStorage struct {
	*database.DBHandle
}

func NewDomainSettingsStorage(dbh *database.DBHandle) *DomainSettingsStorage {
	return &DomainSettingsStorage{DBHandle: dbh}
}

func (d DomainSettingsStorage) Delete(domain string) (bool, error) {
	stmt, err := d.Statement(delete, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(
			ctx,
			`DELETE FROM domain_settings WHERE domain = ?`,
		)
	})
	if err != nil {
		return false, err
	}
	result, err := stmt.ExecContext(d.Ctx, domain)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	switch rows {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("expected 0 or 1 row affected, got %d", rows)
	}
}

func (d DomainSettingsStorage) Fetch(domain string) (*DomainSettings, error) {
	stmt, err := d.Statement(fetch, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(
			ctx,
			`SELECT sitename, fetch_client, user_agent, headers 
			FROM domain_settings WHERE domain = ?`,
		)
	})
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(d.Ctx, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, store.ErrResourceNotFound
	}
	ds, err := d.loadSettingFromRow(rows)
	if err != nil {
		return nil, err
	}
	ds.Domain = domain
	return &ds, nil
}

func (d *DomainSettingsStorage) loadSettingFromRow(rows *sql.Rows) (DomainSettings, error) {
	ds := &DomainSettings{}
	var headers string
	err := rows.Scan(&ds.Sitename, &ds.FetchClient, &ds.UserAgent, &headers)
	if err != nil {
		return *ds, err
	}
	if err := json.Unmarshal([]byte(headers), &ds.Headers); err != nil {
		return *ds, err
	}
	return *ds, nil
}

func (d DomainSettingsStorage) FetchAll() ([]*DomainSettings, error) {
	stmt, err := d.Statement(fetch, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(
			ctx,
			`SELECT sitename, fetch_client, user_agent, headers 
			FROM domain_settings ORDER BY domain ASC`,
		)
	})
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(d.Ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// continue here
	// var domains []*DomainSettings

	return nil, nil
}

func (d DomainSettingsStorage) Save(domain *DomainSettings) error {
	if domain.Domain == "" {
		return ErrDomainRequired
	}
	stmt, err := d.Statement(save, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(
			ctx,
			`REPLACE INTO domain_settings (domain, sitename, fetch_client, user_agent, headers) 
			VALUES (?, ?, ?, ?, ?)`,
		)
	})
	if err != nil {
		return err
	}
	hb, err := json.Marshal(domain.Headers)
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(
		d.Ctx,
		domain.Domain,
		domain.Sitename,
		domain.FetchClient,
		domain.UserAgent,
		string(hb),
	)
	if err != nil {
		return err
	}
	return nil
}
