package store

import (
	"context"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	"peopleops/internal/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct {
	DB  *pgxpool.Pool
	Ctx context.Context
}

func Open() (*Store, error) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = "postgres://peopleops:peopleops@localhost:5432/peopleops?sslmode=disable"
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	//defer cancel()
	cfg.MaxConns = 12
	cfg.MinConns = 1
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	//defer db.Close()
	err = db.Ping(ctx)
	if err != nil {
		return nil, err
	}
	for _, migration := range []string{"migrations/001_init.sql", "migrations/002_app_master_documents.sql", "migrations/003_employee_profiles.sql", "migrations/004_roles_hours_payroll.sql", "migrations/005_employee_hierarchy_password.sql", "migrations/006_role_permissions.sql", "migrations/007_actual_hours_approval.sql", "migrations/008_document_types_and_signing.sql", "migrations/009_document_signing_audit.sql", "migrations/009_leave_balance.sql", "migrations/010_salary_shift_attendance.sql", "migrations/010_bonus_management.sql"} {
		b, readErr := migrationFiles.ReadFile(migration)
		if readErr != nil {
			db.Close()
			return nil, readErr
		}
		if _, execErr := db.Exec(ctx, string(b), pgx.QueryExecModeSimpleProtocol); execErr != nil {
			db.Close()
			return nil, fmt.Errorf("apply %s: %w", migration, execErr)
		}
	}
	s := &Store{DB: db, Ctx: ctx}
	if e := s.seed(); e != nil {
		db.Close()
		return nil, e
	}
	var e error
	// Existing installations also receive the default templates after the shift-management migration.
	var shiftCount int
	if e = db.QueryRow(ctx, `SELECT COUNT(*) FROM shifts;`).Scan(&shiftCount); e != nil {
		db.Close()
		return nil, e
	}
	if shiftCount == 0 {
		now := time.Now().UTC()
		for _, sh := range []struct {
			name, start, end string
			breakM, grace    int
			days             string
		}{{"Day shift", "09:00", "17:00", 60, 5, "Mon,Tue,Wed,Thu,Fri"}, {"Early shift", "06:00", "14:00", 30, 5, "Mon,Tue,Wed,Thu,Fri"}, {"Late shift", "14:00", "22:00", 30, 5, "Mon,Tue,Wed,Thu,Fri"}} {
			if _, e = db.Exec(ctx, `INSERT INTO shifts(id,name,start_time,end_time,break_minutes,grace_minutes,work_days,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,true,$8,$9)`, auth.NewID(), sh.name, parseClock(sh.start), parseClock(sh.end), sh.breakM, sh.grace, sh.days, now, now); e != nil {
				db.Close()
				return nil, e
			}
		}
	}
	return s, nil
}

