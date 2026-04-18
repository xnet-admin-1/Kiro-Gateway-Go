# Kiro Gateway Go - Examples

This directory contains practical examples demonstrating how to use the AWS Security Baseline features in kiro-gateway-go.

## Examples

1. **Bearer Token Authentication** (`bearer_token/`) - Traditional OAuth bearer token authentication
2. **SigV4 Authentication** (`sigv4/`) - AWS Signature Version 4 authentication with IAM credentials
3. **OIDC Device Code Flow** (`oidc_device_code/`) - Browser-less authentication using device codes
4. **Credential Chain** (`credential_chain/`) - AWS credential provider chain usage

## Running Examples

Each example is a standalone Go program that demonstrates specific authentication patterns:

```bash
# Bearer token example
cd examples/bearer_token
go run main.go

# SigV4 example
cd examples/sigv4
go run main.go

# OIDC device code example
cd examples/oidc_device_code
go run main.go

# Credential chain example
cd examples/credential_chain
go run main.go
```

## Prerequisites

- Go 1.21+
- Valid AWS credentials (for SigV4 and credential chain examples)
- Network access to AWS endpoints

## Security Notes

- Examples use test credentials where possible
- Never commit real credentials to version control
- Use environment variables for sensitive data
- Follow the security guidelines in `../SECURITY.md`
