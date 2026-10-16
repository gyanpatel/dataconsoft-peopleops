# Production readiness checklist

This codebase is a functional pilot foundation. Do not process real payroll or upload sensitive employee documents until the items below are completed and independently reviewed.

## Required before real employee data

- [ ] Replace demo login and remove seeded demo users; implement controlled account provisioning, password reset, session revocation and MFA for privileged roles.
- [ ] Set a unique secret using a secret manager. Never commit `.env`; rotate secrets on a defined schedule.
- [ ] Serve only over HTTPS and confirm `Secure`, `HttpOnly`, `SameSite` cookies; add CSRF defenses appropriate to the chosen session model, origin validation and rate limiting.
- [ ] Review every endpoint's RBAC, organization/department scope, employee record scope and field-level access. Current employee listing returns salary fields to all authenticated roles and must be changed before production.
- [ ] Add salary/bank/tax data as separately authorized resources, mask sensitive fields and audit access to sensitive records and downloads.
- [ ] Add server-side schema validation, request-size limits with safe error handling, request tracing and structured log redaction.
- [ ] Review audit storage and retention. PostgreSQL audit rows are application append-only by convention, not cryptographically immutable or tamper-proof.
- [ ] Define privacy notice, lawful basis, location policy, retention schedule, employee access/deletion processes and breach-response runbook.
- [ ] Configure automated encrypted backups and test restore into a clean environment. Keep backups off-host and test data integrity.
- [ ] Pin and scan dependencies, run static analysis, fuzz input parsers, and perform independent penetration testing.

## Payroll is not production-ready

- [ ] Specify and validate the applicable payroll jurisdiction, tax year, tax/pension/statutory rules and effective dates with a qualified payroll professional.
- [ ] Implement exact monetary arithmetic, versioned reproducible calculations, anomaly review, approval, locking, reversal and correction records.
- [ ] Build employee-level payslips, access-controlled PDF delivery and payment reconciliation. Demo payroll values are fictional summaries only.
- [ ] Validate overtime and salary-divisor policy; avoid treating the included sample values as legal or contractual advice.

## Workflow and attendance gaps

- [ ] Implement the reusable configurable workflow engine described in the product specification (step definitions, versioned routing, delegation, escalation, SLA reminders, return-for-correction and history).
- [ ] Add manager/department scoping; the current simplified manager role can view requests broadly and must be narrowed to reporting scope.
- [ ] Add shift templates, time zones, work calendars, breaks, grace periods, timesheets, missed-punch corrections and approved overtime calculations.
- [ ] GPS, geofence checks, location accuracy checks, offline attendance and location-retention controls are not implemented. Browser geolocation is untrusted and must not be treated as conclusive proof.

## Documents, notifications and integrations

- [ ] Document uploads are stored in PostgreSQL BYTEA and downloaded through authenticated handlers with extension/size validation and employee scoping. Before production, add malware scanning, document versioning, retention enforcement and automated expiry alerts. Do not serve uploads from a public static directory.
- [ ] Implement durable outbox/job processing, idempotent schedules, retries, delivery tracking, email templates and notification preferences.
- [ ] Add AppDynamics adapter only for API operations supported by the actual deployment, including timeouts, backoff, rate limits and secret redaction. It must not block core HR functions.
- [ ] Add an AI provider abstraction with authorization before retrieval, prompt-injection defenses, cost limits, confirmation for sensitive actions and audit records. No AI provider is currently connected.

## Operations and scale

- [ ] Set resource limits, run as a non-root user, restrict network access and configure external TLS termination.
- [ ] Monitor API errors, DB latency, job failures, disk space, backup freshness and auth abuse; create incident alerts.
- [ ] PostgreSQL is the application database. Use persistent storage, automated backups, restricted network access, short transactions and load tests for expected concurrency.
- [ ] Add database migration version tracking/rollback guidance, staging deployment, smoke tests, rollback plan and recovery objectives.
- [ ] Configure CORS origin to the actual public origin. The sample Docker deployment assumes `http://localhost:8088`; update this for the deployment.
- [ ] Run a full keyboard/screen-reader and mobile viewport accessibility review.

## Feature status

| Feature | Current status |
|---|---|
| Responsive dashboard / navigation | Implemented |
| Local login and signed session token | Implemented starter; hardening required |
| PostgreSQL schema and fictional seed | Implemented |
| Employee directory and create | Implemented starter; field scoping required |
| Leave request submission and decisions | Implemented starter; configurable workflows not implemented |
| Check-in/check-out | Implemented starter; shifts, geofences and location not implemented |
| Payroll period summary | Demo-only; calculation and payslips not implemented |
| Documents | Upload, BYTEA storage, authenticated download and verification implemented; malware scanning, versioning, retention and expiry alerts remain |
| Email notification queue | Not implemented |
| AppDynamics | Not implemented; integration point documented |
| AI HR assistant | Not implemented; disabled-state guidance provided |
| MFA / OIDC / SSO | Not implemented |
| Master Admin app-field configuration | Implemented for `dcs_app_master`; employee creation is dynamic, other pages currently use their domain-specific UI |
| Full reporting / analytics | Headcount CSV only plus dashboard summaries |


## Application configuration and document storage update

The Master Admin configuration console is backed by `dcs_app_master`. Employee custom fields are persisted in `employees.custom_fields` JSONB. The document vault stores binary content in `dcs_doc_store.doc_file BYTEA` and serves files only through authenticated download endpoints. These features still require organization-specific retention policy, malware scanning, backup/restore testing, and security review before production use.

## Payroll, role access and bank-detail controls

- Migration `004_roles_hours_payroll.sql` adds the role catalogue (`dcs_app_roles`), per-role page access (`dcs_role_pages`), daily actual-hours ledger (`dcs_employees_actual_hours`), salary breakup (`dsc_employee_salary_breakup`), salary algorithm definitions (`dcs_salary_calculation_algorithm`), payment ledger (`dcs_payments`), monthly salary summaries (`dcs_salary_summary`) and employee bank details (`dcs_employee_bank_details`).
- Role/page access is enforced by the API as well as filtered in the frontend. After changing page access, sign out and back in to refresh the user's page list.
- Configure `BANK_DETAILS_KEY` with a high-entropy secret in a secret manager before collecting bank details. The key is required to encrypt/decrypt account numbers and IBANs. Plan key rotation and backups carefully; loss of this key means encrypted values cannot be decrypted.
- Payroll calculations here are an auditable gross-to-net *framework*, not a jurisdiction-compliant payroll engine. Tax, National Insurance, pension, statutory leave, minimum-wage, rounding, and payment-provider integration require separate implementation and validation. Use a payment-provider confirmation before marking a debit cleared; clearing creates the matching credit ledger entry.
- The generated salary-slip PDF is a simple printable summary. Validate layout, retention, access controls, and any legal payslip requirements before production.
