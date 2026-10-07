# Proposal

## Why

People using this app on a bad connection still need an account, a session that ends after they go quiet, and a wallet they can read and send from without a retry spending the money twice. The product is that API, built as one Go service with MySQL, ready to run from Docker Compose.

## What Changes

- A Go HTTP API for registration, login, a protected welcome route, and wallet reads and transfers.
- Registration accepts `email`, `password`, and `confirmPassword`, stores the password only as a one-way hash, and creates the user and a wallet in one database transaction. The wallet opens at 1000.00, stored as 100000 minor units. A duplicate email is rejected by a MySQL unique key.
- Login returns a signed JWT whose claims include the email and a stable user id. The token lifetime is longer than 15 minutes. Logout after 15 minutes of inactivity is a server-side idle clock, not the token expiry.
- The protected route returns exactly `Hello [email], welcome back`. The idle clock moves on a successful login and on each successful authenticated request, including wallet reads and transfers. A failed or timed-out call leaves the clock unchanged. The next authenticated request after 15 idle minutes is rejected, and that rejection does not move the clock.
- A signed-in user can read their balance and their own sent and received transfers, including notes. A transfer takes the recipient's email, a positive amount in minor units, notes, and a caller-generated transfer id. The balance check, debit, and credit commit in one transaction. Repeating the same transfer id with the same payload returns the original result and does not move money again.
- Docker Compose runs the Go API and MySQL 8. MySQL has a named volume and a healthcheck. The API starts only after MySQL is healthy. The same checked-in SQL file is what sqlc compiles and what Compose applies when the data volume is first created. The Go service talks to MySQL only through sqlc-generated code, and that generated package is committed.
- The README documents how to build and run with Compose, and every required environment variable, including the MySQL credentials and the database DSN. It states that money uses two decimal places, stored as integer minor units.
- Go tests prove registration validation, JWT claims, the exact welcome sentence, idle rejection with a fake clock, a transfer that changes both balances once, insufficient funds, an unknown recipient, a self-transfer, and an idempotent retry. Duplicate-email rejection is proven against MySQL.

No frontend is in scope.

### Out of scope

- A frontend of any kind, including Flutter, web, React, and React Native.
- A TypeScript or C# backend, and a Node, npm, pnpm, yarn, or Bun toolchain.
- OAuth, MFA, email verification, password reset, profile editing, and roles.
- Deposits, withdrawals, other currencies, and payment gateways.
- An offline queue that applies transfers later.

## Capabilities

### New Capabilities

- `user-registration`: required fields, email and password checks, matching confirmation, duplicate-email rejection, and creating the user together with an opening wallet.
- `jwt-session`: login, JWT claims and lifetime, the 15-minute idle clock, and the protected route's welcome sentence.
- `wallet`: opening balance, reading one's balance and transfer history, and sending money under the transfer rules, including transfer-id idempotency.

### Modified Capabilities

- None. The project has no existing specs.

## Impact

- New Go module under `api/`: HTTP handlers, registration and session behavior, wallet behavior, checked-in SQL, sqlc output, and tests.
- `compose.yaml` runs MySQL 8 and the API. `.env.example` and the README list the variables Compose and the API require.
- Callers use HTTP only. There is no UI service, no browser bundle, and no second backend.
