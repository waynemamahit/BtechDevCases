# Spec Delta

## Purpose

Lets a registered person sign in, call a protected welcome route with a JWT, and stay signed in only while they keep making successful requests.

## ADDED Requirements

### Requirement: Login returns a signed token
Login SHALL accept `email` and `password`. When they match an account, the endpoint SHALL return a signed JWT whose payload includes that account's email and stable user id. The token lifetime SHALL be longer than 15 minutes. Wrong credentials SHALL NOT return a token.

#### Scenario: Matching credentials
- **WHEN** a client logs in with the email and password of a registered account
- **THEN** the endpoint responds `200` with a signed token whose payload `email` is that account's email and whose payload `sub` is that account's user id

#### Scenario: Wrong password
- **WHEN** a client logs in with a registered email and a password that does not match
- **THEN** the endpoint responds `401` and does not return a token

### Requirement: Protected route returns the welcome sentence
The protected route SHALL return a body that is exactly `Hello [email], welcome back`, with the authenticated account's email in place of `[email]`. The route SHALL accept the login token as a bearer credential.

#### Scenario: Welcome sentence
- **WHEN** a client calls the protected route with a bearer token from a successful login for `ada@example.com`, and the session is inside the idle window
- **THEN** the response status is `200` and the body is exactly `Hello ada@example.com, welcome back`

#### Scenario: Missing credential
- **WHEN** a client calls the protected route without a bearer token
- **THEN** the response status is `401` and the body is not the welcome sentence

### Requirement: Continued activity keeps the session
The idle clock SHALL start at a successful login and SHALL move to the time of each later successful authenticated request, including a wallet read and a wallet transfer. A successful request before 15 minutes of idle SHALL be accepted.

#### Scenario: Activity inside the window
- **WHEN** a client logs in and calls the protected route 14 minutes later
- **THEN** the route responds `200` with that account's welcome sentence

#### Scenario: A later success extends the window
- **WHEN** a client logs in, calls the protected route after 14 minutes, and calls it again 14 minutes after that success
- **THEN** the second protected call responds `200` with the welcome sentence

#### Scenario: A successful wallet read extends the window
- **WHEN** a client logs in, reads their wallet after 14 minutes, and calls the protected route 14 minutes after that read
- **THEN** the protected route responds `200` with the welcome sentence

### Requirement: Idle session is rejected without moving the clock
The API SHALL reject an authenticated request when 15 minutes or more have passed since the last successful login or successful authenticated request. That rejection SHALL NOT move the idle clock. The token signature MAY still be valid when this rejection happens.

#### Scenario: Fifteen idle minutes
- **WHEN** a client logs in and calls the protected route 15 minutes after that login with no successful authenticated request in between
- **THEN** the response status is `401` and the body is not the welcome sentence

#### Scenario: Rejection leaves the clock unchanged
- **WHEN** a protected call is rejected for idle at 15 minutes after the last success, and the client calls the protected route again one minute later with the same token
- **THEN** the second response is also `401` and is not the welcome sentence

### Requirement: A failed or timed-out call does not refresh the session
A request that does not complete as a successful authenticated response SHALL leave the idle clock at the previous success. This includes a failed authenticated call and a call that times out.

#### Scenario: A failed call does not extend the window
- **WHEN** a client logs in, makes an authenticated request that fails 10 minutes later, and calls the protected route 15 minutes after login
- **THEN** the protected route responds `401` and the body is not the welcome sentence

#### Scenario: A timed-out call does not extend the window
- **WHEN** a client logs in, an authenticated request times out 10 minutes later, and the client calls the protected route 15 minutes after login
- **THEN** the protected route responds `401` and the body is not the welcome sentence
