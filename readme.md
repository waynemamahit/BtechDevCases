# Auth wallet

Go API and Flutter app for registration, login, a protected welcome screen, and wallet transfers. Accounts and money are stored in MySQL 8. Docker Compose runs the API and MySQL. The Flutter app runs on an Android device or emulator and calls the API.

## Run the API

Copy `.env.example` to `.env` and set the variables. Then:

```sh
docker compose up --build
```

The API listens on http://localhost:8080. Stop the stack with `docker compose down`.

MySQL keeps data in the named volume `mysql-data`. Compose applies `api/db/schema.sql` when that volume is first created, and the API starts only after MySQL is healthy. To apply the schema again, remove the volume:

```sh
docker compose down -v
```

## Run the Flutter app

Install the Flutter SDK, then from `app/`:

```sh
flutter pub get
flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080
```

`http://10.0.2.2:8080` is how the Android emulator reaches the API published on the host at port 8080. That URL is also the default when `API_BASE_URL` is omitted. A physical device cannot use `10.0.2.2`. Pass the computer's LAN address instead, for example `--dart-define=API_BASE_URL=http://192.168.1.20:8080`, with the phone and the computer on the same network. The API allows cleartext HTTP so this lab URL works.

## Money

There is one currency. Amounts are integer minor units with two decimal places: 100 minor units equal 1.00. The app shows balances that way. A new wallet starts at 100000 minor units, which is 1000.00.

Email comparison is exact, including case. `Ada@example.com` and `ada@example.com` are different accounts.

## Environment variables

| Name | Read by | Default |
| --- | --- | --- |
| `JWT_SECRET` | Go API | No default. The process exits when it is empty. |
| `DATABASE_DSN` | Go API | No default. The process exits when it is empty. Inside Compose the host is `mysql`. Use the same user, password, and database as `MYSQL_USER`, `MYSQL_PASSWORD`, and `MYSQL_DATABASE`. Include `parseTime=true`. |
| `API_PORT` | Go API | `8080` |
| `API_BASE_URL` | Flutter | `http://10.0.2.2:8080`, or the value passed with `--dart-define=API_BASE_URL=...` |
| `MYSQL_ROOT_PASSWORD` | MySQL | No default. Root password for the MySQL service. |
| `MYSQL_DATABASE` | MySQL | No default. Database created on first init. |
| `MYSQL_USER` | MySQL | No default. Application user created on first init. |
| `MYSQL_PASSWORD` | MySQL | No default. Password for `MYSQL_USER`. |

## HTTP

| Method and path | Auth | Success |
| --- | --- | --- |
| `POST /register` | no | `201` JSON `{ "id", "email" }` |
| `POST /login` | no | `200` JSON `{ "token" }` |
| `GET /me` | bearer | `200` text `Hello <email>, welcome back` |
| `GET /wallet` | bearer | `200` JSON `{ "balance", "transfers" }` |
| `POST /transfers` | bearer | `200` JSON of the transfer |

Login returns an HS256 JWT whose `sub` is the user id and whose `email` is the account email. The token expires 24 hours after it is issued. The session ends after 15 minutes without a successful login or a later successful authenticated request. Wallet reads and successful transfers count. A failed or timed-out call does not. The signed-in screen shows `Hello <email>, welcome back`. On resume, 15 idle minutes clears that screen even when the API cannot be reached. `POST /transfers` takes `transferId`, `recipient`, `amount`, and `notes`. Repeating the same transfer id with the same recipient, amount, and notes returns the original transfer and does not move money again. The app retries a failed transfer with that same id.
