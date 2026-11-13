# Sandbox findings (sanitised, 2026-11-13)

| ID | Observation | Evidence |
|---|---|---|
| P0-01 | No official OpenAPI found: 404 on `/api/v2/openapi.json`, `/api/v2/swagger.json`, `/openapi.yaml`, `/swagger/v1/swagger.json` (developer site), and `/swagger/v1/swagger.json`, `/v2/swagger.json` (API sandbox). | six GET requests |
| P0-02 | Bearer `base64(user:sender)` accepted; consultant and assignment reads return 200. | `auth status --verify` |
| P0-03 | `/consultant` 200, `/assignment` 200, `/application` **401** (presumably missing Premium Buyer / Applications licence; not an absent endpoint). Market untested. | smoke reads |
| P0-04 | `DELETE /assignment/delete` worked with `id` in query **and** body (the CLI sends both). Which one the server needs is not isolated. | draft delete |
| P0-07 | Draft create without `publicationEndDate` works; `publicationEndDate` came back `null`; delete works. Draft did not appear in the default `GET /assignment` list but `GET /assignment?id=` returns it. | id 11800, deleted |
| P1-08/P1-07 | Required in practice: `title`, `text`, `competenceAreaId`, `level`, `extent`, **`locations`** (docs say optional; `locations` missing → 400 `[locations] has an invalid value`). Numeric `competenceAreaId`, `extent.type/value`, `countryId/regionId` accepted. | create attempts |
| P1-11 | `GET /consultant?id=` returns the full profile. Top-level keys: active, companyCategories, description, education, email, experiences, firstName, id, industries, languages, lastName, linkedin, mobile, personalGraph, personalityTests, personalityTraits, picture, preferences, roles, skills, strengths, tagline, title, worksamples. (Note `linkedin`, docs say `linkedInUrl`.) | one read |
| – | `GET /assignment?id=` returns an array. After deletion `GET ?id=` returns HTTP 400 with an empty body (not 404). | |
| – | `GET /assignment?status=draft` → 400 invalid value; valid `status` values are unknown. | |
| – | Response headers on 401 carry no rate-limit headers (P2-03). | header names only |
| – | Assignment output uses `assignmentStartDate`/`assignmentEndDate`, timestamps like `2026-11-13T11:02:55.097` **without a time zone**. | |

| P1-11/12 | Consultant create works with only `firstName`, `lastName`, `email` plus optional `title`, `tagline`, `description`, `languages`, `roles[{name,level}]`, `skills[{name,level}]`, `active`. Three synthetic consultants (ids 3639-3641: Java, .NET, Flutter/native mobile; `example.com` addresses) were created. `GET ?id=` returns the skills and roles as sent. Levels seen: `Expert`, `High`, `Advanced`. | create + get |
| – | `consultants list --query flutter` matched the mobile consultant. | 1 search |
| – | `PUT /consultant` with `id` in the body is a **partial update**: sending only name, email and `tagline` kept the existing skills. Response `{id, message}`. | update of 3640 |
| – | Create responses are `{"id": n, "message": "Entity was saved."}`; delete returns an empty 200. | |

| P1-12 | Nested consultant types, verified by round trip on ids 3639-3641 (all sections of the web UI populated except files): experience dates are `startDate`/`endDate` (docs leave the names blank); `education.graduation` must be a year integer (`YYYY-MM-DD`, `YYYY-MM`, ISO timestamps all rejected); the certification flag is `certification: true` (docs say `certificate`, which is ignored); `personalGraph.headline` max 25 chars; `industries` accepts English names; `picture` accepts a Base64 PNG (~1.5 kB); `mobile{countryId,number}` reads back with `fullNumber`. | updates + reads |
| – | Validation is atomic: a 400 saves nothing and lists every invalid field. | 3 rejected updates |
| – | Reliability: one `PUT` with a picture hit the 30 s client timeout yet **was saved**; another returned an empty 200 and was **not** saved. Re-sending worked. Reads are the source of truth; no automatic retries is the right default. | updates of 3639, 3641 |
| – | No API field for profile files (Filer) and no company-profile endpoint in v2. | docs review |

| P0-05, P1-04/05 | Market works in the sandbox (Premium Sales present). Envelope: `results`, `exportedInPeriod`, `volume`, `period{startDate,endDate}` (a 3-day window here), `actualResultCount` (was 0 even with results: do not rely on it). `volume` was **10000**, not 100. Quota counts **distinct exported assignments**: re-running a search returning already-exported assignments did not raise `exportedInPeriod`; a search with 0 results costs nothing. 5 searches exported 17 assignments. No pagination seen (max 11 results). | 5 searches |
| P1-01 | `textMatchType` lowercase `and`/`or` accepted. `and` with `Java Spring` returned 5 hits; `or` over many terms is noisy (matches loosely on tags). | searches |
| P2-02 | Confirmed: result `url` fields point to `https://localhost.brainville.com/...`. Never use them. | results |
| – | The sandbox marketplace is mostly synthetic test data from one company (US-style city names, tag lists with unrelated skills); only a few Swedish assignments. | results |

