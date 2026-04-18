# SigV4 Authentication Example

This example demonstrates how to use AWS Signature Version 4 authentication with IAM credentials.

## Features Demonstrated

- AWS credential chain resolution
- SigV4 request signing for GET and POST requests
- Multiple credential source handling
- Proper header management
- Regional endpoint support

## Running the Example

```bash
cd examples/sigv4
go run main.go
```

## Prerequisites

For full functionality, configure AWS credentials using one of:

1. **Environment Variables:**
   ```bash
   export AWS_ACCESS_KEY_ID=your-access-key
   export AWS_SECRET_ACCESS_KEY=your-secret-key
   export AWS_SESSION_TOKEN=your-session-token  # Optional
   ```

2. **AWS Profile:**
   ```bash
   aws configure --profile your-profile
   export AWS_PROFILE=your-profile
   ```

3. **IAM Role (EC2/ECS):**
   - Attach IAM role to EC2 instance or ECS task

## Expected Output

The example will:
1. Set up AWS credential chain
2. Retrieve credentials from available sources
3. Create SigV4 signer
4. Sign GET request to ListAvailableModels
5. Sign POST request with JSON body
6. Test different credential sources

## Configuration

The example uses:
- Region: us-east-1
- Service: codewhisperer
- Example credentials if no real ones available

## Production Usage

In production:
- Use real AWS credentials
- Configure appropriate IAM permissions
- Handle credential refresh automatically
- Implement proper error handling
