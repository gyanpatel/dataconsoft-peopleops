-- Role catalogue and per-role page access. page_id values match the frontend route keys.
CREATE TABLE IF NOT EXISTS dcs_app_roles (
  id TEXT PRIMARY KEY,
  role_name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  system_role BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS dcs_role_pages (
  role_id TEXT NOT NULL REFERENCES dcs_app_roles(id) ON DELETE CASCADE,
  page_id TEXT NOT NULL,
  page_name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY(role_id,page_id)
);

CREATE TABLE IF NOT EXISTS dcs_employees_actual_hours (
  id TEXT PRIMARY KEY,
  employee_id TEXT NOT NULL REFERENCES employees(id),
  work_date DATE NOT NULL,
  check_in_at TIMESTAMPTZ,
  check_out_at TIMESTAMPTZ,
  assigned_shift_minutes INTEGER NOT NULL DEFAULT 0 CHECK (assigned_shift_minutes >= 0),
  actual_minutes INTEGER NOT NULL DEFAULT 0 CHECK (actual_minutes >= 0),
  normal_minutes INTEGER NOT NULL DEFAULT 0 CHECK (normal_minutes >= 0),
  overtime_minutes INTEGER NOT NULL DEFAULT 0 CHECK (overtime_minutes >= 0),
  payment_pending BOOLEAN NOT NULL DEFAULT TRUE,
  payroll_period TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(employee_id,work_date)
);
CREATE INDEX IF NOT EXISTS idx_actual_hours_pending ON dcs_employees_actual_hours(payment_pending,work_date,employee_id);

CREATE TABLE IF NOT EXISTS dsc_employee_salary_breakup (
  id TEXT PRIMARY KEY,
  employee_id TEXT NOT NULL UNIQUE REFERENCES employees(id),
  gross_pay_minor BIGINT NOT NULL DEFAULT 0 CHECK (gross_pay_minor >= 0),
  hourly_rate_minor BIGINT NOT NULL DEFAULT 0 CHECK (hourly_rate_minor >= 0),
  overtime_multiplier NUMERIC(5,2) NOT NULL DEFAULT 1.5 CHECK (overtime_multiplier >= 1),
  variable_pay_minor BIGINT NOT NULL DEFAULT 0 CHECK (variable_pay_minor >= 0),
  allowances_minor BIGINT NOT NULL DEFAULT 0 CHECK (allowances_minor >= 0),
  deductions_minor BIGINT NOT NULL DEFAULT 0 CHECK (deductions_minor >= 0),
  currency CHAR(3) NOT NULL DEFAULT 'AED',
  effective_from DATE NOT NULL DEFAULT CURRENT_DATE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS dcs_salary_calculation_algorithm (
  id BIGSERIAL PRIMARY KEY,
  step_order INTEGER NOT NULL UNIQUE,
  step_name TEXT NOT NULL,
  formula TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  config JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS dcs_payments (
  id TEXT PRIMARY KEY,
  employee_id TEXT NOT NULL REFERENCES employees(id),
  period_start DATE NOT NULL,
  period_end DATE NOT NULL,
  entry_type TEXT NOT NULL CHECK(entry_type IN ('debit','credit')),
  status TEXT NOT NULL DEFAULT 'Pending' CHECK(status IN ('Pending','Cleared','Failed','Reversed')),
  amount_minor BIGINT NOT NULL CHECK(amount_minor >= 0),
  currency CHAR(3) NOT NULL DEFAULT 'AED',
  reference TEXT NOT NULL DEFAULT '',
  created_by TEXT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  cleared_at TIMESTAMPTZ,
  cleared_by TEXT REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_dcs_payments_employee_period ON dcs_payments(employee_id,period_start,created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_dcs_payments_period_entry_unique ON dcs_payments(employee_id,period_start,period_end,entry_type);
CREATE TABLE IF NOT EXISTS dcs_salary_summary (
  id TEXT PRIMARY KEY,
  employee_id TEXT NOT NULL REFERENCES employees(id),
  period_start DATE NOT NULL,
  period_end DATE NOT NULL,
  currency CHAR(3) NOT NULL DEFAULT 'AED',
  normal_hours NUMERIC(10,2) NOT NULL DEFAULT 0,
  overtime_hours NUMERIC(10,2) NOT NULL DEFAULT 0,
  base_pay_minor BIGINT NOT NULL DEFAULT 0,
  overtime_pay_minor BIGINT NOT NULL DEFAULT 0,
  variable_pay_minor BIGINT NOT NULL DEFAULT 0,
  allowances_minor BIGINT NOT NULL DEFAULT 0,
  deductions_minor BIGINT NOT NULL DEFAULT 0,
  gross_minor BIGINT NOT NULL DEFAULT 0,
  net_minor BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'Calculated' CHECK(status IN ('Calculated','Payment pending','Paid')),
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(employee_id,period_start,period_end)
);
CREATE TABLE IF NOT EXISTS dcs_employee_bank_details (
  id TEXT PRIMARY KEY,
  employee_id TEXT NOT NULL UNIQUE REFERENCES employees(id),
  account_name TEXT NOT NULL,
  bank_name TEXT NOT NULL DEFAULT '',
  sort_code TEXT NOT NULL,
  account_number_encrypted TEXT NOT NULL,
  iban_encrypted TEXT NOT NULL DEFAULT '',
  bic_swift TEXT NOT NULL DEFAULT '',
  updated_by TEXT REFERENCES users(id),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO dcs_app_roles(id,role_name,description,system_role) VALUES
 ('role-master-admin','Master Admin','Configure application, roles and access',true),
 ('role-super-admin','Super Admin','Platform administration',true),
 ('role-hr-admin','HR Admin','Manage employee records and HR operations',true),
 ('role-hr-approver','HR Approver','Review HR requests',true),
 ('role-accounts','Accounts / Payroll','Prepare payroll and issue payments',true),
 ('role-accounts-approver','Accounts Approver','Clear and approve payments',true),
 ('role-employee','Employee','View own information, attendance and payslips',true),
 ('role-manager','Department Manager','Manage team and approvals',true),
 ('role-shift-supervisor','Shift Supervisor','Manage shifts and assignments',true),
 ('role-auditor','Auditor / Read-only','Read-only audit and reports',true)
ON CONFLICT(role_name) DO NOTHING;
INSERT INTO dcs_role_pages(role_id,page_id,page_name,enabled)
SELECT r.id,p.page_id,p.page_name,
 CASE WHEN r.role_name IN ('Master Admin','Super Admin') THEN TRUE
      WHEN r.role_name='Employee' THEN p.page_id IN ('dashboard','my-profile','attendance','leave-management','payroll','documents')
      WHEN r.role_name IN ('Accounts / Payroll','Accounts Approver') THEN p.page_id IN ('dashboard','payroll','employees','reports-analytics')
      WHEN r.role_name IN ('HR Admin','HR Approver') THEN p.page_id IN ('dashboard','employees','attendance','shift-management','leave-management','payroll','documents','approvals','reports-analytics')
      WHEN r.role_name='Shift Supervisor' THEN p.page_id IN ('dashboard','employees','attendance','shift-management','shift-assignments')
      WHEN r.role_name='Auditor / Read-only' THEN p.page_id IN ('dashboard','payroll','reports-analytics','audit-log')
      ELSE p.page_id IN ('dashboard','employees','attendance','shift-management','leave-management','approvals','reports-analytics') END
FROM dcs_app_roles r CROSS JOIN (VALUES
 ('dashboard','Dashboard'),('employees','Employees'),('attendance','Attendance'),('shift-management','Shift management'),('shift-assignments','Shift assignments'),('leave-management','Leave management'),('payroll','Payroll'),('documents','Documents'),('approvals','Approvals'),('reports-analytics','Reports & analytics'),('audit-log','Audit log'),('settings','Settings'),('my-profile','My profile')
) p(page_id,page_name) ON CONFLICT(role_id,page_id) DO NOTHING;
INSERT INTO dcs_salary_calculation_algorithm(step_order,step_name,formula,config) VALUES
 (1,'Normal time pay','normal_minutes / 60 * hourly_rate_minor','{"description":"Normal pay uses assigned shift hours capped at actual worked hours"}'),
 (2,'Overtime pay','overtime_minutes / 60 * hourly_rate_minor * overtime_multiplier','{"description":"Overtime applies only to minutes above assigned shift duration"}'),
 (3,'Gross pay','base_pay + overtime_pay + variable_pay + allowances','{}'),
 (4,'Net pay','gross_pay - deductions','{}')
ON CONFLICT(step_order) DO NOTHING;

-- Bootstrap salary breakups for employees already in the directory. Annual salary is converted
-- to an indicative hourly rate using 2,080 standard hours/year; HR/Accounts should verify it.
INSERT INTO dsc_employee_salary_breakup(id,employee_id,overtime_multiplier,currency,effective_from)
SELECT 'salary-' || e.id,e.id,1.5, 'AED',CURRENT_DATE
FROM employees e
ON CONFLICT(employee_id) DO NOTHING;
