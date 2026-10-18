CREATE TABLE IF NOT EXISTS dcs_leave_role_entitlements (
 role_name TEXT PRIMARY KEY,
 annual_days NUMERIC(7,2) NOT NULL DEFAULT 25 CHECK (annual_days >= 0 AND annual_days <= 366),
 updated_by TEXT REFERENCES users(id),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS dcs_leave_holidays (
 holiday_date DATE PRIMARY KEY,
 name TEXT NOT NULL,
 created_by TEXT REFERENCES users(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO dcs_leave_role_entitlements(role_name,annual_days)
SELECT role_name,25 FROM dcs_app_roles ON CONFLICT(role_name) DO NOTHING;
INSERT INTO dcs_role_permissions(role_id,permission_key,enabled)
SELECT id,'leave.balance.read',true FROM dcs_app_roles ON CONFLICT(role_id,permission_key) DO NOTHING;
INSERT INTO dcs_role_permissions(role_id,permission_key,enabled)
SELECT id,'leave.balance.configure',true FROM dcs_app_roles WHERE role_name='Master Admin' ON CONFLICT(role_id,permission_key) DO NOTHING;
INSERT INTO dcs_role_pages(role_id,page_id,page_name,enabled)
SELECT id,'leave-balance','Leave Balance',true FROM dcs_app_roles ON CONFLICT(role_id,page_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS dcs_leave_weekend_days (
 weekday SMALLINT PRIMARY KEY CHECK (weekday BETWEEN 0 AND 6)
);
INSERT INTO dcs_leave_weekend_days(weekday) VALUES (0),(6) ON CONFLICT DO NOTHING;
