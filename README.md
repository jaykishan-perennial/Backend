# License Management System — Backend

Go/Gin RESTful API for the License Management System: admin and customer JWT APIs, SDK access via API keys, SQLite persistence, and a small static admin web UI.

## Features

- **Authentication**: JWT-based login for **admin** and **customer** roles (`Authorization: Bearer <token>`).
- **SDK authentication**: `POST /sdk/auth/login` and `POST /sdk/auth/signup` return or establish an **API key** for SDK clients (`X-API-Key`).
- **Customer management**: Admin CRUD for customers linked to user accounts.
- **Subscription pack management**: Admin CRUD for subscription plans.
- **Subscription lifecycle**: Request, approve, assign, activate/deactivate, and automatic **expiry** handling (background job marks overdue active subscriptions as expired).
- **Dashboard**: Admin dashboard statistics endpoint.
- **Audit logging**: Admin-accessible audit log listing for security-relevant actions.
- **Operational**: CORS (including `X-API-Key`), **per-IP rate limiting**, seeded default admin user, static **admin UI** at `/admin`.

## Project structure

```
backend/
├── main.go                 # Application entry, CORS, rate limit, routes, static files, expiry checker
├── go.mod / go.sum         # Go module and dependencies
├── Dockerfile              # Multi-stage build (see “Running with Docker”)
├── .dockerignore
├── config/
│   └── config.go           # Environment-driven configuration (PORT, JWT_SECRET, DB_PATH)
├── database/
│   └── database.go         # SQLite + GORM auto-migrate and admin seed
├── models/                 # GORM models (User, Customer, SubscriptionPack, Subscription, AuditLog)
├── handlers/               # HTTP handlers (auth, admin, customer, SDK, validation, audit)
├── middleware/
│   ├── auth.go             # JWT (admin/customer) and API key (SDK) middleware
│   └── ratelimit.go        # Token-bucket rate limiter by client IP
├── routes/
│   └── routes.go           # Route registration and middleware wiring
└── web/                    # Static admin SPA assets (served under /web and /admin)
```

## Setup

### Prerequisites

- **Go**: version required by `go.mod` (currently **Go 1.25**). The `Dockerfile` uses Go **1.24** for the build image; use a local toolchain that satisfies `go.mod`.
- **CGO**: SQLite driver may require a C toolchain on some platforms (the Docker image installs `gcc` / `musl-dev` for this).

### Install dependencies

```bash
cd backend
go mod download
```

### Configure environment variables (optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | TCP port (server listens on **0.0.0.0**:`PORT`) |
| `DB_PATH` | `license_management.db` | SQLite database file path |
| `JWT_SECRET` | `license-management-secret-key` | HMAC secret for JWT signing (**change in production**) |

Example:

```bash
export PORT=8080
export DB_PATH=./license_management.db
export JWT_SECRET=your-secret-key-change-in-production
```

### Run the server

```bash
go run main.go
```

The API is available at **http://localhost:8080** (or your host’s IP on the LAN). Open **http://localhost:8080/admin** for the bundled admin UI.

### Default admin account

On first run, if no admin exists, the database seeder creates:

- **Email**: `admin@example.com`  
- **Password**: `admin123`  

Change this password and restrict access in any real deployment.

## API endpoints

Path parameters below match Gin (`:name`). Send JWTs as `Authorization: Bearer <token>`. Send SDK keys as `X-API-Key: <api_key>`.

### Authentication (no JWT)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/admin/login` | Admin login → JWT |
| POST | `/api/customer/login` | Customer login → JWT |
| POST | `/api/customer/signup` | Customer registration |
| POST | `/sdk/auth/login` | SDK login → API key (and profile fields) |
| POST | `/sdk/auth/signup` | SDK registration → API key |

