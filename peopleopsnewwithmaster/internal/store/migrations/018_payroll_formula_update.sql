-- Align the configured salary algorithm with the approved payroll formulas.
-- Idempotent: re-running this migration updates the six canonical calculation steps.
INSERT INTO dcs_salary_calculation_algorithm(step_order, step_name, formula, enabled, config) VALUES
 (1, 'Total Pay', 'gross_pay_minor + variable_pay_minor + allowances_minor - deductions_minor', TRUE,
  '{"implementation":"payroll_v2","source":"dsc_employee_salary_breakup.gross_pay_minor,variable_pay_minor,allowances_minor,deductions_minor","units":"minor_currency_units"}'),
 (2, 'Overtime Hours', 'MIN(SUM(OT check-in/check-out duration), SUM(assigned OT shift duration - break + grace minutes)) / 60', TRUE,
  '{"implementation":"payroll_v2","source":"attendance_events,shift_assignments,shifts","units":"hours","role":"Employee"}'),
 (3, 'Overtime', 'hourly_rate_minor * overtime_multiplier * OvertimeHours', TRUE,
  '{"implementation":"payroll_v2","source":"dsc_employee_salary_breakup.hourly_rate_minor,overtime_multiplier","units":"minor_currency_units"}'),
 (4, 'Leave Encashment', 'Terminated employees only: remaining_leave_days * 8 * hourly_rate_minor', TRUE,
  '{"implementation":"payroll_v2","source":"leave entitlement, approved annual leave, configured weekends and holidays","units":"minor_currency_units"}'),
 (5, 'Bonus', 'bonus_pay_minor when bonus_month equals the payroll calendar month; otherwise 0', TRUE,
  '{"implementation":"payroll_v2","source":"dsc_employee_salary_breakup.bonus_pay_minor,bonus_month","units":"minor_currency_units"}'),
 (6, 'Net Pay', 'TotalPay + Overtime + Bonus + LeaveEncash', TRUE,
  '{"implementation":"payroll_v2","units":"minor_currency_units"}')
ON CONFLICT(step_order) DO UPDATE SET
 step_name = EXCLUDED.step_name,
 formula = EXCLUDED.formula,
 enabled = EXCLUDED.enabled,
 config = EXCLUDED.config;
