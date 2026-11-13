---
name: konsult
description: Use the local `konsult` CLI (unofficial, Brainville API v2) to search marketplace consultancy assignments and to read or manage your organisation's own assignments, consultants and applications. Use when the user asks about Brainville assignments, consultant profiles or incoming applications.
---

# konsult

`konsult` is an unofficial CLI for the Brainville API v2. Not affiliated with or endorsed by Brainville AB.
Always call the local executable. Never write your own HTTP calls and never handle API keys.

## Check it works
```sh
konsult version && konsult auth status      # shows environment + whether credentials exist, never the values
```
If credentials are missing, tell the user to run `konsult auth login` themselves (it stores the keys in the OS keychain).
**Never ask for, print, copy or save API keys** (not in chat, files, prompts or the workspace).

## Output: always use a machine format
- `--json`: the service JSON, pretty-printed.
- `--ndjson`: one compact record per line. Best for large lists and piping to `jq`/line tools. `market search` emits one line per result.
- Without a flag you get a table on a terminal and JSON when piped. Never parse the table.
- On failure stdout holds the API error JSON (`--json`/`--ndjson`), stderr a message, and the exit code is non-zero: 3 auth/licence, 4 not found, 5 invalid input, 6 rate limit, 7 server error, 8 not confirmed.

## Two different things
- `konsult market ...` searches the **marketplace** (other companies' assignments). Needs Premium Sales and **consumes export quota** (documented: 100 per 30 days). One request per command; never loop, poll, paginate or bulk-export. Read the `market quota:` line on stderr and stop near the limit. Do not forward or republish results to third parties.
- `konsult assignments|consultants|applications ...` work on **your organisation's own** data. `market search` does not return your own assignments; `market get <id>` does (and counts as an export).

## Safety rules
- Brainville's terms allow automated tools and AI agents only through the organisation's own authorised API access. Do not scrape the website, share keys, or work around licence limits or quotas. Marketplace data is for internal use only (see `docs/terms-review.md`).
- Reads are free within the user's authorised scope. Start with `list`/`get`.
- Every change supports `--dry-run`. For any change: run `--dry-run`, show the user the result, get their explicit approval for *that action*, then run it with `--yes`. Never add `--yes` on your own initiative or as a habit.
- Never silently publish (a publication date or `noDelay` makes an assignment visible and may cost money), `close`, `unpublish`, `delete`, `share`, change an application `status`, or `boost`.
- `assignments close` unpublishes, **rejects all pending applications and emails the applicants**. Explain that first.
- `consultants boost` is **paid** (in the sandbox, boosting on currently fails with HTTP 404; the cause is unresolved). Only with the user's specific approval of the charge, then `--accept-charge`.
- Applications, CVs and consultant profiles are personal data: keep downloads out of the repo, don't send them to external services or LLM providers.
- Default environment is **production** (real data, real emails and charges). Use `--env sandbox` (Brainville's test environment) when the user wants to experiment, and always say which environment you are acting on.

## Common workflows
```sh
konsult market search --text 'C# .NET' --match and --ndjson          # one quota-consuming call
konsult market get 12345 --json
konsult assignments list --ndjson                                      # own assignments (published ones; drafts: use get ID)
konsult assignments get 12345 --json
konsult applications list --assignment 12345 --ndjson
konsult applications resume 777 -o ./cv-777.pdf                        # refuses to overwrite without --force
konsult assignments update 12345 --file change.json --dry-run          # preview, then ask the user, then rerun with --yes
```
## Updating a consultant profile
```sh
konsult consultants get 3639 --json                          # 1. read the full profile first
konsult consultants update 3639 --file profile.json --dry-run # 2. preview (personal data redacted), show the user
konsult consultants update 3639 --file profile.json --yes     # 3. only after approval
konsult consultants get 3639 --json                          # 4. read back and verify
```
- The file needs `firstName`, `lastName`, `email`; other sections are optional and **partial**: omitted sections are kept, a section you send **replaces** the old one entirely.
- Known traps: `education.graduation` is a year integer, certifications use `certification: true`, `personalGraph.headline` is max 25 characters, `picture` is a Base64 PNG, experience dates are `startDate`/`endDate`. Files and company pages cannot be edited via the API.
- `unpublish`/`close` return an assignment to a draft-like state; once an assignment was published it can no longer be deleted: `delete` still answers 200 but does nothing (even after unpublish/close), so read the assignment back to confirm.
- A `PUT` may time out yet still save: read back before retrying; never loop. A 400 saves nothing.
- Full field reference and format rules: `references/consultant-profiles.md`.

More detail: `references/workflows.md`, `references/safety.md`, `references/consultant-profiles.md`.

## When something fails
- 401 on `applications` or `market` usually means the account lacks the licence (Premium Buyer / Applications API, Premium Sales), not that the endpoint is missing. Say so.
- Do not invent fields, statuses or results. Unknown fields are preserved in `--json`; report what the API returned. The API contract is partly undocumented, see `docs/api-coverage.md` and `docs/open-questions.md` in the repository.
