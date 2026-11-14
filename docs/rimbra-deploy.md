# Deploy the PockDev engine on Rimbra in 3 minutes

This guide deploys `apps/engine` (the Go engine) to Rimbra Hosting on the
Free tier (512MB RAM). The binary idles well under 30MB RAM and needs only
Rimbra PostgreSQL to run.

## Before you start

You need:

* this repo pushed to GitHub
* a Rimbra account with Hosting, PostgreSQL, and Buckets enabled
* an API token for the dashboard (`apps/web`), any long random string

## Step 1: Create the database (30 seconds)

In the Rimbra dashboard, create a PostgreSQL database named `pockdev`.
Copy its connection string. It is your `DATABASE_URL`.

The engine creates its own tables on startup from
`apps/engine/internal/db/schema.sql`, so there is nothing to migrate by hand.

## Step 2: Create the payload bucket (30 seconds, optional)

Create a Rimbra Bucket named `pockdev-payloads` and copy the S3 endpoint,
access key, and secret key. Raw webhook bodies are spooled to local disk
first (fast catch, never lose a webhook) and mirrored to the bucket in the
background.

On the Free tier you can skip this step and keep bodies on local disk
(`DATA_DIR`) until the volume grows.

## Step 3: Create the service (30 seconds)

In Rimbra Hosting, create a new service from this repository:

* Build: Dockerfile at `apps/engine/Dockerfile`
* Root: `apps/engine`
* Port: 8080 (Rimbra sets `PORT` automatically)

The Dockerfile is multi-stage: it builds with `golang:1.23-alpine` and runs
on plain `alpine:3.20` as a non-root user.

## Step 4: Set the environment (60 seconds)

Set these variables on the service (see `apps/engine/.env.example` for the
full annotated list):

| Variable | Required | Value |
| --- | --- | --- |
| `DATABASE_URL` | yes | the Rimbra PostgreSQL connection string |
| `API_TOKEN` | yes | bearer token the dashboard sends as `Authorization: Bearer` |
| `DATA_DIR` | no | local spool directory, default `.data` |
| `FRONTEND_ORIGIN` | no | the Vercel URL of `apps/web` for CORS |
| `S3_ENDPOINT` `S3_BUCKET` `S3_ACCESS_KEY` `S3_SECRET_KEY` | no | Rimbra Bucket credentials |
| `LLM_BASE_URL` `LLM_API_KEY` `LLM_MODEL` | no | any OpenAI compatible endpoint (DeepSeek, Gemini, Groq, OpenRouter, OpenAI) |
| `GITHUB_TOKEN` | no | needed for reply 1 (open a fix PR) |
| `RIMBRA_API_KEY` | no | bearer token for Rimbra WhatsApp and Email APIs |
| `RIMBRA_WHATSAPP_URL` `RIMBRA_EMAIL_URL` | no | Rimbra send endpoints |
| `RIMBRA_WEBHOOK_SECRET` | no | verifies incoming WhatsApp webhooks |
| `NOTIFY_EMAIL` | no | where email alerts go |

## Step 5: Deploy and verify (30 seconds)

Deploy. Then check:

```sh
curl https://your-service.rimbra.app/healthz
# {"ok": true, "uptimeSeconds": 5}

curl -X POST https://your-service.rimbra.app/api/projects \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "orders-api", "targetUrl": "https://orders.internal/webhooks/rider"}'
# 201 with id, endpointKey, sdkKey, ...

curl -X POST https://your-service.rimbra.app/in/<endpointKey> \
  -H "Content-Type: application/json" \
  -d '{"order": {"id": 1, "rider": {"phone": "+234"}}}'
# {"received": true, "deliveryId": "dlv_..."}
```

The dashboard (`apps/web`) reads the same API with `Authorization: Bearer
$API_TOKEN`. The SDK (`@pockdev/sdk`) sends crashes to `POST /v1/errors`
with `X-PockDev-Key: <sdkKey>`.

## WhatsApp and Email adapters

The Rimbra senders in `internal/notify` post JSON with
`Authorization: Bearer $RIMBRA_API_KEY`:

* WhatsApp: `POST $RIMBRA_WHATSAPP_URL` with `{"to": "<phone>", "message": "<text>"}`
* Email: `POST $RIMBRA_EMAIL_URL` with `{"subject": "<subject>", "text": "<body>"}`

If the live Rimbra API shapes differ, adapt the two small functions in
`internal/notify/rimbra.go`; nothing else changes.

Incoming developer replies arrive at `POST /webhooks/whatsapp`. Configure
Rimbra to send them to your service URL. The engine verifies the HMAC
SHA 256 of the raw body against the `X-Rimbra-Signature` header
(format `sha256=<hex>`) using `RIMBRA_WEBHOOK_SECRET` (falls back to
`RIMBRA_API_KEY`). It accepts these body shapes:

```json
{"from": "+2348012345678", "text": "1"}
{"from": {"phone": "+2348012345678"}, "message": {"text": "1"}}
```

Reply `1` opens a fix PR, `2` replays paused webhooks, `3` mutes for 1 hour.

## Local development

```sh
cd apps/engine
cp .env.example .env   # fill in DATABASE_URL and API_TOKEN
go test ./...          # unit tests, no database needed
go run ./cmd/pockdev   # starts on :8080
```

Integration tests against real Postgres run when `DATABASE_URL` is set:

```sh
DATABASE_URL=postgres://... go test ./internal/db/ -run Integration -v
```

## Troubleshooting

* `config: DATABASE_URL is required`: set the variable from step 1.
* Webhooks stay `retrying`: the project `targetUrl` is unreachable or
  returning 5xx; the engine retries with exponential backoff, then pauses
  the delivery and files an incident.
* No WhatsApp alerts: `RIMBRA_API_KEY` and `RIMBRA_WHATSAPP_URL` must both
  be set, and the project needs `whatsappPhone`.
* Reply `1` says it cannot open a fix PR: set `GITHUB_TOKEN` and the
  project `githubRepo` (format `owner/repo`).
* Free tier memory: the engine is a single static binary with a 6
  connection pool; it fits comfortably in 512MB.
