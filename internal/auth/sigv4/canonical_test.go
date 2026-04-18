package sigv4

import (
	"net/http"
	"net/url"
	"testing"
)

func TestBuildCanonicalRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		query    string
		headers  map[string]string
		body     []byte
		expected string
	}{
		{
			name:   "simple GET request",
			method: "GET",
			path:   "/",
			query:  "",
			headers: map[string]string{
				"Host":           "example.amazonaws.com",
				"X-Amz-Date":     "20150830T123600Z",
			},
			body: []byte{},
			expected: "GET\n/\n\nhost:example.amazonaws.com\nx-amz-date:20150830T123600Z\n\nhost;x-amz-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: tt.method,
				URL:    &url.URL{Path: tt.path, RawQuery: tt.query},
				Header: make(http.Header),
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := buildCanonicalRequest(req, tt.body)
			if result != tt.expected {
				t.Errorf("buildCanonicalRequest() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetCanonicalURI(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"/path", "/path"},
		{"/path/to/resource", "/path/to/resource"},
	}

	for _, tt := range tests {
		result := getCanonicalURI(tt.path)
		if result != tt.expected {
			t.Errorf("getCanonicalURI(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestGetCanonicalQueryString(t *testing.T) {
	tests := []struct {
		name     string
		values   url.Values
		expected string
	}{
		{
			name:     "empty query",
			values:   url.Values{},
			expected: "",
		},
		{
			name:     "single parameter",
			values:   url.Values{"Action": []string{"ListUsers"}},
			expected: "Action=ListUsers",
		},
		{
			name:     "multiple parameters",
			values:   url.Values{"Action": []string{"ListUsers"}, "Version": []string{"2010-05-08"}},
			expected: "Action=ListUsers&Version=2010-05-08",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCanonicalQueryString(tt.values)
			if result != tt.expected {
				t.Errorf("getCanonicalQueryString() = %q, want %q", result, tt.expected)
			}
		})
	}
}
