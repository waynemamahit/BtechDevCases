# Design

## Context

The repo has the assignment brief and no application code. See proposal.md for why this change exists. Behavior is fixed by `user-registration` and `jwt-session`: field checks, duplicate email rejected by a MySQL unique key, a signed login token with email and a stable user id, the exact protected greeting, renewal on a successful authenticated request, rejection after 15 idle minutes, and registration, login, and welcome screens that use Tailwind CSS 4 and daisyUI 5 from a narrow phone width through desktop.

The browser talks to a Go API. React (TypeScript) is built, tested, formatted, and linted with Bun. Docker Compose runs the Bun frontend, the Go API, and MySQL 8. Users live in MySQL. The Go service talks to MySQL only through sqlc-generated code.

## Goals / Non-Goals

**Goals:**

- One Go process, one Bun process, and one MySQL 8 service that implement the two specs, with idle time enforced on the server and cleared on the screen.
- A Compose file and a root README that name every environment variable the processes read, including the database DSN and the MySQL credentials, and how to start the stack.
- Go tests that drive a fake clock, a MySQL proof of duplicate-email rejection, and frontend tests for the exact welcome sentence, one auth form at a time, an accepted registration returning to login without a token, the screen ending the session at token expiry, a `401` from `GET /me` clearing the stored session, and login, registration, and welcome staying on screen at 320×800 and at 1280×800.
- A frontend that passes Biome check.

**Non-Goals:**

- A TypeScript or C# backend, a Node npm pnpm or yarn frontend toolchain, ESLint, Prettier, OAuth, MFA, email verification, password reset, profile editing, roles, an ORM, Postgres or SQLite, and a CSS system other than Tailwind CSS with daisyUI.
- Sharing sessions across API replicas, or revoking a token before its expiry.

## Decisions

### Process layout

- `api/` is a Go module `btechdevcases`. `cmd/api` starts the server. `internal/user` validates registration and stores users through sqlc. `internal/db` is the committed sqlc package. `internal/token` signs and parses JWTs. `internal/httpapi` is the HTTP adapter. Tests sit next to those packages.
- `web/` is a React TypeScript app. Bun runs install, `dev`, `build`, `test`, `lint`, and `typecheck`. Vite is invoked only through Bun scripts. The Compose image is `oven/bun` and does not use npm, pnpm, or yarn.
- Repo root holds `compose.yaml`, `.env.example`, and `README.md`.

Alternative: a single Go binary that serves the React build. Rejected so the browser can call the API on its own origin configuration, which matches the split Compose services.

### Registration and MySQL store

- `POST /register` accepts JSON `{ "email", "password", "confirmPassword" }`.
- Validation order is the spec and nothing more: each field present, `email` accepted by `net/mail.ParseAddress` (this rejects `not-an-email`), `password` non-empty, `confirmPassword` equal to `password`. No trimming and no case folding.
- Success returns `201` and `{ "id", "email" }` only. `id` is a UUID created once and stored on the user. Failure to validate returns `400`. A duplicate email returns `409`. Malformed JSON returns `400` and `{ "error": "invalid JSON" }`. Other error bodies are `{ "error": "<message>" }`. Registration does not return a token.
- The password is a bcrypt hash before insert. The submitted password is never stored.
- The store inserts only through sqlc `CreateUser`. It does not look up the email first. MySQL error 1062 from unique key `users_email_unique` is the duplicate rejection, mapped to the duplicate-email error. `Authenticate` loads the row only through sqlc `GetUserByEmail` and compares the bcrypt hash.
- The `email` column uses `utf8mb4_bin`, so the unique key compares the exact stored string. `Ada@Example.com` and `ada@example.com` are different accounts.

### Checked-in SQL, sqlc, and Compose init