### Admin (`/api/v1/admin/*`) — JWT required, role `admin`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/dashboard` | Dashboard statistics |
| GET | `/api/v1/admin/customers` | List customers |
| POST | `/api/v1/admin/customers` | Create customer |
| GET | `/api/v1/admin/customers/:customer_id` | Get customer |
| PUT | `/api/v1/admin/customers/:customer_id` | Update customer |
| DELETE | `/api/v1/admin/customers/:customer_id` | Delete customer |
| GET | `/api/v1/admin/subscription-packs` | List subscription packs |
| POST | `/api/v1/admin/subscription-packs` | Create pack |
| PUT | `/api/v1/admin/subscription-packs/:pack_id` | Update pack |
| DELETE | `/api/v1/admin/subscription-packs/:pack_id` | Delete pack |
| GET | `/api/v1/admin/subscriptions` | List subscriptions |
| POST | `/api/v1/admin/subscriptions/:subscription_id/approve` | Approve subscription |
| POST | `/api/v1/admin/customers/:customer_id/assign-subscription` | Assign subscription |
| DELETE | `/api/v1/admin/customers/:customer_id/subscription/:subscription_id` | Unassign subscription |
| GET | `/api/v1/admin/audit-logs` | List audit logs |

### Customer (`/api/v1/customer/*`) — JWT required, role `customer`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/customer/subscription` | Current subscription |
| POST | `/api/v1/customer/subscription` | Request subscription |
| DELETE | `/api/v1/customer/subscription` | Deactivate subscription |
| GET | `/api/v1/customer/subscription-history` | Subscription history |

### SDK (`/sdk/v1/*`) — `X-API-Key` required

| Method | Path | Description |
|--------|------|-------------|
| GET | `/sdk/v1/subscription-packs` | List packs (optional query params as implemented) |
| GET | `/sdk/v1/subscription` | Current subscription |
| POST | `/sdk/v1/subscription` | Request subscription |
| DELETE | `/sdk/v1/subscription` | Deactivate subscription |
| GET | `/sdk/v1/subscription-history` | Subscription history |

## Database

The application uses **SQLite** by default via GORM. The file at `DB_PATH` is created and migrated on startup (`AutoMigrate`).

### Schema (logical)

- **users** — Authentication (admin/customer roles, password hash).
- **customers** — Customer profile and **API key** for SDK access.
- **subscription_packs** — Plan catalog.
- **subscriptions** — Assignments and lifecycle (status, dates, etc.).
- **audit_logs** — Recorded admin/security-relevant actions.

## Configuration summary

| Setting | Default |
|---------|---------|
| Listen address | `0.0.0.0:<PORT>` |
| Port | `8080` |
| Database | `./license_management.db` (via `DB_PATH`) |
| JWT secret | `license-management-secret-key` |
| JWT lifetime | **24 hours** (see `handlers/auth.go`) |
| Rate limit | **10** requests/second per IP, burst **20** (see `main.go`) |

## Development

### Running tests

```bash
go test ./...
```

### Building a binary

```bash
go build -o license-management-server .
```

### Running with Docker

Build and run using the provided `Dockerfile` (binary is named `server` in the image; database path is set via `DB_PATH`):

```bash
docker build -t license-management-backend .
docker run -p 8080:8080 \
  -e JWT_SECRET=change-me-in-production \
  -e DB_PATH=license_management.db \
  license-management-backend
```

Mount a volume or bind-mount for `DB_PATH` if you need persistence across container restarts.

## Security notes

- Set a strong, unique **`JWT_SECRET`** in production.
- Prefer **HTTPS** and terminate TLS in front of the API in production.
- **Rate limiting** is enabled globally; tune `middleware.RateLimiter` parameters if needed.
- Request bodies use Gin binding / validation in handlers; keep validating and sanitizing inputs for new endpoints.
- Keep secrets in **environment variables** or a secret manager, not in the image or repo.
- Replace or disable the **default admin** credentials before any public exposure.

## OpenAPI

If your repository includes an OpenAPI document (e.g. at the repo root), it should describe the same HTTP contract as `routes/routes.go` for client generation and testing.
