# Spec Delta

## Purpose

Lets a signed-in person see their balance and their own transfers, and send money to another account once per caller-chosen transfer id.

## ADDED Requirements

### Requirement: A new account starts with an opening balance
A successful registration SHALL create a wallet for that account in the same transaction as the account. The balance SHALL be 100000 minor units, which is 1000.00 of the single currency. The new wallet SHALL have no transfers.

#### Scenario: Balance after registration
- **WHEN** a client registers a new account and then reads that account's wallet
- **THEN** the balance is `100000` and the transfer list is empty

### Requirement: The owner can read balance and transfer history
An authenticated user SHALL be able to read their own balance and their own sent and received transfers, including each transfer's notes. The read SHALL NOT include transfers where the user is neither sender nor recipient. Amounts SHALL be integer minor units.

#### Scenario: History shows sent and received notes
- **WHEN** Ada sends Bob 500 minor units with notes `lunch`, and Bob reads his wallet
- **THEN** Bob's balance has increased by 500 and his history includes that transfer with notes `lunch`

#### Scenario: Another user's unrelated transfer stays hidden
- **WHEN** Ada sends Bob money, and a third account reads its wallet
- **THEN** that third account's history does not include Ada's transfer

### Requirement: The wallet screen shows two decimal places
The Flutter wallet screen SHALL show a balance of minor units with two decimal places. 100 minor units SHALL be shown as 1.00.

#### Scenario: Opening balance display
- **WHEN** the signed-in screen shows a balance of 100000 minor units
- **THEN** the screen shows `1000.00`

### Requirement: Transfer history is one row per transfer
After a successful wallet read, the Flutter screen SHALL show a History section. Each sent or received transfer SHALL be one row that shows `Received` or `Sent`, the amount with two decimal places, the counterparty email, and the notes when the notes are not empty. The notes SHALL be part of that same row. An empty transfer list SHALL show `No transfers yet`. A wallet read that has not succeeded SHALL NOT show `No transfers yet`.

#### Scenario: Sent and received notes stay with their transfer
- **WHEN** the signed-in screen shows a received transfer of `100000` minor units from `grey@gmail.com` with notes `Try send another user`, and a sent transfer of `75000` minor units to `grey@gmail.com` with notes `Send back`
- **THEN** one row shows `Received 1000.00`, `grey@gmail.com`, and `Try send another user`, and another row shows `Sent 750.00`, `grey@gmail.com`, and `Send back`

#### Scenario: Empty history
- **WHEN** a wallet read succeeds and that account has no transfers
- **THEN** the screen shows `No transfers yet`

### Requirement: A transfer moves money once
A transfer SHALL take the recipient's email, a positive integer amount of minor units, notes, and a caller-generated transfer id. When the recipient exists, is not the sender, the notes are at most 200 characters, and the sender's balance is greater than or equal to the amount, the API SHALL debit the sender and credit the recipient by that amount in one transaction. The transfer SHALL appear once in the sender's history and once in the recipient's history.

#### Scenario: Both balances change once
- **WHEN** Ada, whose balance is `100000`, transfers `2500` minor units to Bob with a new transfer id
- **THEN** Ada's balance is `97500`, Bob's balance is `102500`, and the transfer is listed once for each of them

#### Scenario: Empty notes are stored
- **WHEN** Ada transfers a positive amount to Bob with notes `""` and a new transfer id
- **THEN** the transfer succeeds and both histories show that transfer with empty notes

### Requirement: Invalid transfers move no money
The API SHALL reject a transfer to an unknown email, a transfer whose recipient is the sender, an amount that is zero or negative, notes longer than 200 characters, and an amount greater than the sender's balance. A rejected transfer SHALL leave both balances unchanged.

#### Scenario: Unknown recipient
- **WHEN** Ada transfers a positive amount to `nobody@example.com`
- **THEN** the response is `404` with error `recipient not found` and Ada's balance is unchanged

#### Scenario: Self-transfer
- **WHEN** Ada requests a transfer whose recipient email is Ada's email
- **THEN** the response is `400` with error `cannot transfer to self` and Ada's balance is unchanged

