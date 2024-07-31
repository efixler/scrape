// Enable persistent per-domain fetch settings for resources.
package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

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
	fetchRange
	fetchRangeWithQuery
)

const (
	// MaxDomainSettingsBatchSize is the maximum number of domain settings that can be
	// fetched in a single batch.
	DefaultDomainSettingsBatchSize = 100
	MaxDomainSettingsBatchSize     = 1000
)

var (
	ErrDomainRequired = errors.New("domain is required")
	ErrInvalidDomain  = errors.New("invalid domain")
	ErrInvalidQuery   = errors.New("illegal characters in query")
)

type DomainSettings struct {
	Domain      string               `json:"-"`
	Sitename    string               `json:"sitename,omitempty"`
	FetchClient resource.FetchClient `json:"fetch_client,omitempty"`
	UserAgent   ua.UserAgent         `json:"user_agent,omitempty"`
	Headers     MIMEHeader           `json:"headers,omitempty"`
}

// Domain names will be case-folded to lower case.
func NewDomainSettings(domain string) (*DomainSettings, error) {
	if err := ValidateDomain(domain); err != nil {
		return nil, err
	}
	domain = strings.ToLower(domain)
	d := &DomainSettings{
		Domain: domain,
	}
	return d, nil
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
			`DELETE FROM domain_settings WHERE LOWER(domain) = LOWER(?)`,
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
			`SELECT domain, sitename, fetch_client, user_agent, headers 
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
	return ds, nil
}

func (d *DomainSettingsStorage) loadSettingFromRow(rows *sql.Rows) (*DomainSettings, error) {
	ds := &DomainSettings{}
	var headers string
	err := rows.Scan(&ds.Domain, &ds.Sitename, &ds.FetchClient, &ds.UserAgent, &headers)
	if err != nil {
		return ds, err
	}
	if err := json.Unmarshal([]byte(headers), &ds.Headers); err != nil {
		return ds, err
	}
	return ds, nil
}

// FetchRange returns a slice of domain settings, offset by the given offset and limited
// by the given limit. If query is not empty, it will be used to filter the results.
// The query string may contain a leading and/or trailing * to match anything before or after the
// rest of the query. Queries with no asterisks are treated as if they had an asterisk on both sides.
func (d DomainSettingsStorage) FetchRange(offset int, limit int, query string) ([]*DomainSettings, error) {
	switch limit {
	case 0:
		limit = DefaultDomainSettingsBatchSize
	default:
		if limit > MaxDomainSettingsBatchSize {
			limit = MaxDomainSettingsBatchSize
		}
	}
	var (
		stmt        *sql.Stmt
		err         error
		queryParams []any
	)
	if (query == "") || (query == "*") {
		queryParams = []any{limit, offset}
		stmt, err = d.Statement(fetchRangeWithQuery, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
			return db.PrepareContext(
				ctx,
				`SELECT domain, sitename, fetch_client, user_agent, headers FROM domain_settings 
				ORDER BY domain ASC LIMIT ? OFFSET ?`,
			)
		})
	} else {
		// convert leading and trailing * to %
		query, err = parseDomainQuery(query)
		if err != nil {
			return nil, err
		}
		queryParams = []any{query, limit, offset}
		stmt, err = d.Statement(fetchRange, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
			return db.PrepareContext(
				ctx,
				`SELECT domain, sitename, fetch_client, user_agent, headers FROM domain_settings 
				WHERE domain LIKE ? 
				ORDER BY domain ASC LIMIT ? OFFSET ?`,
			)
		})
	}
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(d.Ctx, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dss := make([]*DomainSettings, 0, limit)
	for rows.Next() {
		ds, err := d.loadSettingFromRow(rows)
		if err != nil {
			return nil, err
		}
		dss = append(dss, ds)
	}
	return dss, rows.Err()
}

var starPattern = "^\\*|\\*$" // (separate bc of * unescaping)
var starSyntax = regexp.MustCompile(starPattern)
var queryValidator = regexp.MustCompile(`^(%?)[a-z0-9.-]{0,251}(%?)$`)

func parseDomainQuery(query string) (string, error) {
	query = starSyntax.ReplaceAllString(query, "%")
	query = strings.ToLower(query)
	// check for valid characters
	match := queryValidator.FindStringSubmatch(query)
	if match == nil {
		return "", ErrInvalidQuery
	}
	// If there were no wildcards, wildcard both sides of the query
	if match[1] == "" && match[2] == "" {
		query = "%" + query + "%"
	}
	return query, nil
}

func (d DomainSettingsStorage) Save(domain *DomainSettings) error {
	if domain.Domain == "" {
		return ErrDomainRequired
	}
	domain.Domain = strings.ToLower(domain.Domain)
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

var validDomainChars = regexp.MustCompile(`^[a-zA-Z0-9.-]{4,253}$`)
var validTLDChars = regexp.MustCompile(`^[a-zA-Z]{2,63}$`)
var validDomainElem = regexp.MustCompile(`^[a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62}$`)

func ValidateDomain(domain string) error {
	if !validDomainChars.MatchString(domain) {
		return errors.Join(
			ErrInvalidDomain,
			fmt.Errorf("domain contains non-allowed characters and/or length; %s", domain),
		)
	}
	elem := strings.Split(domain, ".")
	if len(elem) <= 1 {
		return errors.Join(
			ErrInvalidDomain,
			fmt.Errorf("domain must have at least one dot; %s", domain),
		)
	}
	tld := elem[len(elem)-1]
	if !validTLDChars.MatchString(tld) {
		return errors.Join(
			ErrInvalidDomain,
			fmt.Errorf("invalid TLD; %s", tld),
		)
	}
	elem = elem[:len(elem)-1]
	for _, e := range elem {
		if len(e) == 0 {
			return errors.Join(
				ErrInvalidDomain,
				fmt.Errorf("domain element too short; %s", e),
			)
		}
		if !validDomainElem.MatchString(e) {
			return errors.Join(
				ErrInvalidDomain,
				fmt.Errorf("illegal domain element; %s", e),
			)
		}
		if strings.HasSuffix(e, "-") || strings.Contains(e, "--") {
			return errors.Join(
				ErrInvalidDomain,
				fmt.Errorf("illegal domain element (dashes); %s", e),
			)
		}
	}
	return nil
}
