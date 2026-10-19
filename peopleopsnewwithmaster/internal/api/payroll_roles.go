package api

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"peopleops/internal/auth"
)

// hasPermission reads the active role's permission from PostgreSQL on each check,
// so authorization changes take effect without recompiling or restarting the app.
func (s *Server) hasPermission(ctx context.Context, role, permission string) bool {
	var allowed bool
	err := s.Store.DB.QueryRow(ctx, `SELECT COALESCE(bool_or(p.enabled),false) FROM dcs_app_roles r LEFT JOIN dcs_role_permissions p ON p.role_id=r.id AND p.permission_key=$2 WHERE lower(r.role_name)=lower($1) AND r.active=true`, strings.TrimSpace(role), permission).Scan(&allowed)
	if err != nil {
		log.Println("authorization: permission lookup failed for role", role, "permission", permission, ":", err)
		return false
	}
	return allowed
}
func (s *Server) adminRole(ctx context.Context, role string) bool {
	return s.hasPermission(ctx, role, "roles.manage")
}
func (s *Server) accountsRole(ctx context.Context, role string) bool {
	return s.hasPermission(ctx, role, "payroll.manage")
}
func (s *Server) hrAccountsRole(ctx context.Context, role string) bool {
	return s.hasPermission(ctx, role, "hr.payroll.read")
}

func (s *Server) roles(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	// HR users need to read the role catalogue for employee form dropdowns; only admins may mutate it.
	if r.Method != http.MethodGet && !s.adminRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "Master Admin permission required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := s.Store.DB.Query(r.Context(), `SELECT id,role_name,description,active,system_role FROM dcs_app_roles ORDER BY role_name`)
		if err != nil {
			log.Println("roles: list query failed:", err)
			writeError(w, 500, "database_error", "Could not load roles")
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, name, desc string
			var active, system bool
			if err := rows.Scan(&id, &name, &desc, &active, &system); err != nil {
				log.Println("roles: scan failed:", err)
				writeError(w, 500, "database_error", "Could not read roles")
				return
			}
			items = append(items, map[string]any{"id": id, "roleName": name, "description": desc, "active": active, "systemRole": system})
		}
		if err := rows.Err(); err != nil {
			log.Println("roles: iteration failed:", err)
			writeError(w, 500, "database_error", "Could not read roles")
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	case http.MethodPost:
		var in struct {
			RoleName    string `json:"roleName"`
			Description string `json:"description"`
			Active      bool   `json:"active"`
		}
		if err := decode(r, &in); err != nil || strings.TrimSpace(in.RoleName) == "" || len(in.RoleName) > 80 {
			log.Println("roles: invalid create request:", err)
			writeError(w, 400, "validation_error", "Role name is required")
			return
		}
		id := auth.NewID()
		_, err := s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_app_roles(id,role_name,description,active) VALUES($1,$2,$3,$4)`, id, strings.TrimSpace(in.RoleName), strings.TrimSpace(in.Description), in.Active)
		if err != nil {
			log.Println("roles: create failed:", err)
			writeError(w, 409, "role_conflict", "Could not create role; role names must be unique")
			return
		}
		s.Store.Audit(r.Context(), u.ID, "role.created", "role", id, "Created role "+in.RoleName, requestID(r))
		writeJSON(w, 201, map[string]any{"id": id, "message": "Role created"})
	case http.MethodPut:
		var in struct {
			ID          string `json:"id"`
			RoleName    string `json:"roleName"`
			Description string `json:"description"`
			Active      bool   `json:"active"`
		}
		if err := decode(r, &in); err != nil || in.ID == "" || strings.TrimSpace(in.RoleName) == "" {
			log.Println("roles: invalid update request:", err)
			writeError(w, 400, "validation_error", "Role id and name are required")
			return
		}
		tx, err := s.Store.DB.Begin(r.Context())
		if err != nil {
			log.Println("roles: update transaction begin failed:", err)
			writeError(w, 500, "database_error", "Could not update role")
			return
		}
		defer tx.Rollback(r.Context())
		var oldName string
		if err = tx.QueryRow(r.Context(), `SELECT role_name FROM dcs_app_roles WHERE id=$1 AND system_role=false FOR UPDATE`, in.ID).Scan(&oldName); err != nil {
			log.Println("roles: protected or missing role:", err)
			writeError(w, 404, "not_editable", "Role not found or is a protected system role")
			return
		}
		_, err = tx.Exec(r.Context(), `UPDATE dcs_app_roles SET role_name=$2,description=$3,active=$4,updated_at=now() WHERE id=$1 AND system_role=false`, in.ID, strings.TrimSpace(in.RoleName), strings.TrimSpace(in.Description), in.Active)
		if err != nil {
			log.Println("roles: update failed:", err)
			writeError(w, 409, "role_update_failed", "Could not update role; check that the name is unique")
			return
		}
		if oldName != strings.TrimSpace(in.RoleName) {
			if _, err = tx.Exec(r.Context(), `UPDATE users SET role=$1 WHERE role=$2`, strings.TrimSpace(in.RoleName), oldName); err != nil {
				log.Println("roles: update assigned users failed:", err)
				writeError(w, 500, "database_error", "Could not update assigned users")
				return
			}
		}
		if err = tx.Commit(r.Context()); err != nil {
			log.Println("roles: update commit failed:", err)
			writeError(w, 500, "database_error", "Could not save role")
			return
		}
		s.Store.Audit(r.Context(), u.ID, "role.updated", "role", in.ID, "Updated role "+in.RoleName, requestID(r))
		writeJSON(w, 200, map[string]any{"message": "Role updated"})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, 400, "validation_error", "Role id is required")
			return
		}
		tag, err := s.Store.DB.Exec(r.Context(), `DELETE FROM dcs_app_roles r WHERE r.id=$1 AND r.system_role=false AND NOT EXISTS (SELECT 1 FROM users u WHERE u.role=r.role_name)`, id)
		if err != nil {
			log.Println("roles: delete failed:", err)
			writeError(w, 409, "role_delete_failed", "Role is in use or could not be removed")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, 404, "not_removable", "Role not found or is a protected system role")
			return
		}
		s.Store.Audit(r.Context(), u.ID, "role.deleted", "role", id, "Removed role", requestID(r))
		writeJSON(w, 200, map[string]any{"message": "Role removed"})
	default:
		writeError(w, 405, "method_not_allowed", "Method not allowed")
	}
}

func (s *Server) rolePages(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.adminRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "Master Admin permission required")
		return
	}
	roleID := r.URL.Query().Get("roleId")
	if r.Method == http.MethodGet {
		rows, err := s.Store.DB.Query(r.Context(), `SELECT page_id,page_name,enabled FROM dcs_role_pages WHERE role_id=$1 ORDER BY page_name`, roleID)
		if err != nil {
			log.Println("role pages: list failed:", err)
			writeError(w, 500, "database_error", "Could not load page access")
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, name string
			var enabled bool
			if err := rows.Scan(&id, &name, &enabled); err != nil {
				log.Println("role pages: scan failed:", err)
				writeError(w, 500, "database_error", "Could not read page access")
				return
			}
			items = append(items, map[string]any{"pageId": id, "pageName": name, "enabled": enabled})
		}
		writeJSON(w, 200, map[string]any{"items": items})
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, 405, "method_not_allowed", "Method not allowed")
		return
	}
	var in struct {
		RoleID string `json:"roleId"`
		Pages  []struct {
			PageID   string `json:"pageId"`
			PageName string `json:"pageName"`
			Enabled  bool   `json:"enabled"`
		} `json:"pages"`
	}
	if err := decode(r, &in); err != nil || in.RoleID == "" {
		log.Println("role pages: invalid update:", err)
		writeError(w, 400, "validation_error", "Role id and page list are required")
		return
	}
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		log.Println("role pages: begin failed:", err)
		writeError(w, 500, "database_error", "Could not update page access")
		return
	}
	defer tx.Rollback(r.Context())
	for _, p := range in.Pages {
		if strings.TrimSpace(p.PageID) == "" || strings.TrimSpace(p.PageName) == "" {
			continue
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO dcs_role_pages(role_id,page_id,page_name,enabled) VALUES($1,$2,$3,$4) ON CONFLICT(role_id,page_id) DO UPDATE SET page_name=EXCLUDED.page_name,enabled=EXCLUDED.enabled`, in.RoleID, p.PageID, p.PageName, p.Enabled)
		if err != nil {
			log.Println("role pages: upsert failed:", err)
			writeError(w, 500, "database_error", "Could not update page access")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		log.Println("role pages: commit failed:", err)
		writeError(w, 500, "database_error", "Could not save page access")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "role.pages_updated", "role", in.RoleID, "Updated role page access", requestID(r))
	writeJSON(w, 200, map[string]any{"message": "Page access updated"})
}

