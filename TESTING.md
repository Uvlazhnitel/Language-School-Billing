# Manual Testing Guide

This repository is web-only. Manual verification should focus on the HTTP server, browser UI, Docker deployment flow, and storage/backups.

## Prerequisites

- Go 1.24+
- Node.js 22+
- npm
- Docker + Docker Compose

## Automated Test Commands

Backend:

```bash
go test ./...
```

Frontend:

```bash
cd frontend
npm test
npm run lint
```

## Local Manual Run

Start the backend:

```bash
go run ./cmd/web
```

Start the frontend dev server in another terminal:

```bash
cd frontend
npm run dev
```

Verify:

- login screen loads
- `/healthz` responds successfully
- session bootstrap works
- core tabs render without browser console errors

## Docker Manual Run

```bash
docker compose build
docker compose up -d
docker compose logs -f
```

Verify:

- the app responds on the published port
- login works with configured admin credentials
- static frontend is served by `cmd/web`
- SQLite database is created in `${APP_DATA_DIR}`
- backups and invoices persist in mounted volumes

## Core Manual Scenarios

1. Authentication
   - sign in
   - refresh the page and confirm session persistence
   - log out and confirm protected views require re-auth

2. Student and course management
   - create student, teacher, course, and enrollment
   - edit them
   - confirm admin-only destructive actions respect permissions

3. Attendance and invoices
   - create attendance for a `per_lesson` enrollment
   - set subscription lesson count for a `subscription` course
   - generate invoice drafts
   - issue an invoice
   - download the PDF in browser

4. Payments and debtors
   - record payment
   - confirm invoice and debtor summaries update

5. Backups
   - create a backup from settings
   - confirm the file appears in the configured backup directory

## Troubleshooting

- If frontend tests fail, run `npm install` in `frontend/`.
- If the server cannot start, verify `.env` values for data, invoice, backup, and fonts directories.
- If PDF generation fails, confirm the fonts directory contains the required DejaVu font files.
