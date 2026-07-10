# QRSurvey — Implementation Plan

## Context

The `qrsurvey` repo currently contains only a README describing the product vision: a business (mall, doctor's office, etc.) puts up a poster with a QR code; a customer scans it, is walked through a short, delightful multi-step survey about their experience, and ends by entering a prize contest. GitHub issue #2 ("QR Survey Setup") turns this vision into a concrete technical spec: Go server + SQLite database, fronted by Caddy for HTTPS, with an explicit 6-entity data model (Survey, Contest, Poster, Survey Item, Contestant, Answer). The issue also calls out one hard constraint: because the `answer` table references `contestant_id` before contestant info is known, **the whole survey must be submitted as a single atomic form at the end** — no partial/incremental persistence while the user is still answering.

This is a greenfield build (no existing code to reuse). The plan below is the full build-out, from schema to deployment, based on the issue's spec plus README's UX goals (progress bar, satisfying animations, building excitement toward the contest entry).

## Key architecture decisions

| Area | Decision | Why |
|---|---|---|
| Router | stdlib `net/http` (Go 1.22+ method/wildcard patterns) | Handful of simple routes; no need for chi/gorilla |
| SQLite driver | `modernc.org/sqlite` (pure Go, no CGO) | Trivial cross-compilation to a static binary; matches simple systemd+Caddy deploy |
| Migrations | Hand-rolled: embedded `.sql` files (`embed.FS`) applied at startup, tracked in a `schema_migrations` table | Avoids a second CLI/dependency for a single-environment app |
| Survey flow UX | One server-rendered HTML document (all wizard steps pre-rendered) + vanilla JS driving show/hide, progress bar, and CSS animations | Needs snappy transitions with zero build tooling; whole app ships as one Go binary via `go:embed` |
| "No partial persistence" | Client holds answers in an in-memory JS object across steps; **one** `POST /p/{posterID}/submit` at the very end inserts contestant + all answers in a single DB transaction | Matches issue's explicit constraint exactly |
| QR generation | `skip2/go-qrcode`, generated on-demand and cached to disk, served from an admin route | Posters are created rarely; on-demand+cache is simpler than a batch job |
| Admin auth | HTTP Basic Auth, single bcrypt-hashed credential from env config | Single-operator internal tool; session/login system is unjustified overhead |
| Deployment | One Go binary + one SQLite file, Caddy reverse-proxies to `127.0.0.1:8080`, both run via systemd, no Docker | Matches issue's "Caddy for HTTPS" instruction; nothing here needs containers |
| Duplicate-entry / spam control | `UNIQUE(contest_id, phone)` DB constraint + hidden honeypot field + IP rate limiting on submit | Deters bots without adding CAPTCHA friction to a "fun" flow |

## Project layout

```
qrsurvey/
  cmd/qrsurvey/main.go            # wiring: config, db open+migrate, router, graceful shutdown
  internal/
    config/config.go              # env-based config (addr, db path, admin creds, base URL)
    db/
      db.go                       # sql.DB open, PRAGMA foreign_keys=ON, WAL mode
      migrations.go + migrations/0001_init.sql
      queries/{survey,contest,poster,survey_item,contestant,answer}.go
    models/models.go
    handlers/
      public/{scan.go, submit.go}      # GET /p/{posterID}, POST /p/{posterID}/submit
      admin/{surveys,contests,posters,results}.go, auth.go
      middleware/{logging,recover,ratelimit}.go
    qrcode/qrcode.go
    web/
      templates/ (layout, survey_wizard, admin_*)
      static/ (wizard.js, wizard.css) — embedded, no build step
      render.go
  Caddyfile
  deploy/qrsurvey.service
  go.mod
```

`internal/db` owns all SQL; handlers never write raw queries. This centralizes the atomic-submission logic in one place: `db.SubmitEntry(ctx, poster, contestant, answers)`.

## Schema (SQLite, `0001_init.sql`)

Six tables exactly matching issue #2's field lists, plus deployment-necessary additions (timestamps, `sort_order` on survey_item, indexes):

- `survey(id, description, created_at)`
- `contest(id, survey_id → survey, end_date, prize, created_at)` — index on `survey_id`
- `poster(id, contest_id → contest, internal_poster_info, created_at)` — index on `contest_id`
- `survey_item(id, survey_id → survey, question, response_1..5, sort_order)` — index on `survey_id`
- `contestant(id, contest_id → contest, name, phone, address, created_at)` — index on `contest_id`, **`UNIQUE(contest_id, phone)`** (anti-duplicate mechanism)
- `answer(id, contestant_id → contestant CASCADE, date, poster_id → poster, contest_id → contest, survey_item_id → survey_item, value_selected CHECK 1-5)` — indexes on `contestant_id`, `poster_id`, `(contest_id, survey_item_id)` for the results-aggregation query

FKs use `ON DELETE RESTRICT` (survey/contest/poster/contestant referenced by history should not vanish silently) except `answer.contestant_id` which cascades (deleting a contestant should clean up their answers).

## Public survey flow

1. `GET /p/{posterID}` — one join resolves poster → contest → survey → survey_items. If `contest.end_date` has passed, render a "this contest has ended" page instead of the wizard. If the survey has 0 items (data-integrity issue the admin UI should prevent), render a defensive "not ready yet" page and log loudly. Otherwise render the full wizard (all steps in the DOM, later steps hidden) with survey data embedded as JSON.
2. Client JS (`wizard.js`) holds `answers = {surveyItemId: valueSelected}` in memory, advances `currentStep`, drives the progress bar and button/step animations, escalates "almost there!" copy — all with zero network calls.
3. Only the final contest-entry step's submit fires `POST /p/{posterID}/submit` with `{name, phone, address, honeypot, answers[]}`.
4. Server validates (server-side, never trusting client IDs): poster→contest re-resolved fresh; contest not expired; submitted answer set exactly matches the survey's item IDs; each value in 1-5; honeypot empty (else fake-success, no write); rate limit by IP.
5. `db.SubmitEntry` runs one transaction: insert contestant row, then N answer rows. A `UNIQUE(contest_id, phone)` violation rolls back and returns a friendly "you've already entered this contest" response rather than an error page.
6. If the user just closes the tab mid-survey, nothing is ever written — by design.

## QR codes & admin

- `GET /admin/posters/{id}/qrcode.png` generates (via `skip2/go-qrcode`) and caches a PNG encoding `{BASE_URL}/p/{posterID}`.
- Admin surface (all behind Basic Auth): CRUD for survey + survey items (question/5 labels/order), contest (survey picker, end_date, prize), poster (label + QR display/download); results view aggregating `answer` by `(survey_item_id, value_selected)` per contest as simple percentage bars; contestant list with CSV export (needed for running the actual prize drawing).

## Deployment

Caddyfile reverse-proxies the domain to `127.0.0.1:8080` with automatic HTTPS; the Go binary itself speaks plain HTTP and is never exposed directly. Both Caddy and the Go binary run as independent systemd units (`deploy/qrsurvey.service` for the app). Deploy = cross-compile (`CGO_ENABLED=0`), scp binary, `systemctl restart qrsurvey`. Backups = cron `sqlite3 ... .backup` snapshot of the single DB file.

## Testing priorities

1. DB-layer: constraints fire correctly (FK, `value_selected` CHECK, unique phone-per-contest); **atomicity test** for `SubmitEntry` — a forced mid-transaction failure must leave zero rows (this is the highest-value test given the issue's explicit emphasis on all-or-nothing submission).
2. Handler tests for `/p/{posterID}/submit`: missing/extra answers, out-of-range values, expired contest, honeypot triggering, duplicate phone.
3. Admin CRUD: thin smoke tests only (lower risk, single trusted user).
4. Skip browser/E2E testing for v1; validate the wizard UX manually on real phones during Phase 6.

## Phased build order

0. **Scaffolding** — `go.mod`, server boots with `/healthz`, config loading, logging middleware.
1. **Schema & migrations** — all 6 tables + indexes, migration runner, typed query layer. *Done:* migrations apply cleanly and idempotently; constraint tests pass.
2. **Public survey flow (functional, unstyled)** — scan + submit endpoints, full validation, atomic transaction. *Done:* a manually-seeded contest can be scanned, answered, and submitted end-to-end producing correct rows; all edge cases return correct responses.
3. **QR code generation** — on-demand cached PNGs. *Done:* a real phone camera scan lands on the correct survey.
4. **Admin/back-office** — full CRUD + Basic Auth + results view + CSV export. *Done:* an operator can run an entire contest lifecycle with zero direct DB access.
5. **Deployment** — Caddyfile, systemd units, backup cron. *Done:* reachable over HTTPS on a real domain, survives a restart, backup/restore verified.
6. **Polish/animations** — real CSS transitions, mobile-first styling, copy escalation. *Done:* flow feels good on an actual phone (subjective, test with real scans).
7. **Tests/hardening** — fill remaining table-driven tests, verify rate limiter, light burst-submission test. *Done:* suite green, no data corruption under concurrent submissions.

## Verification approach once implemented

- `go run ./cmd/qrsurvey` + seed a survey/contest/poster directly via SQL, then manually walk the flow in a browser at each phase's checkpoint (listed above).
- `go test ./...` for the DB/handler suites in section "Testing priorities."
- End of Phase 3: physically scan a printed/displayed QR code with a phone camera to confirm the whole chain (QR → HTTPS via Caddy → poster resolution → survey render) works outside of localhost.
- End of Phase 5: confirm `systemctl restart qrsurvey` and a Caddy reload both leave the site reachable, and that a `.backup` snapshot restores correctly to a scratch copy.
