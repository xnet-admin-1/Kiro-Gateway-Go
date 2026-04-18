# Configuration Guide

*Last updated: 2026-01-26*

This directory contains configuration examples and documentation for the kiro-gateway project.

## Active Configuration

The active configuration file is `.env` in the project root. This file contains the current environment variables used by the application.

**Important**: The `.env` file is excluded from Git to protect sensitive information. Never commit this file to version control.

## Configuration Examples

The `examples/` directory contains example configuration files for different scenarios:

### .env.codewhisperer-test

Configuration file for specific use case. See file contents for details.

### .env.docker

Configuration file for specific use case. See file contents for details.

### .env.headless

Configuration file for specific use case. See file contents for details.

### .env.qdeveloper-test

Configuration file for specific use case. See file contents for details.

### .env.sigv4-sso-backup

Configuration file for specific use case. See file contents for details.

### .env.token-refresh-test

Configuration file for specific use case. See file contents for details.

## Creating Your Configuration

1. Copy an appropriate example file to the project root:
   ```bash
   cp config/examples/.env.example .env
   ```

2. Edit `.env` with your specific values:
   - AWS credentials and region
   - Q Developer profile ARN
   - Authentication settings
   - Port and logging configuration

3. Verify your configuration:
   ```bash
   go run cmd/kiro-gateway/main.go --validate-config
   ```

## Environment Variables

### Required Variables

- `AWS_REGION` - AWS region (e.g., `us-east-1`)
- `PROFILE_ARN` - Q Developer profile ARN
- `AUTH_MODE` - Authentication mode (`sigv4-sso`, `bearer-token`, or `codewhisperer`)

### Optional Variables

- `PORT` - Server port (default: `8080`)
- `LOG_LEVEL` - Logging level (`debug`, `info`, `warn`, `error`)
- `ENABLE_CORS` - Enable CORS (default: `true`)
- `TIMEOUT` - Request timeout in seconds (default: `30`)

## Security Notes

- **Never commit `.env` files** with real credentials
- Use example files with placeholder values for documentation
- Rotate credentials regularly
- Use AWS IAM roles when possible instead of access keys
- Keep sensitive configuration files out of version control

## Troubleshooting

### Configuration Not Loading

- Verify `.env` file exists in project root
- Check file permissions (should be readable)
- Ensure no syntax errors in the file
- Check for missing required variables

### Authentication Errors

- Verify AWS credentials are valid
- Check profile ARN is correct
- Ensure auth mode matches your setup
- Verify IAM permissions are sufficient

## Additional Resources

- [Quick Start Guide](../docs/guides/docs/guides/docs/guides/docs/guides/docs/guides/QUICK_START.md)
- [Authentication Flow](../docs/architecture/docs/architecture/docs/architecture/COMPLETE_AUTHENTICATION_FLOW.md)
- [Workspace Organization](../WORKSPACE_ORGANIZATION.md)
