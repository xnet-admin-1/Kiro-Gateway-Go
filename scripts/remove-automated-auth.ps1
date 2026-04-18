# Remove automated authentication credentials from Windows Credential Manager

$ErrorActionPreference = "Stop"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Remove Automated Authentication" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Create temporary Go program to remove credentials
$tempFile = [System.IO.Path]::GetTempFileName() + ".go"
@'
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

	fmt.Println("✓ Credentials removed from Windows Credential Manager")
}
'@ | Out-File -FilePath $tempFile -Encoding UTF8

# Remove credentials
go run $tempFile

# Clean up
Remove-Item $tempFile -Force

Write-Host ""
Write-Host "✓ Automated authentication credentials removed" -ForegroundColor Green
Write-Host ""
Write-Host "To set up automated authentication again:"
Write-Host "  .\scripts\setup-automated-auth.ps1"
Write-Host ""
