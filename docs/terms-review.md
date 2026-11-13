# Terms review (P2-06)

Reviewed 2026-11-13. This is a factual summary for developers, **not legal advice**. Read the sources yourself and ask Brainville when in doubt. No separate API licence, developer agreement or third-party-client terms were found.

## Sources
- Terms of Use v1.0 (PDF): <https://www.brainville.com/Content/Legal/Terms%20of%20Use%201_0.pdf>
- Privacy Policy v1.0 (PDF): <https://www.brainville.com/Content/Legal/Privacy%20policy%201_0.pdf>
- Developer pages: [API access](https://developer.brainville.com/api/api-access/), [Authentication](https://developer.brainville.com/api/v2/authentication/), [Market overview](https://developer.brainville.com/api/v2/market/overview/), [Assignment Export](https://developer.brainville.com/assignment-export/introduction/)
- [FAQ](https://www.brainville.com/en/Faq), [public API page](https://www.brainville.com/PublicPage/API?lang=en)

## What applies to this tool
| Topic | What the sources say | Consequence for konsult |
|---|---|---|
| Automated access | ToU 5(h): no bots, crawlers, scrapers, **AI agents**, scripts or other automated tools to access the Service or collect data, "except through APIs or integrations expressly authorised by Brainville", within the scope, purposes and limits of that authorisation. No circumventing access restrictions or rate limits. | Use only your own approved API access (company approved, own keys). Do not use konsult or an agent to scrape the website, share keys, or get around licence limits. The CLI never retries and never paginates by itself. |
| Internal use | Market API / Assignment Export data may only be used inside the approved legal entity. No redistribution, resale, sublicensing or making it available to third parties. No use to build or enrich external marketplaces, aggregation or matching services without Brainville's written approval. | Marketplace data must stay in your organisation. The skill forbids forwarding it. Out of scope: any hosted or shared service. |
| Licences and quota | API access requires company approval. Market needs Premium Sales (up to 100 exported assignments per 30 days; more needs an Assignment Export licence). Applications need Premium Buyer or the Applications API licence. Posting assignments by API is free. | Your licence defines what works. The sandbox showed `volume` 10000, so read `volume` from responses and never assume 100. |
| Commercial exploitation | ToU 5(b): no reproducing or reselling any part of the Service. ToU 8: Brainville Content is protected; copying and publishing it is restricted. | This repo contains original code and sanitised notes only: no scraped marketplace data, no Brainville logos or branding. |
| Personal data | ToU 4: you are the controller of personal data you submit; Brainville is the processor. You must have a lawful basis, inform the individuals and gather consent where required (4.5), and submit only relevant data. | Consultant profiles, CVs and applications are personal data: keep local files protected, do not send them to external services, only submit data you may submit. |
| Liability | ToU 9-10: service is provided as is; liability is limited. | The CLI is unofficial and provided as is (MIT). Test in the sandbox first. |
| Changes | ToU 1: terms may change at any time with immediate effect. | Re-check periodically. API v1 ends 2026-12-31 (v2 only here). |

## Open
- Whether a hand-written third-party client needs any extra approval is not stated anywhere. Brainville can be asked at support@brainville.com.
- The ToU text was read from the PDF with a rough text extraction; a few words were missing. Confirm against the PDF.
