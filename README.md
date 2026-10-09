# Greenlight API

[![CI](https://github.com/daggix02-m/greenlight-let-s-go-further-/actions/workflows/ci.yml/badge.svg)](https://github.com/daggix02-m/greenlight-let-s-go-further-/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/daggix02-m/greenlight-let-s-go-further-)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A production-shaped JSON REST API for a movie-rating application, written in Go
with PostgreSQL. It is a faithful reproduction of the **Greenlight** project
from [Let's Go / Let's Go Further](https://lets-go.alexedwards.net) by Alex
Edwards, built directly from the book with no architectural substitutions —
standard library HTTP routing, hand-rolled middleware, `database/sql`, and
no comments in the production source.

## Highlights

- **Authentication** — JWT bearer tokens, email-based account activation and
  password-reset flows, bcrypt password hashing with constant-time comparison.
- **Authorization** — per-permission checks (`movies:read`, `movies:write`)
  wired through middleware; every response carries the correct 401/403 status.
- **Data layer** — movies, users, tokens, and permissions with input
  validation, pagination (`page`/`page_size`), allow-listed sorting, and a
  stable response envelope including metadata.
- **Operational basics** — structured leveled JSON logging, panic recovery,
  expvar metrics, per-IP token-bucket rate limiting, CORS with trusted
  origins, database connection pooling, and graceful shutdown.
- **Docker-first deployment** — multi-stage build to a distroless non-root
  image, plus a Compose stack that boots Postgres 18, runs migrations, and
  starts the API with Mailpit as a local SMTP sink.
- **Quality gates** — `gofmt`, `go vet`, `go build`, and `go test -race` in CI
  and locally via `make audit`.

## Quick start (Docker)

```bash
git clone https://github.com/daggix02-m/greenlight-let-s-go-further-.git
cd greenlight-let-s-go-further-
docker compose up --build
```

The stack starts, in order: Postgres 18 (with the `citext` extension created
on first boot) → `golang-migrate` (applies all migrations, gated on the DB
healthcheck) → the API on port **4000**. Mailpit listens on ports 1025 (SMTP)
and 8025 (web UI at <http://localhost:8025>), so every welcome/activation
email lands in a browsable inbox.

Verify it:

```bash
curl -i http://localhost:4000/v1/healthcheck
```

```json
{
  "status": "available",
  "system_info": {
    "environment": "development",
    "version": "dev"
  }
}
```

Tear down with `docker compose down` (add `-v` to also drop the database
volume).

## Run natively

Prerequisites: Go 1.26+, a PostgreSQL instance with the `citext` extension,
[golang-migrate](https://github.com/golang-migrate/migrate), and optionally
Mailpit for local email delivery.

```bash
cp .env.example .envrc   # then edit DSN / JWT secret for your machine
make db/migrations/up    # apply migrations
make run/api             # start the API on :4000 (or: go run ./cmd/api)
```

`.envrc` and `.env` are gitignored — never commit real secrets. Both files are
loaded automatically at startup when present in the working directory, so a
bare `go run ./cmd/api` picks them up too (variables already in the
environment are never overwritten). Useful
targets: `make help` lists everything (`build/api`, `run/lan`, `db/psql`,
`audit`, …).

## A complete request flow

```bash
# 1. Register (a welcome email with an activation token appears in Mailpit)
curl -i -X POST http://localhost:4000/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "email": "alice@example.com", "password": "pa55word123"}'

# 2. Activate using the token from the email
curl -i -X PUT http://localhost:4000/v1/users/activated \
  -H 'Content-Type: application/json' \
  -d '{"token": "VGF54QR5FDJOPJVYLOYZZHP5GQ"}'

# 3. Exchange credentials for a JWT
curl -i -X POST http://localhost:4000/v1/tokens/authentication \
  -H 'Content-Type: application/json' \
  -d '{"email": "alice@example.com", "password": "pa55word123"}'

# 4. Call an endpoint that requires an activated account
curl -i http://localhost:4000/v1/movies \
  -H "Authorization: Bearer ${TOKEN}"
```

## API endpoints

| Method | Path                  | Access                                            |
| ------ | --------------------- | ------------------------------------------------- |
| GET    | `/v1/healthcheck`     | Public                                            |
| GET    | `/v1/movies`          | Bearer JWT · activated · `movies:read`            |
| POST   | `/v1/movies`          | Bearer JWT · activated · `movies:write`           |
| GET    | `/v1/movies/:id`      | Bearer JWT · activated · `movies:read`            |
| PATCH  | `/v1/movies/:id`      | Bearer JWT · activated · `movies:write`           |
| DELETE | `/v1/movies/:id`      | Bearer JWT · activated · `movies:write`           |
| POST   | `/v1/users`           | Public (register)                                 |
| PUT    | `/v1/users/activated` | Public (activation token in body)                 |
| PUT    | `/v1/users/password`   | Public (password-reset token in body)             |
| POST   | `/v1/tokens/authentication` | Public (email + password → JWT)             |
| POST   | `/v1/tokens/activation`     | Public (resend activation token)            |
| POST   | `/v1/tokens/password-reset` | Public (request password-reset token)       |
| GET    | `/debug/vars`         | Non-production only (Go `expvar` diagnostics)      |

Activated accounts are granted `movies:read`. `movies:write` (create, update,
delete) can be granted by inserting into the `users_permissions` table.

Errors always return a JSON envelope — for example a validation failure:

```json
{
  "error": {
    "email": "must be a valid email address",
    "password": "must be at least 8 bytes long"
  }
}
```

## Configuration

Settings come from command-line flags; two have environment-variable
defaults. A `.env` or `.envrc` file in the working directory is loaded at
startup before flag defaults are read — existing environment variables and
flags always take precedence. See `.env.example` for a ready-made template.

| Env variable         | Flag           | Default        | Purpose                                |
| -------------------- | -------------- | -------------- | -------------------------------------- |
| `GREENLIGHT_DB_DSN`  | `-db-dsn`      | —              | PostgreSQL connection string           |
| `JWT_SECRET`         | `-jwt-secret`  | —              | Secret used to sign/verify JWTs        |
| —                    | `-port`        | `4000`         | Listen port                            |
| —                    | `-env`         | `development`  | `development` \| `staging` \| `production` |
| —                    | `-smtp-host`   | `smtp.mailtrap.io` | SMTP relay host                    |
| —                    | `-smtp-port`   | `25`           | SMTP relay port                        |
| —                    | `-smtp-sender` | `Greenlight <no-reply@example.com>` | From header          |
| —                    | `-limiter-rps` / `-limiter-burst` / `-limiter-enabled` | `2` / `4` / `true` | Per-IP rate limit  |
| —                    | `-cors-trusted-origins` | —         | Space-separated allowed origins        |
| —                    | `-trusted-proxies`  | —         | Proxy IPs/CIDRs allowed to set `X-Forwarded-For` (rate limiting falls back to the socket address otherwise) |
| —                    | `-db-max-open-conns` / `-db-max-idle-conns` / `-db-max-idle-time` | `25` / `25` / `15m` | Pool tuning |

In production you must set `JWT_SECRET` and point `-smtp-*` at a real relay.
The JWT secret is validated at startup: it must always be set, and when
`-env=production` it must be at least 32 bytes and must not be the bundled
demo value — the API refuses to boot otherwise. `/debug/vars` is only routed
outside production.

## Project structure

```
cmd/api/            entry point: main, routes, handlers, middleware, helpers
internal/data/      models — movies, users, tokens, permissions, filters
internal/jsonlog/   leveled structured JSON logger (io.Writer-compatible)
internal/mailer/    SMTP mailer (embedded templates, per-send connection, 5s dial timeout)
internal/validator/ reusable validation helpers
migrations/         numbered SQL migrations (golang-migrate)
docker/             Postgres initialization (citext extension)
Dockerfile          multi-stage build → distroless, non-root
docker-compose.yml  Postgres + migrate + Mailpit + API
```

## Testing

```bash
go test -race ./...   # or: make audit (also runs gofmt/vet/staticcheck)
```

The suite covers the validator primitives, runtime JSON codec, pagination
rules, user/movie/token validation, the structured logger, JSON request
helpers, client-IP resolution for the rate limiter, the full JWT rejection
matrix (header parsing, signature, claims, issuer/audience, expiry),
`requireActivatedUser` status codes, and handler behavior (healthcheck, 404,
405, expvar gating) through the full middleware chain.

## License

MIT — see [LICENSE](LICENSE). This project is an educational reproduction of
the Greenlight application from Alex Edwards' books
[Let's Go / Let's Go Further](https://lets-go.alexedwards.net); the book's
text and original material remain © Alex Edwards.
