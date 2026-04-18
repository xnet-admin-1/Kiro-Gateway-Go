# Bearer Token Authentication Example

This example demonstrates how to use bearer token authentication with automatic refresh capabilities.

## Features Demonstrated

- Token storage and retrieval
- Automatic expiration checking (1-minute safety margin)
- Token refresh using OIDC client
- Error handling for expired tokens
- Secure token management

## Running the Example

```bash
cd examples/bearer_token
go run main.go
```

## Expected Output

The example will:
1. Check for existing tokens
2. Store an example token
3. Retrieve the stored token
4. Test expiration handling

## Configuration

The example uses:
- In-memory token storage (for demonstration)
- US East 1 region
- Example tokens (not real credentials)

## Production Usage

In production, you would:
- Use keychain storage instead of memory
- Provide real OIDC credentials
- Handle refresh token rotation
- Implement proper error recovery
