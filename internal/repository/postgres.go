package repository

import (
	apperrors "bank-api/internal/errors"
	"bank-api/internal/model"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type userRepo struct{ db *sql.DB }
type accountRepo struct{ db *sql.DB }
type cardRepo struct{ db *sql.DB }
type transactionRepo struct{ db *sql.DB }
type creditRepo struct{ db *sql.DB }
type paymentScheduleRepo struct{ db *sql.DB }

func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		User:            &userRepo{db},
		Account:         &accountRepo{db},
		Card:            &cardRepo{db},
		Transaction:     &transactionRepo{db},
		Credit:          &creditRepo{db},
		PaymentSchedule: &paymentScheduleRepo{db},
	}
}

// Вспомогательная функция для проверки нарушения ограничения по уникальности значения в одном из полей
func isUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}

// UserRepo
func (r *userRepo) Create(ctx context.Context, u *model.User) error {
	query := `INSERT INTO users (username, email, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query, u.Username, u.Email, u.PasswordHash, u.Role).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.ErrConflict
		}
		return apperrors.NewAppError(500, "Ошибка создания пользователя", err)
	}
	return nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u := &model.User{}
	query := `SELECT id, username, email, password_hash, two_factor_key, two_factor_enabled, is_blocked, role, created_at, updated_at FROM users WHERE email=$1`
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.TwoFactorKey, &u.TwoFactorEnabled, &u.IsBlocked, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return u, err
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	u := &model.User{}
	query := `SELECT id, username, email, password_hash, two_factor_key, two_factor_enabled, is_blocked, role, created_at, updated_at FROM users WHERE username=$1`
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.TwoFactorKey, &u.TwoFactorEnabled, &u.IsBlocked, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return u, err
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	u := &model.User{}
	query := `SELECT id, username, email, password_hash, two_factor_key, two_factor_enabled, is_blocked, role, created_at, updated_at FROM users WHERE id=$1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.TwoFactorKey, &u.TwoFactorEnabled, &u.IsBlocked, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return u, err
}

func (r *userRepo) Update(ctx context.Context, u *model.User) error {
	query := `UPDATE users SET username=$1, email=$2, password_hash=$3, two_factor_key=$4, two_factor_enabled=$5, is_blocked=$6, role=$7, updated_at=NOW() WHERE id=$8`
	_, err := r.db.ExecContext(ctx, query, u.Username, u.Email, u.PasswordHash, u.TwoFactorKey, u.TwoFactorEnabled, u.IsBlocked, u.Role, u.ID)
	return err
}

func (r *userRepo) List(ctx context.Context) ([]*model.User, error) {
	query := `SELECT id, username, email, is_blocked, role, created_at FROM users`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsBlocked, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

// AccountRepo
func (r *accountRepo) Create(ctx context.Context, acc *model.Account) error {
	query := `INSERT INTO accounts (user_id, balance, currency) VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, acc.UserID, acc.Balance, acc.Currency).Scan(&acc.ID, &acc.CreatedAt)
}

func (r *accountRepo) GetByID(ctx context.Context, id int64) (*model.Account, error) {
	acc := &model.Account{}
	query := `SELECT id, user_id, balance, currency, created_at FROM accounts WHERE id=$1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.Currency, &acc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return acc, err
}

func (r *accountRepo) ListByUser(ctx context.Context, userID int64) ([]*model.Account, error) {
	query := `SELECT id, user_id, balance, currency, created_at FROM accounts WHERE user_id=$1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []*model.Account
	for rows.Next() {
		a := &model.Account{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Balance, &a.Currency, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *accountRepo) UpdateBalance(ctx context.Context, id int64, delta float64) error {
	query := `UPDATE accounts SET balance = balance + $1 WHERE id=$2 AND balance + $1 >= 0`
	res, err := r.db.ExecContext(ctx, query, delta, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return apperrors.ErrInsufficientFunds
	}
	return nil
}

func (r *accountRepo) GetForUpdate(ctx context.Context, tx interface{}, id int64) (*model.Account, error) {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return nil, fmt.Errorf("неверный тип транзакции")
	}
	acc := &model.Account{}
	query := `SELECT id, user_id, balance, currency, created_at FROM accounts WHERE id=$1 FOR UPDATE`
	err := txx.QueryRowContext(ctx, query, id).Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.Currency, &acc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return acc, err
}

func (r *accountRepo) UpdateBalanceTx(ctx context.Context, tx interface{}, id int64, delta float64) error {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("неверный тип транзакции")
	}
	query := `UPDATE accounts SET balance = balance + $1 WHERE id=$2 AND balance + $1 >= 0`
	res, err := txx.ExecContext(ctx, query, delta, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return apperrors.ErrInsufficientFunds
	}
	return nil
}

// CardRepo
func (r *cardRepo) Create(ctx context.Context, c *model.Card) error {
	query := `INSERT INTO cards (account_id, user_id, encrypted_number, hmac_number, encrypted_expiry, encrypted_cvv, bcrypt_cvv, status) 
              VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, c.AccountID, c.UserID, c.EncryptedNumber, c.HmacNumber, c.EncryptedExpiry, c.EncryptedCVV, c.BcryptCVV, c.Status).
		Scan(&c.ID, &c.CreatedAt)
}

