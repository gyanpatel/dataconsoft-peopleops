-- Persist all Master Admin field metadata, including target column mappings and length constraints.
ALTER TABLE dcs_app_master
  ADD COLUMN IF NOT EXISTS field_mapping TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS min_length INTEGER,
  ADD COLUMN IF NOT EXISTS max_length INTEGER;

-- Explicit mappings for salary breakup values. The employee create/update and payroll
-- APIs persist these fields in dsc_employee_salary_breakup, not employees.custom_fields.
INSERT INTO dcs_app_master
  (page_id,page_name,page_item,field_key,label,data_type,required,visible,display_order,options,config,help_text,field_mapping)
VALUES
  ('salary-breakup','Salary breakup','Gross pay','grossPayMinor','Gross pay (minor units)','number',true,true,10,'[]'::jsonb,'{}'::jsonb,'Gross annual pay stored in minor currency units','dsc_employee_salary_breakup.gross_pay_minor'),
  ('salary-breakup','Salary breakup','Hourly rate','hourlyRateMinor','Hourly rate (minor units)','number',false,true,20,'[]'::jsonb,'{}'::jsonb,'Calculated or configured hourly rate','dsc_employee_salary_breakup.hourly_rate_minor'),
  ('salary-breakup','Salary breakup','Overtime multiplier','overtimeMultiplier','Overtime multiplier','number',false,true,30,'[]'::jsonb,'{}'::jsonb,'Overtime pay multiplier','dsc_employee_salary_breakup.overtime_multiplier'),
  ('salary-breakup','Salary breakup','Variable pay','variablePayMinor','Variable pay (minor units)','number',false,true,40,'[]'::jsonb,'{}'::jsonb,'Variable pay component','dsc_employee_salary_breakup.variable_pay_minor'),
  ('salary-breakup','Salary breakup','Allowances','allowancesMinor','Allowances (minor units)','number',false,true,50,'[]'::jsonb,'{}'::jsonb,'Allowances component','dsc_employee_salary_breakup.allowances_minor'),
  ('salary-breakup','Salary breakup','Deductions','deductionsMinor','Deductions (minor units)','number',false,true,60,'[]'::jsonb,'{}'::jsonb,'Deductions component','dsc_employee_salary_breakup.deductions_minor'),
  ('salary-breakup','Salary breakup','Currency','currency','Currency','text',true,true,70,'[]'::jsonb,'{}'::jsonb,'Currency code from central currency catalogue','dsc_employee_salary_breakup.currency'),
  ('salary-breakup','Salary breakup','Effective from','effectiveFrom','Effective from','date',true,true,80,'[]'::jsonb,'{}'::jsonb,'Date salary breakup becomes effective','dsc_employee_salary_breakup.effective_from'),
  ('bank-details','Bank details','Account name','accountName','Account name','text',true,true,10,'[]'::jsonb,'{}'::jsonb,'Name on the bank account','dcs_employee_bank_details.account_name'),
  ('bank-details','Bank details','Bank name','bankName','Bank name','text',false,true,20,'[]'::jsonb,'{}'::jsonb,'Name of bank','dcs_employee_bank_details.bank_name'),
  ('bank-details','Bank details','Branch','sortCode','Branch','text',true,true,30,'[]'::jsonb,'{}'::jsonb,'UK bank sort code','dcs_employee_bank_details.sort_code'),
  ('bank-details','Bank details','Account number','accountNumber','Account number','text',true,true,40,'[]'::jsonb,'{}'::jsonb,'Stored encrypted at rest','dcs_employee_bank_details.account_number_encrypted'),
  ('bank-details','Bank details','IBAN','iban','IBAN','text',false,true,50,'[]'::jsonb,'{}'::jsonb,'Stored encrypted at rest','dcs_employee_bank_details.iban_encrypted'),
  ('bank-details','Bank details','BIC / SWIFT','bicSwift','BIC / SWIFT','text',false,true,60,'[]'::jsonb,'{}'::jsonb,'International bank identifier','dcs_employee_bank_details.bic_swift')
ON CONFLICT (page_id,field_key) DO UPDATE SET
  page_name=EXCLUDED.page_name,
  page_item=EXCLUDED.page_item,
  label=EXCLUDED.label,
  field_mapping=EXCLUDED.field_mapping,
  updated_at=NOW();
