#!/bin/bash
# Remove automated authentication credentials from OS keychain

set -e

echo "========================================="
echo "Remove Automated Authentication"
echo "========================================="
echo ""

# Create temporary Go program to remove credentials
cat > /tmp/remove-creds.go <<'EOF'
package main

import (
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

func main() {
	if err := keyring.Delete("kiro-gateway-identity-center", "credentials"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete credentials: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Credentials removed from OS keychain")
}
EOF

# Remove credentials
go run /tmp/remove-creds.go

# Clean up temporary file
rm /tmp/remove-creds.go

echo ""
echo "✓ Automated authentication credentials removed"
echo ""
echo "To set up automated authentication again:"
echo "  ./scripts/setup-automated-auth.sh"
echo ""