func (r *cardRepo) GetByID(ctx context.Context, id int64) (*model.Card, error) {
	c := &model.Card{}
	query := `SELECT id, account_id, user_id, encrypted_number, hmac_number, encrypted_expiry, encrypted_cvv, bcrypt_cvv, status, created_at FROM cards WHERE id=$1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&c.ID, &c.AccountID, &c.UserID, &c.EncryptedNumber, &c.HmacNumber, &c.EncryptedExpiry, &c.EncryptedCVV, &c.BcryptCVV, &c.Status, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return c, err
}

func (r *cardRepo) ListByUser(ctx context.Context, userID int64) ([]*model.Card, error) {
	query := `SELECT id, account_id, user_id, encrypted_number, hmac_number, encrypted_expiry, encrypted_cvv, bcrypt_cvv, status, created_at FROM cards WHERE user_id=$1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []*model.Card
	for rows.Next() {
		c := &model.Card{}
		if err := rows.Scan(&c.ID, &c.AccountID, &c.UserID, &c.EncryptedNumber, &c.HmacNumber, &c.EncryptedExpiry, &c.EncryptedCVV, &c.BcryptCVV, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// TransactionRepo
func (r *transactionRepo) Create(ctx context.Context, txn *model.Transaction) error {
	return insertTransaction(ctx, r.db, txn)
}

func (r *transactionRepo) CreateTx(ctx context.Context, tx interface{}, txn *model.Transaction) error {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("неверный тип транзакции")
	}
	return insertTransaction(ctx, txx, txn)
}

func insertTransaction(ctx context.Context, querier interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}, txn *model.Transaction) error {
	query := `INSERT INTO transactions (from_account_id, to_account_id, amount, type, description) 
              VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`
	return querier.QueryRowContext(ctx, query, txn.FromAccountID, txn.ToAccountID, txn.Amount, txn.Type, txn.Description).
		Scan(&txn.ID, &txn.CreatedAt)
}

func (r *transactionRepo) ListByAccount(ctx context.Context, accountID int64, from, to time.Time) ([]*model.Transaction, error) {
	query := `SELECT id, from_account_id, to_account_id, amount, type, description, created_at 
              FROM transactions 
              WHERE (from_account_id=$1 OR to_account_id=$1) AND created_at BETWEEN $2 AND $3
              ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, accountID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *transactionRepo) ListByUser(ctx context.Context, userID int64, from, to time.Time) ([]*model.Transaction, error) {
	query := `SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.type, t.description, t.created_at
              FROM transactions t
              LEFT JOIN accounts a1 ON t.from_account_id = a1.id
              LEFT JOIN accounts a2 ON t.to_account_id = a2.id
              WHERE (a1.user_id = $1 OR a2.user_id = $1) AND t.created_at BETWEEN $2 AND $3
              ORDER BY t.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *transactionRepo) GetIncomeExpenseByUser(ctx context.Context, userID int64, from, to time.Time) (float64, float64, error) {
	query := `SELECT
		COALESCE(SUM(
			CASE
				WHEN a_to.user_id = $1 AND (a_from.user_id IS NULL OR a_from.user_id <> $1) THEN t.amount
				ELSE 0
			END
		), 0) AS income,
		COALESCE(SUM(
			CASE
				WHEN a_from.user_id = $1 AND (a_to.user_id IS NULL OR a_to.user_id <> $1) THEN t.amount
				ELSE 0
			END
		), 0) AS expense
	FROM transactions t
	LEFT JOIN accounts a_from ON t.from_account_id = a_from.id
	LEFT JOIN accounts a_to ON t.to_account_id = a_to.id
	WHERE (a_from.user_id = $1 OR a_to.user_id = $1)
	  AND t.created_at BETWEEN $2 AND $3`

	var income, expense float64
	if err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&income, &expense); err != nil {
		return 0, 0, err
	}

	return income, expense, nil
}

