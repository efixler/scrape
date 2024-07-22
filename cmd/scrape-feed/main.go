package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	nurl "net/url"
	"os"

	"github.com/efixler/scrape/fetch/feed"
)

var (
	flags    flag.FlagSet
	urlsOnly bool
)

func main() {
	args := flags.Args()
	if len(args) != 1 {
		slog.Error("Error: One feed URL is required")
		flags.Usage()
		os.Exit(1)
	}
	feedUrl, err := nurl.Parse(args[0])
	if err != nil {
		slog.Error("Error: Invalid feed URL", "url", args[0], "err", err)
		flags.Usage()
		os.Exit(1)
	}
	feedFetcher := feed.NewFeedFetcher(feed.DefaultOptions)
	resource, err := feedFetcher.Fetch(feedUrl)
	if err != nil {
		slog.Error("Error fetching", "url", feedUrl, "err", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if urlsOnly {
		err = encoder.Encode(resource.ItemLinks())
	} else {
		err = encoder.Encode(resource)
	}
	if err != nil {
		slog.Error("Error encoding", "url", feedUrl, "err", err)
		os.Exit(1)
	}
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	flags.BoolVar(&urlsOnly, "U", false, "Only output URLs from the feed")
	flags.Parse(os.Args[1:])
	logger := slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	))
	slog.SetDefault(logger)
}

func usage() {
	fmt.Println(`Usage: 
	scrape-feed [flags] :feed-url
 
  -h	
  	Show this help message`)

	flags.PrintDefaults()
}
