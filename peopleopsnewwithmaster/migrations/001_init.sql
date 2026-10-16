CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY, 
    email TEXT NOT NULL UNIQUE, 
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL, 
    role TEXT NOT NULL, 
    employee_id TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS employees (
    id TEXT PRIMARY KEY, 
    employee_number TEXT NOT NULL UNIQUE, 
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL, 
    email TEXT NOT NULL UNIQUE, 
    department TEXT NOT NULL,
    job_title TEXT NOT NULL, 
    manager_id TEXT, 
    location TEXT NOT NULL DEFAULT 'Head Office',
    employment_type TEXT NOT NULL DEFAULT 'Full-time', 
    status TEXT NOT NULL DEFAULT 'Active',
    start_date DATE NOT NULL, 
    salary_minor INTEGER NOT NULL DEFAULT 0, 
    currency TEXT NOT NULL DEFAULT 'GBP',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (manager_id) REFERENCES employees(id)
);

CREATE TABLE IF NOT EXISTS attendance_events (
    id TEXT PRIMARY KEY, 
    employee_id TEXT NOT NULL, 
    event_type TEXT NOT NULL CHECK (event_type IN ('check_in', 'check_out')),
    occurred_at TIMESTAMPTZ NOT NULL, 
    source TEXT NOT NULL DEFAULT 'web', 
    latitude DOUBLE PRECISION, 
    longitude DOUBLE PRECISION, 
    accuracy_m DOUBLE PRECISION,
    note TEXT NOT NULL DEFAULT '', 
    FOREIGN KEY (employee_id) REFERENCES employees(id)
);

CREATE TABLE IF NOT EXISTS leave_requests (
    id TEXT PRIMARY KEY, 
    employee_id TEXT NOT NULL, 
    leave_type TEXT NOT NULL,
    start_date DATE NOT NULL, 
    end_date DATE NOT NULL, 
    days REAL NOT NULL,
    reason TEXT NOT NULL DEFAULT '', 
    status TEXT NOT NULL DEFAULT 'Pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    decided_at TIMESTAMPTZ, 
    decided_by TEXT, 
    decision_note TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (employee_id) REFERENCES employees(id), 
    FOREIGN KEY (decided_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY, 
    actor_user_id TEXT, 
    action TEXT NOT NULL, 
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL, 
    summary TEXT NOT NULL, 
    request_id TEXT NOT NULL DEFAULT '', 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (actor_user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS payroll_periods (
    id TEXT PRIMARY KEY, 
    period_label TEXT NOT NULL UNIQUE, 
    start_date DATE NOT NULL, 
    end_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'Draft', 
    employee_count INTEGER NOT NULL DEFAULT 0,
    gross_minor INTEGER NOT NULL DEFAULT 0, 
    deductions_minor INTEGER NOT NULL DEFAULT 0,
    net_minor INTEGER NOT NULL DEFAULT 0, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_employees_department ON employees(department);
CREATE INDEX IF NOT EXISTS idx_attendance_employee_time ON attendance_events(employee_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_leave_status ON leave_requests(status, submitted_at);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_events(created_at);

CREATE TABLE IF NOT EXISTS shifts (
    id TEXT PRIMARY KEY, 
    name TEXT NOT NULL UNIQUE, 
    start_time TIME NOT NULL, 
    end_time TIME NOT NULL,
    break_minutes INTEGER NOT NULL DEFAULT 0, 
    grace_minutes INTEGER NOT NULL DEFAULT 5, 
    work_days TEXT NOT NULL DEFAULT 'Mon,Tue,Wed,Thu,Fri',
    active BOOLEAN NOT NULL DEFAULT TRUE, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shift_assignments (
    id TEXT PRIMARY KEY, 
    shift_id TEXT NOT NULL, 
    employee_id TEXT NOT NULL, 
    shift_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'Assigned', 
    note TEXT NOT NULL DEFAULT '', 
    created_by TEXT, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (shift_id) REFERENCES shifts(id), 
    FOREIGN KEY (employee_id) REFERENCES employees(id),
    FOREIGN KEY (created_by) REFERENCES users(id), 
    UNIQUE (employee_id, shift_date)
);

CREATE INDEX IF NOT EXISTS idx_shift_assignments_date ON shift_assignments(shift_date, shift_id);