| P0-07/P1-09 | Three more synthetic drafts created (11801 Java, 11802 .NET, 11803 Flutter; all with `locations`, tags, roles, `allowRemote: "Partly"`, dates). Drafts are hidden from the default `assignments list` and **do not appear in `market search`** (0 results for their unique title prefix). | create, list, search |
| P1-06 | `maxRate` is **silently dropped on drafts**: four variants (`amount`/`rate`, with/without `currency`, `display` Informative/Mandatory) were accepted with 200 and read back as `null`. Unknown whether it is stored only when published. Market results of published assignments do show `maxRate{amount,currency,type,display}` (output uses `amount`, not `rate`). | 5 creates/updates |

| P0-06/P1-09 | Published 11801 (Java seed) via `PUT` with `publicationEndDate: 2027-03-31`, no `noDelay`. `recipients` stayed `[]`, `noDelay` false; no emails observed (not provable from here). `maxRate` is stored and returned once published. It appears in the default `assignments list`. The other two seeds (11802, 11803) remain drafts. Published assignments can no longer be deleted (documented; only unpublish/close). | publish test |
| P1-05 | `market get <id>` returns the same envelope as search plus `accountEmail`, and **does** return the account's own published assignment (company shown as the account's company); `actualResultCount` is correct there (1), unlike in search (0). It counted as an export (17 -> 18). | market get |
| P1-04 | `market search` does **not** return the account's own published assignment (searches for its title prefix, "Java Kafka" and "Kafka" found nothing): own assignments seem excluded from search, or indexing is delayed (not isolated). A search therefore cannot be used to match consultants against your own seeds. | 3 searches |

| – | All three seeds (11801-11803) are now published (end 2027-03-31, no `noDelay`). The `GET /assignment` list response does not carry `tags`: fetch each assignment with `?id=` for the full record. | publish + reads |

| P0-04 | **id placement, isolated with curl** (body-only vs query-only): `POST /assignment/unpublish`, `POST /assignment/close`, `POST /consultant/share`, `POST /consultant/boost` need **`id` in the query**: body-only gives **404** "no HTTP resource". `DELETE /assignment/delete`: body-only → 400 `[id] invalid`, query → 200. `DELETE /consultant`: body-only → **405**, query → 200. **`active` for share/boost is a query parameter** too; a body `active` is ignored (boost then defaults to on). A POST without any body needs `Content-Length: 0` (else 411; Go sends it). The CLI previously sent body `active`; fixed (id/active in query, no body). | curl + CLI |
| P0-06/P1-09 | `unpublish` (11801) and `close` (11802): both returned 200 and set `publicationEndDate` back to `null` (the assignment then behaves like a draft, hidden from the default list); closed ones appear with `--include-closed`. There were no applications, so no emails were provoked. Share on/off returned 200; the shared state is **not readable** through the API (no field on the profile), so effects are unobservable. | state checks |
| P1-15 | `boost` **on** returns 404 `errorCode 1` with an empty field for every request shape (consultants 3639/3641, id or email); `active=false` returns 200. Probably no payment card or boost capability on the sandbox account; not the documented errorCode 6. No charge observed. Unresolved. | 7 variants |
| P0-08/P1-10 | Assignment files work end to end: `POST /assignmentfile` returned **201**, the file appears in `files`, `GET` streams the exact bytes (saved 0600; second download refused without `--force`), `DELETE /assignmentfile` with a **JSON body** returns 200 and removes it. | draft 11806, deleted |
| – | A throwaway consultant was created and deleted (query id). Existing seeds now: consultants 3638-3641, assignments 11801 (unpublished), 11802 (closed), 11803 (published). | |

| P1-09 | **Delete of a previously published assignment is a silent no-op**: after `unpublish` (11801, 11803) or `close` (11802), `DELETE /assignment/delete` returned **200 but the assignment still exists** (`get` and `list --include-closed` still return it). Never-published drafts are really deleted (verified several times). So a 200 from delete does not prove deletion: read back. Published seeds therefore remain in the sandbox. | 3 deletes |

Not tested on purpose: close, unpublish, share, boost, status change, publication, file operations, consultant delete.
