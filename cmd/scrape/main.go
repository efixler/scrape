package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	nurl "net/url"
	"os"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/store/sqlite"
	//"github.com/efixler/scrape/trafilatura"
)

var (
	flags     flag.FlagSet
	noContent bool
	createDB  bool
	dbPath    string
)

func main() {
	if createDB {
		err := sqlite.CreateDB(context.Background(), dbPath)
		if err != nil {
			log.Fatalf("Error creating database: %s", err)
		}
		log.Printf("Created database %s", dbPath)
		return
	}
	url := flags.Arg(0)
	parsedUrl, err := nurl.Parse(url)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		usage()
	}
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(),
		sqlite.Factory(dbPath),
	)
	if err != nil {
		log.Fatalf("Error creating storage backed fetcher: %s", err)
	}
	err = fetcher.Open(context.TODO())
	if err != nil {
		log.Fatalf("Error opening storage backed fetcher: %s", err)
	}
	defer fetcher.Close() // not sure if this will work right with the waitgroup
	// maybe filter utm_sources here
	page, err := fetcher.Fetch(parsedUrl)
	if err != nil {
		log.Fatalf("Error fetching %s: %s", parsedUrl.String(), err)
	}
	if noContent {
		page.ContentText = ""
	}
	marshaled, err := json.MarshalIndent(page, "", "  ")
	//marshaled, err := result.Metadata.MarshalText(result.Metadata)
	if err != nil {
		log.Fatalf("failed to marshal: %v", err)
	}
	fmt.Println(string(marshaled))
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	flags.BoolVar(&noContent, "notext", false, "Skip text content")
	flags.BoolVar(&createDB, "create", false, "Create the database and exit")
	flags.StringVar(&dbPath, "database", sqlite.DEFAULT_DB_FILENAME, "Database file path")
	// flags automatically adds -h and --help
	flags.Parse(os.Args[1:])
	if createDB {
		return
	}
	if flags.NArg() != 1 {
		fmt.Print("Error: URL is required\n\n")
		flags.Usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`Usage: 
	scrape [flags] :url [...urls]
 
  -C    Don't use the cache to retrieve content
  -p    Prune local storage and exit
  -P    Remove all stored entries from the cache
  -h	Show this help message`)

	flags.PrintDefaults()
}