func scanTransactions(rows *sql.Rows) ([]*model.Transaction, error) {
	var txns []*model.Transaction
	for rows.Next() {
		t := &model.Transaction{}
		if err := rows.Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, nil
}

// CreditRepo
func (r *creditRepo) Create(ctx context.Context, cr *model.Credit) error {
	return insertCredit(ctx, r.db, cr)
}

func (r *creditRepo) CreateTx(ctx context.Context, tx interface{}, cr *model.Credit) error {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("неверный тип транзакции")
	}
	return insertCredit(ctx, txx, cr)
}

func insertCredit(ctx context.Context, querier interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}, cr *model.Credit) error {
	query := `INSERT INTO credits (account_id, user_id, amount, interest_rate, term_months, monthly_payment, remaining_amount, status) 
              VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`
	return querier.QueryRowContext(ctx, query, cr.AccountID, cr.UserID, cr.Amount, cr.InterestRate, cr.TermMonths, cr.MonthlyPayment, cr.RemainingAmount, cr.Status).
		Scan(&cr.ID, &cr.CreatedAt)
}

func (r *creditRepo) GetByID(ctx context.Context, id int64) (*model.Credit, error) {
	cr := &model.Credit{}
	query := `SELECT id, account_id, user_id, amount, interest_rate, term_months, monthly_payment, remaining_amount, status, created_at FROM credits WHERE id=$1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&cr.ID, &cr.AccountID, &cr.UserID, &cr.Amount, &cr.InterestRate, &cr.TermMonths, &cr.MonthlyPayment, &cr.RemainingAmount, &cr.Status, &cr.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return cr, err
}

func (r *creditRepo) ListByUser(ctx context.Context, userID int64) ([]*model.Credit, error) {
	query := `SELECT id, account_id, user_id, amount, interest_rate, term_months, monthly_payment, remaining_amount, status, created_at FROM credits WHERE user_id=$1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var credits []*model.Credit
	for rows.Next() {
		cr := &model.Credit{}
		if err := rows.Scan(&cr.ID, &cr.AccountID, &cr.UserID, &cr.Amount, &cr.InterestRate, &cr.TermMonths, &cr.MonthlyPayment, &cr.RemainingAmount, &cr.Status, &cr.CreatedAt); err != nil {
			return nil, err
		}
		credits = append(credits, cr)
	}
	return credits, nil
}

