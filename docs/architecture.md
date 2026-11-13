# Architecture

Single Go module, `cmd/konsult` (package `main`):

- `client.go`: one HTTP client. Builds `Authorization: Bearer base64(user:sender)`, never follows redirects, no automatic retries (mutations and market calls may not be idempotent or may consume quota), maps HTTP/API errors to exit codes 1-8.
- `config.go`: credentials. Environment variables (`KONSULT_USER_KEY`, `KONSULT_SENDER_KEY`) override the OS keychain (`github.com/zalando/go-keyring`: macOS Keychain, Windows Credential Manager, Linux Secret Service). Keychain entry: service `konsult`, account = environment name. Base URL: production by default (`--env sandbox` for the test API); `--base-url` must be https (http only on loopback).
- `output.go`: table (TTY), JSON, NDJSON. Records stay `json.RawMessage`, so unknown fields survive. Redaction for `--dry-run`.
- `main.go`: cobra root, `read` / `mutation.run` / `download` helpers (confirm and dry-run logic live in one place).
- `commands.go`: command definitions.

Why cobra: completion and help are required; the standard library would mean reimplementing both.
Platform note: go-keyring on macOS runs `/usr/bin/security`; the other platforms use native APIs.
