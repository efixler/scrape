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
	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/cmd"
	jstream "github.com/efixler/scrape/json"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

var (
	flags       flag.FlagSet
	noContent   *envflags.Value[bool]
	dbFlags     *cmd.DatabaseFlags
	csvPath     *envflags.Value[string]
	csvUrlIndex *envflags.Value[int]
	clear       bool
	maintain    bool
	ping        bool
)

func initFetcher() (*scrape.StorageBackedFetcher, error) {
	dbFactory, err := dbFlags.Database()
	if err != nil {
		return nil, fmt.Errorf("error creating database factory: %s", err)
	}
	dbFlags = nil
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(*trafilatura.DefaultOptions),
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

func clearDatabase(fetcher *scrape.StorageBackedFetcher) {
	db, ok := fetcher.Storage.(store.Maintainable)
	if !ok {
		slog.Error("Clearing database not available for this storage backend")
		os.Exit(1)
	}
	err := db.Clear()
	if err != nil {
		slog.Error("Error clearing database", "database", fetcher.Storage, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database cleared", "database", fetcher.Storage)
}

func maintainDatabase(fetcher *scrape.StorageBackedFetcher) {
	db, ok := fetcher.Storage.(store.Maintainable)
	if !ok {
		slog.Error("Maintaining database not available for this storage backend", "database", fetcher.Storage)
		os.Exit(1)
	}
	err := db.Maintain()
	if err != nil {
		slog.Error("Error maintaining database", "database", fetcher.Storage, "err", err)
		os.Exit(1)
	}
	slog.Warn("Database maintenance complete", "database", fetcher.Storage)
}

func main() {
	fetcher, err := initFetcher()
	if err != nil {
		slog.Error("Error initializing fetcher", "err", err)
		os.Exit(1)
	}
	defer fetcher.Close()
	if clear {
		clearDatabase(fetcher)
		return
	} else if maintain {
		maintainDatabase(fetcher)
		return
	} else if ping {
		if err = fetcher.Storage.Ping(); err != nil {
			slog.Error("Error pinging database", "err", err)
			os.Exit(1)
		}
		slog.Warn("Database ping successful", "database", fetcher.Storage)
		return
	}

	args := getArgs()
	if len(args) == 0 {
		slog.Error("Error: At least one URL is required\n\n")
		flags.Usage()
		os.Exit(1)
	}
	encoder := jstream.NewArrayEncoder[*resource.WebPage](os.Stdout, false)

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

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	envflags.EnvPrefix = "SCRAPE_"
	noContent = envflags.NewBool("NOTEXT", false)
	noContent.AddTo(&flags, "notext", "Skip text content")
	dbFlags = cmd.AddDatabaseFlags("DB", &flags)
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
 
  -h	
  	Show this help message`)

	flags.PrintDefaults()
}
