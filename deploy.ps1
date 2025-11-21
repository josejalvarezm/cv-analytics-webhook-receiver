# Deploy Webhook Receiver with Firestore Support
# This script deploys the updated Cloud Function to GCP

$ErrorActionPreference = "Stop"

Write-Host "[DEPLOY] Deploying CV Analytics Webhook Receiver..." -ForegroundColor Cyan
Write-Host ""

# Check if gcloud is authenticated
Write-Host "[1] Checking GCP authentication..." -ForegroundColor Yellow
$project = gcloud config get-value project 2>$null
if (-not $project) {
    Write-Host "[ERROR] Not authenticated with GCP" -ForegroundColor Red
    Write-Host "Run: gcloud auth login" -ForegroundColor Yellow
    exit 1
}
Write-Host "[OK] Authenticated with project: $project" -ForegroundColor Green
Write-Host ""

# Verify we're in the right directory
if (-not (Test-Path "function.go")) {
    Write-Host "[ERROR] Not in webhook receiver directory" -ForegroundColor Red
    Write-Host "Run: cd cv-analytics-webhook-receiver-private" -ForegroundColor Yellow
    exit 1
}

# Get webhook secret (will be passed as environment variable)
Write-Host "[2] Webhook secret configuration..." -ForegroundColor Yellow
$webhookSecret = Read-Host "Enter webhook secret (shared with AWS Lambda)"
if (-not $webhookSecret) {
    Write-Host "[ERROR] Webhook secret is required" -ForegroundColor Red
    exit 1
}
Write-Host "[OK] Secret configured" -ForegroundColor Green
Write-Host ""

# Deploy Cloud Function
Write-Host "[3] Deploying Cloud Function..." -ForegroundColor Yellow
Write-Host "   Region: us-central1" -ForegroundColor Gray
Write-Host "   Runtime: Go 1.23" -ForegroundColor Gray
Write-Host "   Memory: 256MB" -ForegroundColor Gray
Write-Host "   Database: Firestore" -ForegroundColor Gray
Write-Host ""

gcloud functions deploy cv-analytics-webhook `
    --gen2 `
    --runtime=go123 `
    --region=us-central1 `
    --source=. `
    --entry-point=AnalyticsWebhook `
    --trigger-http `
    --allow-unauthenticated `
    --set-env-vars="WEBHOOK_SECRET=$webhookSecret" `
    --memory=256MB `
    --timeout=10s `
    --max-instances=10 `
    --min-instances=0 `
    --project=$project

if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Deployment failed" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[OK] Cloud Function deployed successfully!" -ForegroundColor Green
Write-Host ""

# Get function URL
Write-Host "[4] Getting function URL..." -ForegroundColor Yellow
$webhookUrl = gcloud functions describe cv-analytics-webhook `
    --gen2 `
    --region=us-central1 `
    --project=$project `
    --format="value(serviceConfig.uri)"

Write-Host ""
Write-Host "[URL] WEBHOOK URL (save this for AWS Lambda):" -ForegroundColor Cyan
Write-Host ""
Write-Host "  $webhookUrl" -ForegroundColor White
Write-Host ""

# Save to file
$webhookUrl | Out-File -FilePath "webhook-url.txt" -Encoding UTF8
Write-Host "[OK] URL saved to webhook-url.txt" -ForegroundColor Green
Write-Host ""

# Test webhook endpoint
Write-Host "[5] Testing webhook endpoint..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri $webhookUrl -Method Get -ErrorAction Stop
    Write-Host "[WARN] Endpoint is public but requires POST with signature" -ForegroundColor Yellow
} catch {
    if ($_.Exception.Response.StatusCode -eq 405) {
        Write-Host "[OK] Endpoint active (405 Method Not Allowed for GET is expected)" -ForegroundColor Green
    } else {
        Write-Host "[WARN] Endpoint response: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}
Write-Host ""

# View logs
Write-Host "[6] Recent logs:" -ForegroundColor Yellow
gcloud functions logs read cv-analytics-webhook `
    --gen2 `
    --region=us-central1 `
    --project=$project `
    --limit=10

Write-Host ""
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host "[SUCCESS] DEPLOYMENT COMPLETE!" -ForegroundColor Green
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "[NEXT] Next Steps:" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. Copy this webhook URL:" -ForegroundColor White
Write-Host "   $webhookUrl" -ForegroundColor Yellow
Write-Host ""
Write-Host "2. Update AWS Lambda environment variable:" -ForegroundColor White
Write-Host "   WEBHOOK_URL=$webhookUrl" -ForegroundColor Gray
Write-Host "   WEBHOOK_SECRET=$webhookSecret" -ForegroundColor Gray
Write-Host ""
Write-Host "3. Test with manual webhook (see FIRESTORE-MIGRATION.md)" -ForegroundColor White
Write-Host ""
Write-Host "4. Monitor Cloud Function logs:" -ForegroundColor White
Write-Host "   gcloud functions logs read cv-analytics-webhook --gen2 --region=us-central1 --limit=50" -ForegroundColor Gray
Write-Host ""
