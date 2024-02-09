# About keys and database lookups of url artifacts

## Key Goals

`store.Key(url) uint64` generates a numeric key for storing and retrieving urls from a data store. This key is intended to fulfill the following criteria:

1. Idempotency: always generate the same key for a particular URL
2. Performance
3. Compatibility with most any relational DB storage.

Repeated key generation will always return the same key for a url. `scrape` only stores one instance of content per (canonical) url (more on that below) and isn't intended for storing versioned contents of urls. Any updated content replaces old comtent for a particular url.

Performance here largely boils down using a numeric key, as this is the most economical for storage, indexing, and sorting. Additionally upper bits of thekey provide same-domain grouping that can be used for partitioning if needed.

Compatibility here primarily boils down to the key's internal representaion being an int63 -- not all databases support uint64 natively, so the highest bit is always going to be a 0. 

## Key Structure

Keys are constructed in the following format:

- [bits 0-55]: A 56 bit numeric hash of the url. Currently generated using an `fnv64a` hash rounded down to 56 bits. (This implementation may change in a future iteration)
- [bits 56-62]: A 7 bit checksum of the url's domain. This provides some degree of natural grouping by domain. This _could_ support partitioning or sharding as well, but presently the goal/assumption of the system is that the database is time-constrained in size and should not require paritioning.
- [bit 63] Always 0

## Intended Usage

### Internal use only

Keys are only intended for optimizing database lookups and are not included in shared metadata/API responses. The key format is intended to be a direct representation of a URL for managing internal processes. The system provides a guarantee that it will fetch and return content for any url. It provides no such contract for IDs, nor does it provide any contract that the ID algorithm should not change. 

### Usage in tables 

Usage inside tables is at the discretion of the database implementation and may be implemented differently across storage engines. 

### System assumptions

The following isn't strictly germane to keys, but describes how they are used in the context of the broader system (which did/does inform their construction).

The `resource.WebPage` struct (which is passed into `URLDataStore.store()` implementations) has 3 keys that contain URL data.

1. `OriginalURL` this is literal url that was requested in the API. This value is not stored at all, but is returned to the client to ensure that a client can cross-reference a request.
2. `RequestedURL` This is the URL that was actually requested from the target server, it's the output of `resource.CleanURL(originalURL)`.
3. `URL` This is the URL of the page as reported by the actual content parser, and is considered the canonical URL of the page. It is reliably the content of `og:url` when present.

#### `urls` table

The `urls` table used the stored `URL` (canonical, derived from the content whenever possible) along with the paired key as its `id`.

#### `id_map` table

The `id_map` table stores mappings between `canonical_url` and `requested_url`.

When handling an inbound request, the `id_map` table is consulted first to see if there's a mapping for the `RequestedURL`. If there is, this metadata for this entry is returned to the client.


