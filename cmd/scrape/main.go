package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	nurl "net/url"
	"os"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/store/sqlite"
)

var (
	flags       flag.FlagSet
	noContent   bool
	dbPath      string
	csvPath     string
	csvUrlIndex int
	clear       bool
)

func initFetcher() (*scrape.StorageBackedFetcher, error) {
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(),
		sqlite.Factory(dbPath),
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
			log.Fatalf("Error opening CSV file %s: %s", csvPath, err)
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
				log.Fatalf("error reading CSV file %s: %s", csvPath, err)
			}
			rval = append(rval, record[csvUrlIndex])
		}
		return rval
	}
	return flags.Args()
}

func main() {
	fetcher, err := initFetcher()
	if err != nil {
		log.Fatalf("Error initializing fetcher: %s", err)
	}
	defer fetcher.Close()
	if clear {
		log.Fatal("Clearing database, not yet available here")
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	args := getArgs()
	if len(args) == 0 {
		log.Print("Error: At least one URL is required\n\n")
		flags.Usage()
		os.Exit(1)
	}
	for i := 0; i < len(args); i++ {
		// fmt.Println(args[i])
		url := args[i]
		parsedUrl, err := nurl.Parse(url)
		if err != nil {
			log.Printf("Error: invalue url %s, %s\n", url, err)
			usage()
		}
		page, err := fetcher.Fetch(parsedUrl)
		if err != nil {
			log.Printf("Error fetching %s, skipping: %v", parsedUrl.String(), err)
			continue
		}
		if noContent {
			page.ContentText = ""
		}
		err = encoder.Encode(page)
		if err != nil {
			log.Fatalf("failed to marshal for url %s, skipping: %v", url, err)
			continue
		}
		os.Stdout.Write([]byte(",\n"))
	}
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	flags.BoolVar(&noContent, "notext", false, "Skip text content")
	flags.StringVar(&dbPath, "database", sqlite.DefaultDatabase, "Database file path")
	flags.StringVar(&csvPath, "csv", "", "CSV file path")
	flags.IntVar(&csvUrlIndex, "csv-column", 1, "The index of the column in the CSV that contains the URLs")
	flags.BoolVar(&clear, "clear", false, "Clear the database and exit")
	// flags automatically adds -h and --help
	flags.Parse(os.Args[1:])
}

func usage() {
	fmt.Println(`Usage: 
	scrape [flags] :url [...urls]
 
  -h	
  	Show this help message`)

	flags.PrintDefaults()
}
