# scrape 
Fast web scraping

## Table of Contents

- [Description](#description)
- [Output Format](#output-format)
- [Status](#status)
- [Usage as a CLI Application](#usage-as-a-cli-application)
- [Usage as a Server](#usage-as-a-server)
  - [API](#api)
- [Building and Developing](#building-and-developing)
  - [Building](#building)
  - [Using the Docker](#using-the-docker)
- [Roadmap](#roadmap)

## Description
`scrape` provides a self-contained low-to-no-setup tool to grab metadata and text content from web pages at medium scale.

 Results are stored, so subsequent fetches of a particular URL are fast. Install the binary, and operate it as a shell command or a server with a REST API.

### Features:
- Reliable, accurate and fast parsing of web content using [go-trafilatura](https://github.com/markusmobius/go-trafilatura)
- Scraped content backed by a database to minimize outbound requests and optimize performance
  - Resources stored with a TTL (30 days by default) to limit storage size and prevent staleness
  - Uses [sqlite](https://www.sqlite.org/index.html) - no external server needed
  - Adaptable to other storage backends

## Output Format
JSON output is a superset of Trafilatura format. Empty fields may be omitted in responses.

| Field | Type | Description |
| ----  | ---- | ------------|
| `OriginalURL` | String | Exactly the url that was in the inbound request |
| `RequestedURL` | URL | The URL that was actually requested. (Some URL elements may be rewritten before the outgoing request) |
| `StatusCode` | Int | The status code returned by the target server when fetching this page |
| `Error` | String | Error message(s), if there were any, while processing this page |
| `Hostname` | Domain name | The domain serving this resource |
| `URL` | URL | The (canonical) URL for the page, as reported by the page itself. If the page doesn't supply that, this field will contain  the same value as RequestedURL |
| `Date` | ISO8601 | The publish date of the page, in UTC time |
| `FetchTime` | ISO8601 | The time that URL was retrieved |
| `Sitename` | Text | Identifies the publisher. Can be domain, company name, or other text, IRL usage not consistent |
| `Image` | URL | Hero image link |
| `Title` | Text | The page's title | 
| `Author` | Text | Author |
| `Description` | Text | Page summary or excerpt |
| `Categories` | Array | Content categories, if supplied |
| `Tags` | Array | Tags, if supplied |
| `ID` | Text | Generally empty |
| `License` | Text | Generally empty |
| `Language` | Text | 2-letter language code |
| `PageType` | Text | If it's there it's usually "article" following the `og`` usage |
| `ContentText` | Text | The text of the page, with all HTML removed |

Parsed field content is largely dependent on metadata included in the page. GIGO/YMMV.

Here's an example, with long fields truncated:
```json
{
  "OriginalURL": "https://www.nasa.gov/missions/webb/nasas-webb-stuns-with-new-high-definition-look-at-exploded-star/",
  "RequestedURL": "https://www.nasa.gov/missions/webb/nasas-webb-stuns-with-new-high-definition-look-at-exploded-star/",
  "Title": "NASA’s Webb Stuns With New High-Definition Look at Exploded Star - NASA",
  "Author": "Steve Sabia",
  "URL": "https://www.nasa.gov/missions/webb/nasas-webb-stuns-with-new-high-definition-look-at-exploded-star/",
  "Hostname": "www.nasa.gov",
  "Description": "Like a shiny, round ornament ready to be placed in the perfect spot on a holiday tree, supernova remnant Cassiopeia A (Cas A) gleams in a new image from",
  "Sitename": "NASA",
  "Date": "2023-12-10T00:00:00Z",
  "Categories": [
    "Astrophysics",
    "Goddard Space Flight Center",
    "James Webb Space Telescope (JWST)",
    "Missions",
    "Nebulae",
    "Science \u0026 Research",
    "Stars",
    "Supernovae",
    "The Universe"
  ],
  "Tags": [
    "Astrophysics",
    "Goddard Space Flight Center",
    "James Webb Space Telescope (JWST)",
    "Missions",
    "Nebulae",
    "Science \u0026 Research",
    "Stars",
    "Supernovae",
    "The Universe"
  ],
  "ID": "",
  "Fingerprint": "",
  "License": "",
  "Language": "en",
  "Image": "https://www.nasa.gov/wp-content/uploads/2023/12/webb-stsci-01hggzdyh8ghhssnwzd71mf0xh-2k.png",
  "PageType": "article",
  "ContentText": "Mysterious features hide in near-infrared light Like a shiny, round ornament ready to be placed in the perfect spot on a holiday tree, supernova remnant Cassiopeia A (Cas A) gleams in a new image from NASA’s James Webb Space Telescope. As part of the 2023...(there's about 10x more content in this example, truncated in the docs for readability)",
  "FetchTime": "2023-12-18T03:37:14Z"
}
```


## Status
`scrape` and `scrape-server` are both functional as described here. Both should be buildable in any environment that has `cgo` and runnable wherever there are `sqlite` libs.

On an M1 Mac and a middling internet connection, and with a test sample of about 2K urls, resources are downloaded, stored, and returned at a rate of about 2-3/sec. Repeating that same set with the items having been loaded loads and returns stored items at about 120-150 results/sec. 

Since the above was written the SQLite database has been performance-tuned quite a bit. I'll update the above when I have a systematic benchmark to apply.


## Usage as a CLI Application
### Installing for shell usage
```
go install github.com/efixler/scrape/cmd/scrape@latest
```
The `scrape` command provides single and batch retrieval, using or bypassing the connected storage db. It also provide command to manage the backing store.

### Quickstart
```
> scrape http://www.foo.com/some/url/path
```
That's actually it. The database will be created if it doesn't exist already.

```
scrape % ./scrape -h
Usage: 
        scrape [flags] :url [...urls]
 
  -h
        Show this help message
  -clear
        Clear the database and exit
  -csv string
        CSV file path
  -csv-column int
        The index of the column in the CSV that contains the URLs (default 1)
  -database string
        Database file path (default "scrape_data/scrape.db")
  -maintain
        Execute database maintenance and exit
  -notext
        Skip text content
```
## Usage as a Server
The server provides a REST API to get resource data one-at-a-time or in bulk. The root URL serves up a page that can be used to spot check results for any url.

`scrape-server` is intended for use in closed environments at medium scale. There's no authentication, rate limiting or url sanitization beyond encoding checks. Don't deploy this on an open public network. Do deploy it as a sidecar, in a firewalled environment, or another environment that won't get unbounded quantities of hits.

### Installation
```
go install github.com/efixler/scrape/cmd/scrape-server@latest
```
```
Usage: 
        scrape-server [-port nnnn] [-h]
 
  -h
        Show this help message
  -database string
        Database path. If the database doesn't exist, it will be created. 
        Use ':memory:' for an in-memory database (default "scrape_data/scrape.db")
  -log-level value
        Set the log level [debug|error|info|warn] (default info)
  -port int
        The port to run the server on (default 8080)
```

Use caution when using the in-memory database: There are currently no constraints on database size.

### Web Interface

The root path of the server (`/`) is browsable and provides a simple url to test URLs and results.

### API 

#### batch 
Returns the metadata for the supplied list of URLs. Returned metadatas are not guaranteed to be
in the same order as the request. 

Errors behave differently in single (`extract`) vs `batch` mode. In `batch` mode, an error that prevents
the entire request from executing will result in a 4xx or 5xx error code with error message in the result
body (like `extract`), but errors for individual pages will result in an error message getting included in
the metadata object for that URL; most of the remaining metadata will likely be empty. 

The `Error` key will be absent for pages that have no errors.  Testing for the presence of the `Error` 
key in a page metadata object is sufficient to indicate that there was an error extracting data from 
that page. 

| Param | Description | Required | 
| -------- | ------ | ----------- |
| urls | A JSON array of the urls to fetch | Y |

#### extract [GET, POST]
Fetch the metadata and text content for the specified URL. Errors will be reported using HTTP error codes.
The `Error` key should be absent (behavior subject to change).

Returns JSON payload as decribed above.

| Param | Description | Required | 
| -------- | ------ | ----------- |
| url | The url to fetch. Should be url encoded. | Y |


#### Global Params 
These params work for any endpoint 
| Param | Value | Description |
| ----- | ----- | ----------- |
| pp | 1 | Pretty print JSON output |

## Building and Developing

### Building 

Best to build with `make`. The Makefile has a help target, here is its output:

```
Usage:
  make 
  build            build the binaries, to the build/ folder (post vet)
  clean            clean the build directory
  docker-build     build the docker image
  docker-run       run the docker image, binding to port 8080, or the env value of SCRAPE_PORT
  vet              fmt, vet, and staticcheck
  help             show this help message
```

### Using the Docker
The Docker is mostly intended for distribution and testing. The docker build
pulls the source from the repo via `go install` and the `latest` tag, so, this build will
not be up to date with local changes.

By default, the Docker will run using an in-memory database. This can be changed by modifying the arguments passed to `scrape-server` in the Dockerfile `ENTRYPOINT`.

The `docker-run` make target docker will mount a local folder called `docker/data` and bind that to the container for file storage. If you want to use a file-based db you can use this directory, or update the `Makefile` to mount the desired local directory. 


## Roadmap
### TODOs

- ~~Bulk fetch in the web server~~
  - Test and benchmark concurrency options for bulk fetch (partially done)
- ~~Optimize SQLite for both file-based and in-memry storage~~
- Enforce TTL eviction and/or DB capacity limits
      - TTL currently only forces a re-fetch after TTL expiry 
- Better (more defensive, most structed/easier for callers to understand) error handling
- Improve test coverage
- Headless fallback for pages that require Javascript
- RSS Feed parsing and retrieval of URLs referenced in RSS feeds
- Expose more configuration items as needed
  - Database path
  - Default TTL
- Add an adaptor for MySQL
- Compress text content in the DB, probably using zstd if this can meaningfully reduce database size -- text content entries can be relatively large.