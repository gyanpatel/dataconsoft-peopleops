package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"peopleops/internal/auth"
	"peopleops/internal/store"
)

type Server struct {
	Store  *store.Store
	Secret string
	Origin string
	Logger *slog.Logger
}
type User struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	EmployeeID  string   `json:"employeeId,omitempty"`
	Pages       []string `json:"pages,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}
type ctxKey string

const userKey ctxKey = "user"

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.Handle("GET /api/v1/auth/me", s.protected(http.HandlerFunc(s.me)))
	mux.Handle("GET /api/v1/dashboard", s.protected(http.HandlerFunc(s.dashboard)))
	mux.Handle("GET /api/v1/employees", s.protected(http.HandlerFunc(s.employees)))
	mux.Handle("GET /api/v1/employee-hierarchy", s.protected(http.HandlerFunc(s.employeeHierarchy)))
	mux.Handle("POST /api/v1/auth/change-password", s.protected(http.HandlerFunc(s.changePassword)))
	mux.Handle("GET /api/v1/app-master", s.protected(http.HandlerFunc(s.appMaster)))
	mux.Handle("POST /api/v1/app-master", s.protected(http.HandlerFunc(s.appMaster)))
	mux.Handle("PUT /api/v1/app-master", s.protected(http.HandlerFunc(s.appMaster)))
	mux.Handle("DELETE /api/v1/app-master", s.protected(http.HandlerFunc(s.appMaster)))
	mux.Handle("GET /api/v1/documents", s.protected(http.HandlerFunc(s.documents)))
	mux.Handle("GET /api/v1/document-types", s.protected(http.HandlerFunc(s.documentTypes)))
	mux.Handle("POST /api/v1/documents", s.protected(http.HandlerFunc(s.documents)))
	mux.Handle("GET /api/v1/documents/{id}/download", s.protected(http.HandlerFunc(s.documentDownload)))
	mux.HandleFunc("GET /api/v1/sign/{token}", s.signEmploymentDocument)
	mux.HandleFunc("POST /api/v1/sign/{token}", s.signEmploymentDocument)
	mux.HandleFunc("GET /api/v1/sign/{token}/preview", s.previewEmploymentDocument)
	mux.HandleFunc("GET /sign/{token}/preview", s.previewEmploymentDocument)
	mux.Handle("POST /api/v1/documents/{id}/decision", s.protected(http.HandlerFunc(s.documentDecision)))
	mux.Handle("POST /api/v1/employees", s.protected(http.HandlerFunc(s.createEmployee)))
	mux.Handle("PUT /api/v1/employees/{id}", s.protected(http.HandlerFunc(s.updateEmployee)))
	mux.Handle("GET /api/v1/employees/{id}/photo", s.protected(http.HandlerFunc(s.employeePhoto)))
	mux.Handle("GET /api/v1/attendance", s.protected(http.HandlerFunc(s.attendance)))
	mux.Handle("GET /api/v1/shifts", s.protected(http.HandlerFunc(s.shifts)))
	mux.Handle("POST /api/v1/shifts", s.protected(http.HandlerFunc(s.createShift)))
	mux.Handle("GET /api/v1/shift-assignments", s.protected(http.HandlerFunc(s.shiftAssignments)))
	mux.Handle("POST /api/v1/shift-assignments", s.protected(http.HandlerFunc(s.createShiftAssignment)))
	mux.Handle("DELETE /api/v1/shift-assignments/{id}", s.protected(http.HandlerFunc(s.deleteShiftAssignment)))
	mux.Handle("POST /api/v1/attendance/check-in", s.protected(http.HandlerFunc(s.checkIn)))
	mux.Handle("POST /api/v1/attendance/check-out", s.protected(http.HandlerFunc(s.checkOut)))
	mux.Handle("GET /api/v1/leave-requests", s.protected(http.HandlerFunc(s.leaveRequests)))
	mux.Handle("POST /api/v1/leave-requests", s.protected(http.HandlerFunc(s.createLeave)))
	mux.Handle("POST /api/v1/leave-requests/{id}/decision", s.protected(http.HandlerFunc(s.leaveDecision)))
	mux.Handle("GET /api/v1/payroll", s.protected(http.HandlerFunc(s.payrollAdvanced)))
	mux.Handle("GET /api/v1/payroll/summaries", s.protected(http.HandlerFunc(s.salarySummaries)))
	mux.Handle("POST /api/v1/payroll/calculate", s.protected(http.HandlerFunc(s.calculatePayroll)))
	mux.Handle("POST /api/v1/payroll/issue", s.protected(http.HandlerFunc(s.issuePayroll)))
	mux.Handle("POST /api/v1/payroll/payments/{id}/clear", s.protected(http.HandlerFunc(s.clearPayment)))
	mux.Handle("GET /api/v1/payroll/{id}/payslip.pdf", s.protected(http.HandlerFunc(s.payslipPDF)))
	mux.Handle("GET /api/v1/roles", s.protected(http.HandlerFunc(s.roles)))
	mux.Handle("POST /api/v1/roles", s.protected(http.HandlerFunc(s.roles)))
	mux.Handle("PUT /api/v1/roles", s.protected(http.HandlerFunc(s.roles)))
	mux.Handle("DELETE /api/v1/roles", s.protected(http.HandlerFunc(s.roles)))
	mux.Handle("GET /api/v1/role-pages", s.protected(http.HandlerFunc(s.rolePages)))
	mux.Handle("PUT /api/v1/role-pages", s.protected(http.HandlerFunc(s.rolePages)))
	mux.Handle("GET /api/v1/actual-hours", s.protected(http.HandlerFunc(s.actualHours)))
	mux.Handle("POST /api/v1/actual-hours/approve", s.protected(http.HandlerFunc(s.approveActualHours)))
	mux.Handle("GET /api/v1/employees/{id}/bank-details", s.protected(http.HandlerFunc(s.employeeBank)))
	mux.Handle("PUT /api/v1/employees/{id}/bank-details", s.protected(http.HandlerFunc(s.employeeBank)))
	mux.Handle("GET /api/v1/employees/{id}/salary-breakup", s.protected(http.HandlerFunc(s.salaryBreakup)))
	mux.Handle("PUT /api/v1/employees/{id}/salary-breakup", s.protected(http.HandlerFunc(s.salaryBreakup)))
	mux.Handle("GET /api/v1/salary-algorithm", s.protected(http.HandlerFunc(s.salaryAlgorithm)))
	mux.Handle("GET /api/v1/audit", s.protected(http.HandlerFunc(s.audit)))
	mux.Handle("GET /api/v1/reports/headcount.csv", s.protected(http.HandlerFunc(s.headcountCSV)))
	return s.middleware(mux)
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := auth.NewID()
		w.Header().Set("X-Request-ID", rid)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			if s.Origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", s.Origin)
			}
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.WriteHeader(204)
			return
		}
		if s.Origin != "" && r.Header.Get("Origin") == s.Origin {
			w.Header().Set("Access-Control-Allow-Origin", s.Origin)
			w.Header().Set("Vary", "Origin")
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey("request_id"), rid)))
	})
}
func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		} else if c, e := r.Cookie("peopleops_session"); e == nil {
			token = c.Value
		}
		claims, e := auth.Verify(s.Secret, token)
		if e != nil {
			writeError(w, 401, "unauthorized", "Sign in to continue")
			return
		}
		var u User
		var mustChange bool
		e = s.Store.DB.QueryRow(s.Store.Ctx, `SELECT id,email,display_name,role,COALESCE(employee_id,''),must_change_password FROM users WHERE id=$1 AND active=true`, claims.UserID).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.EmployeeID, &mustChange)
		if e != nil {
			log.Println("auth middleware: error querying user:", e)
			writeError(w, 401, "unauthorized", "Session is no longer valid")
			return
		}
		u.Pages = s.allowedPages(r.Context(), u.Role)
		if mustChange && r.URL.Path != "/api/v1/auth/change-password" && r.URL.Path != "/api/v1/auth/me" && r.URL.Path != "/api/v1/auth/logout" {
			writeError(w, 403, "password_change_required", "Change your temporary password before continuing")
			return
		}
		pageID := requiredPageForPath(r.URL.Path)
		// Payslip access is authenticated self-service for every user; payroll mutation
		// endpoints still enforce their own Accounts role checks.
		selfServicePayroll := r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/payroll")
		roleOptionsRead := r.Method == http.MethodGet && r.URL.Path == "/api/v1/roles"
		if pageID != "" && !selfServicePayroll && !roleOptionsRead && !containsPage(u.Pages, pageID) {
			log.Println("authorization: role", u.Role, "denied page", pageID, "for path", r.URL.Path)
			writeError(w, 403, "page_forbidden", "Your role does not have access to this application page")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}
func requiredPageForPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/dashboard"):
		return "dashboard"
	case strings.HasPrefix(path, "/api/v1/employees"):
		return "employees"
	case strings.HasPrefix(path, "/api/v1/attendance"):
		return "attendance"
	case strings.HasPrefix(path, "/api/v1/shifts"), strings.HasPrefix(path, "/api/v1/shift-assignments"), strings.HasPrefix(path, "/api/v1/actual-hours"):
		return "shift-management"
	case strings.HasPrefix(path, "/api/v1/leave-requests"):
		return "leave-management"
	case strings.HasPrefix(path, "/api/v1/payroll"):
		return "payroll"
	case strings.HasPrefix(path, "/api/v1/documents"):
		return "documents"
	case strings.HasPrefix(path, "/api/v1/audit"):
		return "audit-log"
	case strings.HasPrefix(path, "/api/v1/reports"):
		return "reports-analytics"
	case strings.HasPrefix(path, "/api/v1/app-master"):
		return "employees"
	case strings.HasPrefix(path, "/api/v1/roles"), strings.HasPrefix(path, "/api/v1/role-pages"), strings.HasPrefix(path, "/api/v1/salary-algorithm"):
		return "settings"
	default:
		return ""
	}
}
func containsPage(pages []string, target string) bool {
	for _, p := range pages {
		if p == target {
			return true
		}
	}
	return false
}

func current(r *http.Request) User { u, _ := r.Context().Value(userKey).(User); return u }
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, msg string) {
	log.Println("api error:", "status=", status, "code=", code, "message=", msg)
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
func decode(r *http.Request, v any) error {
	// Employee forms may include a base64-encoded profile photo and configurable fields.
	// Allow a bounded 15 MiB JSON request instead of rejecting normal photo uploads at 1 MiB.
	d := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 15<<20))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if e := s.Store.Check(); e != nil {
		log.Println("health check: database unavailable:", e)
		writeError(w, 503, "unhealthy", "Database unavailable")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if decode(r, &in) != nil || len(in.Email) > 254 || len(in.Password) > 200 {
		writeError(w, 400, "invalid_request", "Enter a valid email and password")
		return
	}
	var u User
	var hash string
	var active bool
	var mustChange bool
	e := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT id,email,display_name,role,COALESCE(employee_id,''),password_hash,active,must_change_password FROM users WHERE lower(email)=lower($1)`, strings.TrimSpace(in.Email)).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.EmployeeID, &hash, &active, &mustChange)
	if e != nil || !active || !auth.CheckPassword(hash, in.Password) {
		if e != nil {
			log.Println("login: user lookup failed:", e)
		} else {
			log.Println("login: invalid credentials or inactive account")
		}
		time.Sleep(200 * time.Millisecond)
		writeError(w, 401, "invalid_credentials", "Email or password is incorrect")
		return
	}
	u.Pages = s.allowedPages(r.Context(), u.Role)
	u.Permissions = s.allowedPermissions(r.Context(), u.Role)
	if len(u.Pages) == 0 {
		log.Println("login error: role has no active page access:", u.Role)
		writeError(w, 403, "role_inactive", "This role is inactive or has no configured page access")
		return
	}
	token, e := auth.Issue(s.Secret, auth.Claims{UserID: u.ID, Email: u.Email, Role: u.Role})
	if e != nil {
		writeError(w, 500, "internal", "Could not create session")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "peopleops_session", Value: token, Path: "/", HttpOnly: true, Secure: os.Getenv("APP_ENV") == "production", SameSite: http.SameSiteLaxMode, MaxAge: 43200})
	s.Store.Audit(r.Context(), u.ID, "auth.login", "user", u.ID, "User signed in", requestID(r))
	writeJSON(w, 200, map[string]any{"user": u, "token": token, "mustChangePassword": mustChange})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "peopleops_session", Value: "", Path: "/", HttpOnly: true, Secure: os.Getenv("APP_ENV") == "production", SameSite: http.SameSiteLaxMode, MaxAge: -1})
	if u := current(r); u.ID != "" {
		s.Store.Audit(r.Context(), u.ID, "auth.logout", "user", u.ID, "User signed out", requestID(r))
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if len(u.Pages) == 0 {
		u.Pages = s.allowedPages(r.Context(), u.Role)
	}
	u.Permissions = s.allowedPermissions(r.Context(), u.Role)
	var mustChange bool
	if err := s.Store.DB.QueryRow(r.Context(), `SELECT must_change_password FROM users WHERE id=$1`, u.ID).Scan(&mustChange); err != nil {
		log.Println("auth me: must-change lookup failed:", err)
	}
	writeJSON(w, 200, map[string]any{"user": u, "mustChangePassword": mustChange})
}
func (s *Server) allowedPermissions(ctx context.Context, role string) []string {
	rows, err := s.Store.DB.Query(ctx, `SELECT p.permission_key FROM dcs_role_permissions p JOIN dcs_app_roles r ON r.id=p.role_id WHERE lower(r.role_name)=lower($1) AND r.active=true AND p.enabled=true ORDER BY p.permission_key`, role)
	if err != nil {
		log.Println("role permissions: lookup failed for", role, ":", err)
		return []string{}
	}
	defer rows.Close()
	permissions := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			log.Println("role permissions: scan failed:", err)
			return []string{}
		}
		permissions = append(permissions, p)
	}
	if err := rows.Err(); err != nil {
		log.Println("role permissions: iteration failed:", err)
		return []string{}
	}
	return permissions
}

