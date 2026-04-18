# Documentation Index

*Last updated: 2026-01-26*

This directory contains all project documentation organized by topic.

## API Documentation

API specifications and technical documentation:

- [Aws Q Cli Trust Relationship Specs](api/AWS_Q_CLI_TRUST_RELATIONSHIP_SPECS.md)
- [Aws Q Developer Api Specifications](api/AWS_Q_DEVELOPER_API_SPECIFICATIONS.md)

## Guides

How-to guides and troubleshooting documentation:

- [Browser Automation Security Fixes](guides/BROWSER_AUTOMATION_SECURITY_FIXES.md)
- [Capture And Compare](guides/CAPTURE_AND_COMPARE.md)
- [Container Deployment Fixed](guides/CONTAINER_DEPLOYMENT_FIXED.md)
- [Direct Api Vision Fixed](guides/DIRECT_API_VISION_FIXED.md)
- [Docker Secrets Setup](guides/DOCKER_SECRETS_SETUP.md)
- [Feature Status Comparison](guides/FEATURE_STATUS_COMPARISON.md)
- [Mcp Go Client Exploration](guides/MCP_GO_CLIENT_EXPLORATION.md)
- [Mcp Security Audit](guides/MCP_SECURITY_AUDIT.md)
- [Project Journey](guides/PROJECT_JOURNEY.md)
- [Python Vs Go Vision Comparison](guides/PYTHON_VS_GO_VISION_COMPARISON.md)
- [Quick Start](guides/QUICK_START.md)
- [Q Cli Log Insights](guides/Q_CLI_LOG_INSIGHTS.md)
- [Q Cli Traffic Capture Setup](guides/Q_CLI_TRAFFIC_CAPTURE_SETUP.md)
- [Security Fixes Implemented](guides/SECURITY_FIXES_IMPLEMENTED.md)
- [Source Code Verification Report](guides/SOURCE_CODE_VERIFICATION_REPORT.md)
- [Test Script Status 20260126](guides/TEST_SCRIPT_STATUS_20260126.md)
- [Vision Complete Success 20260126](guides/VISION_COMPLETE_SUCCESS_20260126.md)
- [Vision Fix Attempt](guides/VISION_FIX_ATTEMPT.md)
- [Vision Fix From Q Cli Source](guides/VISION_FIX_FROM_Q_CLI_SOURCE.md)
- [Vision Modelid Fix](guides/VISION_MODELID_FIX.md)
- [Vision Structure Verified](guides/VISION_STRUCTURE_VERIFIED.md)

## Architecture

Architecture documentation and design decisions:

- [Adapter Flow Analysis](architecture/ADAPTER_FLOW_ANALYSIS.md)
- [Aws Q Services Complete Analysis](architecture/AWS_Q_SERVICES_COMPLETE_ANALYSIS.md)
- [Aws Sdk V2 Sso Services Analysis](architecture/AWS_SDK_V2_SSO_SERVICES_ANALYSIS.md)
- [Complete Authentication Flow](architecture/COMPLETE_AUTHENTICATION_FLOW.md)
- [Credential Chain And Endpoint Analysis](architecture/CREDENTIAL_CHAIN_AND_ENDPOINT_ANALYSIS.md)
- [Mcp Integration Analysis](architecture/MCP_INTEGRATION_ANALYSIS.md)
- [Multi Threading And Caching Analysis](architecture/MULTI_THREADING_AND_CACHING_ANALYSIS.md)
- [Performance Analysis](architecture/PERFORMANCE_ANALYSIS.md)
- [Project Structure](architecture/PROJECT_STRUCTURE.md)
- [Sigv4 Sso Credential Flow Analysis](architecture/SIGV4_SSO_CREDENTIAL_FLOW_ANALYSIS.md)
- [Tool Calling Complete Analysis](architecture/TOOL_CALLING_COMPLETE_ANALYSIS.md)

## Contributing

When adding new documentation:

1. Place files in the appropriate subdirectory (api/, guides/, or architecture/)
2. Use descriptive filenames with appropriate suffixes:
   - API docs: `*_API.md`, `*_SPECIFICATIONS.md`
   - Guides: `*_GUIDE.md`, `*_TROUBLESHOOTING.md`
   - Architecture: `*_ANALYSIS.md`, `*_FLOW.md`, `PROJECT_*.md`
3. Update this index by running: `go run cmd/workspace-organize/main.go --update-docs`

## Organization

For information about workspace organization, see [WORKSPACE_ORGANIZATION.md](../WORKSPACE_ORGANIZATION.md) in the project root.
