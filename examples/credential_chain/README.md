# AWS Credential Chain Example

This example demonstrates how to use the AWS credential provider chain to automatically discover and use AWS credentials from multiple sources.

## Features Demonstrated

- Individual credential provider testing
- Standard AWS credential chain order
- Credential caching mechanism
- Environment variable configuration
- Multiple authentication methods

## Running the Example

```bash
cd examples/credential_chain
go run main.go
```

## Credential Provider Order

The AWS credential chain tries providers in this order:

1. **Environment Variables** (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. **AWS Profile Files** (`~/.aws/credentials`, `~/.aws/config`)
3. **Web Identity Token** (for Kubernetes IRSA)
4. **ECS Container Credentials** (for ECS tasks)
5. **EC2 Instance Metadata** (for EC2 instances)

## Configuration Examples

### Environment Variables
```bash
# Basic credentials
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# With session token (temporary credentials)
export AWS_SESSION_TOKEN=your-session-token

# Profile selection
export AWS_PROFILE=your-profile-name
```

### AWS Profile Files
```bash
# Configure default profile
aws configure

# Configure named profile
aws configure --profile myprofile
export AWS_PROFILE=myprofile
```

### Web Identity Token (Kubernetes)
```bash
export AWS_WEB_IDENTITY_TOKEN_FILE=/var/run/secrets/eks.amazonaws.com/serviceaccount/token
export AWS_ROLE_ARN=arn:aws:iam::123456789012:role/your-role
```

### ECS Task Role
```bash
export AWS_CONTAINER_CREDENTIALS_RELATIVE_URI=/v2/credentials/your-task-id
```

## Expected Output

The example will:
1. Test each credential provider individually
2. Use the credential chain to find the first available credentials
3. Demonstrate credential caching
4. Show environment variable configuration examples
5. Display current environment variables

## Production Usage

In production:
- Credentials are automatically cached until expiration
- Chain stops at first successful provider
- Thread-safe credential access
- Automatic refresh for temporary credentials
- Proper error handling for each provider
