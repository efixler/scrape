package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	nurl "net/url"
	"os"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/envflags"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/cmd"
	"github.com/efixler/scrape/store"
)

var (
	flags       flag.FlagSet
	noContent   *envflags.Value[bool]
	dbSpec      *envflags.Value[cmd.DatabaseSpec]
	csvPath     *envflags.Value[string]
	csvUrlIndex *envflags.Value[int]
	clear       *envflags.Value[bool]
	maintain    *envflags.Value[bool]
)

func initFetcher() (*scrape.StorageBackedFetcher, error) {
	dbFactory, err := cmd.Database(dbSpec.Get())
	if err != nil {
		return nil, fmt.Errorf("error creating database factory: %s", err)
	}
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
		slog.Error("Error clearing database", "err", err)
		os.Exit(1)
	}
	slog.Warn("Database cleared")
}

func maintainDatabase(fetcher *scrape.StorageBackedFetcher) {
	db, ok := fetcher.Storage.(store.Maintainable)
	if !ok {
		slog.Error("Maintaining database not available for this storage backend")
		os.Exit(1)
	}
	err := db.Maintain()
	if err != nil {
		slog.Error("Error maintaining database", "err", err)
		os.Exit(1)
	}
	slog.Warn("Database maintenance complete")
}

func main() {
	fetcher, err := initFetcher()
	if err != nil {
		slog.Error("Error initializing fetcher", "err", err)
		os.Exit(1)
	}
	defer fetcher.Close()
	if clear.Get() {
		clearDatabase(fetcher)
		return
	}
	if maintain.Get() {
		maintainDatabase(fetcher)
		return
	}

	args := getArgs()
	if len(args) == 0 {
		slog.Error("Error: At least one URL is required\n\n")
		flags.Usage()
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	for _, url := range args {
		parsedUrl, err := nurl.Parse(url)
		if err != nil {
			slog.Error("invalid url, skipping", "url", url, "err", err)
			continue
		}
		page, err := fetcher.Fetch(parsedUrl)
		if err != nil {
			slog.Error("fetching url, skipping", "url", parsedUrl.String(), "err", err)
			continue
		}
		if noContent.Get() {
			page.ContentText = ""
		}
		err = encoder.Encode(page)
		if err != nil {
			slog.Error("failed to marshal, skipping: %v", "url", url, "err", err)
			continue
		}
		os.Stdout.Write([]byte(",\n"))
	}
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	envflags.EnvPrefix = "SCRAPE_"
	noContent = envflags.NewBool("NOTEXT", false)
	flags.Var(noContent, "notext", "Skip text content")
	dbSpec = cmd.NewDatabaseValue("DB", cmd.DefaultDatabase)
	flags.Var(dbSpec, "database", "Database type:path")
	csvPath = envflags.NewString("", "")
	flags.Var(csvPath, "csv", "CSV file path")
	csvUrlIndex = envflags.NewInt("CSV_COLUMN", 1)
	flags.Var(csvUrlIndex, "csv-column", "The index of the column in the CSV that contains the URLs")
	clear = envflags.NewBool("", false)
	flags.Var(clear, "clear", "Clear the database and exit")
	maintain = envflags.NewBool("", false)
	flags.Var(maintain, "maintain", "Execute database maintenance and exit")
	logLevel := envflags.NewLogLevel("LOG_LEVEL", slog.LevelWarn)
	flags.Var(logLevel, "log-level", "Set the log level [debug|error|info|warn]")
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
