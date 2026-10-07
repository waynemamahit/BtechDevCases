# Proposal

## Why

The repository is a take-home assignment that asks for registration, login, and a protected welcome, and the submission has to be built and run from Docker Compose. A small greenfield app has to make those flows work end to end, including logout after 15 minutes of inactivity.

## What Changes

- Add a Go API that registers a user, signs a JWT on login, and serves one protected identity endpoint.
- Add a React screen in TypeScript, built and tested with Bun, that registers, logs in, and shows the protected greeting.
- Format and lint the React TypeScript sources, including tests, with Biome through Bun, using a checked-in Biome config. `bun run lint` runs `biome check --write .`. `bun run typecheck` runs `tsc --noEmit`.
- Style the registration, login, and welcome screens with Tailwind CSS 4 and daisyUI 5. The CSS entry is `@import "tailwindcss"` and `@plugin "daisyui"`. The unauthenticated page is one card. daisyUI tabs show either the login form or the registration form, and the page does not render both. It opens on Log in. Choosing Register replaces that form. An accepted registration returns to Log in and does not store a token. The page `flex` layout and the one-column `grid` use responsive prefixes so the screens stay usable from a narrow phone width through tablet and desktop. Components and colors come from daisyUI. Tailwind utilities fill gaps daisyUI does not cover.
- Run the Bun frontend, the Go API, and MySQL 8 with Docker Compose. The browser calls the Go API. The UI API base URL comes from an environment variable.
- Persist users in MySQL. The Go service talks to MySQL only through sqlc-generated code. Queries are checked-in SQL, and the generated Go package is committed.
- Compose runs MySQL with a named volume and a healthcheck. The API starts only after MySQL is healthy. The same SQL file is what sqlc uses as its schema and what Compose applies when the MySQL data volume is first created.
- The database DSN and MySQL credentials come from environment variables.
- Registration validation is only: `email`, `password`, and `confirmPassword` are required; `email` is a valid address; `password` is non-empty; `confirmPassword` matches `password`. A duplicate email is rejected by a MySQL unique key.
- Login returns a signed JWT whose claims include email and a stable user id. The signing secret comes from an environment variable. Store passwords only as a one-way hash.
- The protected React screen shows exactly `Hello [email], welcome back`.
- Inactivity is 15 minutes since the last successful authenticated request. Continued activity keeps the session. After 15 idle minutes the screen ends the session and the Go endpoint rejects the next request.
- Document the Compose run path and every required environment variable, including the MySQL credentials and the DSN, in the project README.
- Add Go tests for registration validation, JWT claims, the protected identity, and idle rejection with a fake clock. Prove duplicate-email rejection against MySQL. Add frontend tests for the exact welcome sentence, one auth form at a time, an accepted registration returning to login without a token, the screen ending the session at token expiry, a `401` from the protected request clearing the stored session, and login, registration, and welcome staying on screen at a narrow phone width and at desktop width.

Out of scope: a TypeScript or C# backend, a Node npm pnpm or yarn frontend toolchain, ESLint, Prettier, OAuth, MFA, email verification, password reset, profile editing, roles, an ORM, Postgres or SQLite, and a CSS system other than Tailwind CSS with daisyUI.

## Capabilities

### New Capabilities

- `user-registration`: Required fields, email format, non-empty password, matching confirmation, and duplicate-email rejection by a MySQL unique key.
- `jwt-session`: Login, JWT claims (`email` and a stable user id), idle logout, the protected React greeting, and Tailwind CSS 4 with daisyUI 5 screens. The unauthenticated page shows one auth form at a time and stays usable from a narrow phone width through desktop.

### Modified Capabilities

- None. The project has no existing specs.

## Impact

- New Go API, sqlc SQL and generated Go, MySQL 8 in Compose, a React TypeScript app (Bun, Biome, Tailwind CSS 4, daisyUI 5), Docker Compose, an environment example, and README run instructions.
- New HTTP surface: register, login (returns a JWT), and one authenticated endpoint whose identity drives `Hello [email], welcome back`.
- Dependencies: a JWT library, a password hash, a MySQL driver, and sqlc in Go; React and React DOM at runtime; Biome, Tailwind CSS 4, daisyUI 5, Playwright, and the Bun test runner as frontend devDependencies.
- Tests: Go coverage of validation, claims, protected identity, and idle rejection with a fake clock; a MySQL proof of duplicate-email rejection; a frontend assertion of the exact welcome sentence; a frontend assertion that login and registration are not on screen together; a frontend assertion that an accepted registration returns to login without a token; a frontend assertion that the welcome sentence ends at token expiry; a frontend assertion that a `401` from the protected request clears the stored session; a Playwright check that login, registration, and welcome stay on screen at 320×800 and at 1280×800.
