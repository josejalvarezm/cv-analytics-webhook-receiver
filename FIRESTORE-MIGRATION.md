# Webhook Receiver - Firestore Migration

## Changes Made

### ✅ Updated to use Firestore instead of Firebase Realtime Database

**Why:** Dashboard now reads from Firestore for real-time updates. Webhook must write to the same database.

## Files Modified

1. **`internal/repositories/firestore.go`** (NEW)
   - Created Firestore repository implementation
   - Uses `requestId` as document ID for idempotency
   - Writes to `analytics` collection

2. **`function.go`**
   - Changed from `firebaseApp.Database()` to `firebaseApp.Firestore()`
   - Updated to use `NewFirestoreRepository` instead of `NewFirebaseRepository`

3. **`internal/config/config.go`**
   - Removed `FIREBASE_DATABASE_URL` requirement
   - Firestore uses project ID from GCP environment automatically

4. **`go.mod`**
   - Promoted `cloud.google.com/go/firestore` to direct dependency

## Deployment Steps

### 1. Deploy Updated Cloud Function

```bash
cd cv-analytics-webhook-receiver-private

gcloud functions deploy cv-analytics-webhook \
  --gen2 \
  --runtime=go121 \
  --region=us-central1 \
  --source=. \
  --entry-point=AnalyticsWebhook \
  --trigger-http \
  --allow-unauthenticated \
  --set-secrets=WEBHOOK_SECRET=webhook-secret:latest \
  --memory=128MB \
  --timeout=10s \
  --max-instances=10 \
  --min-instances=0
```

**Note:** Removed `--set-env-vars FIREBASE_DATABASE_URL` since Firestore doesn't need it!

### 2. Get Webhook URL

```bash
gcloud functions describe cv-analytics-webhook \
  --gen2 \
  --region=us-central1 \
  --format='value(serviceConfig.uri)'
```

Copy the URL (e.g., `https://us-central1-cv-analytics-dashboard.cloudfunctions.net/cv-analytics-webhook`)

### 3. Update AWS Lambda Environment Variable

```bash
aws lambda update-function-configuration \
  --function-name cv-analytics-processor \
  --environment Variables="{
    WEBHOOK_URL=https://us-central1-cv-analytics-dashboard.cloudfunctions.net/cv-analytics-webhook,
    WEBHOOK_SECRET=your-shared-secret-here,
    QUERY_EVENTS_TABLE=cv-analytics-query-events,
    ANALYTICS_TABLE=cv-analytics-analytics
  }" \
  --region us-east-1
```

## Verification

### Test Webhook Manually

```bash
# Get your webhook URL
WEBHOOK_URL="https://us-central1-cv-analytics-dashboard.cloudfunctions.net/cv-analytics-webhook"

# Create test payload
cat > test-payload.json << 'EOF'
{
  "eventType": "analytics_record_created",
  "timestamp": 1732205400000,
  "data": {
    "requestId": "test-001",
    "query": "Test query from manual webhook",
    "matchType": "full",
    "matchScore": 99,
    "reasoning": "Test record",
    "vectorMatches": 5,
    "sessionId": "test-session",
    "week": "2025-W47"
  }
}
EOF

# Generate HMAC signature
WEBHOOK_SECRET="your-shared-secret-here"
SIGNATURE=$(echo -n "$(cat test-payload.json)" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex | awk '{print $2}')

# Send webhook
curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: sha256=$SIGNATURE" \
  -d @test-payload.json

# Expected: 200 OK
# Check Dashboard at http://localhost:3000 - should see "test-001" appear!
```

### Check Firestore

```bash
# View Firestore documents
gcloud firestore documents list analytics --project=cv-analytics-dashboard

# Or use Firebase Console
# https://console.firebase.google.com/project/cv-analytics-dashboard/firestore/data
```

### Check Cloud Function Logs

```bash
gcloud functions logs read cv-analytics-webhook \
  --gen2 \
  --region=us-central1 \
  --limit=50
```

Look for:
- `webhook handler initialized database=firestore`
- `Analytics record written successfully`

## Architecture After Migration

```
AWS Lambda Processor
    ↓ HTTP POST with HMAC
Cloud Function (Go)
    ↓ Validates signature
    ↓ Writes to Firestore collection "analytics"
    ↓ Document ID = requestId (idempotent!)
Firestore
    ↓ Real-time WebSocket subscription
React Dashboard
    ↓ onSnapshot fires
Dashboard updates automatically! ✨
```

## Idempotency Guarantee

**Old (Realtime DB):**
```go
ref.Push(ctx, data)  // Generates random ID → duplicates possible
```

**New (Firestore):**
```go
docRef := client.Collection("analytics").Doc(record.RequestID)
docRef.Set(ctx, data)  // Uses requestId as doc ID → NO duplicates!
```

If AWS Lambda retries webhook:
- Same `requestId` → Firestore overwrites existing document
- No duplicate records in Dashboard ✅

## Rollback Plan

If issues occur, revert to Realtime Database:

```bash
git checkout HEAD~1 -- internal/repositories/firebase.go
git checkout HEAD~1 -- function.go
git checkout HEAD~1 -- internal/config/config.go
git checkout HEAD~1 -- go.mod

# Redeploy with DATABASE_URL
gcloud functions deploy cv-analytics-webhook \
  --gen2 \
  --runtime=go121 \
  --region=us-central1 \
  --source=. \
  --entry-point=AnalyticsWebhook \
  --trigger-http \
  --allow-unauthenticated \
  --set-env-vars=FIREBASE_DATABASE_URL=https://cv-analytics-dashboard.firebaseio.com \
  --set-secrets=WEBHOOK_SECRET=webhook-secret:latest
```

Then update Dashboard `.env.local`:
```
VITE_USE_FIRESTORE=false
```

## Next Steps

1. Deploy webhook receiver with Firestore support
2. Test with manual webhook call
3. Update AWS Lambda Processor to send webhooks after storing analytics
4. Verify real-time updates in Dashboard
5. Monitor Cloud Function logs for any errors

## Status

- ✅ Code changes complete
- ⏳ Deployment pending
- ⏳ AWS Lambda webhook integration pending
- ⏳ End-to-end testing pending
