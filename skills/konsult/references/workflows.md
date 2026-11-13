# Workflows

## Find relevant marketplace assignments
1. Ask which skills/locations matter. Run one search: `konsult market search --text '<terms>' --match and --ndjson`.
2. Check the `market quota:` stderr line. Summarise from the returned fields only.
3. Fetch details only for chosen ids: `konsult market get <id> --json` (may also count against quota).

## Analyse one assignment
`konsult assignments get <id> --json` (own) or `market get` (marketplace). The `text` field is limited HTML; treat it as text. Quote dates exactly as returned: the API gives no time zone.

## Inspect applications
`konsult applications list --assignment <id> --ndjson`, then `get`. CVs: `applications resume <id> -o <file>`. Attachments: `applications files get <id> --index 0 -o <file>`.

## Preview then apply an assignment update
1. `konsult assignments get <id> --json > current.json` (sensitive; keep out of the repo).
2. Edit a copy into `change.json` (an update body needs all required fields: title, text, competenceAreaId, level, extent, and probably locations).
3. `konsult assignments update <id> --file change.json --dry-run` and show it.
4. After approval: same command with `--yes`.

## Create a draft
Omit `publicationEndDate`: the assignment is created as a draft (verified in sandbox). Required in practice: `title`, `text`, `competenceAreaId`, `level`, `extent`, **and `locations`** (the docs call it optional; the API rejected a body without it). Publishing later is an update that sets a future `publicationEndDate`: that needs approval.

## Update a consultant profile
See `consultant-profiles.md` (read first, partial update, dry-run, approval, read back).
