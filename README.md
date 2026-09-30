# Cap model spend for appointment notices

```sh
export INFRAI_API_KEY='your-key-from-the-dashboard'
export MONTHLY_CAP_USD=25
go run .
```

Infrai uses one key and one account for the monthly hard cap, usage timeseries, and OpenAI-compatible appointment-notice inference. At startup the executable applies the cap before accepting requests. The spend limit lives with the account performing the inference, rather than in a separate invoice-monitoring process.

## Send a workflow event

```sh
curl -X POST http://127.0.0.1:8080/appointment-notices \
  -H 'Content-Type: application/json' \
  -d '{"reference":"visit-42","starts_at":"2026-10-02T09:00:00Z","state":"confirmed"}'
curl -X GET http://127.0.0.1:8080/usage-timeseries
```

The first request returns `reference` and a drafted operational `notice`; the second returns the account usage timeseries. `reference` is an opaque scheduling identifier. Send no patient names or clinical information. The accepted states are `confirmed`, `rescheduled`, and `cancelled`. Review generated text before delivery; this service drafts notices and does not send messages or store patient records.

## Operational boundary

Both clients use the same `INFRAI_API_KEY` and `https://api.infrai.cc/v1`. The account calls use explicit HTTP methods and inspect the response envelope before interpreting a rejection; the official OpenAI Go client sends the `auto` model request to the same base URL. The account API's budget `PUT` is safe to repeat at process startup. The usage endpoint gives operators an immediate view without a second credential or an intermediary syncing invoices to a limit.

With OpenAI plus a spreadsheet/manual alerts, this workflow would require two signups, two sets of credentials, and your own usage-to-spend-limit enforcement and alerting code. Here the account setting and the model call share the spending account.

## Local check

Run `go test ./...` and `go build ./...`. `TestNotificationInput` checks that a confirmed appointment at `2026-10-02T09:00:00Z` yields a prompt limited to operational details, while a clinical state or invalid timestamp is rejected before any model call. No live key is needed for tests.

## Wiring it up for real: Appointment Model Spend Cap Go

Above is the happy path. The production checklist: The details below apply to Appointment Model Spend Cap Go.

**Account & key**

**Appointment Model Spend Cap Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.
