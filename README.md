# Language School Billing

Billing and operations app for a small language school.  
The project can run in two modes:

- a desktop app via **Wails**
- a browser-based web app via the Go HTTP server

It is built for one school administrator who needs to keep students, enrollments, attendance, invoices, payments, and debt follow-up in one place.

## What the app does

The current app covers the full monthly workflow:

- students with adult/minor handling and payer contact fields
- courses and teachers
- enrollments with billing mode and discounts
- attendance for `per_lesson` students
- monthly lesson count for `subscription` students
- invoice draft generation, issuing, reopening to draft, PDF generation, and PDF download
- payments and debtor tracking
- web user accounts with roles, password auth, and "remember me"
- backups for both the SQLite database and invoice files

Main UI sections in the web app:

- `Overview`
- `Students`
- `Courses`
- `Enrollments`
- `Attendance`
- `Invoices`
- `Debtors`
- `Files`

## Billing model

The app supports two billing modes:

### `per_lesson`

- each student is billed from their own attendance for the selected month
- attendance is editable per enrollment
- issued / paid / canceled invoices lock attendance until the invoice is returned to draft

### `subscription`

- the invoice is not based on the student's personal attendance
- the invoice is based on the number of lessons actually held for the course in that month
- that monthly lesson count is shared at the course level
- subscription amount is calculated from:

```text
lesson_price × lessons_held × (1 - total_discount_pct / 100)
```

Where:

- `subscription_discount_pct` is the base subscription discount
- `discount_pct` is the personal student discount
- both discounts stack, capped by the backend as needed

## Stack

- **Backend:** Go
- **Desktop shell:** Wails v2
- **Web server:** Go `net/http`
- **Frontend:** React + Vite + TypeScript
- **ORM:** ent
- **Database:** SQLite
- **PDF invoices:** `go-pdf/fpdf`

## Repository layout

- [main.go](/Users/uvlazhnitel/Documents/coding/langschool/langschool/main.go) — Wails desktop entrypoint
- [app.go](/Users/uvlazhnitel/Documents/coding/langschool/langschool/app.go) — desktop-exposed backend methods
- [cmd/web/main.go](/Users/uvlazhnitel/Documents/coding/langschool/langschool/cmd/web/main.go) — web server entrypoint
- [cmd/backupctl/main.go](/Users/uvlazhnitel/Documents/coding/langschool/langschool/cmd/backupctl/main.go) — CLI for DB/full backups
- [internal/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/internal) — business logic, auth, runtime, PDF, web API
- [ent/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/ent) — schema and generated ORM
- [frontend/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/frontend) — React app
- [scripts/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/scripts) — backup and server helper scripts

## Data storage

The app keeps its own working directories for:

- database
- backups
- invoice PDFs
- exports
- fonts

### Desktop default directories

On desktop installs the base app folder is created automatically:

- macOS / Linux: `~/StudentDesk/`
- legacy folder names are migrated from older `~/LangSchool/` layouts when possible

Typical directories:

- `StudentDesk/Data/`
- `StudentDesk/Backups/`
- `StudentDesk/Invoices/`
- `StudentDesk/Exports/`
- `StudentDesk/Fonts/`

SQLite database file:

- `StudentDesk/Data/app.sqlite`

### Web / server directories

In web/server deployments the storage paths are controlled by environment variables:

- `APP_DATA_DIR`
- `INVOICES_DIR`
- `BACKUPS_DIR`
- `LS_FONTS_DIR`

The included Docker setup stores them under:

- `/var/lib/langschool/data`
- `/var/lib/langschool/invoices`
- `/var/lib/langschool/backups`
- `/app/Fonts`

## Backups and migration safety

The runtime is intentionally conservative with data:

- before applying schema migrations to an existing database, it creates a pre-migration backup
- startup stops if that backup cannot be created
- schema updates are additive / non-destructive by default

SQLite is configured for safer local durability:

- `journal_mode=WAL`
- `synchronous=FULL`

Retention defaults:

- latest `30` DB-only backups
- latest `8` full backups

Backup formats:

- `app-YYYYMMDD-HHMMSS.sqlite` — database only
- `full-YYYYMMDD-HHMMSS.tar.gz` — database + invoice files

A full backup contains:

- `data/app.sqlite`
- full `invoices/` tree
- `manifest.json`

## Requirements

### For desktop development

- Go
- Node.js + npm
- Wails CLI

macOS also needs:

