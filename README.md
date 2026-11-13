# konsult

Unofficial command-line interface for the Brainville API v2.
**Not affiliated with or endorsed by Brainville AB.** API docs: <https://developer.brainville.com/api/v2/>.

A single static binary (no runtime needed) for people and AI agents: search marketplace assignments and manage your organisation's own assignments, consultants and applications. Includes an agent skill (`skills/konsult/`).

## Install
Download the archive for your platform from GitHub Releases, check it against `checksums.txt`, and put `konsult` on your `PATH`. Or build: `go build -o konsult ./cmd/konsult`.

## Credentials
Prerequisite: Brainville API access (user key + sender key). Market needs Premium Sales, applications need Premium Buyer or the Applications API licence.

```sh
konsult auth login --env sandbox          # prompts without echo; stores the keys in the OS keychain
konsult auth status --verify              # never prints the keys
konsult auth logout --env sandbox
```
- Storage: macOS Keychain, Windows Credential Manager, Linux Secret Service. Keys are never accepted as arguments.
- Piped: `printf '%s\n%s\n' "$USER_KEY" "$SENDER_KEY" | konsult auth login` (user key first).
- Headless/CI: set `KONSULT_USER_KEY` and `KONSULT_SENDER_KEY`; they take precedence over the keychain.
- Precedence: flag > environment variable > keychain > default. Default environment is **sandbox**; production needs `--env production` (or `KONSULT_ENV`).

## Use
```sh
konsult consultants list --ndjson
konsult assignments list --json
konsult assignments get 12345
konsult assignments create --file draft.json --dry-run
konsult assignments delete 12345 --yes
konsult market search --text 'C# .NET' --match and --ndjson   # consumes export quota
konsult applications list --assignment 12345 --ndjson
konsult applications resume 777 -o cv.pdf
konsult completion zsh
```
Output: table on a terminal, JSON when piped, `--json` (service JSON) or `--ndjson` (one compact record per line; `market search` emits one line per result). In those modes stdout holds only JSON; messages go to stderr.

Safety: every change has `--dry-run` (redacted payload), asks for confirmation, and refuses to run non-interactively without `--yes`. `boost` is paid and needs `--accept-charge`. `close` rejects pending applications and emails applicants. Downloads never overwrite without `--force` and are created with mode 0600.

Exit codes: 0 ok, 1 general, 2 usage, 3 auth/licence, 4 not found, 5 invalid input, 6 rate limited, 7 server error, 8 not confirmed.

## Agent skill
Copy or symlink `skills/konsult` into your agent's skills directory (for Claude Code: `~/.claude/skills/konsult`). It contains no secrets.

## Limits and status
See `docs/api-coverage.md`, `docs/sandbox-findings.md`, `docs/open-questions.md` and `openapi/unofficial-v2.yaml` (community-authored, evidence-tagged). Not yet verified: market, application, close/share/boost and file operations (licence or side effects). Not covered: webhooks, v1, XML. The `macOS` keychain backend runs `/usr/bin/security`.

Troubleshooting: HTTP 401 on `applications`/`market` usually means a missing licence rather than a wrong key. `--debug` logs method, path and status only.

## Security
See `SECURITY.md`. Licence: MIT (see `LICENSE`).