- `api/db/schema.sql` is the shared SQL file. sqlc reads it as the MySQL schema. Compose bind-mounts that same file to `/docker-entrypoint-initdb.d/schema.sql`, which MySQL applies when the data volume is first created.
- `api/db/query.sql` is the checked-in query SQL sqlc compiles. It defines `CreateUser` and `GetUserByEmail`.
- `api/sqlc.yaml` sets engine `mysql`, schema `db/schema.sql`, queries `db/query.sql`, and Go output package `db` at `api/internal/db`. That generated package is committed.
- Table `users` has `id` CHAR(36) primary key, `email` VARCHAR(255) `utf8mb4_bin` NOT NULL, `password_hash` VARCHAR(255) NOT NULL, and `UNIQUE KEY users_email_unique (email)`.

### Login token

- `POST /login` accepts `{ "email", "password" }`. Unknown email, a bcrypt mismatch, and any other authentication failure return `401` and `{ "error": "invalid credentials" }` with no token. Malformed JSON returns `400` and `{ "error": "invalid JSON" }`. Success returns `200` and `{ "token" }`.
- The token is an HMAC JWT (`github.com/golang-jwt/jwt/v5`) signed with `JWT_SECRET`. Claims are `sub` (the stored user id), `email`, `iat`, and `exp`.
- Sign truncates the issuer clock to a UTC second, then sets `iat` to that instant and `exp` to 15 minutes later. The API exits on startup when `JWT_SECRET` is empty.

### Server-enforced idle window

Idle is the JWT `exp` claim, not a row in MySQL.

- Login and every successful `GET /me` issue a token whose `exp` is 15 minutes after the issuer clock, once that clock is truncated to a UTC second.
- `GET /me` requires `Authorization: Bearer <token>`. The handler accepts the token only when the signature matches and `now < exp`. Equality with `exp` is expired. Success returns `200` and `{ "id", "email", "token" }` where `token` is the new credential. Missing, unsigned, or expired tokens return `401` and no new token.
- An active user stays signed in by storing the `token` from the latest successful login or `GET /me` and sending that token on the next request. The credential returned by a request at T+14m is itself valid until T+29m. The credential it replaced still expires at its own `exp`.
- Failed calls do not issue a token, so they do not move the window.
- `internal/token.Issuer` takes a `Clock` interface with `Now() time.Time`. The HTTP handlers sign and parse through that issuer. Production uses the wall clock. Tests use a fake clock that can jump forward, which is how idle rejection and renewal are proved without waiting.

Alternative: store `last_activity` on the user and keep one long-lived token. Rejected because a single timestamp would treat every token for that user as one session, and the spec rejects the credential the last success returned once 15 minutes pass. Expiry on that credential does that without extra session state.

### How the screen hears about idle expiry

- The React app loads `GET /config.json` from its own origin. That JSON is `{ "apiBaseUrl" }` and is the only source for the API base URL.
- Login stores the returned token in `sessionStorage`. The screen then calls `GET /me` and renders the `email` from that response. The protected view is the sentence `Hello ${email}, welcome back` and nothing else in that text node. For `ada@example.com` the text is exactly `Hello ada@example.com, welcome back`.
- Each successful login or `GET /me` replaces the stored token and reads `exp` from the JWT payload (the client does not verify the signature). A timer fires at that `exp`. When it fires, the app deletes the stored token and leaves the protected screen, so the welcome sentence is gone.
- A `401` from `GET /me` clears the same storage immediately. The timer covers a user who sits on the screen. The `401` covers a clock difference or a token that is already expired on load.
- Registration does not store a token. After a `201`, the page returns to the login form and asks the user to log in.

### Screens