```bash
xcode-select --install
```

Install Wails CLI if needed:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Check desktop environment:

```bash
wails doctor
```

### For web development

- Go
- Node.js + npm

## Quick start

Clone the repo:

```bash
git clone https://github.com/Uvlazhnitel/Language-School-Billing.git
cd Language-School-Billing
```

Install backend/frontend dependencies:

```bash
go mod download
cd frontend
npm install
cd ..
```

## Run the web app in development

Start the Go API/server:

```bash
go run ./cmd/web
```

In a second terminal start Vite:

```bash
cd frontend
npm run dev
```

By default Vite proxies:

- `/api`
- `/healthz`

to:

- `http://127.0.0.1:8080`

If needed, create `frontend/.env.local` and override the dev target there.

## Run the web app in production-style mode

Build the frontend:

```bash
cd frontend
npm run build
cd ..
```

Start the Go server:

```bash
go run ./cmd/web
```

If `frontend/dist` exists, the Go server serves:

- the SPA
- the JSON API under `/api`

If `frontend/dist` is missing, the server still starts in API-only mode.

## Desktop development

Build the frontend once:

```bash
cd frontend
npm run build
cd ..
```

Run desktop dev mode:

```bash
wails dev
```

## Build desktop artifacts

### macOS

```bash
cd frontend
npm run build
cd ..
wails build
```

### Windows

```powershell
cd frontend
npm run build
cd ..
wails build
```

Recommended Windows installer build:

```powershell
$env:CGO_ENABLED = "0"
wails build -clean -nsis -webview2 download
```

## Web authentication

The web app uses username/password authentication.

Login payload:

- `username`
- `password`
- `rememberMe`

Behavior:

- with `rememberMe=true`, the session cookie is persistent
- without it, the cookie is session-only

Bootstrap admin account is configured by environment variables:

- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`
- `SESSION_SECRET`

Legacy fallback:

- `ADMIN_EMAIL` still works as a fallback source for the admin username

## Environment variables

See [.env.example](/Users/uvlazhnitel/Documents/coding/langschool/langschool/.env.example) for the current baseline.

Important variables:

- `APP_BASE_URL`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`
- `SESSION_SECRET`
- `APP_DATA_DIR`
- `INVOICES_DIR`
- `BACKUPS_DIR`
- `LS_FONTS_DIR`
- `HOST`
- `PORT`

Web server also supports:

- `ADDR`
- `WEB_DIST_DIR`

## Docker deployment

The repository includes [compose.yaml](/Users/uvlazhnitel/Documents/coding/langschool/langschool/compose.yaml) for a simple server deployment.

Default behavior:

- publishes the app on `8082`
- runs the Go web server inside Docker
- bind-mounts data, invoices, and backups from the host

Basic flow:

1. Copy the repo to the server.
2. Create `.env` from `.env.example`.
3. Make sure the host storage directories exist.
4. Start the app:

```bash
docker compose up -d --build
```

Health check:

```bash
curl http://127.0.0.1:8082/healthz
```

## Backup scripts

Available helper scripts in [scripts/](/Users/uvlazhnitel/Documents/coding/langschool/langschool/scripts):

- `create-db-backup.sh`
- `create-full-backup.sh`
- `restore-backup.sh`
- `install-server-backup-cron.sh`
- `pull-backups-mac.sh`
- `install-mac-backup-launchd.sh`

Examples:

Create DB-only backup:

```bash
./scripts/create-db-backup.sh
```

Create full backup:

```bash
./scripts/create-full-backup.sh
```

Restore full backup:

```bash
./scripts/restore-backup.sh full-20260604-142500.tar.gz
```

## Fonts for PDF invoices

Required fonts:

- `DejaVuSans.ttf`
- `DejaVuSans-Bold.ttf`

Preferred directory:

- desktop: `StudentDesk/Fonts/`
- server/Docker: `/app/Fonts`

Alternative lookup locations are also supported by the runtime, but `LS_FONTS_DIR`
is the most predictable option for deployments.

## Checks during development

Frontend:

```bash
cd frontend
npm run lint
npm test
npm run build
```

Backend:

```bash
go test ./...
```

## Notes

- the web app is stateful because it uses SQLite and server-side generated PDFs
- invoice PDFs are stored on disk, not only generated in memory
- backup/restore is a first-class part of the project, not an afterthought
- the repository currently contains both desktop and web runtime paths, so README and deployment instructions need to reflect both
