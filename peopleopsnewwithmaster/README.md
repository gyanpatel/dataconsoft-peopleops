# PeopleOps HRMS — configurable HR platform

PeopleOps is a React/TypeScript + Vite frontend, Go REST API, and PostgreSQL application. This update adds a Master Admin configuration console backed by `dcs_app_master`, database-backed employee custom fields, and a complete document vault backed by `dcs_doc_store` using PostgreSQL `BYTEA` for the original file bytes.

> **Production status:** This is a production-oriented implementation, not a certified HR/payroll product. Complete the security review, configure TLS, backups, secrets, retention, and jurisdiction-specific payroll rules before using sensitive live data. Payroll includes a calculation and payment ledger workflow, but it is not a statutory payroll engine.

## Features

- React + TypeScript + Vite responsive frontend.
- Go REST API and PostgreSQL using `pgx/v5`.
- Master Admin role can create, edit, hide/show, reorder and delete configurable page fields.
- Initial `dcs_app_master` entries for Employees, Attendance, Shift management, Leave management, Payroll, Documents, Approvals, Reports & analytics, Audit log and Settings.
- Employee creation form is generated from visible `dcs_app_master` employee fields. Extra configured values are stored in `employees.custom_fields` (`JSONB`).
- Document upload, metadata, expiry dates, verification/rejection, authenticated download and employee-scoped self-service. The binary content is stored in `dcs_doc_store.doc_file BYTEA`.
- Supported document extensions: PDF, PNG, JPG/JPEG, DOCX, XLSX and TXT. Maximum file size is 15 MB.
- Employee directory, detail/profile editing and photo support retained; employee profile images remain in `dcs_doc_store`.
- Database-backed roles (`dcs_app_roles`) and page permissions (`dcs_role_pages`), enforced in the API and used to filter navigation after sign-in.
- Check-out calculates daily actual, normal and overtime minutes in `dcs_employees_actual_hours`, using the assigned shift duration minus break time.
- Salary breakup and calculation steps, monthly salary summaries, salary-slip PDF generation, payment debit/credit ledger, and separate employee vs Accounts payroll views.
- Employee bank details are stored in `dcs_employee_bank_details`; account number and IBAN are AES-GCM encrypted using `BANK_DETAILS_KEY`.
- Database migrations are embedded in the Go binary and applied at startup. Migrations are idempotent for existing installations.

## Quick start with Docker

1. Copy the environment template and generate strong secrets:

   ```sh
   cp .env.example .env
   ```

   Set `POSTGRES_PASSWORD` to a long random alphanumeric value, `SESSION_SECRET` to at least 32 random bytes, and `BANK_DETAILS_KEY` to a separate high-entropy secret. For example, generate each with `openssl rand -hex 32`. Keep `.env` private and never commit it.

2. Start PostgreSQL, API and frontend:

   ```sh
   docker compose up --build -d
   docker compose ps
   ```

3. Open <http://localhost:8088>.

On a new database, demo login is `admin@peopleops.local` / `ChangeMe123!`. Change the seeded password mechanism before production. On an upgraded database, the seeded admin account is promoted to `Master Admin` by migration so it can configure the application.

## Local development

Requirements: Go 1.25+, Node.js 20+, npm, and PostgreSQL 16+.

1. Create a PostgreSQL database and user, then set `DATABASE_URL` in your environment or `.env`.
2. Set `SESSION_SECRET` to a value of at least 32 characters.
3. Start the API from the repository root:

   ```sh
   go mod download
   go run ./cmd/server
   ```

4. In another terminal:

   ```sh
   cd web
   npm ci
   npm run dev
   ```

5. Open <http://localhost:5173>. Vite proxies `/api` to `http://localhost:8080`.

## Master Admin configuration

Sign in as Master Admin and open **Settings → Application configuration**. Choose a page, then add or edit a field. Supported types are `text`, `textarea`, `number`, `date`, `email`, `select`, `checkbox`, and `url`. Fields include label, field key, required/visible flags, display order, select options and help text. Page configuration is stored in PostgreSQL table `dcs_app_master`.

