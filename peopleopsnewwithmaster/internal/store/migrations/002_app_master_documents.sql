ALTER TABLE employees ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE IF NOT EXISTS dcs_app_master (
  id BIGSERIAL PRIMARY KEY,
  page_id TEXT NOT NULL,
  page_name TEXT NOT NULL,
  page_item TEXT NOT NULL,
  field_key TEXT NOT NULL,
  label TEXT NOT NULL,
  data_type TEXT NOT NULL DEFAULT 'text' CHECK (data_type IN ('text','textarea','number','date','email','select','checkbox','url')),
  required BOOLEAN NOT NULL DEFAULT FALSE,
  visible BOOLEAN NOT NULL DEFAULT TRUE,
  display_order INTEGER NOT NULL DEFAULT 100,
  options JSONB NOT NULL DEFAULT '[]'::jsonb,
  config JSONB NOT NULL DEFAULT '{}'::jsonb,
  help_text TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(page_id, field_key)
);
CREATE INDEX IF NOT EXISTS idx_dcs_app_master_page_order ON dcs_app_master(page_id, display_order, id);

CREATE TABLE IF NOT EXISTS dcs_doc_store (
  id TEXT PRIMARY KEY,
  employee_id TEXT NOT NULL REFERENCES employees(id),
  document_name TEXT NOT NULL,
  document_type TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  file_size BIGINT NOT NULL CHECK (file_size > 0 AND file_size <= 15728640),
  doc_file BYTEA NOT NULL,
  issue_date DATE,
  expiry_date DATE,
  verification_status TEXT NOT NULL DEFAULT 'Pending' CHECK (verification_status IN ('Pending','Verified','Rejected')),
  notes TEXT NOT NULL DEFAULT '',
  uploaded_by TEXT REFERENCES users(id),
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_dcs_doc_store_employee ON dcs_doc_store(employee_id, uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_dcs_doc_store_expiry ON dcs_doc_store(expiry_date) WHERE expiry_date IS NOT NULL;

INSERT INTO dcs_app_master(page_id,page_name,page_item,field_key,label,data_type,required,visible,display_order,options,help_text)
VALUES
('employees','Employees','First name','firstName','First name','text',true,true,10,'[]','Employee given name'),
('employees','Employees','Last name','lastName','Last name','text',true,true,20,'[]','Employee family name'),
('employees','Employees','Email address','email','Email address','email',true,true,30,'[]','Work email'),
('employees','Employees','Employee number','employeeNumber','Employee number','text',false,true,40,'[]','Unique employee identifier'),
('employees','Employees','Department','department','Department','select',true,true,50,'["People","Engineering","Finance","Operations","Design","Sales","Marketing"]','Organisational department'),
('employees','Employees','Job title','jobTitle','Job title','text',true,true,60,'[]','Role or position'),
('employees','Employees','Location','location','Location','text',false,true,70,'[]','Primary work location'),
('employees','Employees','Employment type','employmentType','Employment type','select',false,true,80,'["Full-time","Part-time","Fixed-term","Contractor"]','Employment arrangement'),
('employees','Employees','Start date','startDate','Start date','date',true,true,90,'[]','Employment start date'),
('employees','Employees','Annual salary','salary','Annual salary (GBP)','number',false,true,100,'[]','Annual salary; access should be limited to authorised HR users'),
('attendance','Attendance','Event type','eventType','Event type','select',true,true,10,'["check_in","check_out"]','Attendance event type'),
('attendance','Attendance','Timestamp','occurredAt','Timestamp','date',true,true,20,'[]','Recorded attendance timestamp'),
('shift-management','Shift management','Shift name','name','Shift name','text',true,true,10,'[]','Reusable shift template name'),
('shift-management','Shift management','Start time','startTime','Start time','text',true,true,20,'[]','24-hour time, HH:MM'),
('shift-management','Shift management','End time','endTime','End time','text',true,true,30,'[]','24-hour time, HH:MM'),
('shift-management','Shift management','Break duration','breakMinutes','Break duration (minutes)','number',false,true,40,'[]','Unpaid break duration'),
('leave-management','Leave management','Leave type','leaveType','Leave type','select',true,true,10,'["Annual leave","Sick leave","Personal leave","Unpaid leave","Parental leave"]','Type of absence'),
('leave-management','Leave management','Start date','startDate','Start date','date',true,true,20,'[]','First day of leave'),
('leave-management','Leave management','End date','endDate','End date','date',true,true,30,'[]','Last day of leave'),
('payroll','Payroll','Period label','periodLabel','Period label','text',true,true,10,'[]','Payroll period label'),
('payroll','Payroll','Payment status','status','Payment status','select',true,true,20,'["Draft","Pending approval","Approved","Paid"]','Payroll run status'),
('documents','Documents','Document type','documentType','Document type','select',true,true,10,'["Identity","Contract","Right to work","Qualification","Other"]','Document category'),
('documents','Documents','Issue date','issueDate','Issue date','date',false,true,20,'[]','Date issued'),
('documents','Documents','Expiry date','expiryDate','Expiry date','date',false,true,30,'[]','Expiry date, if applicable'),
('approvals','Approvals','Decision','decision','Decision','select',true,true,10,'["Approved","Rejected","Return for correction"]','Approval decision'),
('approvals','Approvals','Decision note','decisionNote','Decision note','textarea',false,true,20,'[]','Decision rationale'),
('reports-analytics','Reports & analytics','Date from','dateFrom','Date from','date',false,true,10,'[]','Report start date'),
('reports-analytics','Reports & analytics','Date to','dateTo','Date to','date',false,true,20,'[]','Report end date'),
('audit-log','Audit log','Action','action','Action','text',false,true,10,'[]','Audited action'),
('settings','Settings','Organisation name','organisationName','Organisation name','text',true,true,10,'[]','Display name for the organisation')
ON CONFLICT(page_id,field_key) DO NOTHING;

-- Keep the original seeded administrator as the configuration owner on upgraded installations.
UPDATE users SET role='Master Admin' WHERE email='admin@peopleops.local' AND role='Super Admin';
