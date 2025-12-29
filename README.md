# Language School Billing (Go + Wails)

Single-user desktop app for a language school owner: **students, enrollments, monthly attendance, invoices (PDF)**.  
Stack: **Go 1.22+, Wails v2, ent, SQLite, gofpdf**.

## Features (current)
- Students / Courses / Enrollments (ent schemas; demo data in app).
- **Monthly attendance** for per-lesson billing: quick edit, `+1 to all`, lock month.
- **Invoices**: generate drafts from attendance/subscriptions → **issue** with numbering `PREFIX-YYYYMM-SEQ` → **PDF** saved under `~/LangSchool/Invoices/YYYY/MM/`.
- App folders: `~/LangSchool/{Data,Backups,Invoices,Exports,Fonts}`.

## Quick start

`git clone https://github.com/Uvlazhnitel/Language-School-Billing.git
cd Language-School-Billing
go generate ./ent && go mod download
cd frontend && npm i && npm run build && cd ..
wails dev`

Fonts (once, for Cyrillic PDF)

Place DejaVuSans.ttf and DejaVuSans-Bold.ttf into:
`~/LangSchool/Fonts/`

Usage

  1. Attendance tab → fill counts (or use Demo data) → lock month if needed.
  
  2. Invoices tab → Generate drafts → Issue (or Issue all).
  PDFs appear at ~/LangSchool/Invoices/<YYYY>/<MM>/<NUMBER>.pdf.

Troubleshooting

  * Broken/empty PDF text → check the two DejaVu TTF files in ~/LangSchool/Fonts/.
  
  * No drafts → fill attendance for the month or add subscriptions.
  
  * Wails runtime issues → run wails doctor and install suggested deps.

## Documentation

- **[THESIS.md](THESIS.md)** - Complete Bachelor's thesis documentation for University of Latvia
- **[PROJECT_MAP.md](PROJECT_MAP.md)** - Project structure and architecture overview
- **[TESTING.md](TESTING.md)** - Quick testing guide
- **[docs/REQUIREMENTS.md](docs/REQUIREMENTS.md)** - Detailed software requirements specification
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System architecture and design documentation
- **[docs/TESTING_PROCEDURES.md](docs/TESTING_PROCEDURES.md)** - Comprehensive testing procedures