Employee creation currently renders the configured employee fields dynamically. Additional employee values are persisted in `employees.custom_fields` as JSONB. The other pages have their seeded configuration metadata and can be managed in the same console; each page's bespoke workflow still uses its domain-specific UI and validation.

## Document storage

`dcs_doc_store` stores metadata and binary content in PostgreSQL. The `doc_file` column is `BYTEA`; files are not exposed through a public static directory. Downloads require an authenticated session and are restricted to HR document roles or the employee who owns the document. HR roles can upload for any employee and set verification status; an employee can upload only to their own linked employee record.

The table stores employee ID, original file name, category, MIME type, byte size, issue/expiry dates, verification status, notes, uploader and timestamps. Back up PostgreSQL with your normal database backup process; the binary files are included in database backups.

## API additions

- `GET /api/v1/app-master[?page=employees]` — list configuration; any authenticated user may read field definitions.
- `POST /api/v1/app-master` — create a field (Master Admin only).
- `PUT /api/v1/app-master` — update a field (Master Admin only).
- `DELETE /api/v1/app-master?id=123` — delete a field (Master Admin only).
- `GET /api/v1/documents` — list permitted document metadata.
- `POST /api/v1/documents` — multipart upload (HR roles or employee self-service).
- `GET /api/v1/documents/{id}/download` — authenticated download with record-level authorization.
- `POST /api/v1/documents/{id}/decision` — set `Pending`, `Verified` or `Rejected` (HR roles only).
- `GET /api/v1/health` — database health check.

## Validation

```sh
go test ./...
go vet ./...
cd web && npm ci && npm run build
```

Integration tests require a disposable PostgreSQL database and `DATABASE_URL`. In this environment, dependency downloads may be unavailable; rerun the checks in CI or a network-enabled development environment. Review `docs/PRODUCTION_READINESS.md` before deployment.

## Roles, attendance hours and payroll

- `GET/POST/PUT/DELETE /api/v1/roles` — role catalogue; Master Admin only. Built-in system roles are protected.
- `GET/PUT /api/v1/role-pages?roleId=...` — configure page availability per role. Changes apply at the next sign-in.
- `GET /api/v1/actual-hours?month=YYYY-MM` — daily worked, normal and overtime hours for HR/Accounts/shift managers.
- `GET/PUT /api/v1/employees/{id}/salary-breakup` — salary configuration for HR and Accounts.
- `GET /api/v1/salary-algorithm` — view the configured salary calculation steps.
- `GET /api/v1/payroll` — employee's own paid salary summaries or the Accounts payment ledger, depending on role.
- `GET /api/v1/payroll/summaries` — payroll summaries for Accounts.
- `POST /api/v1/payroll/calculate` — calculate salary for a date range from payment-pending hours.
- `POST /api/v1/payroll/issue` — create debit records after salary review.
- `POST /api/v1/payroll/payments/{id}/clear` — Accounts Approver confirms settlement; creates a credit record and marks hours/salary summary paid.
- `GET /api/v1/payroll/{id}/payslip.pdf` — authenticated employee-owned or Accounts salary slip PDF.
- `GET/PUT /api/v1/employees/{id}/bank-details` — bank details for HR/Accounts roles; encryption requires `BANK_DETAILS_KEY`.

Payroll is deliberately split into calculate, issue and clear steps so the application does not mark a transfer paid before the payment provider confirms settlement.

### Employee manager, hierarchy and onboarding
- Employee creation supports an optional manager selected from existing employee records; the employee's `manager_id` stores the selected employee ID.
- The Employee hierarchy page is available from the main navigation and reads reporting relationships from `/api/v1/employee-hierarchy`.
- Creating an employee also creates a linked `Employee` role login with a cryptographically random temporary password. SMTP must be configured with `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, and optional `SMTP_USERNAME` / `SMTP_PASSWORD`. Account creation is rolled back if the welcome email cannot be sent.
- Temporary-password accounts must change their password before using the rest of the application. New migration `005_employee_hierarchy_password.sql` adds this flag.
- Employees do not see workforce summary metrics or Add employee actions; employee creation is also denied server-side for the Employee role.
