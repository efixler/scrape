package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	nurl "net/url"
	"os"
	"time"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/sqlite"
	_ "github.com/go-shiori/go-readability"
	_ "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	flags      flag.FlagSet
	// content_only bool
	noContent bool
	createDB  bool
	dbPath    string
)

func fetch(url string) (*resource.WebPage, error) {
	// change this interface to work through higher level store.
	db, err := sqlite.Open(context.TODO(), dbPath)
	if err != nil {
		return nil, err
	}
	parsedUrl, err := nurl.Parse(url)
	if err != nil {
		return nil, err
	}
	item, err := db.Fetch(parsedUrl)
	if err != nil {
		return nil, err
	}
	if item != nil {
		return &item.Data, nil
	}
	// if we get here we're not cached

	response, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	topts := trafilatura.Options{
		IncludeImages:      true,
		OriginalURL:        parsedUrl,
		FallbackCandidates: &trafilatura.FallbackConfig{},
	}
	result, err := trafilatura.Extract(response.Body, topts)
	if err != nil {
		return nil, err
	}
	resource := &resource.WebPage{
		Metadata:    result.Metadata,
		ContentText: result.ContentText,
		ParsedUrl:   parsedUrl,
	}
	// this is annoying and dumb
	sd := &store.StoredUrlData{
		Data: *resource,
	}
	// need to make sure the app quit waits for the go func to finish
	// go func() {
	// 	// possibly need a better way to do this, or at least to capture errors
	// 	//
	_, err = db.Store(sd)
	if err != nil {
		log.Printf("Error storing %s: %s", url, err)
	}
	// }()

	return resource, nil
}

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

	// maybe filter utm_sources here
	page, err := fetch(parsedUrl.String())
	if err != nil {
		log.Fatalf("Error fetching %s: %s", url, err)
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
	flags.BoolVar(&noContent, "T", false, "Skip text content")
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
  -h	Show this help message
  -T 	Don't get text content`)

	flags.PrintDefaults()
}