func bankCipher() (cipher.AEAD, error) {
	key := []byte(strings.TrimSpace(os.Getenv("BANK_DETAILS_KEY")))
	if len(key) == 0 {
		return nil, errors.New("BANK_DETAILS_KEY is not configured")
	}
	sum := sha256.Sum256(key)
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func encryptBank(v string) (string, error) {
	a, err := bankCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(append(nonce, a.Seal(nil, nonce, []byte(v), nil)...)), nil
}
func decryptBank(v string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}
	a, err := bankCipher()
	if err != nil {
		return "", err
	}
	if len(raw) < a.NonceSize() {
		return "", errors.New("invalid encrypted bank value")
	}
	plain, err := a.Open(nil, raw[:a.NonceSize()], raw[a.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
func (s *Server) employeeBank(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	empID := r.PathValue("id")
	if !s.hrAccountsRole(r.Context(), u.Role) && u.EmployeeID != empID {
		writeError(w, 403, "forbidden", "Bank details are available only to HR, Accounts and the employee concerned")
		return
	}
	if r.Method == http.MethodGet {
		var name, bank, sort, acct, iban, bic string
		err := s.Store.DB.QueryRow(r.Context(), `SELECT account_name,bank_name,sort_code,account_number_encrypted,iban_encrypted,bic_swift FROM dcs_employee_bank_details WHERE employee_id=$1`, empID).Scan(&name, &bank, &sort, &acct, &iban, &bic)
		if err != nil {
			log.Println("employee bank: read failed:", err)
			writeError(w, 404, "not_found", "Bank details not found")
			return
		}
		acctPlain, err := decryptBank(acct)
		if err != nil {
			log.Println("employee bank: account decrypt failed:", err)
			writeError(w, 500, "bank_crypto_error", "Bank details cannot be decrypted; check BANK_DETAILS_KEY")
			return
		}
		ibanPlain := ""
		if iban != "" {
			ibanPlain, err = decryptBank(iban)
			if err != nil {
				log.Println("employee bank: IBAN decrypt failed:", err)
				writeError(w, 500, "bank_crypto_error", "Bank details cannot be decrypted; check BANK_DETAILS_KEY")
				return
			}
		}
		if !s.hrAccountsRole(r.Context(), u.Role) {
			acctPlain = maskAccount(acctPlain)
			ibanPlain = maskAccount(ibanPlain)
		}
		writeJSON(w, 200, map[string]any{"accountName": name, "bankName": bank, "sortCode": sort, "accountNumber": acctPlain, "iban": ibanPlain, "bicSwift": bic})
		return
	}
	if r.Method != http.MethodPut || !s.hrAccountsRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "HR or Accounts permission required to update bank details")
		return
	}
	var in struct {
		AccountName   string `json:"accountName"`
		BankName      string `json:"bankName"`
		SortCode      string `json:"sortCode"`
		AccountNumber string `json:"accountNumber"`
		IBAN          string `json:"iban"`
		BIC           string `json:"bicSwift"`
	}
	if err := decode(r, &in); err != nil || strings.TrimSpace(in.AccountName) == "" || strings.TrimSpace(in.SortCode) == "" || strings.TrimSpace(in.AccountNumber) == "" {
		log.Println("employee bank: invalid payload:", err)
		writeError(w, 400, "validation_error", "Account name, sort code and account number are required")
		return
	}
	acct, err := encryptBank(in.AccountNumber)
	if err != nil {
		log.Println("employee bank: encryption unavailable:", err)
		writeError(w, 503, "bank_encryption_unavailable", "Configure BANK_DETAILS_KEY before storing bank details")
		return
	}
	iban := ""
	if in.IBAN != "" {
		iban, err = encryptBank(in.IBAN)
		if err != nil {
			log.Println("employee bank: IBAN encryption failed:", err)
			writeError(w, 500, "bank_crypto_error", "Could not encrypt IBAN")
			return
		}
	}
	_, err = s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_employee_bank_details(id,employee_id,account_name,bank_name,sort_code,account_number_encrypted,iban_encrypted,bic_swift,updated_by,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,now()) ON CONFLICT(employee_id) DO UPDATE SET account_name=EXCLUDED.account_name,bank_name=EXCLUDED.bank_name,sort_code=EXCLUDED.sort_code,account_number_encrypted=EXCLUDED.account_number_encrypted,iban_encrypted=EXCLUDED.iban_encrypted,bic_swift=EXCLUDED.bic_swift,updated_by=EXCLUDED.updated_by,updated_at=now()`, auth.NewID(), empID, strings.TrimSpace(in.AccountName), strings.TrimSpace(in.BankName), strings.TrimSpace(in.SortCode), acct, iban, strings.TrimSpace(in.BIC), u.ID)
	if err != nil {
		log.Println("employee bank: save failed:", err)
		writeError(w, 500, "database_error", "Could not save bank details")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "employee.bank_updated", "employee", empID, "Updated employee bank details", requestID(r))
	writeJSON(w, 200, map[string]any{"message": "Bank details saved securely"})
}
func maskAccount(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

func (s *Server) actualHours(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hrAccountsRole(r.Context(), u.Role) && !s.hasPermission(r.Context(), u.Role, "attendance.team") {
		writeError(w, 403, "forbidden", "Actual hours permission required")
		return
	}
	month := r.URL.Query().Get("month")
	query := `SELECT h.employee_id,e.first_name||' '||e.last_name,h.work_date,h.check_in_at,h.check_out_at,h.assigned_shift_minutes,h.actual_minutes,h.normal_minutes,h.overtime_minutes,h.payment_pending,h.approved,COALESCE(h.approved_by,''),h.approved_at,h.approval_note FROM dcs_employees_actual_hours h JOIN employees e ON e.id=h.employee_id WHERE ($1='' OR to_char(h.work_date,'YYYY-MM')=$1) ORDER BY h.work_date DESC,e.last_name LIMIT 1000`
	rows, err := s.Store.DB.Query(r.Context(), query, month)
	if err != nil {
		log.Println("actual hours: list query failed:", err)
		writeError(w, 500, "database_error", "Could not load actual hours")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var emp, name string
		var day time.Time
		var cin, cout sql.NullTime
		var assigned, actual, normal, ot int
		var pending, approved bool
		var approvedBy, approvalNote sql.NullString
		var approvedAt sql.NullTime
		if err := rows.Scan(&emp, &name, &day, &cin, &cout, &assigned, &actual, &normal, &ot, &pending, &approved, &approvedBy, &approvedAt, &approvalNote); err != nil {
			log.Println("actual hours: scan failed:", err)
			writeError(w, 500, "database_error", "Could not read actual hours")
			return
		}
		items = append(items, map[string]any{"employeeId": emp, "employeeName": name, "workDate": day.Format("2006-01-02"), "checkInAt": nullableTime(cin), "checkOutAt": nullableTime(cout), "assignedHours": float64(assigned) / 60, "actualHours": float64(actual) / 60, "normalHours": float64(normal) / 60, "overtimeHours": float64(ot) / 60, "paymentPending": pending, "approved": approved, "approvedBy": approvedBy.String, "approvedAt": nullableTime(approvedAt), "approvalNote": approvalNote.String, "canApprove": s.hasPermission(r.Context(), u.Role, "attendance.approve")})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

// approveActualHours records an auditable approval. It refuses rows without an assigned shift,
// because normal/overtime hours cannot be validated against a zero-hour shift.
func (s *Server) approveActualHours(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "attendance.approve") {
		writeError(w, 403, "forbidden", "Attendance approval permission required")
		return
	}
	var in struct {
		EmployeeID string `json:"employeeId"`
		WorkDate   string `json:"workDate"`
		Note       string `json:"note"`
	}
	if err := decode(r, &in); err != nil || strings.TrimSpace(in.EmployeeID) == "" || len(in.Note) > 500 {
		writeError(w, 400, "validation_error", "Employee, work date and a note of at most 500 characters are required")
		return
	}
	day, err := time.Parse("2006-01-02", in.WorkDate)
	if err != nil {
		writeError(w, 400, "validation_error", "Work date must use YYYY-MM-DD")
		return
	}
	tag, err := s.Store.DB.Exec(r.Context(), `UPDATE dcs_employees_actual_hours SET approved=true,approved_by=$3,approved_at=now(),approval_note=$4,updated_at=now() WHERE employee_id=$1 AND work_date=$2 AND assigned_shift_minutes>0 AND check_in_at IS NOT NULL AND check_out_at IS NOT NULL`, in.EmployeeID, day, u.ID, strings.TrimSpace(in.Note))
	if err != nil {
		log.Println("actual hours: approval update failed:", err)
		writeError(w, 500, "database_error", "Could not approve actual hours")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, 409, "approval_not_ready", "Cannot approve: completed attendance and an assigned shift with positive duration are required")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "attendance.hours_approved", "actual_hours", in.EmployeeID+":"+day.Format("2006-01-02"), "Approved actual hours and overtime", requestID(r))
	writeJSON(w, 200, map[string]any{"message": "Actual hours and overtime approved"})
}

func nullableTime(n sql.NullTime) any {
	if !n.Valid {
		return nil
	}
	return n.Time
}

func (s *Server) recalculateActualHours(employeeID string, day time.Time) {
	ctx := s.Store.Ctx
	var cin, cout sql.NullTime
	err := s.Store.DB.QueryRow(ctx, `SELECT MIN(occurred_at),MAX(occurred_at) FILTER (WHERE event_type='check_out') FROM attendance_events WHERE employee_id=$1 AND occurred_at::date=$2 AND event_type IN ('check_in','check_out')`, employeeID, day.Format("2006-01-02")).Scan(&cin, &cout)
	if err != nil {
		log.Println("actual hours: attendance aggregation failed:", err)
		return
	}
	if !cin.Valid {
		return
	}
	var assigned int
	err = s.Store.DB.QueryRow(ctx, `SELECT GREATEST(0,CASE WHEN s.end_time>=s.start_time THEN EXTRACT(EPOCH FROM (s.end_time-s.start_time))/60 ELSE EXTRACT(EPOCH FROM (s.end_time-s.start_time+interval '24 hours'))/60 END-s.break_minutes)::int FROM shift_assignments a JOIN shifts s ON s.id=a.shift_id WHERE a.employee_id=$1 AND a.shift_date=$2 AND a.status='Assigned'`, employeeID, day.Format("2006-01-02")).Scan(&assigned)
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "no rows") {
			log.Println("actual hours: assigned shift lookup failed:", err)
		}
		assigned = 0
	}
	actual := 0
	if cout.Valid {
		actual = int(cout.Time.Sub(cin.Time).Minutes())
		if actual < 0 {
			actual = 0
		}
	}
	normal := actual
	if assigned > 0 && normal > assigned {
		normal = assigned
	}
	overtime := actual - normal
	id := auth.NewID()
	_, err = s.Store.DB.Exec(ctx, `INSERT INTO dcs_employees_actual_hours(id,employee_id,work_date,check_in_at,check_out_at,assigned_shift_minutes,actual_minutes,normal_minutes,overtime_minutes,payment_pending,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,true,now()) ON CONFLICT(employee_id,work_date) DO UPDATE SET check_in_at=EXCLUDED.check_in_at,check_out_at=EXCLUDED.check_out_at,assigned_shift_minutes=EXCLUDED.assigned_shift_minutes,actual_minutes=EXCLUDED.actual_minutes,normal_minutes=EXCLUDED.normal_minutes,overtime_minutes=EXCLUDED.overtime_minutes,payment_pending=CASE WHEN dcs_employees_actual_hours.payment_pending THEN true ELSE false END,updated_at=now()`, id, employeeID, day.Format("2006-01-02"), cin.Time, nullableTime(cout), assigned, actual, normal, overtime)
	if err != nil {
		log.Println("actual hours: upsert failed:", err)
	}
}

func (s *Server) payrollAdvanced(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.accountsRole(r.Context(), u.Role) {
		if strings.TrimSpace(u.EmployeeID) == "" {
			writeJSON(w, 200, map[string]any{"mode": "employee", "items": []any{}})
			return
		}
		rows, err := s.Store.DB.Query(r.Context(), `SELECT id,period_start,period_end,currency,normal_hours,overtime_hours,gross_minor,deductions_minor,net_minor,status FROM dcs_salary_summary WHERE employee_id=$1 AND status='Paid' ORDER BY period_start DESC`, u.EmployeeID)
		if err != nil {
			log.Println("payroll: employee summaries failed:", err)
			writeError(w, 500, "database_error", "Could not load your payslips")
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, currency, status string
			var start, end time.Time
			var normal, ot float64
			var gross, ded, net int64
			if err := rows.Scan(&id, &start, &end, &currency, &normal, &ot, &gross, &ded, &net, &status); err != nil {
				log.Println("payroll: employee summary scan failed:", err)
				writeError(w, 500, "database_error", "Could not read payslips")
				return
			}
			items = append(items, map[string]any{"id": id, "periodStart": start.Format("2006-01-02"), "periodEnd": end.Format("2006-01-02"), "currency": currency, "normalHours": normal, "overtimeHours": ot, "grossMinor": gross, "deductionsMinor": ded, "netMinor": net, "status": status, "payslipUrl": "/api/v1/payroll/" + id + "/payslip.pdf"})
		}
		writeJSON(w, 200, map[string]any{"mode": "employee", "items": items})
		return
	}
	rows, err := s.Store.DB.Query(r.Context(), `SELECT id,employee_id,period_start,period_end,entry_type,status,amount_minor,currency,reference,created_at FROM dcs_payments ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		log.Println("payroll: payments list failed:", err)
		writeError(w, 500, "database_error", "Could not load payment ledger")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, emp, typ, status, currency, ref string
		var start, end time.Time
		var amount int64
		var created time.Time
		if err := rows.Scan(&id, &emp, &start, &end, &typ, &status, &amount, &currency, &ref, &created); err != nil {
			log.Println("payroll: payment scan failed:", err)
			writeError(w, 500, "database_error", "Could not read payment ledger")
			return
		}
		items = append(items, map[string]any{"id": id, "employeeId": emp, "periodStart": start.Format("2006-01-02"), "periodEnd": end.Format("2006-01-02"), "entryType": typ, "status": status, "amountMinor": amount, "currency": currency, "reference": ref, "createdAt": created})
	}
	writeJSON(w, 200, map[string]any{"mode": "accounts", "items": items})
}

