# CV Analytics Webhook Receiver (GCP Cloud Functions)

Google Cloud Functions 2nd Gen webhook receiver written in Go for real-time analytics processing.

## Overview

This Cloud Function receives webhook notifications from AWS Lambda when new analytics records are created, validates the HMAC signature, and stores the data in Firebase Realtime Database for instant display in the React dashboard.

## Architecture

```
AWS Lambda Processor
    ↓ (HTTP POST with HMAC signature)
Cloud Function (Go)
    ↓ (Validates signature)
    ↓ (Transforms data)
Firebase Realtime Database
    ↓ (WebSocket broadcast)
React Dashboard (real-time update)
```

## Technology Stack

- **Language:** Go 1.21
- **Platform:** Google Cloud Functions 2nd Gen
- **Database:** Firebase Realtime Database
- **Security:** HMAC-SHA256 signature verification
- **Memory:** 128MB (minimum for Go)
- **Cold Start:** ~100ms

## Prerequisites

1. Google Cloud Project with billing enabled
2. Firebase project created and linked to GCP
3. `gcloud` CLI installed and authenticated
4. Go 1.21+ installed (for local testing)

## Environment Variables

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `FIREBASE_DATABASE_URL` | Firebase Realtime Database URL | Yes | `https://your-project.firebaseio.com` |
| `WEBHOOK_SECRET` | HMAC signing secret (shared with AWS Lambda) | Yes | `your-secret-key-here` |

## Local Development

### Install Dependencies

```bash
go mod download
```

### Run Locally (Functions Framework)

```bash
# Set environment variables
export FIREBASE_DATABASE_URL="https://your-project.firebaseio.com"
export WEBHOOK_SECRET="test-secret-123"

# Run function locally
go run cmd/main.go
```

### Test with curl

```bash
# Generate HMAC signature
PAYLOAD='{"eventType":"analytics_record_created","timestamp":1698765432000,"data":{"requestId":"test-123","query":"Do you have Python?","matchType":"full","matchScore":95,"reasoning":"Good match","vectorMatches":5,"sessionId":"session-1","week":"2024-W43"}}'
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "test-secret-123" | sed 's/^.* //')

# Send webhook
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: $SIGNATURE" \
  -d "$PAYLOAD"
```

## Deployment

### 1. Set GCP Project

```bash
gcloud config set project YOUR_PROJECT_ID
```

### 2. Enable Required APIs

```bash
gcloud services enable cloudfunctions.googleapis.com
gcloud services enable cloudbuild.googleapis.com
gcloud services enable firebase.googleapis.com
```

### 3. Store Webhook Secret in Secret Manager

```bash
# Create secret
echo -n "your-webhook-secret-here" | gcloud secrets create webhook-secret --data-file=-

# Grant Cloud Functions access
gcloud secrets add-iam-policy-binding webhook-secret \
  --member="serviceAccount:YOUR_PROJECT_ID@appspot.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### 4. Deploy Function

```bash
gcloud functions deploy cv-analytics-webhook \
  --gen2 \
  --runtime=go121 \
  --region=us-central1 \
  --source=. \
  --entry-point=AnalyticsWebhook \
  --trigger-http \
  --allow-unauthenticated \
  --set-env-vars=FIREBASE_DATABASE_URL=https://YOUR_PROJECT.firebaseio.com \
  --set-secrets=WEBHOOK_SECRET=webhook-secret:latest \
  --memory=128MB \
  --timeout=10s \
  --max-instances=10 \
  --min-instances=0
```

**Note:** Using `--allow-unauthenticated` because authentication is handled via HMAC signature. For additional security, you can:
1. Use `--no-allow-unauthenticated` and configure AWS Lambda with GCP service account credentials
2. Add IP allowlisting via Cloud Armor

### 5. Get Function URL

```bash
gcloud functions describe cv-analytics-webhook --gen2 --region=us-central1 --format="value(serviceConfig.uri)"
```

Copy this URL and add it to your AWS Lambda processor environment variables.

## Firebase Setup

### 1. Create Firebase Project

1. Go to [Firebase Console](https://console.firebase.google.com/)
2. Create new project or link existing GCP project
3. Enable Realtime Database

### 2. Set Security Rules

Navigate to **Realtime Database > Rules** and set:

```json
{
  "rules": {
    "analytics": {
      "live": {
        ".read": "auth != null",
        ".write": false,
        ".indexOn": ["timestamp", "receivedAt"]
      },
      "archive": {
        ".read": "auth != null",
        ".write": false
      }
    }
  }
}
```

**Note:** Only authenticated users can read, only Cloud Function (via Admin SDK) can write.

### 3. Get Service Account Key (for local testing)

1. Go to **Project Settings > Service Accounts**
2. Click "Generate new private key"
3. Save as `firebase-adminsdk.json` (DO NOT commit to git)
4. For local testing, set: `export GOOGLE_APPLICATION_CREDENTIALS=./firebase-adminsdk.json`

## Monitoring

### View Logs

```bash
gcloud functions logs read cv-analytics-webhook --gen2 --region=us-central1 --limit=50
```

### View Metrics

```bash
# Invocations
gcloud monitoring time-series list \
  --filter='metric.type="cloudfunctions.googleapis.com/function/execution_count"' \
  --format=json

