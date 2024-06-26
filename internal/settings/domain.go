// Enable persistent per-domain fetch settings for resources.
package settings

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/ua"
)

type DomainStorage struct {
	*database.DBHandle[int]
}

type Domain struct {
	Domain        string
	PublisherName string
	FetchClient   resource.FetchClient
	UserAgent     ua.UserAgent
	Headers       map[string]string
}

func DomainSettings(domain string) (*Domain, error) {
	d := &Domain{
		Domain: domain,
	}
	return d, nil
}

func (d DomainStorage) Delete(domain string) (bool, error) {
	stmt, err := d.Statement(100, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
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

func (d DomainStorage) Fetch(domain string) (*Domain, error) {
	stmt, err := d.Statement(101, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(
			ctx,
			`SELECT publisher_name, fetch_client, user_agent, headers 
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
		return nil, store.ErrorResourceNotFound
	}
	ds := &Domain{Domain: domain}
	err = rows.Scan(&ds.PublisherName, &ds.FetchClient, &ds.UserAgent, &ds.Headers)
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func (d DomainStorage) FetchAll(domain string) ([]*Domain, error) {
	return nil, nil
}

func (d DomainStorage) Save(domain *Domain) error {
	stmt, err := d.Statement(102, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(
			ctx,
			`REPLACE INTO domain_settings (domain, publisher_name, fetch_client, user_agent, headers) 
			VALUES (?, ?, ?, ?, ?)`,
		)
	})
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(
		d.Ctx,
		domain.Domain,
		domain.PublisherName,
		domain.FetchClient,
		domain.UserAgent,
		domain.Headers,
	)
	if err != nil {
		return err
	}
	return nil
}