func (s *Store) seed() error {
	var n int
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if e := s.DB.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	now := time.Now().UTC()
	pass, e := auth.HashPassword("ChangeMe123!")
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	adminID, empUserID := auth.NewID(), auth.NewID()
	_, e = tx.Exec(ctx, `INSERT INTO users(id,email,password_hash,display_name,role,active,created_at) VALUES($1,$2,$3,$4,$5,true,$6),($7,$8,$9,$10,$11,true,$12)`, adminID, "admin@peopleops.local", pass, "PeopleOps Admin", "Master Admin", now, empUserID, "alex.morgan@peopleops.local", pass, "Alex Morgan", "Employee", now)
	if e != nil {
		return e
	}
	people := []struct {
		id, num, first, last, email, dept, title, manager, loc, typ, status, start string
		salary                                                                     int64
	}{
		{auth.NewID(), "EMP-1001", "Alex", "Morgan", "alex.morgan@peopleops.local", "People", "HR Business Partner", "", "London", "Full-time", "Active", "2022-03-14", 4800000},
		{auth.NewID(), "EMP-1002", "Priya", "Sharma", "priya.sharma@peopleops.local", "Engineering", "Software Engineer", "", "London", "Full-time", "Active", "2023-06-01", 6200000},
		{auth.NewID(), "EMP-1003", "Jordan", "Lee", "jordan.lee@peopleops.local", "Finance", "Payroll Analyst", "", "Manchester", "Full-time", "Active", "2021-11-08", 4100000},
		{auth.NewID(), "EMP-1004", "Sam", "Taylor", "sam.taylor@peopleops.local", "Operations", "Shift Supervisor", "", "Bristol", "Full-time", "On Leave", "2024-01-15", 3600000},
		{auth.NewID(), "EMP-1005", "Casey", "Patel", "casey.patel@peopleops.local", "Design", "Product Designer", "", "London", "Full-time", "Active", "2024-04-22", 4500000},
	}
	for _, p := range people {
		startDate, _ := time.Parse("2006-01-02", p.start)
		_, e = tx.Exec(ctx, `INSERT INTO employees(id,employee_number,first_name,last_name,email,department,job_title,location,employment_type,status,start_date,salary_minor,currency,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, p.id, p.num, p.first, p.last, p.email, p.dept, p.title, p.loc, p.typ, p.status, startDate, p.salary, "GBP", now, now)
		if e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `UPDATE users SET employee_id=$1 WHERE id=$2`, people[0].id, empUserID)
	if e != nil {
		return e
	}
	leaveStart1, _ := time.Parse("2006-01-02", "2026-10-26")
	leaveEnd1, _ := time.Parse("2006-01-02", "2026-10-28")
	leaveStart2, _ := time.Parse("2006-01-02", "2026-10-19")
	_, e = tx.Exec(ctx, `INSERT INTO leave_requests(id,employee_id,leave_type,start_date,end_date,days,reason,status,submitted_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9),($10,$11,$12,$13,$14,$15,$16,$17,$18)`, auth.NewID(), people[1].id, "Annual leave", leaveStart1, leaveEnd1, 3, "Family plans", "Pending", now, auth.NewID(), people[2].id, "Personal leave", leaveStart2, leaveStart2, 1, "Personal appointment", "Pending", now)
	if e != nil {
		return e
	}
	payrollStart, _ := time.Parse("2006-01-02", "2026-09-01")
	payrollEnd, _ := time.Parse("2006-01-02", "2026-09-30")
	_, e = tx.Exec(ctx, `INSERT INTO payroll_periods(id,period_label,start_date,end_date,status,employee_count,gross_minor,deductions_minor,net_minor,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, auth.NewID(), "September 2026", payrollStart, payrollEnd, "Paid", 5, 23100000, 4280000, 18820000, now)
	if e != nil {
		return e
	}
	// Default shift templates make shift management usable immediately in demo environments.
	for _, sh := range []struct {
		name, start, end string
		breakM, grace    int
		days             string
	}{{"Day shift", "09:00", "17:00", 60, 5, "Mon,Tue,Wed,Thu,Fri"}, {"Early shift", "06:00", "14:00", 30, 5, "Mon,Tue,Wed,Thu,Fri"}, {"Late shift", "14:00", "22:00", 30, 5, "Mon,Tue,Wed,Thu,Fri"}} {
		_, e = tx.Exec(ctx, `INSERT INTO shifts(id,name,start_time,end_time,break_minutes,grace_minutes,work_days,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,true,$8,$9)`, auth.NewID(), sh.name, parseClock(sh.start), parseClock(sh.end), sh.breakM, sh.grace, sh.days, now, now)
		if e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_events(id,actor_user_id,action,target_type,target_id,summary,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, auth.NewID(), adminID, "system.seed", "system", "demo", "Seeded fictional demonstration data", now)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s *Store) Audit(ctx context.Context, actor, action, targetType, targetID, summary, requestID string) {
	_, _ = s.DB.Exec(ctx, `INSERT INTO audit_events(id,actor_user_id,action,target_type,target_id,summary,request_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, auth.NewID(), nullable(actor), action, targetType, targetID, summary, requestID, time.Now().UTC())
}

func nullable(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func (s *Store) Check() error {
	if e := s.DB.Ping(s.Ctx); e != nil {
		return fmt.Errorf("database ping: %w", e)
	}
	return nil
}

func parseClock(value string) time.Time { t, _ := time.Parse("15:04", value); return t }
