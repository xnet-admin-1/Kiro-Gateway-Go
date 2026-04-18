package sigv4

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// buildCanonicalRequest creates the canonical request string for SigV4
func buildCanonicalRequest(req *http.Request, body []byte) string {
	canonicalURI := getCanonicalURI(req.URL.Path)
	canonicalQuery := getCanonicalQueryString(req.URL.Query())
	canonicalHeaders, signedHeaders := getCanonicalHeaders(req)
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(body))

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)
}

// getCanonicalURI returns the canonical URI
func getCanonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

// getCanonicalQueryString returns the canonical query string
func getCanonicalQueryString(values url.Values) string {
	if len(values) == 0 {
		return ""
	}

	var keys []string
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		for _, v := range values[k] {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

// getCanonicalHeaders returns canonical headers and signed headers
func getCanonicalHeaders(req *http.Request) (string, string) {
	var keys []string
	headers := make(map[string][]string)

	for k, v := range req.Header {
		lk := strings.ToLower(k)
		keys = append(keys, lk)
		headers[lk] = v
	}
	sort.Strings(keys)

	var canonicalHeaders []string
	for _, k := range keys {
		values := headers[k]
		// Trim spaces and join multiple values with comma
		var trimmedValues []string
		for _, v := range values {
			trimmedValues = append(trimmedValues, strings.TrimSpace(v))
		}
		canonicalHeaders = append(canonicalHeaders, k+":"+strings.Join(trimmedValues, ",")+"\n")
	}

	return strings.Join(canonicalHeaders, ""), strings.Join(keys, ";")
}
