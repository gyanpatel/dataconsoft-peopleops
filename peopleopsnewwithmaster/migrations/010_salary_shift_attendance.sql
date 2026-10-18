-- Salary and shift classifications plus device location support.
ALTER TABLE dsc_employee_salary_breakup ADD COLUMN IF NOT EXISTS base_annual_pay_minor BIGINT NOT NULL DEFAULT 0 CHECK (base_annual_pay_minor >= 0);
ALTER TABLE dsc_employee_salary_breakup ADD COLUMN IF NOT EXISTS bonus_pay_minor BIGINT NOT NULL DEFAULT 0 CHECK (bonus_pay_minor >= 0);
ALTER TABLE dsc_employee_salary_breakup ADD COLUMN IF NOT EXISTS bonus_month SMALLINT NOT NULL DEFAULT 12 CHECK (bonus_month BETWEEN 1 AND 12);
UPDATE dsc_employee_salary_breakup SET base_annual_pay_minor=gross_pay_minor WHERE base_annual_pay_minor=0 AND gross_pay_minor>0;
UPDATE dsc_employee_salary_breakup SET hourly_rate_minor=ROUND(base_annual_pay_minor::numeric/(12*30*8))::bigint WHERE hourly_rate_minor=0 AND base_annual_pay_minor>0;
ALTER TABLE shifts ADD COLUMN IF NOT EXISTS shift_type TEXT NOT NULL DEFAULT 'NT' CHECK (shift_type IN ('NT','OT'));
ALTER TABLE attendance_events ADD COLUMN IF NOT EXISTS shift_type TEXT NOT NULL DEFAULT 'NT' CHECK (shift_type IN ('NT','OT'));
CREATE INDEX IF NOT EXISTS idx_attendance_shift_type_time ON attendance_events(shift_type,occurred_at);

-- Store the monthly salary calculation as auditable, ordered algorithm steps.
INSERT INTO dcs_salary_calculation_algorithm(step_order,step_name,formula,enabled,config) VALUES
 (1,'Monthly overtime hours','MIN(approved OT attendance hours, assigned OT shift hours + assigned grace hours)','true','{"unit":"hours","source":"attendance_events + shift_assignments + shifts","cap_required":true}'),
 (2,'Monthly overtime pay','hourly_rate_minor * capped_overtime_hours * overtime_multiplier','true','{"unit":"minor_currency_units","source":"dsc_employee_salary_breakup.hourly_rate_minor,overtime_multiplier"}'),
 (3,'Monthly base salary','(base_annual_pay_minor + variable_pay_minor + allowances_minor - deductions_minor) / 12','true','{"unit":"minor_currency_units","source":"dsc_employee_salary_breakup"}'),
 (4,'Monthly gross salary','monthly_base_salary + monthly_overtime_pay','true','{"unit":"minor_currency_units"}'),
 (5,'Monthly bonus','bonus_pay_minor when current month equals bonus_month; otherwise zero','true','{"unit":"minor_currency_units","source":"dsc_employee_salary_breakup.bonus_pay_minor,bonus_month"}'),
 (6,'Monthly salary payable','monthly_gross_salary + applicable_bonus','true','{"unit":"minor_currency_units"}')
ON CONFLICT(step_order) DO UPDATE SET step_name=EXCLUDED.step_name,formula=EXCLUDED.formula,enabled=EXCLUDED.enabled,config=EXCLUDED.config;
