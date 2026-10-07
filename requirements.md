# Take-home Assignment: Auth with JWT (TypeScript)

Build a small application in **TypeScript/Go** that supports user **registration** and **login** using **JWT**.
You can choose any stack or structure you want.
As long as the core auth flow works end-to-end, it’s accepted.

---

## Requirements

### 1. Register

- Fields: `email`, `password`, `confirmPassword`

### 2. Login

- Input: `email`, `password`
- Return: **JWT**
- Token should contain at least:

  - `email`
  - `user id` or similar identifier

### 3. Authenticated View / Endpoint

After successful login, calling the protected route / loading the protected screen should show:

```
Hello [email], welcome back
```

user should be logged out after 15 minutes of inacitvity

---

## What to deliver

- Fork this repository and then send the link
- A runnable project (any structure).
- README explaining:

  - How to build and run it (prepare docker compose)
  - Required environment variables

---

## Acceptance criteria

- Registration works with validation.
- Login returns a usable JWT.
- A protected route or screen shows the welcome message using JWT auth.

---

## Optional bonus

- Docker
- Backend built using Go (or their frameworks)
- Frontend built using React/Vue (or their frameworks)
- Tests (unit or integration)

This keeps the scope tight: just registration, login, and a protected “Hello [email]” flow.
