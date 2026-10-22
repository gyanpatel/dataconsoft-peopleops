-- Allow a regular shift and a separate non-overlapping overtime shift on one date.
ALTER TABLE shift_assignments DROP CONSTRAINT IF EXISTS shift_assignments_employee_id_shift_date_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_shift_assignment_template_date ON shift_assignments(employee_id,shift_date,shift_id);
