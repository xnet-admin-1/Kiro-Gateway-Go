package sigv4

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
)

func TestSigner_SignRequest(t *testing.T) {
	creds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "",
		Source:          "test",
	}
	
	signer := NewSigner(creds, "us-east-1", "service")
	
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/", Host: "example.amazonaws.com"},
		Header: make(http.Header),
	}
	req.Header.Set("Host", "example.amazonaws.com")
	
	body := []byte{}
	
	err := signer.SignRequest(req, body)
	if err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}
	
	// Check that required headers are present
	if req.Header.Get("X-Amz-Date") == "" {
		t.Error("X-Amz-Date header not set")
	}
	
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		t.Error("Authorization header not set")
	}
	
	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization header should start with AWS4-HMAC-SHA256, got: %s", authHeader)
	}
	
	if !strings.Contains(authHeader, "Credential=") {
		t.Error("Authorization header should contain Credential=")
	}
	
	if !strings.Contains(authHeader, "SignedHeaders=") {
		t.Error("Authorization header should contain SignedHeaders=")
	}
	
	if !strings.Contains(authHeader, "Signature=") {
		t.Error("Authorization header should contain Signature=")
	}
}

func TestSigner_SignRequestWithSessionToken(t *testing.T) {
	creds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "AQoEXAMPLEH4aoAH0gNCAPyJxz4BlCFFxWNE1OPTgk5TthT+FvwqnKwRcOIfrRh3c/LTo6UDdyJwOOvEVPvLXCrrrUtdnniCEXAMPLE/IvU1dYUg2RVAJBanLiHb4IgRmpRV3zrkuWJOgQs8IZZaIv2BXIa2R4OlgkBN9bkUDNCJiBeb/AXlzBBko7b15fjrBs2+cTQtpZ3CYWFXG8C5zqx37wnOE49mRl/+OtkIKGO7fAE",
		Source:          "test",
	}
	
	signer := NewSigner(creds, "us-east-1", "service")
	
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/", Host: "example.amazonaws.com"},
		Header: make(http.Header),
	}
	req.Header.Set("Host", "example.amazonaws.com")
	
	body := []byte{}
	
	err := signer.SignRequest(req, body)
	if err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}
	
	// Check that session token header is present
	if req.Header.Get("X-Amz-Security-Token") != creds.SessionToken {
		t.Error("X-Amz-Security-Token header not set correctly")
	}
}

func TestBuildAuthorizationHeader(t *testing.T) {
	creds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	
	signer := NewSigner(creds, "us-east-1", "service")
	
	req := &http.Request{
		Header: make(http.Header),
	}
	req.Header.Set("Host", "example.amazonaws.com")
	req.Header.Set("X-Amz-Date", "20150830T123600Z")
	
	timestamp := mustParseTime("20150830T123600Z")
	signature := "5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7"
	
	result := signer.buildAuthorizationHeader(timestamp, signature, req)
	expected := "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20150830/us-east-1/service/aws4_request, SignedHeaders=host;x-amz-date, Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7"
	
	if result != expected {
		t.Errorf("buildAuthorizationHeader() = %q, want %q", result, expected)
	}
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse("20060102T150405Z", s)
	if err != nil {
		panic(err)
	}
	return t
}
