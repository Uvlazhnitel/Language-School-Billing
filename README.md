# LangSchool

Web-only billing and operations system for a small language school.

LangSchool helps a school administrator manage students, enrollments, attendance, invoices, payments, debt follow-up, and backups from a browser-based interface.

## Features

- students with adult/minor handling and payer contact fields
- courses and teachers
- enrollments with billing mode and discounts
- attendance for `per_lesson` students
- shared monthly lesson counts for `subscription` courses
- invoice draft generation, issuing, reopening, PDF generation, and PDF download
- payments and debtor tracking
- role-based browser login with persistent sessions
- database and invoice-file backups

## Stack

- Backend: Go
- HTTP server: Go `net/http`
- Frontend: React + Vite + TypeScript
- ORM: ent
- Database: SQLite
- PDF invoices: `go-pdf/fpdf`
- Deployment: Docker Compose

## Repository Layout

- [cmd/web/main.go](/Users/uvlazhnitel/Documents/coding/langschool/langschool/cmd/web/main.go) — web server entrypoint
- [cmd/backupctl/main.go](/Users/uvlazhnitel/Documents/coding/langschool/langschool/cmd/backupctl/main.go) — backup and restore CLI for deployed environments
- [internal/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/internal) — business logic, auth, runtime, PDF, and HTTP handlers
- [ent/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/ent) — schema and generated ORM
- [frontend/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/frontend) — React application
- [compose.yaml](/Users/uvlazhnitel/Documents/coding/langschool/langschool/compose.yaml) — primary deployment entrypoint

## Runtime Storage

The server runtime is configured through environment variables:

- `APP_BASE_URL`
- `APP_DATA_DIR`
- `INVOICES_DIR`
- `BACKUPS_DIR`
- `LS_FONTS_DIR`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`
- `SESSION_SECRET`

The included Docker setup uses these paths:

- `/var/lib/langschool/data`
- `/var/lib/langschool/invoices`
- `/var/lib/langschool/backups`
- `/app/Fonts`

The SQLite database file is stored at `${APP_DATA_DIR}/app.sqlite`.

## Backups and Safety

- startup creates a pre-migration backup before schema changes
- startup stops if that backup cannot be created
- manual backups are available in the web UI for authorized users
- `backupctl` can create and restore DB/full backups from the server side

Backup formats:

- `app-YYYYMMDD-HHMMSS.sqlite` — database only
- `full-YYYYMMDD-HHMMSS.tar.gz` — database plus invoice files

## Requirements

- Go 1.24+
- Node.js 22+
- npm
- Docker and Docker Compose

## Local Development

Install dependencies:

```bash
go mod download
cd frontend
npm install
cd ..
```

Run the backend:

```bash
go run ./cmd/web
```

Run the frontend dev server in another terminal:

```bash
cd frontend
npm run dev
```

Vite proxies `/api` and `/healthz` to `http://127.0.0.1:8080` by default.

## Docker-First Run

Create `.env` from [.env.example](/Users/uvlazhnitel/Documents/coding/langschool/langschool/.env.example), then start the app:

```bash
docker compose build
docker compose up -d
```

The compose file publishes the app on `http://localhost:8082`.

## Testing

Backend:

```bash
go test ./...
```

Frontend:

```bash
cd frontend
npm test
```

## Billing Model

### `per_lesson`

- each student is billed from their own attendance for the selected month
- attendance is editable per enrollment
- issued / paid / canceled invoices lock attendance until the invoice is returned to draft

### `subscription`

- invoice totals depend on the number of lessons actually held for the course in that month
- that lesson count is shared at the course level
- subscription amount is calculated as:

```text
lesson_price × lessons_held × (1 - total_discount_pct / 100)
```
