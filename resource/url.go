package resource

import (
	"log/slog"
	nurl "net/url"
)

// var StripParamsMatch = regexp.MustCompile(`^(utm_\w+)`)

var illegalParams = []string{
	"utm_source",
	"utm_medium",
	"utm_campaign",
	"utm_term",
	"utm_content",
	"utm_brand",
}

// CleanURL removes utm_ parameters from the URL
func CleanURL(url *nurl.URL) *nurl.URL {
	if url == nil {
		return nil
	}
	v := url.Query()
	for _, p := range illegalParams {
		v.Del(p)
	}
	slog.Debug("CleanURL", "url", url.String(), "rawQuery", url.RawQuery, "to", v.Encode())
	url.RawQuery = v.Encode()
	url.Fragment = ""
	return url
}
