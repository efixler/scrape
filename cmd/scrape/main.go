// Use the `scrape` command line tool to fetch and
// extract metadata from urls and to set up and maintain
// the `scrape` database.
//
// The basic invocation form is:
//
// > scrape https://example.com/path
//
// Run `scrape -h` for complete help and command line options.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal"
	"github.com/efixler/scrape/internal/cmd"
	"github.com/efixler/scrape/internal/headless"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/ua"
	"github.com/efixler/webutil/jsonarray"
)

var (
	flags           flag.FlagSet
	noContent       *envflags.Value[bool]
	dbFlags         *cmd.DatabaseFlags
	userAgent       *envflags.Value[*ua.UserAgent]
	csvPath         *envflags.Value[string]
	csvUrlIndex     *envflags.Value[int]
	headlessEnabled bool
	// clear           bool
	maintain bool
	ping     bool
)

func main() {
	dbh, err := dbFlags.Database()
	if err != nil {
		slog.Error("Error initializing database connection", "err", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	openDatabase(dbh, ctx)

	if dbFlags.IsMigration() {
		migrateDatabase(dbh, dbFlags.MigrationCommand)
		return
	} else if maintain {
		maintainDatabase(dbh)
		return
	} else if ping {
		pingDatabase(dbh)
		return
	}
	fetcher, err := initFetcher(dbh)
	if err != nil {
		slog.Error("Error initializing fetcher", "err", err)
		os.Exit(1)
	}
	defer fetcher.Close()
	args := getArgs()
	if len(args) == 0 {
		slog.Error("Error: At least one URL is required\n\n")
		flags.Usage()
		os.Exit(1)
	}
	encoder := jsonarray.NewEncoder[*resource.WebPage](os.Stdout, false)

	encoder.SetIndent("", "  ")
	rchan := fetcher.Batch(args, fetch.BatchOptions{})
	for page := range rchan {
		// TODO: Make it so we don't have to run a conditional on every iteration
		if noContent.Get() {
			page.ContentText = ""
		}
		err = encoder.Encode(page)
		if err != nil {
			slog.Error("Error encoding page", "page", page, "err", err)
		}
	}
	encoder.Finish()
}

func getArgs() []string {
	if csvPath.Get() != "" {
		csvFile, err := os.Open(csvPath.Get())
		if err != nil {
			slog.Error("Error opening CSV file", "csv", csvPath, "error", err)
			os.Exit(1)
		}
		defer csvFile.Close()
		reader := csv.NewReader(csvFile)
		reader.FieldsPerRecord = -1 // allow variable number of fields, we only care about the first
		reader.TrimLeadingSpace = true
		reader.ReuseRecord = true
		rval := make([]string, 0)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				slog.Error("error reading CSV file", "csv", csvPath, "error", err)
				os.Exit(1)
			}
			rval = append(rval, record[csvUrlIndex.Get()])
		}
		return rval
	}
	return flags.Args()
}

func maintainDatabase(dbh *database.DBHandle) {
	mt, ok := dbh.Engine.(database.Maintainable)
	if !ok {
		slog.Error("Database maintenance not available for this storage backend", "database", dbh)
		os.Exit(1)
	}
	err := mt.Maintain(dbh)
	if err != nil {
		slog.Error("Error maintaining database", "database", dbh, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database maintenance complete", "database", dbh)
}

func migrateDatabase(dbh *database.DBHandle, migrationCommand cmd.MigrationCommand) {
	var err error
	switch migrationCommand {
	case cmd.Up:
		err = dbh.MigrateUp()
	case cmd.Reset:
		err = dbh.MigrateReset()
	case cmd.Status:
		err = dbh.PrintMigrationStatus()
	default:
		err = fmt.Errorf("unsupported migration command: %s", migrationCommand)
	}
	if err != nil {
		slog.Error("Error migrating database", "database", dbh, "err", err)
		os.Exit(1)
	}
}

func pingDatabase(dbh *database.DBHandle) {
	err := dbh.Ping()
	if err != nil {
		slog.Error("Error pinging database", "database", dbh, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database ping successful", "database", dbh)
}

func openDatabase(dbh *database.DBHandle, ctx context.Context) {
	err := dbh.Open(ctx)
	if err != nil {
		slog.Error("Error opening database", "db", dbh, "err", err)
		os.Exit(1)
	}
}

func initFetcher(dbh *database.DBHandle) (*internal.StorageBackedFetcher, error) {
	var err error
	var client fetch.Client
	if headlessEnabled {
		client, err = headless.NewChromeClient(dbh.Ctx, userAgent.Get().String(), 1)
		if err != nil {
			return nil, fmt.Errorf("error creating headless client: %s", err)
		}
	} else {
		client = fetch.MustClient(
			fetch.WithFiles("./"),
			fetch.WithUserAgent(userAgent.Get().String()),
		)
	}
	fetcher, err := internal.NewStorageBackedFetcher(
		trafilatura.MustNew(client),
		storage.NewURLDataStore(dbh),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating storage backed fetcher: %s", err)
	}
	err = fetcher.Open(dbh.Ctx)
	if err != nil {
		return nil, fmt.Errorf("error opening storage backed fetcher: %s", err)
	}
	return fetcher, nil
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	envflags.EnvPrefix = "SCRAPE_"
	noContent = envflags.NewBool("NOTEXT", false)
	noContent.AddTo(&flags, "notext", "Skip text content")
	dbFlags = cmd.AddDatabaseFlags("DB", &flags, true)

	flags.BoolVar(&headlessEnabled, "headless", false, "Use headless browser for extraction")

	dua := ua.UserAgent(fetch.DefaultUserAgent)
	userAgent = envflags.NewText("USER_AGENT", &dua)
	userAgent.AddTo(&flags, "user-agent", "User agent to use for fetching")

	csvPath = envflags.NewString("", "")
	csvPath.AddTo(&flags, "csv", "CSV file path")
	csvUrlIndex = envflags.NewInt("CSV_COLUMN", 1)
	csvUrlIndex.AddTo(&flags, "csv-column", "The index of the column in the CSV that contains the URLs")

	flags.BoolVar(&maintain, "maintain", false, "Execute database maintenance and exit")
	flags.BoolVar(&ping, "ping", false, "Ping the database and exit")

	logLevel := envflags.NewLogLevel("LOG_LEVEL", slog.LevelWarn)
	logLevel.AddTo(&flags, "log-level", "Set the log level [debug|error|info|warn]")
	flags.Parse(os.Args[1:])
	ll := logLevel.Get()
	// Goose prints output directly during migrations, at INFO level,
	// so if we're migrating, make sure we see the Goose messages.
	if dbFlags.IsMigration() && (ll > slog.LevelInfo) {
		ll = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: ll,
		},
	))
	slog.SetDefault(logger)
}

func usage() {
	fmt.Println(`Usage: 
	scrape [flags] :url [...urls]

In addition to http[s] URLs, file:/// urls are supported, using the current working directory as the base path.

Flags:
 
  -h	
  	Show this help message`)
	flags.PrintDefaults()
}
