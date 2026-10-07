# Spec Delta

## Purpose

Accepts a new account when email, password, and confirmation are valid, and rejects a duplicate email with the MySQL unique key.

## ADDED Requirements

### Requirement: Registration fields are required
The system MUST reject registration when `email`, `password`, or `confirmPassword` is missing, and MUST NOT create an account.

#### Scenario: Missing email
- **WHEN** a registration omits `email` and includes a non-empty `password` and the same `confirmPassword`
- **THEN** the system rejects the registration
- **AND** no account is created

#### Scenario: Missing password
- **WHEN** a registration includes a valid `email` and a `confirmPassword` but omits `password`
- **THEN** the system rejects the registration
- **AND** no account is created

#### Scenario: Missing confirmation
- **WHEN** a registration includes a valid `email` and a non-empty `password` but omits `confirmPassword`
- **THEN** the system rejects the registration
- **AND** no account is created

### Requirement: Email must be a valid address
The system MUST reject registration when `email` is not a valid email address, and MUST NOT create an account.

#### Scenario: Email without a valid address shape
- **WHEN** a registration submits `email` `not-an-email` with a non-empty `password` and the same `confirmPassword`
- **THEN** the system rejects the registration
- **AND** no account is created

### Requirement: Password must be non-empty
The system MUST reject registration when `password` is empty, and MUST NOT create an account.

#### Scenario: Empty password
- **WHEN** a registration includes a valid `email`, an empty `password`, and an empty `confirmPassword`
- **THEN** the system rejects the registration
- **AND** no account is created

### Requirement: Confirmation must match the password
The system MUST reject registration when `confirmPassword` differs from `password`, and MUST NOT create an account.

#### Scenario: Confirmation does not match
- **WHEN** a registration includes a valid `email`, a non-empty `password`, and a different `confirmPassword`
- **THEN** the system rejects the registration
- **AND** no account is created

### Requirement: Duplicate email is rejected by the MySQL unique key
The system MUST reject a second registration that uses the same `email` string as an existing account because the MySQL unique key on `email` rejects the insert, and MUST leave the original account unchanged.

#### Scenario: Same email registers twice
- **WHEN** an account already exists for an email
- **AND** another registration submits that same email string with a valid password and matching confirmation
- **THEN** the MySQL unique key rejects the second insert
- **AND** the original account remains
