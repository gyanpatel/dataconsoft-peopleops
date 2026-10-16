-- Database-driven action permissions. Role assignments are configurable without code changes.
CREATE TABLE IF NOT EXISTS dcs_role_permissions (
  role_id TEXT NOT NULL REFERENCES dcs_app_roles(id) ON DELETE CASCADE,
  permission_key TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY(role_id, permission_key)
);

-- Default permission matrix is seeded as data; administrators can change it in the database.
WITH defaults(role_name, permission_key) AS (VALUES
 ('Master Admin','roles.manage'),('Super Admin','roles.manage'),
 ('Master Admin','app.configure'),
 ('Master Admin','documents.manage'),('Super Admin','documents.manage'),('HR Admin','documents.manage'),('HR Approver','documents.manage'),
 ('Master Admin','employees.manage'),('Super Admin','employees.manage'),('HR Admin','employees.manage'),
 ('Master Admin','shift.manage'),('Super Admin','shift.manage'),('HR Admin','shift.manage'),('Shift Supervisor','shift.manage'),
 ('Master Admin','shift.assign'),('Super Admin','shift.assign'),('HR Admin','shift.assign'),('Department Manager','shift.assign'),('Shift Supervisor','shift.assign'),
 ('Master Admin','leave.review'),('Super Admin','leave.review'),('HR Admin','leave.review'),('HR Approver','leave.review'),('Department Manager','leave.review'),
 ('Master Admin','payroll.manage'),('Super Admin','payroll.manage'),('Accounts / Payroll','payroll.manage'),('Accounts Approver','payroll.manage'),
 ('Master Admin','payroll.issue'),('Super Admin','payroll.issue'),('Accounts / Payroll','payroll.issue'),
 ('Master Admin','payroll.approve'),('Super Admin','payroll.approve'),('Accounts Approver','payroll.approve'),
 ('Master Admin','hr.payroll.read'),('Super Admin','hr.payroll.read'),('HR Admin','hr.payroll.read'),('Accounts / Payroll','hr.payroll.read'),('Accounts Approver','hr.payroll.read'),
 ('Master Admin','reports.read'),('Super Admin','reports.read'),('HR Admin','reports.read'),('HR Approver','reports.read'),('Accounts / Payroll','reports.read'),('Accounts Approver','reports.read'),('Auditor / Read-only','reports.read'),('Department Manager','reports.read'),
 ('Master Admin','audit.read'),('Super Admin','audit.read'),('Auditor / Read-only','audit.read'),
 ('Master Admin','attendance.manage'),('Super Admin','attendance.manage'),('HR Admin','attendance.manage'),('Shift Supervisor','attendance.manage'),
 ('Master Admin','attendance.team'),('Super Admin','attendance.team'),('HR Admin','attendance.team'),('Department Manager','attendance.team'),('Shift Supervisor','attendance.team')
)
INSERT INTO dcs_role_permissions(role_id,permission_key,enabled)
SELECT r.id,d.permission_key,true FROM defaults d JOIN dcs_app_roles r ON r.role_name=d.role_name
ON CONFLICT(role_id,permission_key) DO NOTHING;
