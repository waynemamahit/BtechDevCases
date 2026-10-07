# Spec Delta

## Purpose

Signs a user in with a JWT, shows the protected welcome, keeps the session while the user stays active, and ends it after 15 idle minutes. The registration, login, and welcome screens use Tailwind CSS 4 and daisyUI 5. The unauthenticated page shows one auth form at a time.

## ADDED Requirements

### Requirement: Login returns a signed token
A login with the registered email and password MUST return a signed JWT whose claims include that email and the account's stable user id.

#### Scenario: Successful login
- **WHEN** a registered user submits that account's email and password
- **THEN** the system returns a signed JWT
- **AND** the token claims include that email and the account's user id

#### Scenario: Later login keeps the same user id
- **WHEN** the same account logs in again
- **THEN** the new token carries the same user id as the earlier token

### Requirement: Login rejects unknown credentials
The system MUST reject login when the email is not registered or the password does not match, and MUST NOT return a token.

#### Scenario: Unknown email
- **WHEN** a login submits an email that has no account
- **THEN** the system rejects the login
- **AND** no token is returned

#### Scenario: Wrong password
- **WHEN** a login submits a registered email with a password that does not match
- **THEN** the system rejects the login
- **AND** no token is returned

### Requirement: Protected identity requires a valid token
The protected endpoint MUST return the token's email and user id when the token is valid, and MUST reject a call with no token or an invalid token.

#### Scenario: Valid token returns the account identity
- **WHEN** a client calls the protected endpoint with a valid token
- **THEN** the endpoint returns the email and user id from that token

#### Scenario: Missing token is rejected
- **WHEN** a client calls the protected endpoint with no token
- **THEN** the endpoint rejects the request

#### Scenario: Invalid token is rejected
- **WHEN** a client calls the protected endpoint with a token the system did not sign
- **THEN** the endpoint rejects the request

### Requirement: Protected screen shows the welcome sentence
The protected screen MUST show exactly `Hello <email>, welcome back`, where `<email>` is the email from the protected identity.

#### Scenario: Greeting uses the authenticated email
- **WHEN** the protected screen loads for the authenticated email `ada@example.com`
- **THEN** the screen shows exactly `Hello ada@example.com, welcome back`

### Requirement: A successful authenticated request renews the credential
A successful authenticated request MUST be accepted and MUST return a new credential. The protected endpoint MUST accept that new credential until 15 minutes after the request, and MUST reject it once those 15 minutes have passed.

#### Scenario: Activity before the idle limit renews access
- **WHEN** the client makes a successful authenticated request
- **AND** makes another authenticated request 14 minutes later
- **THEN** the later request is accepted
- **AND** its response includes a new credential
- **AND** the protected endpoint accepts that new credential 14 minutes after the later request
- **AND** the protected endpoint rejects that new credential 15 minutes after the later request

### Requirement: Idle time ends the session
When 15 minutes pass with no successful authenticated request, the protected screen MUST end the session, and the protected endpoint MUST reject the credential that the last successful request returned.

#### Scenario: Screen ends the session after idle time
- **WHEN** 15 minutes pass after the last successful authenticated request
- **THEN** the protected screen ends the session
- **AND** it no longer shows the welcome sentence

#### Scenario: Endpoint rejects the idle credential
- **WHEN** 15 minutes pass after a successful authenticated request
- **AND** the client calls the protected endpoint with the credential that request returned
- **THEN** the endpoint rejects the request

### Requirement: The page shows one auth form
The unauthenticated page MUST show either the login form or the registration form, and MUST NOT show both at the same time. It MUST open on the login form. Choosing Register MUST show the registration form and hide the login form. An accepted registration MUST return to the login form and MUST NOT store a token.

#### Scenario: Login is the only form on first load
- **WHEN** the unauthenticated page loads
- **THEN** the login form is visible
- **AND** the registration fields are not visible

#### Scenario: Register replaces the login form
- **WHEN** the visitor selects Register
- **THEN** the registration form is visible
- **AND** the login submit control is not visible

#### Scenario: Accepted registration returns to login
- **WHEN** registration is accepted
- **THEN** the page shows the login form
- **AND** no token is stored

### Requirement: Screens use Tailwind CSS and daisyUI
The registration screen, the login screen, and the protected greeting MUST use Tailwind CSS 4 and daisyUI 5, and MUST stay usable from a narrow phone width through tablet and desktop. The unauthenticated page MUST present those two forms as tabs on one card.

#### Scenario: Narrow phone width
- **WHEN** the registration, login, or welcome screen is shown at a narrow phone width
- **THEN** the controls and the welcome sentence stay on screen without a horizontal page scroll

#### Scenario: Desktop width
- **WHEN** the registration, login, or welcome screen is shown at a desktop width
- **THEN** the controls and the welcome sentence stay usable
