# Employee, leave, roles and shift update

This change implements the approved usability and workflow items. It does not create or elevate an administrator account and does not change employee payslip publication, payroll approval or payment-settlement behavior.

## Included

- One employee editor and **Save all changes** action. Employee/profile/photo, changed salary fields and changed bank fields are saved in one PostgreSQL transaction. Invalid financial fields roll back the entire update. Unchanged financial sections are not submitted; account encryption remains in place.
- Accounts users retain their existing financial-edit access through the same editor. Employment fields remain read-only for users without employee-management permission; a dedicated transactional financial-save endpoint rejects employment fields.
- Documents use an employee autocomplete backed by case-insensitive server search for name, employee number and email. Upload requires an actual employee selection.
- **Roles & Permissions** has clickable role names, a complete server-provided module catalogue, **Available Modules**, and an **Allowed Actions** editor. Existing administrator action permissions are protected. Module availability and action permissions remain distinct.
- Shared control styling and responsive form layouts, including keyboard-operable employee search and shift cards.
- Leave days are read-only, calculated from the selected inclusive date range using configured weekends and holidays, and recalculated by the server at submission. This follows the existing working-day balance policy; it does not introduce monthly accruals or a new UAE leave-policy engine.
- Bulk leave decisions operate on explicit request IDs, report individual results, reject self-decisions and already-decided requests, and write an audit event per successful decision. Department-level reviewers are constrained to their reporting tree; existing HR reviewers retain organization-wide scope.
- Individual and bulk leave decisions use the same transaction-protected service. Duplicate IDs in a bulk request are processed once. Individual decision buttons are disabled while processing.
- Shift creation/listing exposes **Regular** and **Overtime**. Punches within an assigned shift inherit its type; checkout preserves the check-in type. Regular assigned minutes are aggregated when multiple shifts exist.
- Clickable shift-template cards open an employee multi-selection panel with department search, date/date range, conflict preview and per-employee/per-date results. Maximum 31 dates and 500 assignments per batch. Existing assignments are never silently replaced. Conflicts include overnight overlaps on adjacent dates.

## Database deployment requirement

Apply `migrations/019_shift_assignment_sessions.sql` before using same-day regular and overtime assignments. It removes the former one-assignment-per-employee-per-day constraint and adds employee/date/template uniqueness. It preserves existing records and is safe to apply more than once.

The identical SQL is supplied in `internal/store/migrations/` for embedded distribution. The repository's existing startup migration list is empty, so the new file is **not automatically applied**. Apply this specific migration through the deployment database change process, then rebuild/restart the API and deploy the rebuilt frontend. No live database was modified during this work.

## Verification

- TypeScript checking and Vite production build passed.
- Go backend build passed.
- Existing password/session and role helper tests passed.
- New PostgreSQL regression tests passed against a disposable local instance: one employee's second leave request remains pending when the first is approved; duplicate decisions and self-approval are rejected; batch IDs are deduplicated; invalid bank fields roll back employee and salary changes; successful bank writes are encrypted; shift previews do not write data; regular/overtime shifts coexist; duplicate, overlapping and overnight-conflicting shifts are rejected. The new migration was applied twice in the isolated test setup to verify idempotence.
- Headless browser tests with synthetic API fixtures passed for unified employee payloads, document search, single and bulk leave decisions, read-only duration, shift-type selection, batch assignment, and role module changes. Screens were inspected at desktop and narrow viewport sizes. These browser fixtures are not live-service integration tests.
- `go vet ./...` still reports the pre-existing unreachable SMTP code after the unconditional `return nil` in `sendWelcomeCredentials`. That unrelated behavior was not changed.

Run the isolated regression tests with a disposable PostgreSQL connection:

```powershell
$env:HRMS_TEST_DATABASE_URL = 'postgres://test_user@127.0.0.1:55447/postgres?sslmode=disable'
go test ./internal/api -run 'TestWorkflow|TestWorkingDays' -v
```

Each database test creates and drops its own uniquely named schema. Do not point the test variable at a production database.

## Leave incident finding

The reported "approving one request approved both" behavior was not reproduced in the reviewed source. The original SQL already targeted a request primary key. The implementation now shares a locked, request-specific decision service across individual and batch actions, with regression coverage and visible request references. Diagnosing the historical incident still requires the affected request IDs, audit records and deployed application version; no historical leave records were altered.

## Limits retained

The existing single-stage approval model remains. Bulk selection means deciding multiple requests; it does not implement the separate six-level sequential approval engine from the original scope. Payroll, pension, WPS and Business Central work is outside this update.
