# API coverage

Levels: **documented** (public docs only), **sandbox-verified**, **inferred**, **unknown**, **blocked** (licence/role).
Sandbox base: `https://apisandbox.brainville.com/v2`. (`sandbox.brainville.com` is the web UI, not the API.)

| Method | Path | Command | Status |
|---|---|---|---|
| POST | /market/search | `market search` | **sandbox-verified** (counts distinct exported assignments) |
| GET | /market?id= | `market get` | documented; not tested (quota) |
| GET | /assignment | `assignments list/get` | **sandbox-verified** (returns an array even for `id=`) |
| POST | /assignment | `assignments create` | **sandbox-verified** (draft) |
| PUT | /assignment | `assignments update` | documented, not tested |
| DELETE | /assignment/delete | `assignments delete` | **sandbox-verified** (draft; id sent in query *and* body) |
| POST | /assignment/unpublish?id= | `assignments unpublish` | **sandbox-verified** (id in query) |
| POST | /assignment/close | `assignments close` | **sandbox-verified** (id in query; no applications, so emails untested) |
| POST/GET/DELETE | /assignmentfile | `assignments files add/get/delete` | **sandbox-verified** (POST returns 201; DELETE accepts a JSON body) |
| POST | /assignment/suggestedtags | `assignments tags suggest` | documented, not tested |
| GET | /application | `applications list/get` | **blocked**: 401 with the sandbox keys (licence/role presumed) |
| PUT | /application | `applications status` | documented; blocked |
| GET | /application/resume, /application/file | `applications resume`, `files get` | documented; blocked |
| GET | /consultant | `consultants list/get` | **sandbox-verified** (`id` returns the full profile) |
| POST/PUT | /consultant | `consultants create/update` | **sandbox-verified** (update is partial) |
| DELETE | /consultant | `consultants delete` | **sandbox-verified** (id in query) |
| POST | /consultant/share, /consultant/boost | `consultants share/boost` | share **sandbox-verified** (200, effect unobservable); boost off 200, **boost on 404 unresolved** |

Authentication `Bearer base64(userKey:senderKey)`: **sandbox-verified**.
Deferred: webhooks (XML, out of scope), countries/industries XML (reachable, `text/xml`, not cached by the CLI).
