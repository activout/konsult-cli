# Updating consultant profiles

Field names and formats below are **sandbox-verified** (2026-11-13) unless marked otherwise. The public docs are incomplete or wrong in places; trust this file and the API's validation errors over the docs.

## Procedure
1. Read first: `konsult consultants get <id|email> --json`. The profile read is the full profile; the list is not.
2. Write a JSON file with only what should change, plus `firstName`, `lastName`, `email` (the API requires them). **Updates are partial**: sections you leave out are kept. A section you send is **replaced whole** (send all items of a list, not just the new one).
3. `konsult consultants update <id> --file profile.json --dry-run`: shows the payload with personal data redacted. Show it to the user.
4. After approval: same command with `--yes`.
5. **Always read back** (`consultants get`) and compare. Do not retry on a timeout or an odd reply: a `PUT` can time out *and still be saved*, and one update returned an empty 200 without saving. Check the profile, then repeat once only if it is really missing.
- Validation is all-or-nothing: on HTTP 400 nothing is saved, and the errors name each bad field (e.g. `education[0].graduation`). Fix all of them and resend.
- Profiles hold personal data. Keep files outside the repo and delete them afterwards. For tests use obviously synthetic people and `example.com` addresses.

## Sections (UI name → field)
| UI section | Field | Format |
|---|---|---|
| basics | `title`, `tagline`, `description` | strings |
| Erfarenheter | `experiences[]` | `{title*, company*, location, description, startDate, endDate, extent{type,value}, tags[]}`. Dates `YYYY-MM-DD` (read back as `...T00:00:00`). The docs leave the two date names blank: they are `startDate`/`endDate`. |
| Utbildningar | `education[]` | `{title*, school*, graduation, certification, url, tags[]}`. `graduation` is a **year as integer** (`2008`); `YYYY-MM-DD` is rejected. A certification/course is flagged with **`certification: true`** (boolean). The docs call it `certificate`, which is accepted but ignored. |
| Personlig graf | `personalGraph` | `{headline, items[{name, value}]}`. `headline` max **25 characters**; values are integers (use ~100 in total). |
| Styrkor | `strengths[]` | list of strings |
| Personliga punkter | `personalityTraits[]` | `{question, answer}` |
| Personlighetstest | `personalityTests[]` | `{personalityType, url, description}` |
| Arbetsprover | `worksamples[]` | `{headline, description, url}` |
| Profilbild | `picture` | Base64 **PNG** (no `data:` prefix). Large pictures make the request slow. |
| Branscher | `industries[]` | industry **names** in English as in `industries.xml` (e.g. `"IT & Telecom"`). Ids not tested. |
| skills, roles | `skills[]`, `roles[]` | `{name, level}`; levels seen: `Medium`, `High`, `Advanced`, `Expert` |
| other | `languages[]` | list of strings |
| other | `mobile` | `{countryId, number}` (Sweden = 22, number without leading 0); reads back with `fullNumber` |
| other | `preferences` | `{locations[{countryId, regionId, city}], availability (YYYY-MM-DD), allowRemote, allowOnsite, extent{type,value}, preferredRate}` |
| other | `active` | boolean, visibility in lists |
| other | `linkedInUrl` | written as `linkedInUrl`, read back as `linkedin` (not tested with a value) |

`*` required. **Files (Filer) have no API field**: attachments can only be added in the web UI. Company profile pages are not in API v2.

## Effects to warn about
- `consultants share` makes the profile visible to every company on the marketplace; `boost` costs money. A profile update alone does neither, but check `active` and whether the consultant is already shared before editing a live profile.
- An update changes real data the consultant may see. Ask before overwriting existing text.

## Minimal example
```json
{"firstName":"Ann","lastName":"Example","email":"ann@example.com",
 "tagline":"Flutter and native apps",
 "education":[{"title":"BSc Design","school":"Example University","graduation":2016}],
 "personalGraph":{"headline":"Where my time goes","items":[{"name":"Coding","value":60},{"name":"Design","value":40}]}}
```
