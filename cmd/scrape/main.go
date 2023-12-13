package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	nurl "net/url"
	"os"
	"time"

	_ "github.com/go-shiori/go-readability"
	_ "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	flags      flag.FlagSet
	content    bool
)

func main() {
	url := flags.Arg(0)
	parsedUrl, err := nurl.Parse(url)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		usage()
	}
	response, err := httpClient.Get(url)
	if err != nil {
		log.Fatalf("Could not get url: %s", err)
	}
	defer response.Body.Close()
	topts := trafilatura.Options{
		IncludeImages:      true,
		OriginalURL:        parsedUrl,
		FallbackCandidates: &trafilatura.FallbackConfig{},
	}
	result, err := trafilatura.Extract(response.Body, topts)
	if err != nil {
		log.Fatalf("failed to extract: %v", err)
	}
	if content {
		fmt.Println(result.ContentText)
		return
	}
	marshaled, err := json.MarshalIndent(result.Metadata, "", "  ")
	//marshaled, err := result.Metadata.MarshalText(result.Metadata)
	if err != nil {
		log.Fatalf("failed to marshal: %v", err)
	}
	fmt.Println(string(marshaled))
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.BoolVar(&content, "t", false, "Get text content only")
	// flags automatically adds -h and --help
	flags.Usage = usage
	flags.Parse(os.Args[1:])
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
  -t	Get text content only
  -T 	Don't get text content`)

	flags.PrintDefaults()
}
