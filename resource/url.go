package resource

import (
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

func CleanURL(url *nurl.URL) *nurl.URL {
	if url == nil {
		return nil
	}
	v := url.Query()
	for _, p := range illegalParams {
		v.Del(p)
	}
	url.RawQuery = v.Encode()
	return url
}
