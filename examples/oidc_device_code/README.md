# OIDC Device Code Flow Example

This example demonstrates the OIDC device code flow for browser-less authentication with AWS SSO.

## Features Demonstrated

- Device registration with AWS SSO-OIDC
- Device authorization flow
- User code display and verification
- Token polling with proper backoff
- Secure token storage
- Token expiration and refresh handling

## Running the Example

```bash
cd examples/oidc_device_code
go run main.go
```

## How Device Code Flow Works

1. **Device Registration**: Register application with AWS SSO-OIDC
2. **Device Authorization**: Get device code and user code
3. **User Interaction**: User opens URL and enters code in browser
4. **Token Polling**: Application polls for access token
5. **Token Storage**: Store tokens securely for future use

## Expected Output

The example will:
1. Create OIDC client
2. Simulate device registration
3. Start device authorization
4. Display user instructions
5. Simulate successful token retrieval
6. Store token securely
7. Demonstrate token validation

## Real Implementation Notes

This example simulates the flow for demonstration. In a real implementation:

- Device registration calls `RegisterClient` API
- Device authorization calls `StartDeviceAuthorization` API  
- Token polling calls `CreateToken` API with device code
- Handle `authorization_pending` and `slow_down` responses
- Implement exponential backoff for polling

## Configuration

The example uses:
- Region: us-east-1
- Simulated AWS SSO endpoints
- Example tokens (not real credentials)

## Production Usage

In production:
- Use real AWS SSO-OIDC endpoints
- Implement proper error handling
- Use keychain storage for tokens
- Handle network timeouts and retries
- Support custom start URLs for IAM Identity Center
