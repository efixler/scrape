# scrape

# Description
A tool to grab metadata (including description, image) and text content from web pages. Resource parsing is handled is handled by the excellent [go-trafilatura](https://github.com/markusmobius/go-trafilatura) module. Intended to be an out-of-the-box solution for grabbing web reources one-at-a-time of in batches as part of a classification or summarization toolchain. 

Built binaries are self-contained, and operate as a shell command or a server with a REST API. Scraped data may be stored/cached locally in a sqlite database to improve performance and reduce the nmber of outgoing requests. 

## Table of Contents

- [Status](#status)
- [Installing](#installing)
- [Usage as a CLI Application](#usage-as-a-cli-application)
- [Usage as a server](#usage-as-a-server)
- [Backing Storage](#backing-storage)
- [Roadmap](#roadmap)

## Status

## Installing


## Usage as a CLI Application
### Installing for shell usage
```
go get -u -v github.com/efixler/scrape/cmd/scrape
```
The `scrape` command provides single and batch retrieval, using or bypassing the connected storage db. It also provide command to manage the backing store.

```
scrape % ./scrape -h
Usage: 
        scrape [flags] :url [...urls]

  -C    Don't use the cache to retrieve content
  -p    Prune local storage and exit
  -P    Remove all stored entries from the cache
  -h    Show this help message
  -t    Get text content only
  -T    Don't get text content
  -t    Get text content only
```


