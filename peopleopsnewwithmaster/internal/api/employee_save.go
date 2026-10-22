package api

import (
	"context"
	"fmt"
	"net/http"
	"peopleops/internal/auth"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Accounts users retain their existing financial-edit permission without gaining
// permission to change employment records or login roles.
func (s *Server) updateEmployeeFinancials(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hrAccountsRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "HR or Accounts permission required")
		return
	}
	var in struct {
		Salary *salaryInput `json:"salary"`
		Bank   *bankInput   `json:"bank"`
	}
	if decode(r, &in) != nil || (in.Salary == nil && in.Bank == nil) {
		writeError(w, 400, "invalid_request", "Change salary or bank details before saving")
		return
	}
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "database_error", "Could not start employee save")
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	if err = tx.QueryRow(r.Context(), `SELECT id FROM employees WHERE id=$1 FOR UPDATE`, r.PathValue("id")).Scan(&id); err != nil {
		writeError(w, 404, "not_found", "Employee not found")
		return
	}
	if err = saveEmployeeFinancials(r.Context(), tx, id, u.ID, in.Salary, in.Bank); err != nil {
		writeError(w, 400, "employee_financial_validation", err.Error())
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "database_error", "Could not save employee financial details")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "employee.financials_updated", "employee", id, "Updated salary/bank details", requestID(r))
	writeJSON(w, 200, map[string]any{"id": id, "message": "Employee financial details saved"})
}

type salaryInput struct {
	Gross      int64   `json:"grossPayMinor"`
	Rate       int64   `json:"hourlyRateMinor"`
	Multiplier float64 `json:"overtimeMultiplier"`
	Variable   int64   `json:"variablePayMinor"`
	Allow      int64   `json:"allowancesMinor"`
	Ded        int64   `json:"deductionsMinor"`
	Currency   string  `json:"currency"`
	Effective  string  `json:"effectiveFrom"`
}
type bankInput struct {
	AccountName   string `json:"accountName"`
	BankName      string `json:"bankName"`
	SortCode      string `json:"sortCode"`
	AccountNumber string `json:"accountNumber"`
	IBAN          string `json:"iban"`
	BIC           string `json:"bicSwift"`
}

func saveEmployeeFinancials(ctx context.Context, tx pgx.Tx, empID, actor string, salary *salaryInput, bank *bankInput) error {
	if salary != nil {
		in := salary
		effective, err := time.Parse("2006-01-02", in.Effective)
		if err != nil || len(in.Currency) != 3 || in.Gross < 0 || in.Rate < 0 || in.Multiplier < 1 || in.Multiplier > 5 || in.Variable < 0 || in.Allow < 0 || in.Ded < 0 {
			return fmt.Errorf("Salary: enter valid amounts, currency and effective date")
		}
		_, err = tx.Exec(ctx, `INSERT INTO dsc_employee_salary_breakup(id,employee_id,gross_pay_minor,hourly_rate_minor,overtime_multiplier,variable_pay_minor,allowances_minor,deductions_minor,currency,effective_from,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,now()) ON CONFLICT(employee_id) DO UPDATE SET gross_pay_minor=EXCLUDED.gross_pay_minor,hourly_rate_minor=EXCLUDED.hourly_rate_minor,overtime_multiplier=EXCLUDED.overtime_multiplier,variable_pay_minor=EXCLUDED.variable_pay_minor,allowances_minor=EXCLUDED.allowances_minor,deductions_minor=EXCLUDED.deductions_minor,currency=EXCLUDED.currency,effective_from=EXCLUDED.effective_from,updated_at=now()`, auth.NewID(), empID, in.Gross, in.Rate, in.Multiplier, in.Variable, in.Allow, in.Ded, strings.ToUpper(in.Currency), effective)
		if err != nil {
			return fmt.Errorf("Salary could not be saved; no employee changes were saved")
		}
	}
	if bank != nil {
		in := bank
		if strings.TrimSpace(in.AccountName) == "" || strings.TrimSpace(in.SortCode) == "" || strings.TrimSpace(in.AccountNumber) == "" {
			return fmt.Errorf("Bank details: account name, sort code and account number are required")
		}
		var oldAcct, oldIBAN string
		if strings.Contains(in.AccountNumber, "*") || strings.Contains(in.IBAN, "*") {
			if err := tx.QueryRow(ctx, `SELECT account_number_encrypted,iban_encrypted FROM dcs_employee_bank_details WHERE employee_id=$1`, empID).Scan(&oldAcct, &oldIBAN); err != nil {
				return fmt.Errorf("Existing bank details could not be preserved")
			}
		}
		acct, err := encryptBank(in.AccountNumber)
		if err != nil {
			return fmt.Errorf("Bank encryption is unavailable; no employee changes were saved")
		}
		iban := ""
		if in.IBAN != "" {
			iban, err = encryptBank(in.IBAN)
			if err != nil {
				return fmt.Errorf("Could not encrypt IBAN")
			}
		}
		if strings.Contains(in.AccountNumber, "*") {
			acct = oldAcct
		}
		if strings.Contains(in.IBAN, "*") {
			iban = oldIBAN
		}
		_, err = tx.Exec(ctx, `INSERT INTO dcs_employee_bank_details(id,employee_id,account_name,bank_name,sort_code,account_number_encrypted,iban_encrypted,bic_swift,updated_by,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,now()) ON CONFLICT(employee_id) DO UPDATE SET account_name=EXCLUDED.account_name,bank_name=EXCLUDED.bank_name,sort_code=EXCLUDED.sort_code,account_number_encrypted=EXCLUDED.account_number_encrypted,iban_encrypted=EXCLUDED.iban_encrypted,bic_swift=EXCLUDED.bic_swift,updated_by=EXCLUDED.updated_by,updated_at=now()`, auth.NewID(), empID, strings.TrimSpace(in.AccountName), strings.TrimSpace(in.BankName), strings.TrimSpace(in.SortCode), acct, iban, strings.TrimSpace(in.BIC), actor)
		if err != nil {
			return fmt.Errorf("Bank details could not be saved; no employee changes were saved")
		}
	}
	return nil
}