# Execution times
gcloud monitoring time-series list \
  --filter='metric.type="cloudfunctions.googleapis.com/function/execution_times"' \
  --format=json
```

### Set Up Alerts (Optional)

Create alert for error rate > 5%:

```bash
gcloud alpha monitoring policies create \
  --notification-channels=YOUR_CHANNEL_ID \
  --display-name="Webhook Function Errors" \
  --condition-display-name="Error rate > 5%" \
  --condition-threshold-value=0.05 \
  --condition-threshold-duration=300s \
  --condition-filter='resource.type="cloud_function" AND metric.type="cloudfunctions.googleapis.com/function/execution_count" AND metric.label.status!="ok"'
```

## Performance Optimization

### Current Configuration
- **Memory:** 128MB (lowest for Go)
- **Timeout:** 10s (enough for Firebase write)
- **Max Instances:** 10 (prevents runaway costs)
- **Min Instances:** 0 (scales to zero when idle)

### Cold Start Optimization
- Go starts in ~100ms (vs 400ms Node.js, 600ms Python)
- Firebase client initialized once in `init()` and reused
- No heavy dependencies loaded

### Cost Optimization
- Uses smallest memory allocation (128MB)
- Scales to zero when idle
- 2M invocations/month FREE
- 400K GB-seconds/month FREE
- Expected cost at 10K requests/month: **$0.00**

## Troubleshooting

### Error: "Invalid signature"

**Cause:** HMAC secret mismatch between Lambda and Cloud Function

**Solution:**
```bash
# Check secret in GCP
gcloud secrets versions access latest --secret=webhook-secret

# Update Lambda environment variable to match
```

### Error: "Failed to store data"

**Cause:** Firebase permissions issue

**Solution:**
```bash
# Ensure service account has Firebase Admin role
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="serviceAccount:YOUR_PROJECT_ID@appspot.gserviceaccount.com" \
  --role="roles/firebase.admin"
```

### Error: "Function timeout"

**Cause:** Slow Firebase write or network issue

**Solution:**
```bash
# Increase timeout (max 60s for gen2)
gcloud functions deploy cv-analytics-webhook \
  --timeout=30s \
  --update-env-vars=TIMEOUT_INCREASED=true
```

## Security Best Practices

1. ✅ **HMAC Signature Verification** - Validates every request
2. ✅ **Secret Manager** - Secrets not in code or env vars
3. ✅ **Firebase Admin SDK** - Server-side write access only
4. ✅ **CORS Headers** - Restricted origins
5. ✅ **Input Validation** - Checks required fields
6. ⚠️ **Rate Limiting** - Consider adding Cloud Armor if needed
7. ⚠️ **IP Allowlisting** - Consider restricting to Lambda NAT Gateway IP

## Testing

### Unit Tests (TODO)

```bash
go test ./... -v
```

### Integration Test

```bash
# Deploy to test environment
gcloud functions deploy cv-analytics-webhook-test \
  --gen2 \
  --runtime=go121 \
  --region=us-central1 \
  --trigger-http \
  --allow-unauthenticated

# Test endpoint
./test/integration_test.sh
```

## Maintenance

### Update Dependencies

```bash
go get -u firebase.google.com/go/v4
go get -u github.com/GoogleCloudPlatform/functions-framework-go
go mod tidy
```

### Redeploy

```bash
gcloud functions deploy cv-analytics-webhook --gen2 --region=us-central1
```

## Related Projects

- **AWS Lambda Processor:** `cv-analytics-processor/`
- **React Dashboard:** `cv-analytics-dashboard/`
- **Infrastructure:** `cv-analytics-infrastructure/`

## License

Private - Jose Alvarez

## Support

For issues or questions, see main project documentation.