func (s *Server) calculatePayroll(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.accountsRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "Accounts / Payroll permission required")
		return
	}
	var in struct {
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
	}
	if err := decode(r, &in); err != nil {
		log.Println("payroll calculation: invalid payload:", err)
		writeError(w, 400, "validation_error", "Start and end dates are required")
		return
	}
	start, e1 := time.Parse("2006-01-02", in.StartDate)
	end, e2 := time.Parse("2006-01-02", in.EndDate)
	if e1 != nil || e2 != nil || end.Before(start) || start.Day() != 1 || end.Year() != start.Year() || end.Month() != start.Month() || end.Day() != start.AddDate(0, 1, -1).Day() {
		log.Println("payroll calculation: period must be a complete calendar month", in.StartDate, in.EndDate)
		writeError(w, 400, "validation_error", "Payroll must cover one complete calendar month")
		return
	}
	// The database algorithm is the authoritative ordered configuration. Its steps map to
	// reviewed, typed calculations below; formula text is descriptive, not executable SQL/code.
	algorithmRows, err := s.Store.DB.Query(r.Context(), `SELECT step_order,step_name,enabled FROM dcs_salary_calculation_algorithm ORDER BY step_order`)
	if err != nil {
		log.Println("payroll calculation: could not load dcs_salary_calculation_algorithm:", err)
		writeError(w, 500, "database_error", "Could not load salary calculation algorithm")
		return
	}
	steps := map[int]struct {
		name    string
		enabled bool
	}{}
	for algorithmRows.Next() {
		var order int
		var name string
		var enabled bool
		if err := algorithmRows.Scan(&order, &name, &enabled); err != nil {
			algorithmRows.Close()
			log.Println("payroll calculation: algorithm scan failed:", err)
			writeError(w, 500, "database_error", "Could not read salary calculation algorithm")
			return
		}
		steps[order] = struct {
			name    string
			enabled bool
		}{name, enabled}
	}
	if err := algorithmRows.Err(); err != nil {
		algorithmRows.Close()
		log.Println("payroll calculation: algorithm iteration failed:", err)
		writeError(w, 500, "database_error", "Could not read salary calculation algorithm")
		return
	}
	algorithmRows.Close()
	expectedSteps := []string{"Capped monthly overtime hours", "Monthly overtime pay", "Monthly base components", "Monthly salary before bonus", "Applicable monthly bonus", "Monthly salary payable"}
	for i, expected := range expectedSteps {
		step, ok := steps[i+1]
		if !ok || !step.enabled || step.name != expected {
			log.Println("payroll calculation: required algorithm step missing, disabled, or renamed:", i+1, expected, step.name, ok, step.enabled)
			writeError(w, 409, "salary_algorithm_invalid", "The configured salary algorithm is incomplete or disabled; restore the six required calculation steps")
			return
		}
	}
	var issued int
	if err := s.Store.DB.QueryRow(r.Context(), `SELECT COUNT(*) FROM dcs_payments WHERE period_start=$1 AND period_end=$2 AND entry_type='debit'`, start, end).Scan(&issued); err != nil {
		log.Println("payroll calculation: issued payment check failed:", err)
		writeError(w, 500, "database_error", "Could not validate payroll period")
		return
	}
	if issued > 0 {
		writeError(w, 409, "payroll_locked", "Payments have already been issued for this period; do not recalculate a locked payroll period")
		return
	}
	nextMonth := end.AddDate(0, 0, 1)
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		log.Println("payroll calculation: begin failed:", err)
		writeError(w, 500, "database_error", "Could not calculate payroll")
		return
	}
	defer tx.Rollback(r.Context())
	query := `WITH actual AS (
		SELECT employee_id, COALESCE(SUM(normal_minutes),0)::bigint AS normal_minutes
		FROM dcs_employees_actual_hours
		WHERE work_date >= $1 AND work_date < $2 AND payment_pending=true AND approved=true
		GROUP BY employee_id
	), event_stream AS (
		SELECT employee_id, occurred_at, event_type, shift_type,
			LEAD(event_type) OVER (PARTITION BY employee_id ORDER BY occurred_at,id) AS next_type,
			LEAD(occurred_at) OVER (PARTITION BY employee_id ORDER BY occurred_at,id) AS next_at
		FROM attendance_events
		WHERE occurred_at >= $1 AND occurred_at < $2 AND shift_type='OT'
	), actual_ot AS (
		SELECT employee_id, COALESCE(SUM(EXTRACT(EPOCH FROM (next_at-occurred_at))/60),0)::numeric AS minutes
		FROM event_stream
		WHERE event_type='check_in' AND next_type='check_out' AND next_at>occurred_at
			AND next_at-occurred_at <= interval '24 hours'
		GROUP BY employee_id
	), assigned_ot AS (
		SELECT sa.employee_id,
			COALESCE(SUM(GREATEST(0, EXTRACT(EPOCH FROM ((sa.shift_date + s.end_time + CASE WHEN s.end_time <= s.start_time THEN interval '1 day' ELSE interval '0 day' END) - (sa.shift_date + s.start_time)))/60 - s.break_minutes) + s.grace_minutes),0)::numeric AS cap_minutes
		FROM shift_assignments sa JOIN shifts s ON s.id=sa.shift_id
		WHERE sa.shift_date >= $1::date AND sa.shift_date < $2::date AND sa.status='Assigned' AND s.shift_type='OT'
		GROUP BY sa.employee_id
	)
	SELECT e.id, COALESCE(NULLIF(b.base_annual_pay_minor,0),NULLIF(b.gross_pay_minor,0),e.salary_minor,0)::bigint,
		COALESCE(b.hourly_rate_minor,0)::bigint, COALESCE(b.overtime_multiplier,1.5)::numeric,
		COALESCE(b.variable_pay_minor,0)::bigint, COALESCE(b.allowances_minor,0)::bigint, COALESCE(b.deductions_minor,0)::bigint,
		COALESCE(b.bonus_pay_minor,0)::bigint, COALESCE(b.bonus_month,12)::int, COALESCE(b.currency,e.currency,'GBP'),
		COALESCE(a.normal_minutes,0)::bigint,
		LEAST(COALESCE(ot.minutes,0),COALESCE(ass.cap_minutes,0))::numeric AS capped_ot_minutes
	FROM employees e
	LEFT JOIN dsc_employee_salary_breakup b ON b.employee_id=e.id
	LEFT JOIN actual a ON a.employee_id=e.id
	LEFT JOIN actual_ot ot ON ot.employee_id=e.id
	LEFT JOIN assigned_ot ass ON ass.employee_id=e.id
	WHERE e.status='Active'
	ORDER BY e.last_name,e.first_name`
	rows, err := tx.Query(r.Context(), query, start, nextMonth)
	if err != nil {
		log.Println("payroll calculation: employee, attendance or assigned OT query failed:", err)
		writeError(w, 500, "database_error", "Could not load salary, attendance and shift data for payroll")
		return
	}
	type calc struct {
		emp, currency                             string
		annual, rate, variable, allow, ded, bonus int64
		mult                                      float64
		bonusMonth                                int
		normalMinutes                             int64
		cappedOTMinutes                           float64
	}
	list := []calc{}
	for rows.Next() {
		var x calc
		if err := rows.Scan(&x.emp, &x.annual, &x.rate, &x.mult, &x.variable, &x.allow, &x.ded, &x.bonus, &x.bonusMonth, &x.currency, &x.normalMinutes, &x.cappedOTMinutes); err != nil {
			rows.Close()
			log.Println("payroll calculation: employee salary scan failed:", err)
			writeError(w, 500, "database_error", "Could not read salary and attendance data")
			return
		}
		if x.annual < 0 || x.variable < 0 || x.allow < 0 || x.ded < 0 || x.bonus < 0 || x.rate < 0 || x.mult < 1 || x.bonusMonth < 1 || x.bonusMonth > 12 {
			rows.Close()
			log.Println("payroll calculation: invalid salary configuration for employee:", x.emp)
			writeError(w, 409, "salary_setup_invalid", "One or more employees have invalid salary configuration")
			return
		}
		list = append(list, x)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		log.Println("payroll calculation: employee iteration failed:", err)
		writeError(w, 500, "database_error", "Could not read payroll employees")
		return
	}
	rows.Close()
	created := 0
	for _, x := range list {
		// Step 1: OT time from check-in/check-out events is capped by assigned OT shift
		// duration plus that shift's grace minutes. Convert to fractional hours.
		otHours := x.cappedOTMinutes / 60.0
		// Step 2: hourly rate is stored in minor currency units per hour.
		overtimePay := int64(math.Round(float64(x.rate) * otHours * x.mult))
		// Step 3: monthly components are the annual amounts divided by 12 as requested.
		basePay := int64(math.Round(float64(x.annual) / 12.0))
		variablePay := int64(math.Round(float64(x.variable) / 12.0))
		allowances := int64(math.Round(float64(x.allow) / 12.0))
		deductions := int64(math.Round(float64(x.ded) / 12.0))
		// Step 4: monthly salary before bonus follows (base + variable + allowances - deductions)/12
		// by dividing each annual component once above, then adding monthly overtime.
		monthlyBeforeBonus := basePay + variablePay + allowances - deductions + overtimePay
		// Step 5: bonus is paid only in the configured calendar month.
		bonusPay := int64(0)
		if int(start.Month()) == x.bonusMonth {
			bonusPay = x.bonus
		}
		// Step 6: this is the payable monthly amount; avoid negative payroll amounts.
		payable := monthlyBeforeBonus + bonusPay
		if payable < 0 {
			payable = 0
		}
		// Keep gross/net semantics consistent with the existing payment workflow: net_minor
		// is the amount issued. Deductions are already applied in the formula above.
		gross := basePay + variablePay + allowances + overtimePay + bonusPay
		_, err = tx.Exec(r.Context(), `INSERT INTO dcs_salary_summary(id,employee_id,period_start,period_end,currency,normal_hours,overtime_hours,base_pay_minor,overtime_pay_minor,variable_pay_minor,allowances_minor,deductions_minor,gross_minor,net_minor,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'Payment pending') ON CONFLICT(employee_id,period_start,period_end) DO UPDATE SET normal_hours=EXCLUDED.normal_hours,overtime_hours=EXCLUDED.overtime_hours,base_pay_minor=EXCLUDED.base_pay_minor,overtime_pay_minor=EXCLUDED.overtime_pay_minor,variable_pay_minor=EXCLUDED.variable_pay_minor,allowances_minor=EXCLUDED.allowances_minor,deductions_minor=EXCLUDED.deductions_minor,gross_minor=EXCLUDED.gross_minor,net_minor=EXCLUDED.net_minor,status='Payment pending',generated_at=now()`, auth.NewID(), x.emp, start, end, strings.TrimSpace(x.currency), float64(x.normalMinutes)/60.0, otHours, basePay, overtimePay, variablePay, allowances, deductions, gross, payable)
		if err != nil {
			log.Println("payroll calculation: summary upsert failed for employee", x.emp, ":", err)
			writeError(w, 500, "database_error", "Could not save salary summary")
			return
		}
		created++
	}
	if err = tx.Commit(r.Context()); err != nil {
		log.Println("payroll calculation: commit failed:", err)
		writeError(w, 500, "database_error", "Could not save payroll calculation")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "payroll.calculated", "payroll", in.StartDate+"/"+in.EndDate, fmt.Sprintf("Calculated monthly payroll for %d employees using configured salary algorithm", created), requestID(r))
	writeJSON(w, 200, map[string]any{"calculatedEmployees": created, "message": "Monthly salary, capped overtime and applicable bonus calculated. Review summaries before issuing payments."})
}