#### Scenario: Zero amount
- **WHEN** Ada requests a transfer with amount `0` to Bob
- **THEN** the response is `400` with error `amount must be a positive integer` and both balances are unchanged

#### Scenario: Negative amount
- **WHEN** Ada requests a transfer with amount `-1` to Bob
- **THEN** the response is `400` with error `amount must be a positive integer` and both balances are unchanged

#### Scenario: Notes longer than 200 characters
- **WHEN** Ada requests a transfer whose notes are 201 characters
- **THEN** the response is `400` with error `notes must be at most 200 characters` and both balances are unchanged

#### Scenario: Insufficient funds
- **WHEN** Ada's balance is `100000` and Ada requests a transfer of `100001` minor units to Bob
- **THEN** the response is `409` with error `insufficient funds` and both balances are unchanged

### Requirement: A transfer id is idempotent for the same payload
The transfer id belongs to the authenticated sender. Repeating that id with the same recipient, amount, and notes SHALL return the original result and SHALL NOT move money again, including when the repeat starts before the first request commits. The same id with a different recipient, amount, or notes SHALL be rejected and SHALL NOT move money.

#### Scenario: Repeated transfer id
- **WHEN** Ada completes a transfer to Bob and then submits the same transfer id, recipient, amount, and notes again
- **THEN** the second response returns the original transfer and Ada's and Bob's balances stay at the post-transfer amounts

#### Scenario: Overlapping retry
- **WHEN** Ada submits a transfer to Bob and, before that request commits, submits the same transfer id, recipient, amount, and notes again
- **THEN** both responses return the original transfer and Ada's and Bob's balances change once

#### Scenario: Same id with a different amount
- **WHEN** Ada completes a transfer and then submits the same transfer id with a different amount
- **THEN** the second response is `409` with error `transfer id reused with a different payload` and both balances stay at the post-transfer amounts

### Requirement: The wallet screen shows local validation errors together
Before a transfer request is sent, the Flutter screen SHALL show every failing local check in one error alert and SHALL NOT send the request. An empty recipient is `recipient is required`. An amount that is not a positive number with at most two decimal places is `amount must be a positive number with at most two decimal places`. A non-positive amount is `amount must be positive`. Notes longer than 200 characters are `notes must be at most 200 characters`.

#### Scenario: Several local problems are all visible
- **WHEN** Ada submits a transfer with an empty recipient, amount `0`, and notes of 201 characters
- **THEN** the screen shows `recipient is required`, `amount must be positive`, and `notes must be at most 200 characters`, and no transfer request is sent

#### Scenario: Too many decimal places
- **WHEN** Ada submits a transfer to Bob with amount `1.234`
- **THEN** the screen shows `amount must be a positive number with at most two decimal places` and no transfer request is sent

### Requirement: The wallet screen shows an API validation error
A transfer response of `400`, `404`, or `409` whose JSON body has a non-empty `error` string SHALL show that string in the error alert. The displayed balance SHALL stay at the last successful read, and the stored activity time SHALL NOT move.

#### Scenario: A self-transfer shows the API error
- **WHEN** Ada submits a transfer whose recipient is Ada's email and the API responds `400` with error `cannot transfer to self`
- **THEN** the screen shows `cannot transfer to self` and the displayed balance stays at the last successful read

#### Scenario: An unknown recipient and insufficient funds stay visible
- **WHEN** Ada submits a transfer and the API responds `404` with error `recipient not found`, then submits again and the API responds `409` with error `insufficient funds`
- **THEN** the screen shows `recipient not found` after the first response and `insufficient funds` after the second, and the displayed balance stays at the last successful read

### Requirement: A lost transfer is shown as a failure and retried with the same id
When a transfer request fails or times out, the Flutter app SHALL show that failure and SHALL NOT treat it as a successful send. The displayed balance SHALL stay at the last successful read. A retry of that same submission SHALL send the same transfer id, recipient, amount, and notes.

#### Scenario: Timeout is a visible failure
- **WHEN** Ada submits a transfer and the request times out
- **THEN** the screen shows `Request failed` and the displayed balance stays at the last successful read

#### Scenario: Retry uses the same transfer id
- **WHEN** Ada retries that timed-out submission without changing recipient, amount, or notes
- **THEN** the app sends the same transfer id as the first submission
