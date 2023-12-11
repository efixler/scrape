package main

import (
	"encoding/json"
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
)

func main() {
	if len(os.Args) < 2 {
		usage(1)
	}
	url := os.Args[1]
	if url == "-h" || url == "--help" {
		usage(0)
	}
	parsedUrl, err := nurl.Parse(url)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		usage(1)
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
	// fmt.Printf("Result for %s: \n%#v\n", url, result.Metadata)
	marshaled, err := json.MarshalIndent(result.Metadata, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal: %v", err)
	}
	fmt.Println(string(marshaled))
	fmt.Println("Extracted content to follow:\n-----------------------------")
	fmt.Println(result.ContentText)

}

func usage(exitCode int) {
	fmt.Println(`Usage: 
	scrape :url
	`)
}