func (r *creditRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE credits SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *creditRepo) UpdateStatusTx(ctx context.Context, tx interface{}, id int64, status string) error {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("неверный тип транзакции")
	}
	_, err := txx.ExecContext(ctx, `UPDATE credits SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *creditRepo) ReduceRemainingAmountTx(ctx context.Context, tx interface{}, id int64, amount float64) (float64, error) {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return 0, fmt.Errorf("неверный тип транзакции")
	}

	var remaining float64
	err := txx.QueryRowContext(ctx, `
		UPDATE credits
		SET remaining_amount = GREATEST(remaining_amount - $1, 0)
		WHERE id = $2
		RETURNING remaining_amount
	`, amount, id).Scan(&remaining)
	if err != nil {
		return 0, err
	}

	return remaining, nil
}

func (r *creditRepo) GetActiveByUser(ctx context.Context, userID int64) ([]*model.Credit, error) {
	query := `SELECT id, account_id, user_id, amount, interest_rate, term_months, monthly_payment, remaining_amount, status, created_at FROM credits WHERE user_id=$1 AND status='active'`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var credits []*model.Credit
	for rows.Next() {
		cr := &model.Credit{}
		if err := rows.Scan(&cr.ID, &cr.AccountID, &cr.UserID, &cr.Amount, &cr.InterestRate, &cr.TermMonths, &cr.MonthlyPayment, &cr.RemainingAmount, &cr.Status, &cr.CreatedAt); err != nil {
			return nil, err
		}
		credits = append(credits, cr)
	}
	return credits, nil
}

// PaymentScheduleRepo
func (r *paymentScheduleRepo) CreateBatch(ctx context.Context, schedules []*model.PaymentSchedule) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := r.CreateBatchTx(ctx, tx, schedules); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *paymentScheduleRepo) CreateBatchTx(ctx context.Context, tx interface{}, schedules []*model.PaymentSchedule) error {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("неверный тип транзакции")
	}

	stmt, err := txx.PrepareContext(ctx, `INSERT INTO payment_schedules (credit_id, due_date, amount, status) VALUES ($1,$2,$3,$4) RETURNING id, created_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, s := range schedules {
		err := stmt.QueryRowContext(ctx, s.CreditID, s.DueDate, s.Amount, s.Status).Scan(&s.ID, &s.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *paymentScheduleRepo) GetByCreditID(ctx context.Context, creditID int64) ([]*model.PaymentSchedule, error) {
	query := `SELECT id, credit_id, due_date, amount, paid_amount, status, created_at FROM payment_schedules WHERE credit_id=$1 ORDER BY due_date`
	rows, err := r.db.QueryContext(ctx, query, creditID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []*model.PaymentSchedule
	for rows.Next() {
		s := &model.PaymentSchedule{}
		if err := rows.Scan(&s.ID, &s.CreditID, &s.DueDate, &s.Amount, &s.PaidAmount, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (r *paymentScheduleRepo) GetNextUnpaidByCredit(ctx context.Context, creditID int64) (*model.PaymentSchedule, error) {
	query := `SELECT id, credit_id, due_date, amount, paid_amount, status, created_at
		FROM payment_schedules
		WHERE credit_id = $1 AND status IN ('pending', 'overdue')
		ORDER BY due_date
		LIMIT 1`

	schedule := &model.PaymentSchedule{}
	err := r.db.QueryRowContext(ctx, query, creditID).Scan(
		&schedule.ID,
		&schedule.CreditID,
		&schedule.DueDate,
		&schedule.Amount,
		&schedule.PaidAmount,
		&schedule.Status,
		&schedule.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}

	return schedule, err
}

func (r *paymentScheduleRepo) GetOverdue(ctx context.Context) ([]*model.PaymentSchedule, error) {
	query := `SELECT id, credit_id, due_date, amount, paid_amount, status, created_at FROM payment_schedules WHERE due_date < NOW() AND status IN ('pending','overdue')`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []*model.PaymentSchedule
	for rows.Next() {
		s := &model.PaymentSchedule{}
		if err := rows.Scan(&s.ID, &s.CreditID, &s.DueDate, &s.Amount, &s.PaidAmount, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (r *paymentScheduleRepo) MarkPaid(ctx context.Context, id int64, amount float64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE payment_schedules SET paid_amount = paid_amount + $1, status = CASE WHEN paid_amount + $1 >= amount THEN 'paid' ELSE status END WHERE id=$2`, amount, id)
	return err
}

func (r *paymentScheduleRepo) MarkPaidTx(ctx context.Context, tx interface{}, id int64, amount float64) error {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("неверный тип транзакции")
	}
	_, err := txx.ExecContext(ctx, `UPDATE payment_schedules SET paid_amount = paid_amount + $1, status = CASE WHEN paid_amount + $1 >= amount THEN 'paid' ELSE status END WHERE id=$2`, amount, id)
	return err
}

func (r *paymentScheduleRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE payment_schedules SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *paymentScheduleRepo) UpdateStatusTx(ctx context.Context, tx interface{}, id int64, status string) error {
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("неверный тип транзакции")
	}
	_, err := txx.ExecContext(ctx, `UPDATE payment_schedules SET status=$1 WHERE id=$2`, status, id)
	return err
}
