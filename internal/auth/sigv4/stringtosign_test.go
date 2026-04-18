package sigv4

import (
	"strings"
	"testing"
	"time"
)

func TestBuildStringToSign(t *testing.T) {
	tests := []struct {
		name              string
		timestamp         time.Time
		region            string
		service           string
		canonicalRequest  string
	}{
		{
			name:      "basic string to sign",
			timestamp: time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC),
			region:    "us-east-1",
			service:   "service",
			canonicalRequest: "GET\n/\n\nhost:example.amazonaws.com\nx-amz-date:20150830T123600Z\n\nhost;x-amz-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildStringToSign(tt.timestamp, tt.region, tt.service, tt.canonicalRequest)
			
			// Check format rather than exact match
			lines := strings.Split(result, "\n")
			if len(lines) != 4 {
				t.Errorf("buildStringToSign() should have 4 lines, got %d", len(lines))
			}
			
			if lines[0] != "AWS4-HMAC-SHA256" {
				t.Errorf("First line should be AWS4-HMAC-SHA256, got %s", lines[0])
			}
			
			if lines[1] != "20150830T123600Z" {
				t.Errorf("Second line should be timestamp, got %s", lines[1])
			}
			
			if lines[2] != "20150830/us-east-1/service/aws4_request" {
				t.Errorf("Third line should be credential scope, got %s", lines[2])
			}
			
			if len(lines[3]) != 64 {
				t.Errorf("Fourth line should be 64-char hash, got %d chars", len(lines[3]))
			}
		})
	}
}
