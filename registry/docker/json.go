package docker

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

var ErrNoMorePages = errors.New("no more pages")

// Matches an RFC 5988 (https://tools.ietf.org/html/rfc5988#section-5)
// Link header. For example,
//
//	<http://registry.example.com/v2/_catalog?n=5&last=tag5>; type="application/json"; rel="next"
//
// The URL is _supposed_ to be wrapped by angle brackets `< ... >`,
// but e.g., quay.io does not include them. Similarly, params like
// `rel="next"` may not have quoted values in the wild.
var nextLinkRE = regexp.MustCompile(`^ *<?([^;>]+)>? *(?:;[^;]*)*; *rel="?next"?(?:;.*)?`)

func getNextLink(resp *http.Response) (string, error) {
	for _, link := range resp.Header[http.CanonicalHeaderKey("Link")] {
		parts := nextLinkRE.FindStringSubmatch(link)
		if parts != nil {
			return parts[1], nil
		}
	}
	return "", ErrNoMorePages
}

// getPaginatedJson accepts a string and a pointer, and returns the
// next page URL while updating pointed-to variable with a parsed JSON
// value. When there are no more pages it returns `ErrNoMorePages`.
func (r *Registry) getPaginatedJSON(url string, response interface{}) (string, error) {
	resp, err := r.Client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(response)
	if err != nil {
		return "", err
	}
	next, err := getNextLink(resp)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(next, r.URL) {
		next = r.URL + next
	}

	return next, nil
}
