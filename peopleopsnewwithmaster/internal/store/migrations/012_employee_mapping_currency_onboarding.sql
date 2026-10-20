-- Application field mapping, global currency catalogue, onboarding activation and employee data normalization.
ALTER TABLE dcs_app_master ADD COLUMN IF NOT EXISTS field_mapping TEXT NOT NULL DEFAULT '';
ALTER TABLE dcs_app_master ADD COLUMN IF NOT EXISTS min_length INTEGER;
ALTER TABLE dcs_app_master ADD COLUMN IF NOT EXISTS max_length INTEGER;
ALTER TABLE dcs_app_master ADD CONSTRAINT dcs_app_master_length_bounds CHECK ((min_length IS NULL OR min_length >= 0) AND (max_length IS NULL OR max_length >= 0) AND (min_length IS NULL OR max_length IS NULL OR min_length <= max_length)) NOT VALID;

CREATE TABLE IF NOT EXISTS dcs_app_currency (
  currency_code CHAR(3) PRIMARY KEY CHECK (currency_code = upper(currency_code)),
  currency_name TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO dcs_app_currency(currency_code,currency_name,active,is_default) VALUES ('GBP','Pound sterling',TRUE,TRUE) ON CONFLICT(currency_code) DO UPDATE SET active=TRUE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_dcs_app_currency_single_default ON dcs_app_currency(is_default) WHERE is_default=TRUE;

-- Salary and bank data are stored in their dedicated tables, not employees.
ALTER TABLE dsc_employee_salary_breakup ADD COLUMN IF NOT EXISTS base_annual_pay_minor BIGINT NOT NULL DEFAULT 0;
UPDATE dsc_employee_salary_breakup SET base_annual_pay_minor=gross_pay_minor WHERE base_annual_pay_minor=0 AND gross_pay_minor>0;
ALTER TABLE employees DROP COLUMN IF EXISTS salary_minor;
ALTER TABLE employees DROP COLUMN IF EXISTS currency;

-- Existing records stay as-is; new employees are inactive until HR verifies onboarding documents.
ALTER TABLE employees ALTER COLUMN status SET DEFAULT 'Inactive';
UPDATE employees SET status='Inactive' WHERE status IS NULL OR status='';

-- Canonical document types used by the activation gate.
UPDATE dcs_app_master SET options='["Passport","Emirates ID","IBAN letter from bank","Contract","Right to work","Qualification","Other"]'::jsonb WHERE page_id='documents' AND field_key='documentType';