- `web/src/app.css` contains `@import "tailwindcss";` and `@plugin "daisyui";`. Vite loads Tailwind through `@tailwindcss/vite`. React and React DOM are the runtime dependencies. Tailwind CSS 4, daisyUI 5, `@tailwindcss/vite`, and Playwright are Bun devDependencies. Vite compiles Tailwind and daisyUI into the build. There is no `tailwind.config.js`.
- The unauthenticated page is one daisyUI `card`. Inside it, a `tabs tabs-box` tablist uses two radio inputs, labeled Log in and Register. React renders only the selected form, so the other form is not in the document. Each form is a `fieldset` with `input`, `label`, and a default `btn`. Notices use `alert`. The welcome screen is a `card` whose text node is the greeting. Tailwind utilities such as width and spacing fill gaps those components do not cover.
- The page is a `flex` column. The auth card sits in a one-column `grid` that grows from `max-w-sm` through `sm:max-w-md` to `md:max-w-lg`. The welcome card uses the same flex page and the same width steps. Those prefixes keep the controls and the welcome sentence usable from a narrow phone width through tablet and desktop. The page opens on Log in. Choosing Register replaces the login form. An accepted registration selects Log in again.
- `web/biome.json` is the checked-in Biome config. `web/package.json` script `lint` runs `biome check --write .` on the React TypeScript sources, including tests. `typecheck` runs `tsc --noEmit`. Biome covers the frontend only. `web/bunfig.toml` preloads `web/src/happydom.ts` for `bun test`.

### Environment variables and Compose

| Variable | Process | Required | Meaning |
| --- | --- | --- | --- |
| `JWT_SECRET` | Go API | yes, no default | HMAC secret. Empty value aborts startup. |
| `WEB_ORIGIN` | Go API | yes, no default | Browser origin allowed by CORS. |
| `API_PORT` | Go API | no, default `8080` | Listen port inside the API container. |
| `DATABASE_DSN` | Go API | yes, no default | MySQL DSN. Empty value aborts startup. |
| `MYSQL_ROOT_PASSWORD` | MySQL | yes, no default | Root password for the MySQL image. |
| `MYSQL_DATABASE` | MySQL | yes, no default | Database created on first init. |
| `MYSQL_USER` | MySQL | yes, no default | Application user created on first init. |
| `MYSQL_PASSWORD` | MySQL | yes, no default | Password for `MYSQL_USER`. |
| `API_BASE_URL` | Bun web | yes, no default | Browser-reachable API origin, no path suffix. Empty value aborts the web server. |
| `WEB_PORT` | Bun web | no, default `5173` | Port the Bun static server listens on. |

`compose.yaml` supplies those variables to the three services:

- `mysql` uses image `mysql:8`, named volume `mysql-data`, and a healthcheck. It receives `MYSQL_ROOT_PASSWORD`, `MYSQL_DATABASE`, `MYSQL_USER`, and `MYSQL_PASSWORD` from the Compose project env. It mounts `api/db/schema.sql` at `/docker-entrypoint-initdb.d/schema.sql`.
- `api` publishes `8080:8080`, starts only after `mysql` is healthy, and sets `API_PORT=8080`, `WEB_ORIGIN=http://localhost:5173`, `JWT_SECRET` from `${JWT_SECRET}`, and `DATABASE_DSN` from `${DATABASE_DSN}`.
- `web` publishes `5173:5173`, depends on `api`, and sets `API_BASE_URL=http://localhost:8080` and `WEB_PORT=5173`.

`.env.example` sets `JWT_SECRET`, `MYSQL_ROOT_PASSWORD`, `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD`, and `DATABASE_DSN`. The DSN user, password, database, and `tcp(mysql:3306)` host match the MySQL service. The README lists all ten variables, states which have defaults, and shows `docker compose up --build` after copying the example env file. CORS allows `WEB_ORIGIN` for `GET`, `POST`, and `OPTIONS`, with `Content-Type` and `Authorization` request headers.

The Bun server reads `API_BASE_URL` at process start and serves it on `/config.json`, so the image does not bake the URL in at build time. An empty `API_BASE_URL` aborts the process. A missing `WEB_PORT` listens on `5173`. A `WEB_PORT` that is not an integer from 1 through 65535 aborts the process. The API opens `DATABASE_DSN` with the MySQL driver and pings before it listens.

### Tests

