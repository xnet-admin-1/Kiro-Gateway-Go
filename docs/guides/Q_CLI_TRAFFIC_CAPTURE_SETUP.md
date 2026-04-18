# Q Developer CLI Traffic Capture System - Setup Complete

## Date
January 25, 2026

## Overview

Created a complete system to capture and analyze AWS Q Developer CLI network traffic using mitmproxy. This allows us to reverse engineer the exact API format by intercepting real Q CLI requests.

## Files Created

### 1. Capture Scripts

**`scripts/capture-q-cli-traffic.py`** - mitmproxy addon
- Intercepts Q Developer API traffic
- Saves requests and responses as JSON
- Handles event stream responses
- Redacts sensitive headers
- Parses JSON bodies automatically

**`scripts/capture-q-cli.ps1`** - PowerShell helper
- Starts mitmproxy with the addon
- Creates capture directory
- Shows usage instructions
- Handles errors gracefully

### 2. Analysis Scripts

**`scripts/analyze-q-cli-captures.ps1`** - Traffic analyzer
- Reads captured JSON files
- Extracts key information
- Shows request/response structure
- Highlights vision/multimodal data
- Supports detailed output modes

### 3. Documentation

**`scripts/CAPTURE_README.md`** - Complete guide
- Installation instructions
- Usage examples
- Troubleshooting tips
- Analysis workflow
- Security notes

## Quick Start

### Install mitmproxy
```powershell
pip install mitmproxy
```

### Start Capture
```powershell
.\scripts\capture-q-cli.ps1
```

### Configure Q CLI (in another terminal)
```powershell
$env:HTTPS_PROXY = "http://localhost:8080"
$env:HTTP_PROXY = "http://localhost:8080"
```

### Run Q CLI Commands
```powershell
# Interactive mode (default)
qchat chat
# Type: "What is Amazon S3?"
# Drag/drop image for vision testing

# Non-interactive mode
qchat chat -i
# Type: "What is Amazon S3?"
# Exits after answer
```

### Analyze Captures
```powershell
.\scripts\analyze-q-cli-captures.ps1 -ShowFull
```

## What This Solves

### Problem
We've been guessing at the Q Developer API format based on:
- Source code analysis (Rust → Go translation)
- Documentation (incomplete)
- Trial and error

### Solution
Now we can:
- **See exact requests** the official Q CLI sends
- **Compare byte-for-byte** with our implementation
- **Identify missing fields** we didn't know about
- **Verify data types** and encoding
- **Understand event stream format** in detail

## Use Cases

### 1. Vision/Multimodal Debugging
Capture Q CLI sending an image request:
```powershell
q chat "What's in this image?" --image test.png
```

Then compare with our gateway's request to find differences.

### 2. Request Format Verification
Capture various request types:
- Simple text queries
- Multi-turn conversations
- Tool use requests
- Code analysis requests

### 3. Response Format Analysis
See how Q CLI handles:
- Event stream responses
- Error responses
- Metadata events
- Code references

### 4. Authentication Analysis
Observe:
- SigV4 signature format
- Required headers
- Token handling
- Session management

## Expected Workflow

1. **Capture Q CLI traffic** for a specific feature (e.g., vision)
2. **Analyze the JSON** to understand exact format
3. **Compare with our code** to find differences
4. **Update our implementation** to match Q CLI
5. **Test our gateway** with same request
6. **Capture our traffic** and compare
7. **Iterate** until identical

## Key Insights We'll Gain

### Request Structure
- Exact field names (case-sensitive)
- Field ordering (if it matters)
- Required vs optional fields
- Data type expectations
- Nested structure details

### Image Encoding
- How Q CLI encodes images
- Base64 format specifics
- Image metadata fields
- Multiple image handling

### Headers
- Required headers
- Header values
- Authentication format
- Content-Type specifics

### Event Stream
- Event types used
- Event structure
- Parsing requirements
- Error handling

## Next Steps

1. **Install mitmproxy** if not already installed
2. **Run first capture** with simple text query
3. **Verify capture works** by checking JSON files
4. **Capture vision request** with image
5. **Analyze differences** between text and vision
6. **Update our code** based on findings
7. **Test and verify** our implementation

## Success Criteria

We'll know the capture system is working when:
- ✅ JSON files are created in `q-cli-captures/`
- ✅ Request bodies are properly parsed
- ✅ Response bodies are captured
- ✅ Image data is visible in captures
- ✅ Analysis script shows clear structure

We'll know our implementation is correct when:
- ✅ Our JSON matches Q CLI JSON exactly
- ✅ AWS Q Developer responds correctly
- ✅ Vision requests work with images
- ✅ No generic error responses

## Security Notes

- Captured files contain sensitive data (tokens, credentials)
- Only use on trusted local machine
- Review files before sharing
- Delete captures after analysis
- Don't commit captures to git

## Files Location

All capture-related files are in `scripts/`:
- `capture-q-cli-traffic.py` - mitmproxy addon
- `capture-q-cli.ps1` - Start capture
- `analyze-q-cli-captures.ps1` - Analyze captures
- `CAPTURE_README.md` - Full documentation

Captured traffic goes to:
- `q-cli-captures/` - Default capture directory

## Summary

We now have a complete system to reverse engineer the Q Developer API by capturing real Q CLI traffic. This eliminates guesswork and lets us see exactly what the official implementation sends to AWS.

**Next action**: Run the capture system and analyze Q CLI's vision requests to fix our implementation.
