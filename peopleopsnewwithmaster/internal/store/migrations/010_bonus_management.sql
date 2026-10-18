ALTER TABLE dsc_employee_salary_breakup
  ADD COLUMN IF NOT EXISTS bonus_pay_minor BIGINT NOT NULL DEFAULT 0 CHECK (bonus_pay_minor >= 0),
  ADD COLUMN IF NOT EXISTS bonus_month SMALLINT NOT NULL DEFAULT 12 CHECK (bonus_month BETWEEN 1 AND 12);

CREATE TABLE IF NOT EXISTS dcs_bonus_history (
  id TEXT PRIMARY KEY,
  employee_id TEXT NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
  bonus_pay_minor BIGINT NOT NULL CHECK (bonus_pay_minor >= 0),
  bonus_month SMALLINT NOT NULL CHECK (bonus_month BETWEEN 1 AND 12),
  updated_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dcs_bonus_history_employee_created
  ON dcs_bonus_history(employee_id, created_at DESC);

-- Make the page available in role navigation; bonus API still enforces write permissions.
INSERT INTO dcs_role_pages(role_id,page_id,page_name,enabled)
SELECT r.id,'bonus-management','Bonus management',true
FROM dcs_app_roles r
ON CONFLICT(role_id,page_id) DO UPDATE SET page_name=EXCLUDED.page_name, enabled=true;
