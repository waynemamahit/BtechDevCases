# Spec Delta

## Purpose

Lets a person create an account with an email and a password, and rejects a submission that fails field validation or reuses an email.

## ADDED Requirements

### Requirement: Registration fields are required
The registration endpoint SHALL reject a submission that omits `email`, `password`, or `confirmPassword`. A field is omitted when it is absent or null. The endpoint SHALL identify which required field is missing.

#### Scenario: Email is omitted
- **WHEN** a client submits registration without `email` and with a non-empty `password` and a matching `confirmPassword`
- **THEN** the endpoint responds `400` with error `email is required` and does not create an account

#### Scenario: Password is omitted
- **WHEN** a client submits registration with an email and `confirmPassword` but without `password`
- **THEN** the endpoint responds `400` with error `password is required` and does not create an account

#### Scenario: Confirmation is omitted
- **WHEN** a client submits registration with an email and a non-empty `password` but without `confirmPassword`
- **THEN** the endpoint responds `400` with error `confirmPassword is required` and does not create an account

### Requirement: Email must be a valid address
The registration endpoint SHALL reject an `email` that is not a valid email address. An empty string is not a valid address.

#### Scenario: Email has no domain
- **WHEN** a client submits registration with `email` `not-an-email`, a non-empty `password`, and a matching `confirmPassword`
- **THEN** the endpoint responds `400` with error `email is not a valid address` and does not create an account

#### Scenario: Email is an empty string
- **WHEN** a client submits registration with `email` `""`, a non-empty `password`, and a matching `confirmPassword`
- **THEN** the endpoint responds `400` with error `email is not a valid address` and does not create an account

### Requirement: Password must be non-empty
The registration endpoint SHALL reject a `password` that is an empty string. The endpoint SHALL apply this check before it compares `confirmPassword`.

#### Scenario: Password is empty
- **WHEN** a client submits registration with a valid email, `password` `""`, and `confirmPassword` `""`
- **THEN** the endpoint responds `400` with error `password must be non-empty` and does not create an account

### Requirement: Confirmation must match the password
The registration endpoint SHALL reject a submission whose `confirmPassword` differs from `password` when `password` is non-empty.

#### Scenario: Confirmation differs
- **WHEN** a client submits registration with a valid email, `password` `secret`, and `confirmPassword` `other`
- **THEN** the endpoint responds `400` with error `confirmPassword does not match password` and does not create an account

### Requirement: Valid registration creates one account
The registration endpoint SHALL create one account for a valid submission and return that account's stable id and email. It SHALL NOT return the password.

#### Scenario: Valid submission
- **WHEN** a client submits registration with `email` `ada@example.com`, `password` `secret`, and `confirmPassword` `secret`
- **THEN** the endpoint responds `201` with that email and a stable user id, and the response does not contain the password

### Requirement: Duplicate email is rejected
The registration endpoint SHALL reject a second registration of an email that already belongs to an account. The stored email is unique by exact match, including case. The first account SHALL remain usable.

#### Scenario: Same email registers twice
- **WHEN** a client registers `ada@example.com` successfully and a client registers `ada@example.com` again with a valid password and matching confirmation
- **THEN** the second response is `409` with error `email is already registered`, and login still succeeds for the first account only

#### Scenario: Differing case is a different email
- **WHEN** `Ada@example.com` is already registered and a client registers `ada@example.com` with a valid password and matching confirmation
- **THEN** the endpoint responds `201` and creates a separate account

### Requirement: The register screen shows each validation error
The Flutter register screen SHALL show a rejected registration in an error alert and SHALL NOT sign the user in. A `400` or `409` whose JSON body has a non-empty `error` string SHALL show that string. A timeout, a status of 500 or higher, or a dropped connection SHALL show `Request failed`.

#### Scenario: Invalid email is visible
- **WHEN** the register screen submits an email that is not a valid address
- **THEN** the screen shows `email is not a valid address` in the error alert and stays on the register screen

#### Scenario: Duplicate email is visible
- **WHEN** the register screen submits an email that is already registered
- **THEN** the screen shows `email is already registered` in the error alert and does not sign the user in
