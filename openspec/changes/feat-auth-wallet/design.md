# Design

## Context

See `proposal.md` for why this app exists. The specs in `specs/user-registration`, `specs/jwt-session`, and `specs/wallet` are the behavior contract.

The finished program is a Go module, `btechdevcases`, under `api/`, and a Flutter project under `app/`. Docker Compose runs the API and MySQL 8. The Flutter app runs on an Android device or emulator and calls the API over HTTP. The service reaches MySQL only through sqlc. Checked-in SQL is both the sqlc input and the script Compose applies the first time the MySQL volume is created.

People use this app on links that fail and time out. A retried transfer must not move money twice, and a call that fails or times out must not count as activity. The phone must be able to end an idle session without waiting for the API.

## Goals / Non-Goals

**Goals:**

- One Go process serves registration, login, the protected welcome route, and wallet reads and transfers.
- One Flutter app registers, logs in, shows the welcome sentence, shows the wallet, and sends a transfer.
- The idle clock and the JWT expiry are different mechanisms. Tests can move the server clock. The Flutter session clock can be moved without a network call.
- A transfer's balance check, debit, and credit commit or roll back together.
- Compose starts MySQL, waits until it is healthy, then starts the API.

**Non-Goals:**

- Postgres, a TypeScript or C# backend, a React or React Native frontend, and a Node, npm, pnpm, yarn, or Bun toolchain.
- OAuth, MFA, email verification, password reset, profile editing, and roles.
- Deposits, withdrawals, other currencies, and payment gateways.
- An offline queue that applies transfers later. The caller retries the same transfer id. The server does not store a request to run after the link returns.
- A browser client. The API does not read a browser origin and does not emit CORS headers.

## Decisions

### Module layout

The API is created under these paths:

- `api/cmd/api/main.go` loads config, opens MySQL, and serves HTTP.
- `api/internal/httpapi` owns routes, request deadlines, and the idle check.
- `api/internal/user` owns field validation, the password hash, registration, and login lookup.
- `api/internal/token` owns signing and parsing.
- `api/internal/wallet` owns balance reads and transfers.
- `api/db/schema.sql` and `api/db/query.sql` are the checked-in SQL. `api/sqlc.yaml` generates `api/internal/db`, and that package is committed.
- `api/Dockerfile` builds the API. `compose.yaml` runs MySQL and the API. `.env.example` lists the Compose and API variables. `readme.md` documents Compose, the Android run path, and every variable the API and the Flutter app read.

The Flutter app is created under these paths:

- `app/lib/config.dart` reads the API base URL.
- `app/lib/session.dart` stores the token, the email, and the last successful activity time.
- `app/lib/money.dart` converts minor units and two-decimal text.
- `app/lib/transfer_draft.dart` keeps one transfer id for one payload and collects every local validation problem.
- `app/lib/api_client.dart` calls the API with a finite timeout. A `4xx` JSON `error` string is kept. A `5xx`, a timeout, or a dropped connection becomes `Request failed`.
- `app/lib/error_alert.dart` shows that message in the theme error container.
- `app/lib/transfer_history.dart` shows one history row per transfer, including its notes.
- `app/lib/welcome.dart` shows the welcome sentence.
- `app/lib/main.dart` builds the register, login, and signed-in screens.
- `app/test/welcome_test.dart` covers the welcome sentence.
- `app/test/session_test.dart` covers local idle logout.
- `app/test/wallet_client_test.dart` covers decimal display, transfer-id retry, and which failures keep the API error text.
- `app/test/wallet_screen_test.dart` covers the history rows and the error alert.

Handlers call the user and wallet packages. Those packages call sqlc. Handlers do not embed SQL.

Alternative considered: a second service for wallets. One process is enough, and a transfer must share the registration database transaction model.

### HTTP API

| Method and path | Auth | Success |
| --- | --- | --- |
| `POST /register` | no | `201` JSON `{ "id", "email" }`. The password is not in the body. |
| `POST /login` | no | `200` JSON `{ "token" }`. |
| `GET /me` | bearer | `200` `text/plain` body exactly `Hello <email>, welcome back`, with no extra characters. |
| `GET /wallet` | bearer | `200` JSON `{ "balance", "transfers" }`. |
| `POST /transfers` | bearer | `200` JSON of the transfer. |

Error bodies are JSON `{ "error" }`. Validation failures are `400`. An unknown recipient is `404`. A duplicate email, insufficient funds, and a reused transfer id with a different payload are `409`. Wrong credentials, a missing or invalid token, and an idle session are `401`.

The `error` string is the one the Flutter alert shows for a `4xx` response:

