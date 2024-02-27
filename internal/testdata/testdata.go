package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	nurl "net/url"
	"os"
	"time"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
)

var (
	flags       flag.FlagSet
	csvPath     string
	csvUrlIndex int
	maxURLs     int
	client      *http.Client
	fetcher     *trafilatura.TrafilaturaFetcher
	saveDir     string = "data"
)

func main() {
	for _, url := range getArgs() {
		basename, err := getHtml(url)
		if err != nil {
			slog.Error("Error fetching", "url", url, "err", err)
			continue
		}
		parseAndStore(basename)
	}
}

func parseAndStore(basename string) error {
	localUrl, _ := nurl.Parse(fmt.Sprintf("file://%s/%s.html", saveDir, basename))
	resource, err := fetcher.Fetch(localUrl)
	if err != nil {
		slog.Error("Error re-loading", "file", localUrl, "err", err)
		return err
	}
	file, err := os.Create(fmt.Sprintf("%s/%s.json", saveDir, basename))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(resource)
	return nil
}

func getHtml(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Error creating request", "err", err, "url", url)
		return "", err
	}
	req.Header.Set("User-Agent", fetch.DefaultUserAgent)
	basename := basenameForUrl(url)
	slog.Info("Fetching", "url", url, "basename", basename)
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	file, err := os.Create(fmt.Sprintf("data/%s.html", basename))
	if err != nil {
		return "", err
	}
	defer file.Close()
	defer resp.Body.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}
	return basename, nil
}

func basenameForUrl(url string) string {
	sha := md5.New()
	sha.Write([]byte(url))
	bs := sha.Sum(nil)
	return fmt.Sprintf("%x", string(bs))
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
		for i := 0; i < maxURLs; i++ {
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

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	flags.StringVar(&csvPath, "csv", "", "CSV file path")
	flags.IntVar(&csvUrlIndex, "csv-column", 1, "The index of the column in the CSV that contains the URLs")
	flags.IntVar(&maxURLs, "max", 100, "The maximum number of URLs to process")
	flags.Parse(os.Args[1:])
	fetcher, _ = trafilatura.New(trafilatura.WithFiles("./data"), trafilatura.WithTimeout(30*time.Second))
}

func usage() {
	fmt.Println(`Usage: 
	scrape-test-capture [flags] :url [...urls]
 
  -h	
  	Show this help message`)
	flags.PrintDefaults()
}
