# Open questions

Resolved items are in `sandbox-findings.md` (P0-04 id placement, P0-08 downloads, P1-10 DELETE body, P1-05 market get envelope, P0-07 drafts, P1-08 required fields).

| ID | Question | Status / test plan |
|---|---|---|
| P1-15 | Why does `boost` on return 404 `errorCode 1` (off works)? | Likely missing payment card or boost capability. Ask Brainville; register a test card only with explicit user approval. |
| P0-06 | Do publish, close, share or status change send email in the sandbox? | Not observable here (no applications, no subscribers). Needs a second account to receive mail. |
| P1-13 | Application field types and statuses. | Blocked: 401 (Premium Buyer / Applications licence missing). |
| P1-01..03 | `market search`: `textMatchType` casing (lowercase works), `extent` casing, `id: 0`. | `extent` and profile `id` untested. |
| P1-04 | Does search exclude own assignments, or is indexing delayed? | Not isolated. `market get <id>` returns them. |
| P1-06 | `maxRate` dropped on drafts, stored once published. Why? | Observed, cause unknown. |
| P1-09 | Reversibility. | unpublish/close return the record to a draft-like state (`publicationEndDate` null); a previously published assignment cannot be deleted even after unpublish/close (delete returns a misleading 200); re-publishing is untested. |
| – | Valid `GET /assignment?status=` values. | Unknown (`draft` rejected). |
| P2-01 | v2 webhooks XML or JSON. | Out of scope for v1. |
| P2-04 | Countries/industries XML caching terms. | Reachable; the CLI does not cache. |
| P2-05 | Company profile / me endpoint. | None in the v2 docs; the company page cannot be edited through the API. |
| P2-06 | Third-party client terms. | MIT confirmed by the owner; review Brainville terms before the public release. |