func (s *Server) issuePayroll(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "payroll.issue") {
		writeError(w, 403, "forbidden", "Accounts / Payroll permission required to issue payments")
		return
	}
	var in struct {
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
		Reference string `json:"reference"`
	}
	if err := decode(r, &in); err != nil {
		log.Println("payroll issue: invalid payload:", err)
		writeError(w, 400, "validation_error", "Payroll dates are required")
		return
	}
	start, e1 := time.Parse("2006-01-02", in.StartDate)
	end, e2 := time.Parse("2006-01-02", in.EndDate)
	if e1 != nil || e2 != nil || end.Before(start) {
		writeError(w, 400, "validation_error", "Enter valid payroll dates")
		return
	}
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		log.Println("payroll issue: begin failed:", err)
		writeError(w, 500, "database_error", "Could not issue payroll")
		return
	}
	defer tx.Rollback(r.Context())
	rows, err := tx.Query(r.Context(), `SELECT employee_id,net_minor,currency FROM dcs_salary_summary WHERE period_start=$1 AND period_end=$2 AND status='Payment pending'`, start, end)
	if err != nil {
		log.Println("payroll issue: summary query failed:", err)
		writeError(w, 500, "database_error", "Could not load salary summaries")
		return
	}
	type item struct {
		emp, currency string
		amount        int64
	}
	items := []item{}
	for rows.Next() {
		var x item
		if err := rows.Scan(&x.emp, &x.amount, &x.currency); err != nil {
			rows.Close()
			log.Println("payroll issue: summary scan failed:", err)
			writeError(w, 500, "database_error", "Could not read salary summaries")
			return
		}
		items = append(items, x)
	}
	rows.Close()
	for _, x := range items {
		_, err = tx.Exec(r.Context(), `INSERT INTO dcs_payments(id,employee_id,period_start,period_end,entry_type,status,amount_minor,currency,reference,created_by) VALUES($1,$2,$3,$4,'debit','Pending',$5,$6,$7,$8)`, auth.NewID(), x.emp, start, end, x.amount, x.currency, strings.TrimSpace(in.Reference), u.ID)
		if err != nil {
			log.Println("payroll issue: debit insert failed:", err)
			writeError(w, 500, "database_error", "Could not create payment debit records")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		log.Println("payroll issue: commit failed:", err)
		writeError(w, 500, "database_error", "Could not issue payment records")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "payroll.issued", "payroll", in.StartDate+"/"+in.EndDate, fmt.Sprintf("Issued %d debit payment records", len(items)), requestID(r))
	writeJSON(w, 201, map[string]any{"issued": len(items), "message": "Payment debit records created. Mark cleared only after the payment provider confirms settlement."})
}
func (s *Server) clearPayment(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "payroll.approve") {
		writeError(w, 403, "forbidden", "Accounts Approver permission required to clear payments")
		return
	}
	id := r.PathValue("id")
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		log.Println("payment clear: begin failed:", err)
		writeError(w, 500, "database_error", "Could not clear payment")
		return
	}
	defer tx.Rollback(r.Context())
	var emp string
	var start, end time.Time
	var amount int64
	var currency, reference string
	err = tx.QueryRow(r.Context(), `UPDATE dcs_payments SET status='Cleared',cleared_at=now(),cleared_by=$2 WHERE id=$1 AND entry_type='debit' AND status='Pending' RETURNING employee_id,period_start,period_end,amount_minor,currency,reference`, id, u.ID).Scan(&emp, &start, &end, &amount, &currency, &reference)
	if err != nil {
		log.Println("payment clear: pending debit not found or update failed:", err)
		writeError(w, 409, "payment_not_pending", "Payment is not pending or could not be cleared")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO dcs_payments(id,employee_id,period_start,period_end,entry_type,status,amount_minor,currency,reference,created_by,cleared_at,cleared_by) VALUES($1,$2,$3,$4,'credit','Cleared',$5,$6,$7,$8,now(),$8)`, auth.NewID(), emp, start, end, amount, currency, reference, u.ID)
	if err != nil {
		log.Println("payment clear: credit insert failed:", err)
		writeError(w, 500, "database_error", "Debit was updated but credit record failed; transaction will be rolled back")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE dcs_salary_summary SET status='Paid' WHERE employee_id=$1 AND period_start=$2 AND period_end=$3`, emp, start, end)
	if err != nil {
		log.Println("payment clear: summary update failed:", err)
		writeError(w, 500, "database_error", "Could not update salary summary")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE dcs_employees_actual_hours SET payment_pending=false,payroll_period=$3 WHERE employee_id=$1 AND work_date BETWEEN $2 AND $4`, emp, start, start.Format("2006-01"), end)
	if err != nil {
		log.Println("payment clear: hours update failed:", err)
		writeError(w, 500, "database_error", "Could not mark actual hours as paid")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		log.Println("payment clear: commit failed:", err)
		writeError(w, 500, "database_error", "Could not clear payment")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "payroll.payment_cleared", "payment", id, "Cleared payment and recorded matching credit", requestID(r))
	writeJSON(w, 200, map[string]any{"message": "Payment cleared and credit record created"})
}

func (s *Server) payslipPDF(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	id := r.PathValue("id")
	var emp, name, currency, status string
	var start, end time.Time
	var normal, ot float64
	var base, overtime, variable, allow, ded, gross, net int64
	err := s.Store.DB.QueryRow(r.Context(), `SELECT x.employee_id,e.first_name||' '||e.last_name,x.period_start,x.period_end,x.currency,x.normal_hours,x.overtime_hours,x.base_pay_minor,x.overtime_pay_minor,x.variable_pay_minor,x.allowances_minor,x.deductions_minor,x.gross_minor,x.net_minor,x.status FROM dcs_salary_summary x JOIN employees e ON e.id=x.employee_id WHERE x.id=$1`, id).Scan(&emp, &name, &start, &end, &currency, &normal, &ot, &base, &overtime, &variable, &allow, &ded, &gross, &net, &status)
	if err != nil {
		log.Println("payslip: summary lookup failed:", err)
		http.NotFound(w, r)
		return
	}
	if !s.accountsRole(r.Context(), u.Role) && u.EmployeeID != emp {
		writeError(w, 403, "forbidden", "You can only view your own payslip")
		return
	}
	body := fmt.Sprintf("PEOPLEOPS - SALARY SLIP\nEmployee: %s\nPeriod: %s to %s\nStatus: %s\nCurrency: %s\nNormal hours: %.2f\nOvertime hours: %.2f\nBase pay: %s %.2f\nOvertime pay: %s %.2f\nVariable pay: %s %.2f\nAllowances: %s %.2f\nDeductions: %s %.2f\nGross pay: %s %.2f\nNet pay: %s %.2f\n", name, start.Format("2006-01-02"), end.Format("2006-01-02"), status, currency, normal, ot, currency, float64(base)/100, currency, float64(overtime)/100, currency, float64(variable)/100, currency, float64(allow)/100, currency, float64(ded)/100, currency, float64(gross)/100, currency, float64(net)/100)
	pdf := simplePDF(body)
	w.Header().Set("Content-Type", "application/pdf")
	disposition := "attachment"
	if r.URL.Query().Get("view") == "1" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=salary-slip-%s.pdf", disposition, start.Format("2006-01")))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(200)
	if _, err := w.Write(pdf); err != nil {
		log.Println("payslip: response write failed:", err)
	}
}
func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	return strings.ReplaceAll(s, ")", "\\)")
}
func simplePDF(content string) []byte {
	lines := strings.Split(content, "\n")
	var stream strings.Builder
	stream.WriteString("BT /F1 12 Tf 50 790 Td 16 TL ")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("T*")
		}
		stream.WriteString(" (" + pdfEscape(line) + ") Tj ")
	}
	stream.WriteString("ET")
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>", fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream.String()), stream.String())}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xref)
	return out.Bytes()
}

func (s *Server) salaryBreakup(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hrAccountsRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "HR or Accounts permission required")
		return
	}
	empID := r.PathValue("id")
	if r.Method == http.MethodGet {
		var gross, rate, variable, allow, ded int64
		var mult float64
		var currency string
		var effective time.Time
		err := s.Store.DB.QueryRow(r.Context(), `SELECT gross_pay_minor,hourly_rate_minor,overtime_multiplier,variable_pay_minor,allowances_minor,deductions_minor,currency,effective_from FROM dsc_employee_salary_breakup WHERE employee_id=$1`, empID).Scan(&gross, &rate, &mult, &variable, &allow, &ded, &currency, &effective)
		if err != nil {
			log.Println("salary breakup: lookup failed:", err)
			writeError(w, 404, "not_found", "Salary breakup not found")
			return
		}
		writeJSON(w, 200, map[string]any{"grossPayMinor": gross, "hourlyRateMinor": rate, "overtimeMultiplier": mult, "variablePayMinor": variable, "allowancesMinor": allow, "deductionsMinor": ded, "currency": strings.TrimSpace(currency), "effectiveFrom": effective.Format("2006-01-02")})
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, 405, "method_not_allowed", "Method not allowed")
		return
	}
	var in struct {
		Gross      int64   `json:"grossPayMinor"`
		Rate       int64   `json:"hourlyRateMinor"`
		Multiplier float64 `json:"overtimeMultiplier"`
		Variable   int64   `json:"variablePayMinor"`
		Allow      int64   `json:"allowancesMinor"`
		Ded        int64   `json:"deductionsMinor"`
		Currency   string  `json:"currency"`
		Effective  string  `json:"effectiveFrom"`
	}
	if err := decode(r, &in); err != nil || in.Gross < 0 || in.Rate < 0 || in.Multiplier < 1 || in.Multiplier > 5 || in.Variable < 0 || in.Allow < 0 || in.Ded < 0 {
		log.Println("salary breakup: invalid payload:", err)
		writeError(w, 400, "validation_error", "Enter valid non-negative pay values and overtime multiplier between 1 and 5")
		return
	}
	if in.Currency == "" {
		in.Currency = "GBP"
	}
	if len(in.Currency) != 3 {
		writeError(w, 400, "validation_error", "Currency must be a 3-letter code")
		return
	}
	effective, err := time.Parse("2006-01-02", in.Effective)
	if err != nil {
		effective = time.Now().UTC()
	}
	_, err = s.Store.DB.Exec(r.Context(), `INSERT INTO dsc_employee_salary_breakup(id,employee_id,gross_pay_minor,hourly_rate_minor,overtime_multiplier,variable_pay_minor,allowances_minor,deductions_minor,currency,effective_from,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,now()) ON CONFLICT(employee_id) DO UPDATE SET gross_pay_minor=EXCLUDED.gross_pay_minor,hourly_rate_minor=EXCLUDED.hourly_rate_minor,overtime_multiplier=EXCLUDED.overtime_multiplier,variable_pay_minor=EXCLUDED.variable_pay_minor,allowances_minor=EXCLUDED.allowances_minor,deductions_minor=EXCLUDED.deductions_minor,currency=EXCLUDED.currency,effective_from=EXCLUDED.effective_from,updated_at=now()`, auth.NewID(), empID, in.Gross, in.Rate, in.Multiplier, in.Variable, in.Allow, in.Ded, strings.ToUpper(in.Currency), effective)
	if err != nil {
		log.Println("salary breakup: save failed:", err)
		writeError(w, 500, "database_error", "Could not save salary breakup")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "employee.salary_updated", "employee", empID, "Updated salary breakup", requestID(r))
	if contractErr := s.createEmploymentContract(empID); contractErr != nil {
		log.Println("employment contract: draft/signature email workflow could not complete:", contractErr)
		// Salary was saved successfully. Do not misreport a successful salary save as failed;
		// the contract workflow issue is logged for administrator follow-up.
	}
	writeJSON(w, 200, map[string]any{"message": "Salary breakup saved"})
}

func (s *Server) salaryAlgorithm(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.adminRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "Master Admin permission required")
		return
	}
	if r.Method == http.MethodGet {
		rows, err := s.Store.DB.Query(r.Context(), `SELECT step_order,step_name,formula,enabled,config FROM dcs_salary_calculation_algorithm ORDER BY step_order`)
		if err != nil {
			log.Println("salary algorithm: list failed:", err)
			writeError(w, 500, "database_error", "Could not load salary algorithm")
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var order int
			var name, formula string
			var enabled bool
			var config []byte
			if err := rows.Scan(&order, &name, &formula, &enabled, &config); err != nil {
				log.Println("salary algorithm: scan failed:", err)
				writeError(w, 500, "database_error", "Could not read salary algorithm")
				return
			}
			var cfg any = map[string]any{}
			if len(config) > 0 {
				_ = json.Unmarshal(config, &cfg)
			}
			items = append(items, map[string]any{"stepOrder": order, "stepName": name, "formula": formula, "enabled": enabled, "config": cfg})
		}
		writeJSON(w, 200, map[string]any{"items": items})
		return
	}
	writeError(w, 405, "method_not_allowed", "Algorithm editing is read-only in this release; edit configuration through a reviewed migration")
}

func intQuery(r *http.Request, key string, def int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return v
}

func (s *Server) salarySummaries(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.accountsRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "Accounts / Payroll permission required")
		return
	}
	rows, err := s.Store.DB.Query(r.Context(), `SELECT x.id,x.employee_id,e.first_name||' '||e.last_name,x.period_start,x.period_end,x.currency,x.normal_hours,x.overtime_hours,x.base_pay_minor,x.overtime_pay_minor,x.variable_pay_minor,x.allowances_minor,x.deductions_minor,x.gross_minor,x.net_minor,x.status FROM dcs_salary_summary x JOIN employees e ON e.id=x.employee_id ORDER BY x.period_start DESC,e.last_name LIMIT 1000`)
	if err != nil {
		log.Println("salary summaries: list query failed:", err)
		writeError(w, 500, "database_error", "Could not load salary summaries")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, emp, name, currency, status string
		var start, end time.Time
		var normal, ot float64
		var base, overtime, variable, allow, ded, gross, net int64
		if err := rows.Scan(&id, &emp, &name, &start, &end, &currency, &normal, &ot, &base, &overtime, &variable, &allow, &ded, &gross, &net, &status); err != nil {
			log.Println("salary summaries: scan failed:", err)
			writeError(w, 500, "database_error", "Could not read salary summaries")
			return
		}
		items = append(items, map[string]any{"id": id, "employeeId": emp, "employeeName": name, "periodStart": start.Format("2006-01-02"), "periodEnd": end.Format("2006-01-02"), "currency": strings.TrimSpace(currency), "normalHours": normal, "overtimeHours": ot, "basePayMinor": base, "overtimePayMinor": overtime, "variablePayMinor": variable, "allowancesMinor": allow, "deductionsMinor": ded, "grossMinor": gross, "netMinor": net, "status": status})
	}
	if err := rows.Err(); err != nil {
		log.Println("salary summaries: iteration failed:", err)
		writeError(w, 500, "database_error", "Could not read salary summaries")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
