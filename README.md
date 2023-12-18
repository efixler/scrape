# scrape 
Fast web scraping

## Table of Contents

- [Description](#description)
- [Status](#status)
- [Installing](#installing)
- [Usage as a CLI Application](#usage-as-a-cli-application)
- [Usage as a server](#usage-as-a-server)
- [Roadmap](#roadmap)

## Description
`scrape` provides a self-contained low-to-no-setup tool to grab metadata and text content from web pages at medium scale.

Install or build the binary, and operate 

A tool to grab metadata (including description, image) and text content from web pages. Results are stored, so subsequent fetches of a particular URL are fast. Install the binary, and operate it as a shell command or a server with a REST API.

### Features:
- Reliable, accurate and fast parsing of web content using [go-trafilatura](https://github.com/markusmobius/go-trafilatura)
- Content stored in a database to minimize outbound requests and optimize performance
- Uses sqlite by default - no database server needed
- Adaptable to other storage backends

### Output Format
JSON output is a superset of Trafilatura format. 

| Field | Type | Description |
| ----  | ---- | ------------|
| `Hostname` | Domain name | The domain serving this resource |
| `RequestedURL` | URL | The URL that was requested |
| `URL` | URL | The (canonical) URL for the page. If not present, will report the same value as RequestedURL |
| `Date` | ISO8601 | The publish date of the page, in UTC time |
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
| `ContentText` | Text | The text of the page, will all HML removed |

Parsed field content is largely dependent on metadata included in the page. GIGO/YMMV.

Here's an example, with long fields truncated:
```json
{
  "RequestedURL": "https://www.theverge.com/2023/12/6/23990466/google-gemini-llm-ai-model",
  "Title": "Google launches Gemini, the AI model it hopes will take down GPT-4",
  "Author": "David Pierce",
  "URL": "https://www.theverge.com/2023/12/6/23990466/google-gemini-llm-ai-model",
  "Hostname": "www.theverge.com",
  "Description": "Google let OpenAI take the lead in the AI race — now, it’s mounting a comeback.",
  "Sitename": "The Verge",
  "Date": "2023-12-06T00:00:00Z",
  "Categories": null,
  "Tags": [
    "verge",
    "front-page",
    "google",
    "tech",
    "ai-artificial-intelligence",
    "news",
    "stream-23374468"
  ],
  "ID": "",
  "Fingerprint": "",
  "License": "",
  "Language": "en",
  "Image": "https://cdn.vox-cdn.com/thumbor/VyXNzykQp5u4OHAsOFeo2cJc34E=/0x0:1920x1080/1200x628/filters:focal(960x540:961x541)/cdn.vox-cdn.com/uploads/chorus_asset/file/25137172/gemini_mm_02.png",
  "PageType": "article",
  "ContentText": "It’s the beginning of a new era of AI at Google, says CEO Sundar Pichai: the Gemini era. Gemini is Google....",
  "FetchTime": "2023-12-18T02:32:48Z"
}
```




## Status
`scrape` is functional as a CLI program, accepting urls on the command line or CSV.

On an M1 Mac and a middling internet connection, and with a test sample of about 2K urls, resources are downloaded, stored, and returned at a rate of about 2-3/sec. Repeating that same set with the items having been loaded loads and returns stored items at about 120-150 results/sec. 

Both the code and the database could get optimized (it's basically single threaded right now, with miminal DB optimizations)


## Installing


## Usage as a CLI Application
### Installing for shell usage
```
go install github.com/efixler/scrape/cmd/scrape@latest
```
The `scrape` command provides single and batch retrieval, using or bypassing the connected storage db. It also provide command to manage the backing store.

#### Quickstart
```
> scrape -create
> scrape http://www.foo.com/some/url/path
```


```
scrape % ./scrape -h
Usage: 
        scrape [flags] :url [...urls]
 
  -h
        Show this help message
  -clear
        Clear the database and exit
  -create
        Create the database and exit
  -csv string
        CSV file path
  -csv-column int
        The index of the column in the CSV that contains the URLs (default 1)
  -database string
        Database file path (default "scrape_data/scrape.db")
  -notext
        Skip text content
```
# Usage as a server

# Roadmap