func (s *Server) allowedPages(ctx context.Context, role string) []string {
	rows, err := s.Store.DB.Query(ctx, `SELECT p.page_id FROM dcs_role_pages p JOIN dcs_app_roles r ON r.id=p.role_id WHERE r.role_name=$1 AND r.active=true AND p.enabled=true ORDER BY p.page_name`, role)
	if err != nil {
		log.Println("role access: could not load page permissions for", role, ":", err)
		return []string{"dashboard"}
	}
	defer rows.Close()
	pages := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			log.Println("role access: page scan failed:", err)
			return []string{"dashboard"}
		}
		pages = append(pages, p)
	}
	if err := rows.Err(); err != nil {
		log.Println("role access: page iteration failed:", err)
	}
	return pages
}
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	var employees, active, pending, departments int
	if err := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT COUNT(*),COALESCE(SUM(CASE WHEN status='Active' THEN 1 ELSE 0 END),0),COUNT(DISTINCT department) FROM employees`).Scan(&employees, &active, &departments); err != nil {
		log.Println("dashboard: employee metrics query failed:", err)
	}
	if err := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT COUNT(*) FROM leave_requests WHERE status='Pending'`).Scan(&pending); err != nil {
		log.Println("dashboard: pending leave count failed:", err)
	}
	var payroll struct {
		Label, Status string
		Net           int64
	}
	if err := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT period_label,status,net_minor FROM payroll_periods ORDER BY created_at DESC LIMIT 1`).Scan(&payroll.Label, &payroll.Status, &payroll.Net); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Println("dashboard: latest payroll query failed:", err)
	}
	rows, queryErr := s.Store.DB.Query(s.Store.Ctx, `SELECT department,COUNT(*) FROM employees GROUP BY department ORDER BY COUNT(*) DESC`)
	if queryErr != nil {
		log.Println("dashboard: department breakdown query failed:", queryErr)
	}
	byDept := []map[string]any{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var d string
			var n int
			if err := rows.Scan(&d, &n); err != nil {
				log.Println("dashboard: department row scan failed:", err)
				continue
			}
			byDept = append(byDept, map[string]any{"department": d, "count": n})
		}
	}
	writeJSON(w, 200, map[string]any{"metrics": map[string]any{"employees": employees, "activeEmployees": active, "pendingApprovals": pending, "departments": departments, "payrollNetMinor": payroll.Net, "payrollPeriod": payroll.Label, "payrollStatus": payroll.Status}, "departments": byDept})
}

type Employee struct {
	ID           string         `json:"id"`
	Number       string         `json:"employeeNumber"`
	First        string         `json:"firstName"`
	Last         string         `json:"lastName"`
	Email        string         `json:"email"`
	Department   string         `json:"department"`
	Title        string         `json:"jobTitle"`
	Location     string         `json:"location"`
	Type         string         `json:"employmentType"`
	Status       string         `json:"status"`
	StartDate    time.Time      `json:"startDate"`
	SalaryMinor  int64          `json:"salaryMinor"`
	Currency     string         `json:"currency"`
	CustomFields map[string]any `json:"customFields,omitempty"`
	PhotoID      string         `json:"photoId,omitempty"`
	ManagerID    string         `json:"managerId,omitempty"`
	ManagerName  string         `json:"managerName,omitempty"`
	RoleName     string         `json:"roleName,omitempty"`
}

func (s *Server) employees(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	like := "%" + strings.ReplaceAll(strings.ReplaceAll(q, "%", "\\%"), "_", "\\_") + "%"
	limit := 100
	if v, e := strconv.Atoi(r.URL.Query().Get("limit")); e == nil && v > 0 && v <= 200 {
		limit = v
	}
	rows, e := s.Store.DB.Query(s.Store.Ctx, `SELECT id,employee_number,first_name,last_name,email,department,job_title,location,employment_type,status,start_date,salary_minor,currency,custom_fields,COALESCE(photo_document_id, ''),COALESCE(manager_id,''),COALESCE((SELECT first_name||' '||last_name FROM employees m WHERE m.id=employees.manager_id),''),COALESCE((SELECT u.role FROM users u WHERE u.employee_id=employees.id AND u.active=true ORDER BY u.id LIMIT 1),'') FROM employees WHERE ($1='' OR first_name LIKE $2 OR last_name LIKE $3 OR email LIKE $4 OR employee_number LIKE $5 OR department LIKE $6) ORDER BY last_name,first_name LIMIT $7`, q, like, like, like, like, like, limit)
	if e != nil {
		writeError(w, 500, "database_error", "Could not load employees")
		return
	}
	defer rows.Close()
	out := []Employee{}
	for rows.Next() {
		var x Employee
		var sDate sql.NullTime
		var customJSON []byte
		if e = rows.Scan(&x.ID, &x.Number, &x.First, &x.Last, &x.Email, &x.Department, &x.Title, &x.Location, &x.Type, &x.Status, &sDate, &x.SalaryMinor, &x.Currency, &customJSON, &x.PhotoID, &x.ManagerID, &x.ManagerName, &x.RoleName); e != nil {
			log.Println("[ERROR]", e)
			writeError(w, 500, "database_error", "Could not read employees")
			return
		}
		x.StartDate = sDate.Time
		if len(customJSON) > 0 {
			_ = json.Unmarshal(customJSON, &x.CustomFields)
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		log.Println("employees: row iteration failed:", err)
		writeError(w, 500, "database_error", "Could not read employees")
		return
	}
	writeJSON(w, 200, map[string]any{"items": out, "count": len(out)})
}
func (s *Server) updateEmployee(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "employees.manage") {
		writeError(w, 403, "forbidden", "HR administrator permission required")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Number       string         `json:"employeeNumber"`
		First        string         `json:"firstName"`
		Last         string         `json:"lastName"`
		Email        string         `json:"email"`
		Department   string         `json:"department"`
		Title        string         `json:"jobTitle"`
		Location     string         `json:"location"`
		Type         string         `json:"employmentType"`
		Status       string         `json:"status"`
		StartDate    string         `json:"startDate"`
		SalaryMinor  int64          `json:"salaryMinor"`
		CustomFields map[string]any `json:"customFields"`
		PhotoData    string         `json:"photoData"`
		PhotoName    string         `json:"photoName"`
		ManagerID    string         `json:"managerId"`
		RoleName     string         `json:"roleName"`
	}
	if decode(r, &in) != nil || strings.TrimSpace(in.First) == "" || strings.TrimSpace(in.Last) == "" || !strings.Contains(in.Email, "@") || in.Department == "" || in.Title == "" || in.SalaryMinor < 0 {
		writeError(w, 400, "validation_error", "Complete the required fields with valid values")
		return
	}
	in.RoleName = strings.TrimSpace(in.RoleName)
	if in.RoleName == "" {
		writeError(w, 400, "validation_error", "Select a login role for this employee account")
		return
	}
	var activeRole string
	if err := s.Store.DB.QueryRow(r.Context(), `SELECT COALESCE((SELECT role_name FROM dcs_app_roles WHERE lower(role_name)=lower($1) AND active=true), '')`, in.RoleName).Scan(&activeRole); err != nil {
		log.Println("employee update: role validation query failed:", err)
		writeError(w, 500, "database_error", "Could not validate selected login role")
		return
	}
	if activeRole == "" {
		writeError(w, 400, "invalid_role", "The selected login role does not exist or is inactive")
		return
	}
	in.RoleName = activeRole
	start, err := time.Parse("2006-01-02", in.StartDate)
	if err != nil {
		writeError(w, 400, "validation_error", "Start date must use YYYY-MM-DD")
		return
	}
	if in.Status == "" {
		in.Status = "Active"
	}
	if in.Number == "" {
		writeError(w, 400, "validation_error", "Employee number is required")
		return
	}
	custom, _ := json.Marshal(in.CustomFields)
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "database_error", "Could not update employee")
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `UPDATE employees SET employee_number=$2,first_name=$3,last_name=$4,email=$5,department=$6,job_title=$7,location=$8,employment_type=$9,status=$10,start_date=$11,salary_minor=$12,custom_fields=$13::jsonb,updated_at=$14 WHERE id=$1`, id, in.Number, strings.TrimSpace(in.First), strings.TrimSpace(in.Last), strings.ToLower(strings.TrimSpace(in.Email)), in.Department, in.Title, in.Location, in.Type, in.Status, start, in.SalaryMinor, string(custom), time.Now().UTC())
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, 409, "update_failed", "Could not update employee; check the employee exists and the email/number is unique")
		return
	}
	userTag, roleErr := tx.Exec(r.Context(), `UPDATE users SET role=$2,email=$3,display_name=$4 WHERE employee_id=$1 AND active=true`, id, in.RoleName, strings.ToLower(strings.TrimSpace(in.Email)), strings.TrimSpace(in.First)+" "+strings.TrimSpace(in.Last))
	if roleErr != nil {
		log.Println("employee update: linked login role update failed:", roleErr)
		writeError(w, 500, "database_error", "Could not update the employee login role")
		return
	}
	// Existing employee records may predate linked login accounts. Do not block
	// editing their employee details when no active linked user exists.
	_ = userTag
	if in.PhotoData != "" {
		parts := strings.SplitN(in.PhotoData, ",", 2)
		if len(parts) != 2 {
			writeError(w, 400, "invalid_photo", "Invalid image data")
			return
		}
		raw, er := base64.StdEncoding.DecodeString(parts[1])
		if er != nil || len(raw) == 0 || len(raw) > 5*1024*1024 {
			writeError(w, 400, "invalid_photo", "Profile photo must be a valid image under 5 MB")
			return
		}
		mime := "image/jpeg"
		if strings.HasPrefix(parts[0], "data:image/png;") {
			mime = "image/png"
		} else if strings.HasPrefix(parts[0], "data:image/webp;") {
			mime = "image/webp"
		} else if !strings.HasPrefix(parts[0], "data:image/jpeg;") {
			writeError(w, 400, "invalid_photo", "Only JPEG, PNG or WebP images are supported")
			return
		}
		photoID := auth.NewID()
		name := strings.TrimSpace(in.PhotoName)
		if name == "" {
			name = "profile-photo"
		}
		_, er = tx.Exec(r.Context(), `INSERT INTO dcs_doc_store(id,employee_id,document_name,document_type,mime_type,file_size,doc_file,verification_status,notes,uploaded_by) VALUES($1,$2,$3,'Profile photo',$4,$5,$6,'Verified','Employee profile image',$7)`, photoID, id, name, mime, len(raw), raw, u.ID)
		if er == nil {
			_, er = tx.Exec(r.Context(), `UPDATE employees SET photo_document_id=$2 WHERE id=$1`, id, photoID)
		}
		if er != nil {
			writeError(w, 500, "photo_save_failed", "Could not save profile photo")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "database_error", "Could not commit employee update")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "employee.updated", "employee", id, fmt.Sprintf("Updated employee %s %s", in.First, in.Last), requestID(r))
	writeJSON(w, 200, map[string]any{"id": id, "message": "Employee updated"})
}
func (s *Server) employeePhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var mime string
	var data []byte
	err := s.Store.DB.QueryRow(r.Context(), `SELECT d.mime_type,d.doc_file FROM employees e JOIN dcs_doc_store d ON d.id=e.photo_document_id WHERE e.id=$1 AND d.document_type='Profile photo'`, id).Scan(&mime, &data)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) createEmployee(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "employees.manage") {
		writeError(w, 403, "forbidden", "HR administrator permission required")
		return
	}
	var in struct {
		Status       string         `json:"status"`
		Number       string         `json:"employeeNumber"`
		First        string         `json:"firstName"`
		Last         string         `json:"lastName"`
		Email        string         `json:"email"`
		Department   string         `json:"department"`
		Title        string         `json:"jobTitle"`
		Location     string         `json:"location"`
		Type         string         `json:"employmentType"`
		StartDate    string         `json:"startDate"`
		SalaryMinor  int64          `json:"salaryMinor"`
		CustomFields map[string]any `json:"customFields"`
		PhotoData    string         `json:"photoData"`
		PhotoName    string         `json:"photoName"`
		ManagerID    string         `json:"managerId"`
		RoleName     string         `json:"roleName"`
	}
	if err := decode(r, &in); err != nil {
		log.Println("employee create: request JSON decode failed:", err)
		writeError(w, 400, "invalid_request", "Could not read employee form data: "+err.Error())
		return
	}
	if len(strings.TrimSpace(in.First)) < 1 || len(strings.TrimSpace(in.First)) > 80 || len(strings.TrimSpace(in.Last)) < 1 || len(strings.TrimSpace(in.Last)) > 80 || !strings.Contains(strings.TrimSpace(in.Email), "@") || len(in.Email) > 254 || len(strings.TrimSpace(in.Department)) < 1 || len(strings.TrimSpace(in.Title)) < 1 || in.SalaryMinor < 0 {
		log.Printf("employee create: validation failed: firstName=%q lastName=%q email=%q department=%q jobTitle=%q salaryMinor=%d", in.First, in.Last, in.Email, in.Department, in.Title, in.SalaryMinor)
		writeError(w, 400, "validation_error", "Complete the required fields with valid values: first name, last name, email, department, job title and a non-negative salary")
		return
	}
	// Validate the requested login role against the active database role catalogue.
	in.RoleName = strings.TrimSpace(in.RoleName)
	if in.RoleName == "" {
		writeError(w, 400, "validation_error", "Select a login role for the new employee account")
		return
	}
	var activeRole string
	if err := s.Store.DB.QueryRow(r.Context(), `SELECT COALESCE((SELECT role_name FROM dcs_app_roles WHERE lower(role_name)=lower($1) AND active=true), '')`, in.RoleName).Scan(&activeRole); err != nil {
		log.Println("employee create: role validation query failed:", err)
		writeError(w, 500, "database_error", "Could not validate selected login role")
		return
	}
	if activeRole == "" {
		writeError(w, 400, "invalid_role", "The selected login role does not exist or is inactive")
		return
	}
	in.RoleName = activeRole
	if in.Number == "" {
		in.Number = "EMP-" + strings.ToUpper(auth.NewID()[:8])
	}
	if in.Location == "" {
		in.Location = "Head Office"
	}
	if in.Type == "" {
		in.Type = "Full-time"
	}
	startDate, dateErr := time.Parse("2006-01-02", in.StartDate)
	if dateErr != nil {
		writeError(w, 400, "validation_error", "Start date must use YYYY-MM-DD")
		return
	}
	// Required fields configured by Master Admin are enforced server-side as well as in the UI.
	configRows, configErr := s.Store.DB.Query(r.Context(), `SELECT field_key,label FROM dcs_app_master WHERE page_id='employees' AND visible=true AND required=true`)
	if configErr != nil {
		writeError(w, 500, "database_error", "Could not validate employee configuration")
		return
	}
	values := map[string]any{"firstName": in.First, "lastName": in.Last, "email": in.Email, "employeeNumber": in.Number, "department": in.Department, "jobTitle": in.Title, "location": in.Location, "employmentType": in.Type, "startDate": in.StartDate, "salary": in.SalaryMinor, "salaryMinor": in.SalaryMinor}
	for k, v := range in.CustomFields {
		values[k] = v
	}
	for configRows.Next() {
		var key, label string
		if configRows.Scan(&key, &label) != nil {
			configRows.Close()
			writeError(w, 500, "database_error", "Could not validate employee configuration")
			return
		}
		v, ok := values[key]
		if !ok || v == nil || strings.TrimSpace(fmt.Sprint(v)) == "" {
			configRows.Close()
			writeError(w, 400, "required_field", "Required field is missing: "+label)
			return
		}
	}
	configRows.Close()
	// Treat an omitted manager selection as SQL NULL. PostgreSQL foreign keys
	// reject an empty string because it is a non-NULL value with no matching employee.
	in.ManagerID = strings.TrimSpace(in.ManagerID)
	var managerArg any
	if in.ManagerID != "" && !strings.EqualFold(in.ManagerID, "null") && !strings.EqualFold(in.ManagerID, "undefined") {
		var managerExists bool
		if err := s.Store.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND status='Active')`, in.ManagerID).Scan(&managerExists); err != nil {
			log.Println("employee create: manager validation query failed:", err)
			writeError(w, 500, "database_error", "Could not validate selected manager")
			return
		}
		if !managerExists {
			writeError(w, 400, "invalid_manager", "Select an existing active employee as manager")
			return
		}
		managerArg = in.ManagerID
	}
	now := time.Now().UTC()
	id := auth.NewID()
	customJSON, _ := json.Marshal(in.CustomFields)
	_, e := s.Store.DB.Exec(r.Context(), `INSERT INTO employees(id,employee_number,first_name,last_name,email,department,job_title,location,employment_type,status,start_date,salary_minor,currency,custom_fields,manager_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'Active',$10,$11,'GBP',$12::jsonb,$13,$14,$15)`, id, in.Number, strings.TrimSpace(in.First), strings.TrimSpace(in.Last), strings.ToLower(strings.TrimSpace(in.Email)), strings.TrimSpace(in.Department), strings.TrimSpace(in.Title), in.Location, in.Type, startDate, in.SalaryMinor, string(customJSON), managerArg, now, now)
	if e != nil {
		log.Println("employee create: insert failed:", e)
		if strings.Contains(strings.ToLower(e.Error()), "unique") {
			writeError(w, 409, "conflict", "Employee number or email already exists")
		} else {
			writeError(w, 500, "database_error", "Could not create employee")
		}
		return
	}
	// Every employee receives a linked employee-role login with a random one-time password.
	tempPassword, passErr := randomTemporaryPassword()
	if passErr != nil {
		log.Println("employee create: temporary password generation failed:", passErr)
		writeError(w, 500, "account_create_failed", "Could not create employee login")
		return
	}
	passwordHash, hashErr := auth.HashPassword(tempPassword)
	if hashErr != nil {
		log.Println("employee create: password hash failed:", hashErr)
		writeError(w, 500, "account_create_failed", "Could not create employee login")
		return
	}
	userID := "user-" + id
	_, userErr := s.Store.DB.Exec(r.Context(), `INSERT INTO users(id,email,password_hash,display_name,role,employee_id,active,must_change_password) VALUES($1,$2,$3,$4,$5,$6,TRUE,TRUE)`, userID, strings.ToLower(strings.TrimSpace(in.Email)), passwordHash, strings.TrimSpace(in.First)+" "+strings.TrimSpace(in.Last), in.RoleName, id)
	if userErr != nil {
		log.Println("employee create: linked user insert failed:", userErr)
		_, _ = s.Store.DB.Exec(r.Context(), `DELETE FROM employees WHERE id=$1`, id)
		writeError(w, 500, "account_create_failed", "Could not create employee login; employee creation was rolled back")
		return
	}
	if err := sendWelcomeCredentials(in.Email, strings.TrimSpace(in.First), tempPassword); err != nil {
		log.Println("employee create: welcome email failed:", err)
		_, _ = s.Store.DB.Exec(r.Context(), `DELETE FROM users WHERE id=$1`, userID)
		_, _ = s.Store.DB.Exec(r.Context(), `DELETE FROM employees WHERE id=$1`, id)
		writeError(w, 502, "welcome_email_failed", "Employee account was not created because the welcome email could not be sent. Check SMTP configuration and try again.")
		return
	}
	if _, salaryErr := s.Store.DB.Exec(r.Context(), `INSERT INTO dsc_employee_salary_breakup(id,employee_id,gross_pay_minor,hourly_rate_minor,overtime_multiplier,currency,effective_from) VALUES($1,$2,$3,GREATEST(0,ROUND($3::numeric/2080)::bigint),1.5,'GBP',$4) ON CONFLICT(employee_id) DO NOTHING`, "salary-"+id, id, in.SalaryMinor, startDate); salaryErr != nil {
		log.Println("employee create: initial salary breakup failed:", salaryErr)
	}
	if contractErr := s.createEmploymentContract(id); contractErr != nil {
		log.Println("employee create: contract draft/signature email workflow could not complete:", contractErr)
	}
	if in.PhotoData != "" {
		parts := strings.SplitN(in.PhotoData, ",", 2)
		if len(parts) == 2 {
			raw, er := base64.StdEncoding.DecodeString(parts[1])
			mime := "image/jpeg"
			if strings.HasPrefix(parts[0], "data:image/png;") {
				mime = "image/png"
			} else if strings.HasPrefix(parts[0], "data:image/webp;") {
				mime = "image/webp"
			}
			if er == nil && len(raw) > 0 && len(raw) <= 5*1024*1024 && (mime == "image/jpeg" && strings.HasPrefix(parts[0], "data:image/jpeg;") || mime == "image/png" && strings.HasPrefix(parts[0], "data:image/png;") || mime == "image/webp" && strings.HasPrefix(parts[0], "data:image/webp;")) {
				photoID := auth.NewID()
				name := strings.TrimSpace(in.PhotoName)
				if name == "" {
					name = "profile-photo"
				}
				_, er = s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_doc_store(id,employee_id,document_name,document_type,mime_type,file_size,doc_file,verification_status,notes,uploaded_by) VALUES($1,$2,$3,'Profile photo',$4,$5,$6,'Verified','Employee profile image',$7)`, photoID, id, name, mime, len(raw), raw, u.ID)
				if er == nil {
					_, er = s.Store.DB.Exec(r.Context(), `UPDATE employees SET photo_document_id=$2 WHERE id=$1`, id, photoID)
				}
				if er != nil {
					writeError(w, 500, "photo_save_failed", "Employee was created but the profile photo could not be saved")
					return
				}
			}
		}
	}
	s.Store.Audit(r.Context(), u.ID, "employee.created", "employee", id, fmt.Sprintf("Created employee %s %s", in.First, in.Last), requestID(r))
	writeJSON(w, 201, map[string]any{"id": id, "message": "Employee created"})
}

type Shift struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	BreakMinutes int    `json:"breakMinutes"`
	GraceMinutes int    `json:"graceMinutes"`
	WorkDays     string `json:"workDays"`
	Active       bool   `json:"active"`
}

func (s *Server) shifts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.DB.Query(s.Store.Ctx, `SELECT id,name,start_time,end_time,break_minutes,grace_minutes,work_days,active FROM shifts ORDER BY active DESC,name`)
	if err != nil {
		writeError(w, 500, "database_error", "Could not load shifts")
		return
	}
	defer rows.Close()
	items := []Shift{}
	for rows.Next() {
		var x Shift
		var active bool
		if err := rows.Scan(&x.ID, &x.Name, &x.StartTime, &x.EndTime, &x.BreakMinutes, &x.GraceMinutes, &x.WorkDays, &active); err != nil {
			writeError(w, 500, "database_error", "Could not read shifts")
			return
		}
		x.Active = active
		items = append(items, x)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) createShift(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "attendance.manage") {
		writeError(w, 403, "forbidden", "Shift management permission required")
		return
	}
	var in struct {
		Name         string `json:"name"`
		StartTime    string `json:"startTime"`
		EndTime      string `json:"endTime"`
		BreakMinutes int    `json:"breakMinutes"`
		GraceMinutes int    `json:"graceMinutes"`
		WorkDays     string `json:"workDays"`
	}
	if decode(r, &in) != nil || strings.TrimSpace(in.Name) == "" || len(in.Name) > 80 || !validClock(in.StartTime) || !validClock(in.EndTime) || in.BreakMinutes < 0 || in.BreakMinutes > 360 || in.GraceMinutes < 0 || in.GraceMinutes > 180 {
		writeError(w, 400, "validation_error", "Enter a shift name, valid start/end times, and valid break/grace minutes")
		return
	}
	if strings.TrimSpace(in.WorkDays) == "" {
		in.WorkDays = "Mon,Tue,Wed,Thu,Fri"
	}
	id, now := auth.NewID(), time.Now().UTC()
	_, err := s.Store.DB.Exec(s.Store.Ctx, `INSERT INTO shifts(id,name,start_time,end_time,break_minutes,grace_minutes,work_days,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,true,$8,$9)`, id, strings.TrimSpace(in.Name), parseShiftClock(in.StartTime), parseShiftClock(in.EndTime), in.BreakMinutes, in.GraceMinutes, in.WorkDays, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, 409, "conflict", "A shift with this name already exists")
		} else {
			writeError(w, 500, "database_error", "Could not create shift")
		}
		return
	}
	s.Store.Audit(r.Context(), u.ID, "shift.created", "shift", id, "Created shift template: "+in.Name, requestID(r))
	writeJSON(w, 201, map[string]any{"id": id, "message": "Shift created"})
}
func validClock(v string) bool { _, err := time.Parse("15:04", v); return err == nil }
func (s *Server) shiftAssignments(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.DB.Query(s.Store.Ctx, `SELECT a.id,a.shift_id,s.name,a.employee_id,e.first_name||' '||e.last_name,a.shift_date,a.status,a.note,a.created_at FROM shift_assignments a JOIN shifts s ON s.id=a.shift_id JOIN employees e ON e.id=a.employee_id ORDER BY a.shift_date DESC,e.last_name LIMIT 500`)
	if err != nil {
		writeError(w, 500, "database_error", "Could not load shift assignments")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var (
			id, sid, sname, eid, ename, status, note string
			date                                     sql.NullTime
			created                                  sql.NullTime
		)

		if err := rows.Scan(&id, &sid, &sname, &eid, &ename, &date, &status, &note, &created); err != nil {
			writeError(w, 500, "database_error", "Could not read shift assignments")
			return
		}
		items = append(items, map[string]any{"id": id, "shiftId": sid, "shiftName": sname, "employeeId": eid, "employeeName": ename, "date": date.Time, "status": status, "note": note, "createdAt": created.Time})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) createShiftAssignment(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "shift.assign") {
		writeError(w, 403, "forbidden", "Shift assignment permission required")
		return
	}
	var in struct {
		ShiftID    string `json:"shiftId"`
		EmployeeID string `json:"employeeId"`
		Date       string `json:"date"`
		Note       string `json:"note"`
	}
	if decode(r, &in) != nil || in.ShiftID == "" || in.EmployeeID == "" {
		writeError(w, 400, "validation_error", "Select a shift, employee and date")
		return
	}
	shiftDate, dateErr := time.Parse("2006-01-02", in.Date)
	if dateErr != nil {
		writeError(w, 400, "validation_error", "Date must use YYYY-MM-DD")
		return
	}
	var active int
	if err := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT COUNT(*) FROM shifts WHERE id=$1 AND active=true`, in.ShiftID).Scan(&active); err != nil || active == 0 {
		writeError(w, 400, "invalid_shift", "Select an active shift")
		return
	}
	var emp int
	if err := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT COUNT(*) FROM employees WHERE id=$1 AND status='Active'`, in.EmployeeID).Scan(&emp); err != nil || emp == 0 {
		writeError(w, 400, "invalid_employee", "Select an active employee")
		return
	}
	id, now := auth.NewID(), time.Now().UTC()
	_, err := s.Store.DB.Exec(s.Store.Ctx, `INSERT INTO shift_assignments(id,shift_id,employee_id,shift_date,status,note,created_by,created_at) VALUES($1,$2,$3,$4,'Assigned',$5,$6,$7)`, id, in.ShiftID, in.EmployeeID, shiftDate, strings.TrimSpace(in.Note), u.ID, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, 409, "assignment_conflict", "This employee already has a shift assigned for that date")
		} else {
			writeError(w, 500, "database_error", "Could not assign shift")
		}
		return
	}
	s.Store.Audit(r.Context(), u.ID, "shift.assigned", "shift_assignment", id, "Assigned shift on "+in.Date, requestID(r))
	writeJSON(w, 201, map[string]any{"id": id, "message": "Shift assigned"})
}
func (s *Server) deleteShiftAssignment(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "shift.assign") {
		writeError(w, 403, "forbidden", "Shift assignment permission required")
		return
	}
	id := r.PathValue("id")
	res, err := s.Store.DB.Exec(s.Store.Ctx, `DELETE FROM shift_assignments WHERE id=$1`, id)
	if err != nil {
		writeError(w, 500, "database_error", "Could not remove assignment")
		return
	}
	n := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "not_found", "Shift assignment not found")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "shift.assignment_removed", "shift_assignment", id, "Removed shift assignment", requestID(r))
	w.WriteHeader(204)
}

func (s *Server) attendance(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	rows, e := s.Store.DB.Query(s.Store.Ctx, `SELECT a.id,a.employee_id,e.first_name||' '||e.last_name,a.event_type,a.occurred_at,a.source,a.note FROM attendance_events a JOIN employees e ON e.id=a.employee_id WHERE ($1='' OR a.employee_id=$2) ORDER BY a.occurred_at DESC LIMIT 100`, u.EmployeeID, u.EmployeeID)
	if e != nil {
		writeError(w, 500, "database_error", "Could not load attendance")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, emp, name, typ, source, note string
		var at sql.NullTime
		if err := rows.Scan(&id, &emp, &name, &typ, &at, &source, &note); err != nil {
			log.Println("attendance: row scan failed:", err)
			writeError(w, 500, "database_error", "Could not read attendance")
			return
		}
		var occurred any
		if at.Valid {
			occurred = at.Time
		}
		items = append(items, map[string]any{"id": id, "employeeId": emp, "employeeName": name, "type": typ, "occurredAt": occurred, "source": source, "note": note})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) punch(w http.ResponseWriter, r *http.Request, typ string) {
	u := current(r)
	if u.EmployeeID == "" {
		writeError(w, 403, "forbidden", "Your user account is not linked to an employee profile")
		return
	}
	var last string
	e := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT event_type FROM attendance_events WHERE employee_id=$1 ORDER BY occurred_at DESC LIMIT 1`, u.EmployeeID).Scan(&last)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		writeError(w, 500, "database_error", "Could not validate attendance")
		return
	}
	if typ == "check_in" && last == "check_in" {
		writeError(w, 409, "already_checked_in", "You are already checked in")
		return
	}
	if typ == "check_out" && last != "check_in" {
		writeError(w, 409, "not_checked_in", "Check in before checking out")
		return
	}
	now := time.Now().UTC()
	id := auth.NewID()
	_, e = s.Store.DB.Exec(s.Store.Ctx, `INSERT INTO attendance_events(id,employee_id,event_type,occurred_at,source) VALUES($1,$2,$3,$4,$5)`, id, u.EmployeeID, typ, now, "web")
	if e != nil {
		writeError(w, 500, "database_error", "Could not record attendance")
		return
	}
	if typ == "check_out" || typ == "check_in" {
		s.recalculateActualHours(u.EmployeeID, now)
	}
	s.Store.Audit(r.Context(), u.ID, "attendance."+typ, "attendance", id, "Attendance event recorded", requestID(r))
	writeJSON(w, 201, map[string]any{"id": id, "type": typ, "occurredAt": now})
}
func (s *Server) checkIn(w http.ResponseWriter, r *http.Request)  { s.punch(w, r, "check_in") }
func (s *Server) checkOut(w http.ResponseWriter, r *http.Request) { s.punch(w, r, "check_out") }
func (s *Server) leaveRequests(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	query := `SELECT l.id,l.employee_id,e.first_name||' '||e.last_name,l.leave_type,l.start_date,l.end_date,l.days,l.reason,l.status,l.submitted_at,COALESCE(l.decision_note,'') FROM leave_requests l JOIN employees e ON e.id=l.employee_id`
	args := []any{}
	if !s.hasPermission(r.Context(), u.Role, "leave.review") {
		query += ` WHERE l.employee_id=$1`
		args = append(args, u.EmployeeID)
	}
	query += ` ORDER BY CASE l.status WHEN 'Pending' THEN 0 ELSE 1 END,l.submitted_at DESC LIMIT 200`
	rows, e := s.Store.DB.Query(s.Store.Ctx, query, args...)
	if e != nil {
		writeError(w, 500, "database_error", "Could not load leave requests")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, emp, name, typ, reason, status, note string
		var start, end, submitted sql.NullTime
		var days float64
		if err := rows.Scan(&id, &emp, &name, &typ, &start, &end, &days, &reason, &status, &submitted, &note); err != nil {
			log.Println("leave requests: row scan failed:", err)
			writeError(w, 500, "database_error", "Could not read leave requests")
			return
		}
		dateText := func(v sql.NullTime) any {
			if !v.Valid {
				return nil
			}
			return v.Time.Format("2006-01-02")
		}
		items = append(items, map[string]any{"id": id, "employeeId": emp, "employeeName": name, "leaveType": typ, "startDate": dateText(start), "endDate": dateText(end), "days": days, "reason": reason, "status": status, "submittedAt": nullableTime(submitted), "decisionNote": note})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) createLeave(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if u.EmployeeID == "" {
		writeError(w, 403, "forbidden", "Your user account is not linked to an employee profile")
		return
	}
	var in struct {
		Type   string  `json:"leaveType"`
		Start  string  `json:"startDate"`
		End    string  `json:"endDate"`
		Days   float64 `json:"days"`
		Reason string  `json:"reason"`
	}
	if decode(r, &in) != nil || in.Type == "" || in.Days <= 0 || in.Days > 365 || len(in.Reason) > 1000 {
		writeError(w, 400, "validation_error", "Enter a leave type, valid dates, days and reason")
		return
	}
	start, e1 := time.Parse("2006-01-02", in.Start)
	end, e2 := time.Parse("2006-01-02", in.End)
	if e1 != nil || e2 != nil || end.Before(start) {
		writeError(w, 400, "validation_error", "End date must be on or after start date")
		return
	}
	id := auth.NewID()
	now := time.Now().UTC()
	_, e := s.Store.DB.Exec(s.Store.Ctx, `INSERT INTO leave_requests(id,employee_id,leave_type,start_date,end_date,days,reason,status,submitted_at) VALUES($1,$2,$3,$4,$5,$6,$7,'Pending',$8)`, id, u.EmployeeID, in.Type, start, end, in.Days, strings.TrimSpace(in.Reason), now)
	if e != nil {
		writeError(w, 500, "database_error", "Could not submit leave request")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "leave.submitted", "leave_request", id, "Leave request submitted", requestID(r))
	writeJSON(w, 201, map[string]any{"id": id, "status": "Pending"})
}
func (s *Server) leaveDecision(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "leave.review") {
		writeError(w, 403, "forbidden", "You cannot approve leave requests")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if decode(r, &in) != nil || (in.Decision != "Approved" && in.Decision != "Rejected") || len(in.Note) > 1000 {
		writeError(w, 400, "validation_error", "Decision must be Approved or Rejected")
		return
	}
	var emp, status string
	e := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT employee_id,status FROM leave_requests WHERE id=$1`, id).Scan(&emp, &status)
	if e != nil {
		writeError(w, 404, "not_found", "Leave request not found")
		return
	}
	if status != "Pending" {
		writeError(w, 409, "already_decided", "This request has already been decided")
		return
	}
	if u.EmployeeID != "" && u.EmployeeID == emp {
		writeError(w, 403, "self_approval", "You cannot approve your own request")
		return
	}
	now := time.Now().UTC()
	res, e := s.Store.DB.Exec(s.Store.Ctx, `UPDATE leave_requests SET status=$1,decided_at=$2,decided_by=$3,decision_note=$4 WHERE id=$5 AND status='Pending'`, in.Decision, now, u.ID, strings.TrimSpace(in.Note), id)
	if e != nil {
		writeError(w, 500, "database_error", "Could not decide request")
		return
	}
	n := res.RowsAffected()
	if n != 1 {
		writeError(w, 409, "already_decided", "This request has already been decided")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "leave."+strings.ToLower(in.Decision), "leave_request", id, "Leave request decision: "+in.Decision, requestID(r))
	writeJSON(w, 200, map[string]any{"id": id, "status": in.Decision})
}
func (s *Server) payroll(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "reports.read") {
		writeError(w, 403, "forbidden", "Payroll permission required")
		return
	}
	rows, e := s.Store.DB.Query(s.Store.Ctx, `SELECT id,period_label,start_date,end_date,status,employee_count,gross_minor,deductions_minor,net_minor FROM payroll_periods ORDER BY start_date DESC`)
	if e != nil {
		writeError(w, 500, "database_error", "Could not load payroll periods")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, label, status string
		var start, end sql.NullTime
		var count int
		var gross, ded, net int64
		if err := rows.Scan(&id, &label, &start, &end, &status, &count, &gross, &ded, &net); err != nil {
			log.Println("payroll periods: row scan failed:", err)
			writeError(w, 500, "database_error", "Could not read payroll periods")
			return
		}
		dateText := func(v sql.NullTime) any {
			if !v.Valid {
				return nil
			}
			return v.Time.Format("2006-01-02")
		}
		items = append(items, map[string]any{"id": id, "periodLabel": label, "startDate": dateText(start), "endDate": dateText(end), "status": status, "employeeCount": count, "grossMinor": gross, "deductionsMinor": ded, "netMinor": net})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "audit.read") {
		writeError(w, 403, "forbidden", "Audit permission required")
		return
	}
	rows, e := s.Store.DB.Query(s.Store.Ctx, `SELECT id,COALESCE(actor_user_id,''),action,target_type,target_id,summary,created_at FROM audit_events ORDER BY created_at DESC LIMIT 200`)
	if e != nil {
		writeError(w, 500, "database_error", "Could not load audit events")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, actor, action, typ, target, summary string
		var at sql.NullTime
		if err := rows.Scan(&id, &actor, &action, &typ, &target, &summary, &at); err != nil {
			log.Println("audit: row scan failed:", err)
			writeError(w, 500, "database_error", "Could not read audit events")
			return
		}
		items = append(items, map[string]any{"id": id, "actorUserId": actor, "action": action, "targetType": typ, "targetId": target, "summary": summary, "createdAt": nullableTime(at)})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) headcountCSV(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "reports.read") {
		writeError(w, 403, "forbidden", "Report permission required")
		return
	}
	rows, e := s.Store.DB.Query(s.Store.Ctx, `SELECT employee_number,first_name,last_name,email,department,job_title,status,start_date FROM employees ORDER BY department,last_name`)
	if e != nil {
		writeError(w, 500, "database_error", "Could not create report")
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=headcount.csv")
	_, _ = w.Write([]byte("Employee Number,First Name,Last Name,Email,Department,Job Title,Status,Start Date\r\n"))
	for rows.Next() {
		var a, b, c, d, e, f, g string
		var h sql.NullTime
		if err := rows.Scan(&a, &b, &c, &d, &e, &f, &g, &h); err != nil {
			log.Println("headcount report: row scan failed:", err)
			writeError(w, 500, "database_error", "Could not read headcount report")
			return
		}
		dateValue := ""
		if h.Valid {
			dateValue = h.Time.Format("2006-01-02")
		}
		vals := []string{a, b, c, d, e, f, g, dateValue}
		for i, v := range vals {
			vals[i] = "\"" + strings.ReplaceAll(v, "\"", "\"\"") + "\""
		}
		_, _ = fmt.Fprintf(w, "%s\r\n", strings.Join(vals, ","))
	}
	s.Store.Audit(r.Context(), u.ID, "report.export", "report", "headcount", "Exported headcount CSV", requestID(r))
}
func hasRole(role string, allowed ...string) bool {
	for _, x := range allowed {
		if role == x {
			return true
		}
	}
	return false
}
func requestID(r *http.Request) string {
	x, _ := r.Context().Value(ctxKey("request_id")).(string)
	return x
}

func randomTemporaryPassword() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "P!" + hex.EncodeToString(b), nil
}
func sendWelcomeCredentials(to, first, password string) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	user := os.Getenv("SMTP_USERNAME")
	pass := os.Getenv("SMTP_PASSWORD")
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if host == "" || from == "" {
		return errors.New("SMTP_HOST and SMTP_FROM must be configured")
	}
	if port == "" {
		port = "587"
	}
	addr := host + ":" + port
	body := "To: " + to + "\r\nSubject: Your PeopleOps https://peopleops.dataconsoft.com/ account\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nHello " + first + ",\r\n\r\nYour PeopleOps account has been created.\r\nUsername: " + to + "\r\nTemporary password: " + password + "\r\n\r\nYou must change this password after your first login.\r\n"
	var authSMTP smtp.Auth
	if user != "" {
		authSMTP = smtp.PlainAuth("", user, pass, host)
	}
	return smtp.SendMail(addr, authSMTP, from, []string{to}, []byte(body))
}
func (s *Server) employeeHierarchy(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.DB.Query(r.Context(), `SELECT e.id,e.employee_number,e.first_name,e.last_name,e.email,e.department,e.job_title,COALESCE(e.manager_id,''),COALESCE(m.first_name||' '||m.last_name,'') FROM employees e LEFT JOIN employees m ON m.id=e.manager_id WHERE e.status <> 'Terminated' ORDER BY e.last_name,e.first_name`)
	if err != nil {
		log.Println("employee hierarchy query failed:", err)
		writeError(w, 500, "database_error", "Could not load employee hierarchy")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, num, first, last, email, dept, title, managerID, managerName string
		if err := rows.Scan(&id, &num, &first, &last, &email, &dept, &title, &managerID, &managerName); err != nil {
			log.Println("employee hierarchy scan failed:", err)
			writeError(w, 500, "database_error", "Could not read employee hierarchy")
			return
		}
		items = append(items, map[string]any{"id": id, "employeeNumber": num, "name": first + " " + last, "email": email, "department": dept, "jobTitle": title, "managerId": managerID, "managerName": managerName})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	var in struct {
		Current string `json:"currentPassword"`
		New     string `json:"newPassword"`
	}
	if err := decode(r, &in); err != nil || len(in.New) < 12 || len(in.New) > 200 {
		writeError(w, 400, "invalid_password", "New password must be at least 12 characters")
		return
	}
	var hash string
	if err := s.Store.DB.QueryRow(r.Context(), `SELECT password_hash FROM users WHERE id=$1`, u.ID).Scan(&hash); err != nil || !auth.CheckPassword(hash, in.Current) {
		writeError(w, 400, "invalid_password", "Current password is incorrect")
		return
	}
	newHash, err := auth.HashPassword(in.New)
	if err != nil {
		writeError(w, 500, "internal", "Could not update password")
		return
	}
	if _, err = s.Store.DB.Exec(r.Context(), `UPDATE users SET password_hash=$2,must_change_password=FALSE WHERE id=$1`, u.ID, newHash); err != nil {
		log.Println("change password update failed:", err)
		writeError(w, 500, "database_error", "Could not update password")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
