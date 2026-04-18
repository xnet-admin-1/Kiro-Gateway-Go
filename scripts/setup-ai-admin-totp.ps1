# Setup AI Admin TOTP for Automated Q Developer Access
# This script helps complete the ai-admin setup with TOTP MFA

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "AI Admin TOTP Setup for Q Developer Pro" -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host ""

# Get user info
$identityStoreId = "d-90661d5348"
$userId = "74884448-90c1-7007-c06d-e69b630145dc"
$ssoStartUrl = "https://xnetinc.awsapps.com/start"

Write-Host "User Details:" -ForegroundColor Yellow
Write-Host "  Username: ai-admin" -ForegroundColor White
Write-Host "  User ID: $userId" -ForegroundColor White
Write-Host "  Email: ai-admin@xnetinc.com" -ForegroundColor White
Write-Host "  SSO Portal: $ssoStartUrl" -ForegroundColor White
Write-Host ""

Write-Host "Status: User created and assigned to Q Developer Pro ✓" -ForegroundColor Green
Write-Host ""

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "Next Steps - Complete Setup Manually" -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "STEP 1: Check Email" -ForegroundColor Yellow
Write-Host "  1. Check ai-admin@xnetinc.com inbox" -ForegroundColor White
Write-Host "  2. Look for 'Invitation to join AWS IAM Identity Center'" -ForegroundColor White
Write-Host "  3. Click the activation link" -ForegroundColor White
Write-Host ""

Write-Host "STEP 2: Set Password" -ForegroundColor Yellow
Write-Host "  1. Create a strong password for ai-admin" -ForegroundColor White
Write-Host "  2. Save it securely (you'll need it for automation)" -ForegroundColor White
Write-Host ""

Write-Host "STEP 3: Set Up TOTP MFA" -ForegroundColor Yellow
Write-Host "  1. Choose 'Authenticator app' as MFA method" -ForegroundColor White
Write-Host "  2. You'll see a QR code and a text secret key" -ForegroundColor White
Write-Host "  3. IMPORTANT: Copy the secret key (not the QR code)" -ForegroundColor White
Write-Host "     Example: JBSWY3DPEHPK3PXP" -ForegroundColor Gray
Write-Host "  4. Use an authenticator app to scan QR or enter secret" -ForegroundColor White
Write-Host "  5. Enter the 6-digit code to complete setup" -ForegroundColor White
Write-Host ""

Write-Host "STEP 4: Update Environment Variables" -ForegroundColor Yellow
Write-Host "  Add these to your .env file:" -ForegroundColor White
Write-Host ""
Write-Host "  AUTOMATION_USERNAME=ai-admin" -ForegroundColor Cyan
Write-Host "  AUTOMATION_PASSWORD=<your-password>" -ForegroundColor Cyan
Write-Host "  MFA_TOTP_SECRET=<your-totp-secret>" -ForegroundColor Cyan
Write-Host ""

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "Alternative: Use AWS Console" -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "If you prefer to use the AWS Console:" -ForegroundColor Yellow
Write-Host ""
Write-Host "1. Open IAM Identity Center Console:" -ForegroundColor White
$consoleUrl = "https://console.aws.amazon.com/singlesignon/home?region=us-east-1#!/users"
Write-Host "   $consoleUrl" -ForegroundColor Cyan
Write-Host ""
Write-Host "2. Find 'ai-admin' user and click on it" -ForegroundColor White
Write-Host ""
Write-Host "3. Click 'Reset password' to send a new activation email" -ForegroundColor White
Write-Host ""
Write-Host "4. Follow the email link to complete setup" -ForegroundColor White
Write-Host ""

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "After Setup Complete" -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Test the automated authentication:" -ForegroundColor Yellow
Write-Host ""
Write-Host "  1. Update .env with credentials" -ForegroundColor White
Write-Host "  2. Build: go build -o kiro-gateway.exe ./cmd/kiro-gateway" -ForegroundColor White
Write-Host "  3. Run: .\kiro-gateway.exe" -ForegroundColor White
Write-Host "  4. Test: .\scripts\test_automated_auth_simple.ps1" -ForegroundColor White
Write-Host ""

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "Cost Information" -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Q Developer Pro Subscription: `$19/month per user" -ForegroundColor Yellow
Write-Host "ai-admin subscription cost: `$19/month" -ForegroundColor Yellow
Write-Host ""

Write-Host "Press any key to open the SSO portal in your browser..." -ForegroundColor Green
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

Start-Process $ssoStartUrl

Write-Host ""
Write-Host "Setup script complete!" -ForegroundColor Green
Write-Host ""