- Go registration tests in `api/internal/user/register_test.go` cover each validation failure without MySQL, and accept a mixed-case address. HTTP tests in `api/internal/httpapi/register_test.go` use a test directory: validation failures return `400` and create no account, an accepted registration returns `201` and `{ "id", "email" }` with no token or password, and a duplicate email returns `409` while the original password remains.
- The duplicate-email proof in `api/internal/user/duplicate_mysql_test.go` starts MySQL 8 with testcontainers, applies `api/db/schema.sql`, and registers through the sqlc store. The second insert of the same email fails because of `users_email_unique`, the stored secret is a bcrypt hash of the first password, and a different-case email is a different account.
- Go token tests in `api/internal/token/token_test.go` use a fake clock and assert `sub`, `email`, `iat`, and `exp`, acceptance at 14 minutes, and rejection at exactly 15 minutes.
- Go login tests in `api/internal/httpapi/login_test.go` assert that the body is `{ "token" }` and that a second login for the same user reuses `sub`. An unknown email and a wrong password each return `401` and `{ "error": "invalid credentials" }` with no token.
- Go session tests in `api/internal/httpapi/session_test.go` use the fake clock: `GET /me` returns the identity for a fresh token, missing and foreign tokens are `401`, a request at +14 minutes succeeds and returns a token that still works +14 minutes later and fails at +15 minutes from that renewal, and the unrenewed token fails at +15 minutes from its own issue. That file also checks CORS for `GET`, `POST`, and `OPTIONS` with `Content-Type` and `Authorization`. `api/internal/httpapi/config_test.go` rejects an empty `JWT_SECRET`, `WEB_ORIGIN`, or `DATABASE_DSN`, and defaults `API_PORT` to `8080`.
- `web/src/welcome.test.tsx` renders the protected screen from identity email `ada@example.com` and expects the exact string `Hello ada@example.com, welcome back`. `web/src/auth.test.tsx` expects the login form without a confirm-password field, then selects Register and expects the registration form without the login submit control. It also expects an accepted registration to return to the login form with no token stored, the welcome sentence to disappear when the token `exp` is reached and the stored token to be removed, and a `401` from `GET /me` to remove the stored token and leave the protected screen. A login whose following `GET /me` returns `401` shows the login form, shows no welcome sentence, and stores no token. `web/src/server.test.ts` checks the `/config.json` body, a missing `API_BASE_URL`, process exit, and the `WEB_PORT` default. `web/src/layout.test.ts` runs `web/src/layout.browser.ts` in a child process so Playwright can measure layout. That browser test opens the login, registration, and welcome screens at 320×800 and at 1280×800. At 320px the controls and the welcome sentence stay inside the viewport and the page does not scroll horizontally, including a long welcome email. At 1280px those same controls and the welcome sentence stay usable, and the auth grid and welcome card use the `md:max-w-lg` width. Runner is `bun test`. Component tests use Testing Library and happy-dom. The layout test uses Playwright because happy-dom does not measure page scroll and replaces `setTimeout`. The web image sets `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD` so its install does not download browsers. `bun run lint` passes on those sources.

## Risks / Trade-offs

- [A replaced token still works until its own `exp`] → The client stores only the newest token. Short overlap is limited to the remaining life of the previous token, at most 15 minutes.
- [The screen timer and the API clock can disagree] → The timer clears the UI. `GET /me` remains the authority and a `401` clears the stored token.
- [JWT expiry assumes one API process] → Compose runs one `api` service. Horizontal scaling is out of scope.
- [Binary email collation treats case variants as different users] → This matches the decision not to trim or case-fold. The unique key still rejects the exact string.
- [MySQL init runs only when the data volume is first created] → Changing `schema.sql` after the volume exists does not alter the existing database. Recreate the volume to reapply it.

## Migration Plan

Greenfield. Add the `api` and `web` trees, `compose.yaml`, and `.env.example`, and replace the root README with the runbook described above. Deploy with `docker compose up --build`. Roll back with `docker compose down`. The named volume `mysql-data` holds accounts until it is removed.