| Failure | `error` |
| --- | --- |
| Missing email, password, or confirmPassword | `email is required`, `password is required`, or `confirmPassword is required` |
| Invalid or empty email | `email is not a valid address` |
| Empty password | `password must be non-empty` |
| Confirmation differs | `confirmPassword does not match password` |
| Duplicate email | `email is already registered` |
| Wrong password | `invalid credentials` |
| Missing, invalid, or idle token | `unauthorized` |
| Unknown recipient | `recipient not found` |
| Self-transfer | `cannot transfer to self` |
| Zero, negative, or non-integer amount | `amount must be a positive integer` or `amount must be an integer` |
| Notes longer than 200 characters | `notes must be at most 200 characters` |
| Amount above the balance | `insufficient funds` |
| Same transfer id with a different payload | `transfer id reused with a different payload` |

A `5xx` body, including `504` `timeout`, is not shown as that text. The Flutter client shows `Request failed` for it.

`GET /wallet` returns `balance` as an integer number of minor units and `transfers` as that user's sent and received rows, oldest first. Each row has `transferId`, `direction` (`sent` or `received`), `counterparty` (the other account's email), `amount`, and `notes`.

`POST /transfers` accepts `{ "transferId", "recipient", "amount", "notes" }`. `notes` may be omitted or `""`. `amount` must be a JSON integer. `transferId` is a non-empty string of at most 64 characters, chosen by the caller.

Registration JSON uses `email`, `password`, and `confirmPassword`. A field that is absent or null is missing. An empty string is present. Email is accepted only when it parses as a bare address, with no display name, and it is stored exactly as submitted. Comparison is case-sensitive.

### Flutter screens

The app has three screens.

- Register collects `email`, `password`, and `confirmPassword` and posts them to `POST /register`. Success goes to the login screen. A `400` or `409` shows the JSON `error` string in the error alert and does not sign the user in.
- Login collects `email` and `password` and posts them to `POST /login`. Success stores the token, the submitted email, and the current time, then opens the signed-in screen. A `401` shows `invalid credentials` in the error alert and stays on login.
- The signed-in screen shows `Hello <email>, welcome back` from the stored email, then reads `GET /wallet`. It shows the balance with two decimal places. After that read succeeds, `TransactionHistory` shows a History heading and one `TransactionTile` per transfer. The tile title is `Received` or `Sent` plus the two-decimal amount. The same tile shows the counterparty email and, when notes are not empty, the notes. An empty list shows `No transfers yet`. A read that has not succeeded does not show that empty sentence. A form collects recipient, amount, and notes. Amount is typed as decimal text and sent as minor units. Logout clears the session and returns to login.

The signed-in screen is a lifecycle observer. On resume it checks the stored activity time locally. At 15 minutes or more it clears the session and shows login, and it does not call the API to make that decision. Before 15 minutes it stays on the screen and still shows the welcome sentence.

A failed or timed-out call shows the failure in the error alert and does not update the stored activity time. The last successfully read balance stays on screen. A `4xx` JSON `error` string is the alert text. A timeout, a `5xx`, a body without that string, and a dropped connection use `Request failed`.

### Passwords and JWT

Passwords are stored as a bcrypt hash and never returned. Login compares the submitted password with that hash.

The token is an HS256 JWT. The payload `sub` is the user id, and `email` is the account email. `iat` and `exp` come from the server clock. **The lifetime is 24 hours** (`exp = iat + 24h`), which is longer than 15 minutes.

The 24-hour expiry is not the idle logout. A token that is still inside 24 hours is rejected when the idle clock says so. A token older than 24 hours is rejected even if the user has been active, because each login issues one token and activity does not mint a new one. `GET /me` returns only the welcome sentence, so it cannot also return a refreshed token. The Flutter screen builds the same sentence from the email stored at login, so it does not need a new token to keep showing it.

Alternative considered: a 15-minute JWT expiry. That would log the user out on the token lifetime and would make a successful call unable to extend the session without a new token. Alternative considered: no expiry. A stolen token that the attacker keeps using would then work until the signing secret changed.

The signing secret is `JWT_SECRET`.

### Idle clock

`users.last_activity_at` is the server idle clock. It is set to the server time on a successful login. It moves to the server time of a later authenticated request only when that request finishes as a `2xx` and the request context is still active. Wallet reads, successful transfers, and idempotent replays are `2xx`, so they move the clock.

A `4xx` or `5xx` does not update the column. If the request deadline fires first, the handler returns `504`, the transfer transaction rolls back if it had started, and the column stays where it was.

