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
	"strings"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/sqlite"
)

var (
	flags       flag.FlagSet
	logLevel    slog.Level = slog.LevelWarn
	noContent   bool
	dbPath      string = sqlite.DefaultDatabase
	csvPath     string
	csvUrlIndex int
	clear       bool
	maintain    bool
)

func initFetcher() (*scrape.StorageBackedFetcher, error) {
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(*trafilatura.DefaultOptions),
		sqlite.Factory(sqlite.WithFile(dbPath)),
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
	if csvPath != "" {
		csvFile, err := os.Open(csvPath)
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
			rval = append(rval, record[csvUrlIndex])
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
	if clear {
		clearDatabase(fetcher)
		return
	}
	if maintain {
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
		if noContent {
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
	flags.BoolVar(&noContent, "notext", false, "Skip text content")
	flags.StringVar(&dbPath, "database", dbPath, "Database file path")
	flags.StringVar(&csvPath, "csv", "", "CSV file path")
	flags.IntVar(&csvUrlIndex, "csv-column", 1, "The index of the column in the CSV that contains the URLs")
	flags.BoolVar(&clear, "clear", false, "Clear the database and exit")
	flags.BoolVar(&maintain, "maintain", false, "Execute database maintenance and exit")
	flags.Func(
		"log-level",
		"Set the log level [debug|error|info|warn]\n (default info)",
		func(s string) error {
			switch strings.ToLower(s) {
			case "debug":
				logLevel = slog.LevelDebug
			case "info":
				logLevel = slog.LevelInfo
			case "warn":
				logLevel = slog.LevelWarn
			case "error":
				logLevel = slog.LevelError
			}
			return nil
		})
	// flags automatically adds -h and --help
	flags.Parse(os.Args[1:])
	logger := slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: logLevel,
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
