package api

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"peopleops/internal/store"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorkingDaysCalendar(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-10-23")
	end := start.AddDate(0, 0, 4)
	got := countWorkingDays(start, end, map[time.Weekday]bool{time.Saturday: true, time.Sunday: true}, map[string]bool{"2026-10-26": true})
	if got != 2 {
		t.Fatalf("Friday–Tuesday with weekend and Monday holiday: got %v want 2", got)
	}
	if got = countWorkingDays(start, start, map[time.Weekday]bool{}, map[string]bool{}); got != 1 {
		t.Fatalf("same-day leave: %v", got)
	}
}

func workflowTestServer(t *testing.T) *Server {
	t.Helper()
	dsn := os.Getenv("HRMS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("HRMS_TEST_DATABASE_URL must point to a disposable local PostgreSQL instance")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("hrms_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close(); admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	_, err = db.Exec(ctx, `
 CREATE TABLE dcs_app_roles(id text PRIMARY KEY,role_name text,active bool);
 CREATE TABLE dcs_role_permissions(role_id text,permission_key text,enabled bool);
 CREATE TABLE users(id text PRIMARY KEY,role text,email text,display_name text,employee_id text,active bool);
 CREATE TABLE employees(id text PRIMARY KEY,employee_number text,first_name text,last_name text,email text,department text,job_title text,location text,employment_type text,status text,start_date date,custom_fields jsonb,manager_id text,updated_at timestamptz);
 CREATE TABLE leave_requests(id text PRIMARY KEY,employee_id text,status text,decided_at timestamptz,decided_by text,decision_note text);
 CREATE TABLE audit_events(id text PRIMARY KEY,actor_user_id text,action text,target_type text,target_id text,summary text,request_id text,created_at timestamptz);
 CREATE TABLE dcs_leave_weekend_days(weekday int); INSERT INTO dcs_leave_weekend_days VALUES(0),(6);
 CREATE TABLE dcs_leave_holidays(holiday_date date);
 CREATE TABLE shifts(id text PRIMARY KEY,start_time time,end_time time,active bool);
 CREATE TABLE shift_assignments(id text PRIMARY KEY,shift_id text,employee_id text,shift_date date,status text,note text,created_by text,created_at timestamptz,UNIQUE(employee_id,shift_date));
 CREATE TABLE dsc_employee_salary_breakup(id text PRIMARY KEY,employee_id text UNIQUE,gross_pay_minor bigint,hourly_rate_minor bigint,overtime_multiplier numeric,variable_pay_minor bigint,allowances_minor bigint,deductions_minor bigint,currency text,effective_from date,updated_at timestamptz);
 CREATE TABLE dcs_employee_bank_details(id text PRIMARY KEY,employee_id text UNIQUE,account_name text,bank_name text,sort_code text,account_number_encrypted text,iban_encrypted text,bic_swift text,updated_by text,updated_at timestamptz);
 INSERT INTO dcs_app_roles VALUES('hr','HR Admin',true),('employee','Employee',true);
 INSERT INTO dcs_role_permissions VALUES('hr','leave.review',true),('hr','employees.manage',true),('hr','hr.payroll.read',true),('hr','shift.assign',true);
 INSERT INTO employees(id,first_name,last_name,email,department,job_title,employee_number,status,start_date) VALUES('emp','Original','Employee','e@example.test','Operations','Trainer','E1','Active','2026-01-01');
 INSERT INTO users VALUES('hruser','HR Admin','hr@example.test','HR','hr-employee',true),('empuser','Employee','e@example.test','Original Employee','emp',true);
 `)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../store/migrations/019_shift_assignment_sessions.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = db.Exec(ctx, string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	return &Server{Store: &store.Store{DB: db, Ctx: ctx}}
}

func TestWorkflowLeaveDecisionIsolation(t *testing.T) {
	s := workflowTestServer(t)
	ctx := context.Background()
	u := User{ID: "hruser", Role: "HR Admin", EmployeeID: "hr-employee"}
	_, err := s.Store.DB.Exec(ctx, `INSERT INTO leave_requests VALUES('request-a','emp','Pending',NULL,NULL,''),('request-b','emp','Pending',NULL,NULL,'')`)
	if err != nil {
		t.Fatal(err)
	}
	code, err := s.decideLeave(ctx, u, "request-a", "Approved", "", "test")
	if err != nil || code != 200 {
		t.Fatalf("decision: %d %v", code, err)
	}
	var a, b string
	s.Store.DB.QueryRow(ctx, `SELECT status FROM leave_requests WHERE id='request-a'`).Scan(&a)
	s.Store.DB.QueryRow(ctx, `SELECT status FROM leave_requests WHERE id='request-b'`).Scan(&b)
	if a != "Approved" || b != "Pending" {
		t.Fatalf("approving A changed B: A=%s B=%s", a, b)
	}
	if code, err = s.decideLeave(ctx, u, "request-a", "Rejected", "", "test"); err == nil || code != 409 {
		t.Fatal("duplicate decision must fail")
	}
	u.EmployeeID = "emp"
	if code, err = s.decideLeave(ctx, u, "request-b", "Approved", "", "test"); err == nil || code != 403 {
		t.Fatal("self approval must fail")
	}
	u.EmployeeID = "hr-employee"
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"ids":["request-a","request-b","request-b"],"decision":"Rejected"}`))
	req = req.WithContext(context.WithValue(req.Context(), userKey, u))
	w := httptest.NewRecorder()
	s.bulkLeaveDecision(w, req)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var n int
	s.Store.DB.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE target_id='request-b'`).Scan(&n)
	if n != 1 {
		t.Fatalf("duplicate ID processed %d times", n)
	}
}

