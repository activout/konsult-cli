# Safety policy

| Action | Effect | Needs |
|---|---|---|
| `list`, `get`, `tags suggest`, downloads | read-only | nothing special |
| `market search`, `market get` | consumes export quota, licence-gated | user knows quota is used; never loop |
| `assignments create/update` | draft, or visible if publication date / `noDelay` set | dry-run + approval |
| `assignments unpublish` | removes from marketplace | dry-run + approval |
| `assignments close` | unpublishes, **rejects pending applications, emails applicants** | dry-run + explicit approval after explaining |
| `assignments delete`, `files delete`, `consultants delete` | permanent | dry-run + approval |
| `consultants share` | profile searchable by all Brainville companies | dry-run + approval |
| `consultants boost` | **paid** | specific approval of the charge + `--accept-charge` |
| `applications status` | applicant may be notified | dry-run + approval |

- Non-interactive runs without `--yes` are refused (exit 8). That is intentional: do not work around it.
- Keys: only via `konsult auth login` (OS keychain) or `KONSULT_USER_KEY`/`KONSULT_SENDER_KEY` injected by the user. Never in chat or files.
- `--debug` logs only method, path and status.
- Do not send marketplace data, CVs or profiles to other services, and stay inside the organisation's approved tooling.
