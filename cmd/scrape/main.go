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
	"github.com/efixler/jsonarray"
	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/cmd"
	"github.com/efixler/scrape/internal/headless"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

var (
	flags          flag.FlagSet
	noContent      *envflags.Value[bool]
	dbFlags        *cmd.DatabaseFlags
	csvPath        *envflags.Value[string]
	csvUrlIndex    *envflags.Value[int]
	headlessConfig *cmd.ProxyFlags
	clear          bool
	maintain       bool
	ping           bool
)

func main() {
	dbFactory, err := dbFlags.Database()
	if err != nil {
		slog.Error("Error initializing database connection", "err", err)
		os.Exit(1)
	}
	if clear {
		clearDatabase(dbFactory)
		return
	} else if dbFlags.Create {
		createDatabase(dbFactory)
		return
	} else if maintain {
		maintainDatabase(dbFactory)
		return
	} else if ping {
		pingDatabase(dbFactory)
		return
	}
	fetcher, err := initFetcher(dbFactory)
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

func clearDatabase(dbFactory store.Factory) {
	db, ok := openDatabase(dbFactory).(store.Maintainable)
	if !ok {
		slog.Error("Clearing database not available for this storage backend")
		os.Exit(1)
	}
	defer db.(store.URLDataStore).Close()
	err := db.Clear()
	if err != nil {
		slog.Error("Error clearing database", "database", db, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database cleared", "database", db)
}

func maintainDatabase(dbFactory store.Factory) {
	db, ok := openDatabase(dbFactory).(store.Maintainable)
	if !ok {
		slog.Error("Maintaining database not available for this storage backend", "database", db)
		os.Exit(1)
	}
	defer db.(store.URLDataStore).Close()
	err := db.Maintain()
	if err != nil {
		slog.Error("Error maintaining database", "database", db, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database maintenance complete", "database", db)
}

func createDatabase(dbFactory store.Factory) {
	db, ok := openDatabase(dbFactory).(store.Maintainable)
	if !ok {
		slog.Error("Creating database not available for this storage backend", "database", db)
		os.Exit(1)
	}
	defer db.(store.URLDataStore).Close()
	err := db.Create()
	if err != nil {
		slog.Error("Error creating database", "database", db, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database creation complete", "database", db)
}

func pingDatabase(dbFactory store.Factory) {
	db := openDatabase(dbFactory)
	defer db.Close()
	err := db.Ping()
	if err != nil {
		slog.Error("Error pinging database", "database", db, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database ping successful", "database", db)
}

func openDatabase(dbFactory store.Factory) store.URLDataStore {
	db, err := dbFactory()
	if err != nil {
		slog.Error("Error opening database factory", "db", db, "err", err)
		os.Exit(1)
	}
	err = db.Open(context.TODO())
	if err != nil {
		slog.Error("Error opening database", "db", db, "err", err)
		os.Exit(1)
	}
	return db
}

func initFetcher(dbFactory store.Factory) (*scrape.StorageBackedFetcher, error) {
	tfopts := []fetch.ClientOption{}
	if headlessConfig.Enabled() {
		if headlessConfig.ProxyURL() == "" {
			slog.Error("Headless mode requires a proxy URL")
			os.Exit(1)
		}
		ht, err := headless.NewRoundTripper(
			headless.Address(headlessConfig.ProxyURL()),
		)
		if err != nil {
			return nil, err
		}
		tfopts = append(tfopts, fetch.WithTransport(ht))
	} else {
		tfopts = append(tfopts, fetch.WithFiles("./"))
	}
	client, err := fetch.NewClient(tfopts...)
	if err != nil {
		return nil, fmt.Errorf("error creating default client: %s", err)
	}
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(client),
		dbFactory,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating storage backed fetcher: %s", err)
	}
	err = fetcher.Open(context.TODO())
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

	// TODO: Add headless support
	headlessConfig = cmd.AddProxyFlags("headless", true, &flags)

	csvPath = envflags.NewString("", "")
	csvPath.AddTo(&flags, "csv", "CSV file path")
	csvUrlIndex = envflags.NewInt("CSV_COLUMN", 1)
	csvUrlIndex.AddTo(&flags, "csv-column", "The index of the column in the CSV that contains the URLs")

	flags.BoolVar(&clear, "clear", false, "Clear the database and exit")
	flags.BoolVar(&maintain, "maintain", false, "Execute database maintenance and exit")
	flags.BoolVar(&ping, "ping", false, "Ping the database and exit")

	logLevel := envflags.NewLogLevel("LOG_LEVEL", slog.LevelWarn)
	logLevel.AddTo(&flags, "log-level", "Set the log level [debug|error|info|warn]")
	flags.Parse(os.Args[1:])
	logger := slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: logLevel.Get(),
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
