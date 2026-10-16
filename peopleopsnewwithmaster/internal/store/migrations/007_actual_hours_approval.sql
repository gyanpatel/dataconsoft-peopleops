-- Explicit review workflow for actual hours and overtime. Payroll only consumes approved rows.
ALTER TABLE dcs_employees_actual_hours
  ADD COLUMN IF NOT EXISTS approved BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS approved_by TEXT,
  ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS approval_note TEXT;

WITH defaults(role_name, permission_key) AS (VALUES
 ('Master Admin','attendance.approve'),('Super Admin','attendance.approve'),
 ('HR Admin','attendance.approve'),('HR Approver','attendance.approve'),
 ('Shift Supervisor','attendance.approve')
)
INSERT INTO dcs_role_permissions(role_id,permission_key,enabled)
SELECT r.id,d.permission_key,true FROM defaults d JOIN dcs_app_roles r ON lower(r.role_name)=lower(d.role_name)
ON CONFLICT(role_id,permission_key) DO NOTHING;