func TestWorkflowEmployeeSaveRollback(t *testing.T) {
	s := workflowTestServer(t)
	u := User{ID: "hruser", Role: "HR Admin"}
	t.Setenv("BANK_DETAILS_KEY", "test-bank-key-that-is-long-enough-123456")
	base := `"employeeNumber":"E1","firstName":"Changed","lastName":"Employee","email":"e@example.test","department":"Operations","jobTitle":"Trainer","status":"Active","startDate":"2026-01-01","roleName":"Employee","salary":{"grossPayMinor":500000,"hourlyRateMinor":2000,"overtimeMultiplier":1.5,"currency":"AED","effectiveFrom":"2026-01-01"}`
	call := func(bank string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PUT", "/", strings.NewReader("{"+base+`,"bank":`+bank+"}"))
		r.SetPathValue("id", "emp")
		r = r.WithContext(context.WithValue(r.Context(), userKey, u))
		w := httptest.NewRecorder()
		s.updateEmployee(w, r)
		return w
	}
	w := call(`{"accountName":"Name","sortCode":"","accountNumber":"123"}`)
	if w.Code != 400 {
		t.Fatalf("expected validation error: %d %s", w.Code, w.Body.String())
	}
	var name string
	var n int
	s.Store.DB.QueryRow(context.Background(), `SELECT first_name FROM employees WHERE id='emp'`).Scan(&name)
	s.Store.DB.QueryRow(context.Background(), `SELECT count(*) FROM dsc_employee_salary_breakup`).Scan(&n)
	if name != "Original" || n != 0 {
		t.Fatalf("partial save occurred: name=%s salary rows=%d", name, n)
	}
	w = call(`{"accountName":"Name","sortCode":"123","accountNumber":"12345678"}`)
	if w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	var encrypted string
	s.Store.DB.QueryRow(context.Background(), `SELECT account_number_encrypted FROM dcs_employee_bank_details WHERE employee_id='emp'`).Scan(&encrypted)
	if encrypted == "12345678" || encrypted == "" {
		t.Fatal("bank details not encrypted")
	}
}

func TestWorkflowShiftConflicts(t *testing.T) {
	s := workflowTestServer(t)
	ctx := context.Background()
	u := User{ID: "hruser", Role: "HR Admin"}
	_, err := s.Store.DB.Exec(ctx, `INSERT INTO shifts VALUES('regular','09:00','17:00',true),('ot','17:00','19:00',true),('overlap','16:00','18:00',true),('night','22:00','06:00',true),('early','05:00','07:00',true)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.assignShift(ctx, u, "regular", "emp", "2026-10-21", "", true); err != nil {
		t.Fatal(err)
	}
	var before int
	s.Store.DB.QueryRow(ctx, `SELECT count(*) FROM shift_assignments`).Scan(&before)
	if before != 0 {
		t.Fatal("preview wrote an assignment")
	}
	for _, id := range []string{"regular", "ot", "night"} {
		if _, err = s.assignShift(ctx, u, id, "emp", "2026-10-21", ""); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	for _, id := range []string{"regular", "overlap"} {
		if _, err = s.assignShift(ctx, u, id, "emp", "2026-10-21", ""); err == nil {
			t.Fatalf("expected conflict for %s", id)
		}
	}
	if _, err = s.assignShift(ctx, u, "early", "emp", "2026-10-22", ""); err == nil {
		t.Fatal("overnight overlap accepted")
	}

}

func TestWorkflowAccountsFinancialOnly(t *testing.T) {
	s := workflowTestServer(t)
	ctx := context.Background()
	_, err := s.Store.DB.Exec(ctx, `INSERT INTO dcs_app_roles VALUES('accounts','Accounts / Payroll',true); INSERT INTO dcs_role_permissions VALUES('accounts','hr.payroll.read',true)`)
	if err != nil {
		t.Fatal(err)
	}
	u := User{ID: "hruser", Role: "Accounts / Payroll"}
	call := func(payload string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PUT", "/", strings.NewReader(payload))
		r.SetPathValue("id", "emp")
		r = r.WithContext(context.WithValue(r.Context(), userKey, u))
		w := httptest.NewRecorder()
		s.updateEmployeeFinancials(w, r)
		return w
	}
	w := call(`{"salary":{"grossPayMinor":500000,"hourlyRateMinor":2000,"overtimeMultiplier":1.5,"currency":"AED","effectiveFrom":"2026-01-01"}}`)
	if w.Code != 200 {
		t.Fatalf("financial-only save failed: %d %s", w.Code, w.Body.String())
	}
	w = call(`{"firstName":"Unauthorized","salary":{"grossPayMinor":500000,"hourlyRateMinor":2000,"overtimeMultiplier":1.5,"currency":"AED","effectiveFrom":"2026-01-01"}}`)
	if w.Code != 400 {
		t.Fatalf("financial-only endpoint accepted employment fields: %d", w.Code)
	}
	var name string
	s.Store.DB.QueryRow(ctx, `SELECT first_name FROM employees WHERE id='emp'`).Scan(&name)
	if name != "Original" {
		t.Fatal("Accounts changed the employee profile")
	}
}
