# Proposal

## Why

Someone on a bad connection still needs to register, stay signed in only while they are actually using the app, and move money without a retry spending it twice. The product is a Go API and a Flutter app. Docker Compose runs the API and MySQL 8. The Flutter app runs on an Android device or emulator and calls that API.

## What Changes

- A Go HTTP API for registration, login, a protected welcome route, and wallet reads and transfers, plus a Flutter app that performs that same flow on screen.
- Registration accepts `email`, `password`, and `confirmPassword`. The password is stored only as a one-way hash. The user and a wallet are created in one database transaction. The wallet opens at 1000.00, stored as 100000 minor units. A duplicate email is rejected by a MySQL unique key. Email comparison is exact, including case.
- Login accepts `email` and `password` and returns a signed JWT whose claims include the email and a stable user id. The token lifetime is 24 hours, which is longer than the idle limit. Logout after 15 minutes of inactivity is a separate clock, not the token expiry.
- The protected route returns exactly `Hello [email], welcome back`. The signed-in Flutter screen shows that same sentence.
- The idle clock starts at a successful login and moves on each later successful authenticated request, including wallet reads and transfers. A failed or timed-out call leaves the clock unchanged. After 15 idle minutes the API rejects the next authenticated request and does not move the clock. The Flutter app stores the time of the last successful login or authenticated request and, on resume, clears the session locally even when the API cannot be reached.
- A signed-in user can see their balance and their own sent and received transfers, including notes. The screen shows amounts with two decimal places. After a successful wallet read, each transfer is one history row: `Received` or `Sent`, the two-decimal amount, the other account, and the notes when they are not empty. An empty history says there are no transfers yet.
- A transfer takes the recipient's email, a positive amount, notes, and a client-generated transfer id. The balance check, debit, and credit commit in one transaction. The Flutter app shows a validation failure in an error alert, using the API error text for a client error and `Request failed` for a timeout, a server fault, or a lost response. The last successful balance stays on screen. Retrying that same submission sends the same transfer id and does not move money twice.
- Calls use finite timeouts. Docker Compose runs the Go API and MySQL 8. MySQL keeps data in the named volume `mysql-data`, applies the checked-in schema when that volume is first created, and the API starts only after MySQL is healthy. Go reaches MySQL only through sqlc, and the generated package is committed.
- The README explains how to run Compose, how an Android emulator or device reaches the API, and every environment variable the API and the Flutter app read.
- Go tests cover registration validation, JWT claims, the protected identity, idle rejection with a fake clock, a successful transfer that changes both balances, insufficient funds, an unknown recipient, a self-transfer, and an idempotent retry. Flutter tests cover the exact welcome sentence, the history rows, and the error alert for validation failures and a timed-out transfer.

### Out of scope

- Postgres, a TypeScript or C# backend, a React or React Native frontend, and a Node, npm, pnpm, yarn, or Bun toolchain.
- OAuth, MFA, email verification, password reset, profile editing, and roles.
- Deposits, withdrawals, other currencies, and payment gateways.
- An offline queue that applies transfers later.

## Capabilities

### New Capabilities

- `user-registration`: required fields, email and password checks, matching confirmation, duplicate-email rejection, and the register screen showing each of those errors.
- `jwt-session`: login, JWT claims, the 15-minute idle clock on the API and on the Flutter screen, the protected welcome sentence, and the login screen showing a rejected sign-in.
- `wallet`: the opening balance, reading one's balance and transfer history as one row per transfer, and sending money under the transfer rules, including transfer-id idempotency and visible validation errors.

### Modified Capabilities

- None. The project has no existing specs.

## Impact

- A Go module under `api/` serves HTTP, owns registration, sessions, and wallets, and stores rows in MySQL 8 through sqlc.
- A Flutter project under `app/` is the client. It is not a Compose service. Android is the documented run path.
- `compose.yaml` runs MySQL 8 and the API. `.env.example` and the README list the variables those processes read, and the README also lists the Flutter API base URL.