The check runs before the protected handler. When `now` is at least 15 minutes after `last_activity_at`, the handler is not called, the response is `401`, and the column is not updated. A following request with the same token stays rejected until a new login. A null `last_activity_at` is treated as idle.

The token package and the idle check share one `Clock`. Production uses the system clock. Tests pass a clock they can set, so they can prove a call at 14 minutes succeeds, a call at 15 minutes is rejected, and a rejection does not make a call one minute later succeed.

The Flutter session stores its own time in memory for the life of the process. Login success sets it. Each later successful `GET /me`, `GET /wallet`, or `POST /transfers` moves it. Failures and timeouts do not. The resume check uses that stored time and a clock the session tests can replace. Clearing the session drops the token, the email, and the stored time, and the widget tree leaves the signed-in screen.

### Money

There is one currency. Amounts are `BIGINT` minor units. 100 minor units are 1.00. The README states that. A new wallet starts at `100000`. The Flutter screen formats a balance with two fraction digits and parses a typed amount that has at most two decimal places into minor units. More than two decimal places, or a non-positive amount, is rejected on the device before the request is sent.

Tables:

- `users`: `id` `CHAR(36)`, `email` `VARCHAR(255)` with `utf8mb4_bin` and a unique key, `password_hash`, `last_activity_at` `DATETIME(6)` null.
- `wallets`: `user_id` primary key, `balance_minor` `BIGINT`, foreign key to `users`.
- `transfers`: primary key `(sender_id, id)`, plus `recipient_id`, `amount_minor`, `notes` `VARCHAR(200)`, `created_at`. `id` is the caller transfer id. The same id from another sender is a different transfer.

Registration inserts the user and the wallet in one database transaction. A duplicate email hits the unique key, the transaction rolls back, and the API returns `409`.

A transfer locks the two wallets with `SELECT ... FOR UPDATE` in user-id order, so two transfers cannot overdraw the same wallet and the locks cannot deadlock. Inside that transaction the server:

1. Loads any row with this sender and transfer id using `SELECT ... FOR UPDATE` on `transfers` only. A locking read sees a row committed while this transaction waited on the wallet locks. A plain `SELECT` would keep the earlier snapshot and miss that row.
2. If the row exists and the recipient, amount, and notes match, returns that row and does not change balances.
3. If the row exists and any of those differ, rolls back and returns `409`.
4. If the recipient email is unknown, rolls back and returns `404`. If it is the sender, the API returns `400` before taking money locks.
5. If the amount is greater than the locked sender balance, rolls back and returns `409`.
6. Otherwise subtracts the amount from the sender, adds it to the recipient, and inserts the transfer. If the insert hits the primary key, the transaction rolls back those balance changes, locking-reads the stored row, and returns it when the recipient, amount, and notes match. A different payload still returns `409`.

Zero, negative, and non-integer amounts, missing ids, and notes longer than 200 characters are rejected with `400` before the transaction. An amount equal to the balance is allowed.

### Transfer retry on the device

The signed-in screen keeps a transfer draft: a transfer id plus the recipient, amount text, and notes currently in the form. The id is created when the draft is created. Editing recipient, amount, or notes replaces the id, because those edits are a different payload. Before the request, the draft collects every local problem: an empty recipient, an amount that is not a positive number with at most two decimal places, a non-positive amount, and notes longer than 200 characters. The alert shows those messages together, and the request is not sent. A failed or timed-out submit leaves the draft, including the id, in place, and the screen shows the failure. The retry action submits that same draft again. Success clears the draft so the next transfer gets a new id, refreshes the wallet, and moves the activity time.

### Request deadlines

The HTTP server sets `ReadHeaderTimeout` to 5 seconds, `ReadTimeout` and `WriteTimeout` to 10 seconds, and connection `IdleTimeout` to 60 seconds. That connection idle timeout is not the 15-minute session clock.

Each request context has an 8-second deadline. sqlc calls use that context. When it expires, the API responds `504` and does not move `last_activity_at`.

The Flutter HTTP client times out after 12 seconds, which is long enough to receive that `504`. On timeout, on a status of 500 or higher, and on a dropped connection, the screen shows `Request failed` and does not move its activity time. On a `400`, `401`, `404`, or `409` with a non-empty JSON `error` string, the screen shows that string in the same alert.

### Environment variables

Compose reads `.env` and passes variables into the API and MySQL. The Flutter app is not a Compose service. It receives the API base URL from `--dart-define=API_BASE_URL=...`. When that define is absent, the app uses `http://10.0.2.2:8080`, which is how the Android emulator reaches the host's published port 8080. A physical device uses the computer's LAN address instead, and the README says so.

