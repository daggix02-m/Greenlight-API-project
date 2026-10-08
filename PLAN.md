# Plan: Book-Faithful Greenlight API — *Let's Go Further* (Alex Edwards)

## 0. Current state & ground rules

- Project dir is empty (only `.idea/`) → from-scratch build, chapter by chapter.
- **Source of truth**: `Alex Edwards - Let's Go Further` PDF (923 pp.), extracted to
  `/tmp/opencode/lgtf.txt` (re-extractable with `pdftotext -layout`). Every stage begins by
  reading the corresponding chapter section from this text — not memory.
- **Loop per stage**: read section → extract requirements → implement → `go fmt` → build →
  run/test → verify against book → next stage.
- **Zero comments** in source (only syntactically required directives like `//go:embed`).
  No TODOs, no placeholders, no architectural substitutions (httprouter, database/sql + lib/pq,
  application struct, book's middleware/model patterns).

## 1. Environment provisioning (before Chapter 2)

1. **PostgreSQL** — Docker `postgres:18` container on `localhost:5432` with the book's dev
   DSN: `postgres://greenlight:pa55word@localhost/greenlight` (role `greenlight`, db `greenlight`).
2. **SMTP sink** — temporary Mailpit (Docker, SMTP `:1025`, UI `:8025`) *outside the repo*;
   API run with `-smtp-host localhost -smtp-port 1025` so welcome/activation/reset mail is
   actually deliverable and inspectable.
3. **Tooling**: `go install honnef.co/go/tools/cmd/staticcheck@latest` (§20.3);
   golang-migrate CLI (`migrate`) for `make db/migrations/new|up` (§6.2).
4. **Git**: `git init` + commits at stage boundaries so `git describe --always --dirty`
   version burn-in works (§20.6).
5. **Env**: `.envrc` (book §20.2) with `GREENLIGHT_DB_DSN` and `JWT_SECRET` (dev-only);
   root `.gitignore` containing `.envrc`. Makefile does `include .envrc`.

## 2. Fixed decisions (cross-checked against the book)

| Item | Value |
|---|---|
| Module path | `greenlight.alexedwards.net` (book's; all internal imports use it) |
| go directive | `go 1.16` as printed in the book's `go.mod`; fallback to toolchain default only if modern deps/staticcheck hard-fail it |
| Deps (pinned where book pins) | `httprouter@v1.3.0`, `lib/pq@v1.10.0`, `go-mail/mail/v2@v2.3.0`, `httpsnoop@v1.0.1`, `pascaldekloe/jwt@v1.10.0`, `tomasen/realip@latest`, `golang.org/x/crypto/bcrypt@latest`, `golang.org/x/time/rate@latest`, transitive `gopkg.in/alexcesaro/quotedprintable.v3` |
| Migrations | exactly `000001_create_movies_table` … `000006_add_permissions` (book explicitly deletes the demo `000007`) |
| Final auth | book's progression: stateful tokens (Ch.16) then JWT appendix (§22.3) replaces issue+verify → final behavior = `Authorization: Bearer <jwt>`, HS256, issuer/audience `greenlight.alexedwards.net`, 24 h expiry |
| SMTP creds | flag structure identical to book; username/password defaults empty (no secrets in source) |
| Includes | `cmd/examples/cors/{simple,preflight}` (in book's final `make audit` output), `.envrc`, `.gitignore` |

## 3. Build stages (book order)

**Stage 1 — Ch.2.1 Skeleton**: `bin/`, `cmd/api/main.go`, `internal/`, `migrations/`,
`remote/`, `go.mod`, `Makefile`; `git init` + commit.

**Stage 2 — Ch.2.2 HTTP server**: `config` struct + flags (`-port`, `-env`),
`application` struct, `http.Server` with read/write/idle timeouts + error log,
`/v1/healthcheck`.

**Stage 3 — Ch.2.3 Routing**: `httprouter`, `cmd/api/routes.go`, `/v1/` versioning.

**Stage 4 — Ch.3-4 JSON**: `envelope`, `writeJSON`, error responses; `readJSON`
(max 1 MB, `DisallowUnknownFields`, malformed/unknown-field/trailing-data/empty-body errors);
`internal/validator/validator.go` (`New()`, `Valid()`, `AddError()`, `Check()`).

**Stage 5 — Ch.5 PostgreSQL**: `open()` with DSN, `PingContext`, pool settings
(`MaxOpenConns` 25, `MaxIdleConns` 25, `ConnMaxIdleTime` 15m) via
`-db-max-open-conns/-db-max-idle-conns/-db-max-idle-time`; `data.Models` in `application`.

**Stage 6 — Ch.6 Migrations**: golang-migrate workflow; pairs 000001–000002;
Makefile `db/migrations/new`, `db/migrations/up` (+ `confirm`, `help`).

**Stage 7 — Ch.7 CRUD**: `internal/data/movie.go` (Create/Get/Update/Delete),
`ErrRecordNotFound`, `cmd/api/movies.go`, `readIDParam`, error helpers in `errors.go`,
router NotFound/MethodNotAllowed handlers.

**Stage 8 — Ch.8 Advanced CRUD**: PATCH partial updates, optimistic concurrency
(`version` → 409), context query timeouts.

**Stage 9 — Ch.9 Filtering/sorting/pagination**: `internal/data/filters.go`,
`GetAll` with title/genres/full-text search, sorting whitelist, `LIMIT/OFFSET`,
`count(*) OVER()` pagination metadata, query-string validation.

**Stage 10 — Ch.10 Logging & errors**: `internal/jsonlog/jsonlog.go`; logger as app
dependency; `recoverPanic`; centralized error helpers.

**Stage 11 — Ch.11 Rate limiting**: global limiter → per-IP map with `x/time/rate`,
background cleanup, `-limiter-enabled/-limiter-rps/-limiter-burst`, 429 + `Retry-After`.

**Stage 12 — Ch.12 Graceful shutdown**: `cmd/api/server.go` — `signal.NotifyContext`
(SIGINT/SIGTERM), 5 s `srv.Shutdown(ctx)`, error channel, shutdown logging.

**Stage 13 — Ch.13 Users**: migration `000004`, `internal/data/user.go` + `users.go`
(bcrypt, versioned Update), validators, `POST /v1/users` (duplicate email → 422).

**Stage 14 — Ch.14 Email**: `internal/mailer/mailer.go` + embedded templates
(`user_welcome`, `token_activation`, `token_password_reset`), `go-mail/mail/v2`,
SMTP flags, `app.background(fn)`; WaitGroup coordination in shutdown (§14.5).

**Stage 15 — Ch.15 Activation**: migration `000005`, `internal/data/tokens.go`
(SHA-256 hashes, 45 min expiry), `PUT /v1/users/activated`.

**Stage 16 — Ch.16 Authentication**: `POST /v1/tokens/authentication`,
`cmd/api/context.go`, `authenticate()` middleware with anonymous user — stateful first.

**Stage 17 — Ch.17 Authorization**: migration `000006` (`movies:read`/`movies:write`),
`internal/data/permissions.go`, `requirePermission → requireActivatedUser →
requireAuthenticatedUser`, 403 responses, movie routes wrapped.

**Stage 18 — Ch.18 CORS**: `enableCORS`, `-cors-trusted-origins` via `flag.Func`,
`Vary` headers, preflight; `cmd/examples/cors/{simple,preflight}`.

**Stage 19 — Ch.19 Metrics**: `expvar` + `GET /debug/vars`, custom metrics,
`httpsnoop` request/status-code instrumentation.

**Stage 20 — Ch.20 Build/quality**: full Makefile — `help`, `confirm`, `include .envrc`,
`run/api`, `db/psql`, `db/migrations/new|up`, `audit` (depends `vendor`), `vendor`,
`build/api` with `-ldflags='-s -X main.buildTime=… -X main.version=$(git describe …)'`
→ `./bin/api` and `./bin/linux_amd64/api`; `-version` flag; `go mod vendor`.

**Stage 21 — Ch.21 Deployment**: `remote/setup/01.sh`, `remote/production/api.service`
(systemd), `remote/production/Caddyfile` (reverse_proxy, `/debug/*` 403, HTTPS),
Makefile PRODUCTION section (`production/connect`, `production/deploy/api`,
`production/configure/api.service`, `production/configure/caddyfile`);
`tomasen/realip` in rate limiter for Caddy's `X-Forwarded-For`.

**Stage 22 — Appendices (final state)**:
- §22.1 `POST /v1/tokens/password-reset`, `PUT /v1/users/password`
- §22.2 `POST /v1/tokens/activation`
- §22.3 JWT: `-jwt-secret`, `claims.HMACSign(jwt.HS256, …)`, `authenticate()` verifies
  signature/time/issuer/audience/subject
- §22.4/22.5 JSON encoding/decoding nuances
- §22.6 request context timeouts

**Stage 23 — Final QA**:

```
go fmt ./... && go vet ./... && go test -race -vet=off ./... && staticcheck ./...
go mod tidy && go mod verify && go mod vendor
make audit && make build/api
```

Behavioral sweep: healthcheck; movie CRUD (version conflict → 409); list filter/search/sort/
pagination; user registration → background welcome mail in SMTP sink; activation; JWT auth;
anonymous vs authenticated vs inactive vs missing-permission (403); rate limit 429 + disable
flag; CORS trusted/untrusted + preflight; `/debug/vars`; SIGTERM graceful shutdown with
in-flight background mail completed; zero `TODO`/placeholder/tutorial comments in the tree.

## 4. Risks / fallbacks

- **`go 1.16` directive vs modern x/crypto+staticcheck**: attempt as printed; if rejected,
  raise minimally and document.
- **golang-migrate CLI**: verify postgres support; fallback to official release binary.
- **SMTP sink**: test infra only — never committed to the repo.
- **Deployment**: files + Makefile rules implemented; actual DigitalOcean provisioning is
  out of scope (no server) — systemd/Caddy artifacts verified by inspection.
