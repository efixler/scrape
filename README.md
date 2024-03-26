# scrape 
Fast web scraping

[![Go Reference](https://pkg.go.dev/badge/github.com/efixler/scrape.svg)](https://pkg.go.dev/github.com/efixler/scrape)
[![Build status](https://github.com/efixler/scrape/actions/workflows/test.yml/badge.svg)](https://github.com/efixler/scrape/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/efixler/scrape)](https://goreportcard.com/report/github.com/efixler/scrape)
[![License GPL](https://img.shields.io/badge/License-GPL--3.0-informational)](https://github.com/efixler/scrape/?tab=GPL-3.0-1-ov-file)

## Table of Contents

- [Description](#description)
- [Output Format](#output-format)
- [Usage as a CLI Application](#usage-as-a-cli-application)
- [Usage as a Server](#usage-as-a-server)
  - [Web Interface](#web-interface)
  - [API](#api)
  - [Healthchecks](#healthchecks)
- [Database Options](#database-options)
- [Building and Developing](#building-and-developing)
  - [Building](#building)
  - [Using the Docker](#using-the-docker)
- [Roadmap](#roadmap)
- [Acknowledgements](#acknowledgements)

## Description
`scrape` provides a self-contained low-to-no-setup tool to grab metadata and text content from web pages at medium scale. 

 Results are stored, so subsequent fetches of a particular URL are fast. Install the binary, and operate it as a shell command or as a server with a REST API. The default SQLite storage backend is performance-optimized and can store to disk or in memory. MySQL is also supported. Resources are stored with a configurable TTL. 

 The `scrape` cli tool provides shell access to scraped content via command-line entry or CSV files, and also provides database management functionality. `scrape-server` provides web and API access to content metadata in one-offs or batches.

 RSS and Atom feeds are supported via an endpoint in `scrape-server`. Loading a feed returns the parsed results for all item links in the feed. 

 The `scrape` and `scrape-server` binaries should be buildable and runnable in any `cgo`-enabled environment where `SQLite3` is present. A docker build is also included.

## Output Format
JSON output is a superset of Trafilatura fields. Empty fields may be omitted in responses.

| Field | Type | Description |
| ----  | ---- | ------------|
| `url` | String (URL) | The (canonical) URL for the page, as reported by the page itself. If the page doesn't supply that, this field will contain  the same value as RequestedURL |
| `requested_url` | String (URL) | The URL that was actually requested. (Some URL params (e.g. utm_*) may be stripped before the outbound request) |
| `original_url` | String (URL) | Exactly the url that was in the inbound request |
| `fetch_time` | ISO8601 | The time that URL was retrieved |
| `status_code` | Int | The status code returned by the target server when fetching this page |
| `error` | String | Error message(s), if there were any, while processing this page |
| `hostname` | Domain name | The domain serving this resource |
| `date` | ISO8601 | The publish date of the page, in UTC time |
| `sitename` | String | Identifies the publisher. Can be domain, company name, or other text, IRL usage not consistent |
| `title` | String | The page's title | 
| `authors` | []String | Authors |
| `description` | String | Page summary or excerpt |
| `categories` | []String | Content categories, if supplied |
| `tags` | []String | Tags, if supplied |
| `language` | String | 2-letter language code |
| `page_type` | String | If it's there it's usually "article" following the `og`` usage |
| `image` | String (URL) | Hero image link |
| `license` | String | Generally empty |
| `content_text` | String | The text of the page, with all HTML removed |

Parsed field content is largely dependent on metadata included in the page. GIGO/YMMV.

Here's an example, with long fields truncated:
```json
{
  "url": "https://www.nasa.gov/missions/webb/nasas-webb-stuns-with-new-high-definition-look-at-exploded-star/",
  "requested_url": "https://www.nasa.gov/missions/webb/nasas-webb-stuns-with-new-high-definition-look-at-exploded-star/",
  "original_url": "https://www.nasa.gov/missions/webb/nasas-webb-stuns-with-new-high-definition-look-at-exploded-star/",
  "fetch_time": "2024-01-09T03:57:44Z",
  "status_code": 200,
  "hostname": "www.nasa.gov",
  "date": "2023-12-10T00:00:00Z",
  "sitename": "NASA",
  "title": "NASA’s Webb Stuns With New High-Definition Look at Exploded Star - NASA",
  "authors": [
      "Steve Sabia"
  ],
  "description": "Like a shiny, round ornament ready to be placed in the perfect spot on a holiday tree, supernova remnant Cassiopeia A (Cas A) gleams in a new image from",
  "categories": [
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
  "tags": [
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
  "language": "en",
  "image": "https://www.nasa.gov/wp-content/uploads/2023/12/webb-stsci-01hggzdyh8ghhssnwzd71mf0xh-2k.png",
  "page_type": "article",
  "content_text": "Mysterious features hide in near-infrared light Like a shiny, round ornament ready to be placed in the perfect spot on a holiday tree, supernova remnant Cassiopeia A (Cas A) gleams in a new image from NASA’s James Webb Space Telescope. As part of the 2023...(there's about 10x more content in this example, truncated in the docs for readability)",
}
```

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

In addition to http[s] URLs, file:/// urls are supported, using the current working directory as the base path.

Flags:
 
  -h
        Show this help message
  -clear
        Clear the database and exit
  -create
        Create the database and exit
  -csv value
        CSV file path
  -csv-column value
        The index of the column in the CSV that contains the URLs
        Environment: SCRAPE_CSV_COLUMN (default 1)
  -database value
        Database type:path
        Environment: SCRAPE_DB (default sqlite:scrape_data/scrape.db)
  -db-password value
        Database password
        Environment: SCRAPE_DB_PASSWORD
  -db-user value
        Database user
        Environment: SCRAPE_DB_USER
  -log-level value
        Set the log level [debug|error|info|warn]
        Environment: SCRAPE_LOG_LEVEL (default WARN)
  -maintain
        Execute database maintenance and exit
  -notext
        Skip text content
        Environment: SCRAPE_NOTEXT
  -ping
        Ping the database and exit
```
## Usage as a Server
The server provides a REST API to get resource data one-at-a-time or in bulk. The root URL serves up a page that can be used to spot check results for any url.

`scrape-server` is intended for use in closed environments at medium scale. There's no authentication, rate limiting or url sanitization beyond encoding checks. Don't deploy this on an open public network. Do deploy it as a sidecar, in a firewalled environment, or another environment that won't get unbounded quantities of hits.

### Installation
```
go install github.com/efixler/scrape/cmd/scrape-server@latest
```
```
scrape % ./build/scrape-server -h

Usage:
-----
scrape-server [-port nnnn] [-h]

Some options have environment variable equivalents. Invalid environment settings
are ignored. Command line options override environment variables.
	
If environment variables are set, they'll override the defaults displayed in this 
help message.
 
Command line options:
--------------------

  -h	
  	Show this help message
  -database value
    	Database type:path
    	Environment: SCRAPE_DB (default sqlite:scrape_data/scrape.db)
  -db-password value
    	Database password
    	Environment: SCRAPE_DB_PASSWORD
  -db-user value
    	Database user
    	Environment: SCRAPE_DB_USER
  -log-level value
    	Set the log level [debug|error|info|warn]
    	Environment: SCRAPE_LOG_LEVEL
  -port value
    	Port to run the server on
    	Environment: SCRAPE_PORT (default 8080)
  -profile
    	Enable profiling at /debug/pprof
    	Environment: SCRAPE_PROFILE
  -ttl value
    	TTL for fetched resources
    	Environment: SCRAPE_TTL (default 720h0m0s)
  -user-agent value
    	User agent to use for fetching
    	Environment: SCRAPE_USER_AGENT (default Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0)
```

### Web Interface

The root path of the server (`/`) is browsable and provides a simple url to test URLs and results.

![Alt text](internal/server/pages/webui-control.png)

The pulldown on the right lets you select between loading results for a page url or for a feed.

### API 

#### batch [POST]
Returns the metadata for the supplied list of URLs. Returned metadatas are not guaranteed to be
in the same order as the request. 

The `batch` endpoint behaves indentically to the `extract` endpoint in all ways except two:
1. The endpoint returns an array of the JSON payload described above
1. When individual items have errors, the request will still return with a 200 status code. Inspect the 
payload for individual items to determine the status of an individual item request.

| Param | Description | Required | 
| -------- | ------ | ----------- |
| urls | A JSON array of the urls to fetch | Y |

#### extract [GET, POST]
Fetch the metadata and text content for the specified URL. Returns JSON payload as decribed above.

If the server encounters an error fetching a requested URL, the status code for the request will be set to 422 (Unprocessable Entity). This may change.

The returned JSON payload will include a `StatusCode` field in all cases, along with an `Error` field when
there's an error fetching or parsing the requested content.

##### Params

| Param | Description | Required | 
| -------- | ------ | ----------- |
| url | The url to fetch. Should be url encoded. | Y |

##### Errors

| StatusCode | Description | 
| ---------- | ----------- |
| 415 | The requested resource was for a content type not supported by this service |
| 422 | The request could not be completed |
| 504 | The request for the target url timed out |

In all other cases, requests should return a 200 status code, and any errors received when fetching a resource
will be included in the returned JSON payload.

#### feed [GET, POST]

Feed parses an RSS or Atom feed and returns the parsed results for each of the item links in the feed.

##### Params

| Param | Description | Required | 
| -------- | ------ | ----------- |
| url | The feed url to fetch. Should be url encoded. | Y |

##### Errors

| StatusCode | Description | 
| ---------- | ----------- |
| 422 | The url was not a valid feed |
| 504 | Request for the feed timed out |

#### Global Params 
These params work for any endpoint 
| Param | Value | Description |
| ----- | ----- | ----------- |
| pp | 1 | Pretty print JSON output |


### Healthchecks 

`scrape` has two healthchecks:

#### /.well-known/health

This is a JSON endpoint that returns data on the application's state, including memory and
database runtime info.

#### /.well-known/heartbeat

This just returns a status `200` with the content `OK`

## Database Options

`scrape` supports SQLite or MySQL for data storage. Your choice depends on your requirements and environment.

### SQLite

SQLite is the default storage engine for `scrape`. There's no need for setup -- the database will autocreate if it doesn't exist. 

SQLite is ideal when there's a 1:1 relationship between the service and its backing store, or if you're running the service on a workstation or a 'real' computer.

If you deploy the Docker container to a cloud provider like GCP or AWS there will usually be a few hundred MB of disk storage associated with the container. Storage requirements typically translate to
about 3000 url metadatas per 100MB. 

When your container shuts down, previously stored data will be lost. This may or may not be a problem, depending on your application. Disk space can be monitored with the healthcheck.

It is also possible to mount a block drive to a container for persistent storage independent of the 
container lifecycle.

To specify a path to a SQLite database using the command line `-database` switch or the equivalent `SCRAPE_DB` environment variable, use the form `sqlite:/path/to.db`. The special form `sqlite::memory:` is supported for a transient, in-memory database.

### MySQL

MySQL will be a better choice for applications that run under higher volumes and/or where multiple
service instances want to share a storage backend.

Here are the configuration options for MySQL:

| Flag | Environment | Description | Example | 
| -------- | ------ | ----------- | -------- |
| -database | SCRAPE_DB | mysql: + addr:port | `mysql:mysql.domain.co:3306` |
| -db-password | SCRAPE_DB_PASSWORD | Password | lkajd901e109i^jhj% |
| -db-user | SCRAPE_DB_USER | Username for mysql connections | `scrape_app` (default) |

Create the MySQL database by running `scrape -create` with the applicable values above. For
database creation a privileged user is required. The database will be provisioned with two
roles; `scrape_app` for app operations and a `scrape_admin` role with full privileges to the
schema. Assign these roles to users as appropriate.

## Building and Developing

### Building 

Best to build with `make`. The Makefile has a help target, here is its output:

```
scrape % make help

Usage:
  make 
  build            build the binaries, to the build/ folder (default target)
  clean            clean the build directory
  docker-build     build a docker image on the current platform, for local use
  docker-push      push an amd64/arm64 image to Docker Hub or to a registry specfied by CONTAINER_REGISTRY
  docker-run       run the docker image, binding to port 8080, or the env value of SCRAPE_PORT
  test             run the tests
  vet              fmt, vet, and staticcheck
  cognitive        run the cognitive complexity checker
  help             show this help message
```

### Using the Docker
The `docker-build` target will build a docker on the current architecture. Run this image with
`docker-run` to bring up a local instance of the service for testing. You don't need any Go 
tooling installed locally to build and run via the Docker.

To push a image to a registry, use `docker-push`. This will build a multiplatform amd64/arm64 image
and deploy it upstream. This image is appropriate for cloud platform deployment. The registry username 
or organization should match the username or organization of the working repo, and you need the 
appropriate permissions. 

The docker builds with local sources so use caution when pushing to registries.

By default, the Docker will run using a sqlite database at `/scrape_data/scrape.db` on the container itself. This can be changed via the `SCRAPE_DB` environment variable. You can also
use a mount to mount this locally.

The `docker-run` make target will mount a local folder called `docker/data` and bind that to the container for file storage. If you want to use a file-based db you can use this directory, or update the `Makefile` to mount the desired local directory. 


## Roadmap
- Outbound request pacing
- Expose outbound request options (headers, timeouts, etc)
- Headless fallback for pages that require Javascript
- Explore performance optimizations, e.g.
  - Batch request parallelization
  - zstd compression for stored resources
- Explore alternate fetch/parse backends

Feature request or bug? Post issues [here](https://github.com/efixler/scrape/issues).


## Acknowledgements

`scrape` is powered by:
-  [go-trafilatura](https://github.com/markusmobius/go-trafilatura) HTML parsing
-  [gofeed](https://github.com/mmcdole/gofeed) Atom and RSS parsing
-  [sqlite](https://www.sqlite.org/index.html) data storage
-  [go-sqlite3](https://github.com/mattn/go-sqlite3) SQLite client
-  [go-mysql](https://github.com/go-sql-driver/mysql) MySQL client