The Android manifest allows cleartext HTTP and declares the internet permission so that documented URL works.

| Variable | Consumer | Rule |
| --- | --- | --- |
| `JWT_SECRET` | API | Required. The process exits when it is empty. |
| `DATABASE_DSN` | API | Required. The process exits when it is empty. Inside Compose the host is `mysql`, and the DSN includes `parseTime=true`. |
| `API_PORT` | API | Default `8080`. |
| `API_BASE_URL` | Flutter | `--dart-define`. Default `http://10.0.2.2:8080`. |
| `MYSQL_ROOT_PASSWORD` | MySQL | Required root password. |
| `MYSQL_DATABASE` | MySQL | Required database name, created on first init. |
| `MYSQL_USER` | MySQL | Required application user, created on first init. |
| `MYSQL_PASSWORD` | MySQL | Required password for `MYSQL_USER`. |

`DATABASE_DSN` uses the same user, password, and database as `MYSQL_USER`, `MYSQL_PASSWORD`, and `MYSQL_DATABASE`.

The API publishes `8080`. The API `depends_on` MySQL with `condition: service_healthy`. The healthcheck runs `mysqladmin ping`. The data volume is named `mysql-data`. Compose mounts `api/db/schema.sql` at `/docker-entrypoint-initdb.d/schema.sql`. MySQL uses image `mysql:8`.

### Tests

Go tests cover the spec scenarios that the API owns. A manual clock drives the JWT and the idle check. HTTP tests cover registration validation, login claims, the exact welcome bytes, a session that survives activity at 14 minutes, idle rejection at 15 minutes, a second rejection one minute later, a failed call that does not extend the window, and a canceled request that does not extend the window. Wallet HTTP tests cover both balances changing once, empty notes, insufficient funds, an unknown recipient, a self-transfer, a non-positive amount, notes of 201 characters, an idempotent retry, and the same id with a different amount.

Duplicate-email rejection, the opening balance, and a transfer that changes two stored balances run against MySQL 8 started with `api/db/schema.sql`, so the unique key and the transfer transaction are the ones Compose applies. One of those tests starts the second submission of a transfer id before the first transaction commits, and both calls return the original transfer with the balances changed once.

Flutter tests cover the screen and the client rules. `app/test/welcome_test.dart` pumps the welcome widget for `ada@example.com` and expects the exact sentence. `app/test/session_test.dart` uses a controllable clock to show that resume at 14 minutes keeps the session, resume at 15 minutes clears it without a network call, and a failed call does not move the stored time. `app/test/wallet_client_test.dart` shows that 100000 minor units format as `1000.00`, that a failed submit keeps the transfer id while an edit of the amount replaces it, and that a `4xx` `error` string is kept while a `5xx` or a dropped connection becomes `Request failed`. `app/test/wallet_screen_test.dart` shows the history rows with their notes, an empty history, every local validation message in one alert, a self-transfer's API error, an unknown recipient, insufficient funds, a timeout as `Request failed`, and the login and register alerts.

## Risks / Trade-offs

- [A stolen token that the attacker keeps using refreshes the idle clock] → The token still dies 24 hours after issue, and an unused stolen token dies after 15 idle minutes. Rotating tokens would change the welcome body, which the spec fixes as one sentence.
- [MySQL applies `schema.sql` only when the data volume is first created] → The README tells the operator to remove `mysql-data` when the schema must be applied again.
- [Case-sensitive email rejects `Ada@example.com` as a login for `ada@example.com`] → This matches the spec. The README states that email match is exact.
- [An activity update that fails after a committed transfer leaves the money moved but the clock old] → The transfer response is already success. The next call may be idle sooner. The money transaction is not reopened to fix the clock.
- [The 8-second deadline is shorter than a very slow query on a bad network] → The caller retries with the same transfer id. A timeout rolls back and does not double-spend.
- [The Flutter activity time lives in memory] → Process death returns the user to login. Resume of a living process still ends the session locally when the API is unreachable.
- [Cleartext HTTP on Android] → The documented lab URL is `http`, not TLS. The manifest allows cleartext so the emulator can reach Compose. A production host would terminate TLS in front of the same API.

## Migration Plan

This is the whole schema, not a patch on an older one. The first Compose start with an empty `mysql-data` volume creates `users`, `wallets`, and `transfers`. Rollback is `docker compose down`. Recreate the volume to drop the data and reapply `schema.sql`. No production data migration is in scope.

Apply writes this design into `api/`, `app/`, `compose.yaml`, `.env.example`, and `readme.md`, replacing files that disagree with it. `openspec/` stays as the plan.
