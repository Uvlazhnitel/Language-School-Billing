# Comprehensive Testing Procedures

This document describes the recommended regression pass for the web-only LangSchool application.

## Environment Setup

1. Install Go 1.24+, Node.js 22+, npm, Docker, and Docker Compose.
2. Copy `.env.example` to `.env`.
3. Fill in:
   - `APP_BASE_URL`
   - `ADMIN_USERNAME`
   - `ADMIN_PASSWORD`
   - `SESSION_SECRET`
4. Confirm these directories are valid and writable:
   - `APP_DATA_DIR`
   - `INVOICES_DIR`
   - `BACKUPS_DIR`
   - `LS_FONTS_DIR`

## Automated Regression Pass

Run backend tests:

```bash
go test ./...
```

Run frontend checks:

```bash
cd frontend
npm test
npm run lint
```

## Browser Regression Pass

Start locally:

```bash
go run ./cmd/web
cd frontend && npm run dev
```

Verify these workflows:

1. Authentication
   - invalid login shows an error
   - valid login creates a session
   - logout returns to sign-in

2. Master data
   - create and edit teacher
   - create and edit course
   - create and edit student
   - create and edit enrollment

3. Attendance and billing
   - edit `per_lesson` attendance
   - set `subscription` monthly lesson count
   - generate invoice drafts
   - issue invoice
   - reopen issued invoice to draft when allowed

4. PDF behavior
   - open invoice details
   - click download PDF
   - confirm browser download works
   - confirm there is no local folder reveal action

5. Payments and debtors
   - record invoice payment
   - delete payment as admin
   - verify debtor totals refresh correctly

6. Backups
   - create backup from settings
   - verify a new backup appears in `BACKUPS_DIR`

## Docker Regression Pass

```bash
docker compose build
docker compose up -d
docker compose logs --tail=200
```

Verify:

- `/healthz` returns ready
- the frontend is served by the Go server
- mounted volumes receive database, invoices, and backups
- admin login works in the containerized app

## Failure Checklist

- If auth fails, check `SESSION_SECRET`, cookies, and `APP_BASE_URL`.
- If PDFs fail, check `LS_FONTS_DIR` and the bundled font files.
- If backups fail, check write permissions for `BACKUPS_DIR`.
- If the UI cannot load data, verify routing to `/api`.
