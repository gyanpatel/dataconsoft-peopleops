-- Make the configured monthly salary algorithm match the approved business formula.
-- Safe to re-run: columns are guarded and algorithm rows are upserted by step_order.
ALTER TABLE dsc_employee_salary_breakup
  ADD COLUMN IF NOT EXISTS base_annual_pay_minor BIGINT NOT NULL DEFAULT 0 CHECK (base_annual_pay_minor >= 0),
  ADD COLUMN IF NOT EXISTS bonus_pay_minor BIGINT NOT NULL DEFAULT 0 CHECK (bonus_pay_minor >= 0),
  ADD COLUMN IF NOT EXISTS bonus_month SMALLINT NOT NULL DEFAULT 12 CHECK (bonus_month BETWEEN 1 AND 12);

UPDATE dsc_employee_salary_breakup
SET base_annual_pay_minor = gross_pay_minor
WHERE base_annual_pay_minor = 0 AND gross_pay_minor > 0;

ALTER TABLE shifts
  ADD COLUMN IF NOT EXISTS shift_type TEXT NOT NULL DEFAULT 'NT' CHECK (shift_type IN ('NT','OT'));
ALTER TABLE attendance_events
  ADD COLUMN IF NOT EXISTS shift_type TEXT NOT NULL DEFAULT 'NT' CHECK (shift_type IN ('NT','OT'));

INSERT INTO dcs_salary_calculation_algorithm(step_order,step_name,formula,enabled,config) VALUES
 (1,'Capped monthly overtime hours','MIN(SUM(OT check-in/check-out duration), SUM(assigned OT shift duration - break + grace minutes)) / 60',TRUE,
  '{"implementation":"monthly_v1","source":"attendance_events,shift_assignments,shifts","units":"hours","cap":"assigned OT shift duration plus grace minutes"}'),
 (2,'Monthly overtime pay','hourly_rate_minor * capped_monthly_overtime_hours * overtime_multiplier',TRUE,
  '{"implementation":"monthly_v1","source":"dsc_employee_salary_breakup.hourly_rate_minor,overtime_multiplier","units":"minor_currency_units"}'),
 (3,'Monthly base components','(base_annual_pay_minor + variable_pay_minor + allowances_minor - deductions_minor) / 12',TRUE,
  '{"implementation":"monthly_v1","source":"dsc_employee_salary_breakup","units":"minor_currency_units per month"}'),
 (4,'Monthly salary before bonus','monthly_base_components + monthly_overtime_pay',TRUE,
  '{"implementation":"monthly_v1","units":"minor_currency_units"}'),
 (5,'Applicable monthly bonus','bonus_pay_minor when payroll calendar month equals bonus_month; otherwise 0',TRUE,
  '{"implementation":"monthly_v1","source":"dsc_employee_salary_breakup.bonus_pay_minor,bonus_month","units":"minor_currency_units"}'),
 (6,'Monthly salary payable','MAX(0, monthly_salary_before_bonus + applicable_monthly_bonus)',TRUE,
  '{"implementation":"monthly_v1","units":"minor_currency_units","negative_payable":"clamp_to_zero"}')
ON CONFLICT(step_order) DO UPDATE SET
 step_name=EXCLUDED.step_name,
 formula=EXCLUDED.formula,
 enabled=EXCLUDED.enabled,
 config=EXCLUDED.config;

CREATE INDEX IF NOT EXISTS idx_attendance_shift_type_time ON attendance_events(shift_type,occurred_at);
CREATE INDEX IF NOT EXISTS idx_shift_assignments_date_employee ON shift_assignments(shift_date,employee_id);
