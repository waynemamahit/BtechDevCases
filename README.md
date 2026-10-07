# Auth

Registration, login, and a protected welcome screen. The API is Go. The web app is React, built and served with Bun. Users are stored in MySQL through sqlc. Docker Compose runs the web app, the API, and MySQL 8.

## Run

Copy `.env.example` to `.env` and set the variables that have no default. Then run:

```sh
docker compose up --build
```

The web app is published on http://localhost:5173 and the API on http://localhost:8080. Stop the stack with `docker compose down`.

MySQL keeps accounts in the named volume `mysql-data`. Compose applies `api/db/schema.sql` when that volume is first created, and the API starts only after MySQL is healthy.

## Environment variables

| Name | Read by | Default |
| --- | --- | --- |
| `JWT_SECRET` | Go API | No default. The process exits when it is empty. |
| `WEB_ORIGIN` | Go API | No default. The process exits when it is empty. Compose sets `http://localhost:5173`. |
| `API_PORT` | Go API | `8080` |
| `DATABASE_DSN` | Go API | No default. The process exits when it is empty. Compose passes the DSN from `.env`. Use host `mysql` inside Compose. |
| `MYSQL_ROOT_PASSWORD` | MySQL | No default. Root password for the MySQL service. |
| `MYSQL_DATABASE` | MySQL | No default. Database created on first init. |
| `MYSQL_USER` | MySQL | No default. Application user created on first init. |
| `MYSQL_PASSWORD` | MySQL | No default. Password for `MYSQL_USER`. |
| `API_BASE_URL` | Bun web server | No default. The process exits when it is empty. Compose sets `http://localhost:8080`. |
| `WEB_PORT` | Bun web server | `5173` |

`DATABASE_DSN` must use the same user, password, and database as `MYSQL_USER`, `MYSQL_PASSWORD`, and `MYSQL_DATABASE`.
