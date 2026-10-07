# Auth wallet API

Go HTTP API for registration, login, a protected welcome route, and wallet transfers. Accounts and money are stored in MySQL 8. Docker Compose runs the API and MySQL. There is no UI service.

## Run

Copy `.env.example` to `.env` and set the variables. Then:

```sh
docker compose up --build
```

The API listens on http://localhost:8080. Stop the stack with `docker compose down`.

MySQL keeps data in the named volume `mysql-data`. Compose applies `api/db/schema.sql` when that volume is first created, and the API starts only after MySQL is healthy. To apply the schema again, remove the volume:

```sh
docker compose down -v
```

## Money

There is one currency. Amounts are integer minor units with two decimal places: 100 minor units equal 1.00. A new wallet starts at 100000 minor units, which is 1000.00.

Email comparison is exact, including case. `Ada@example.com` and `ada@example.com` are different accounts.

## Environment variables

| Name | Read by | Default |
| --- | --- | --- |
| `JWT_SECRET` | Go API | No default. The process exits when it is empty. |
| `DATABASE_DSN` | Go API | No default. The process exits when it is empty. Inside Compose the host is `mysql`. Use the same user, password, and database as `MYSQL_USER`, `MYSQL_PASSWORD`, and `MYSQL_DATABASE`. Include `parseTime=true`. |
| `API_PORT` | Go API | `8080` |
| `WEB_ORIGIN` | Go API | No default. The process exits when it is empty. Browser origin allowed by CORS. |
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

Login returns an HS256 JWT whose `sub` is the user id and whose `email` is the account email. The token expires 24 hours after it is issued. The session ends after 15 minutes without a successful login or a later successful authenticated request. Wallet reads and successful transfers count. A failed or timed-out call does not. `POST /transfers` takes `transferId`, `recipient`, `amount`, and `notes`. Repeating the same transfer id with the same recipient, amount, and notes returns the original transfer and does not move money again.
