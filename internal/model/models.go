package model

import "time"

// User хранит данные пользователя системы и его параметры безопасности
type User struct {
	ID               int64     `json:"id" db:"id"`
	Username         string    `json:"username" db:"username"`
	Email            string    `json:"email" db:"email"`
	PasswordHash     string    `json:"-" db:"password_hash"`
	TwoFactorKey     string    `json:"-" db:"two_factor_key"`
	TwoFactorEnabled bool      `json:"two_factor_enabled" db:"two_factor_enabled"`
	IsBlocked        bool      `json:"is_blocked" db:"is_blocked"`
	Role             string    `json:"role" db:"role"` // "user" или "admin"
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Account хранит банковский счёт пользователя
type Account struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Balance   float64   `json:"balance" db:"balance"`
	Currency  string    `json:"currency" db:"currency"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Card хранит данные банковской карты
type Card struct {
	ID              int64     `json:"id" db:"id"`
	AccountID       int64     `json:"account_id" db:"account_id"`
	UserID          int64     `json:"user_id" db:"user_id"`
	EncryptedNumber string    `json:"-" db:"encrypted_number"`
	HmacNumber      string    `json:"-" db:"hmac_number"`
	EncryptedExpiry string    `json:"-" db:"encrypted_expiry"`
	EncryptedCVV    string    `json:"-" db:"encrypted_cvv"`
	BcryptCVV       string    `json:"-" db:"bcrypt_cvv"`
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Transaction описывает определённую операцию по счёту
type Transaction struct {
	ID            int64     `json:"id" db:"id"`
	FromAccountID *int64    `json:"from_account_id,omitempty" db:"from_account_id"`
	ToAccountID   *int64    `json:"to_account_id,omitempty" db:"to_account_id"`
	Amount        float64   `json:"amount" db:"amount"`
	Type          string    `json:"type" db:"type"`
	Description   string    `json:"description" db:"description"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// Credit описывает параметры выданного кредита
type Credit struct {
	ID              int64     `json:"id" db:"id"`
	AccountID       int64     `json:"account_id" db:"account_id"`
	UserID          int64     `json:"user_id" db:"user_id"`
	Amount          float64   `json:"amount" db:"amount"`
	InterestRate    float64   `json:"interest_rate" db:"interest_rate"`
	TermMonths      int       `json:"term_months" db:"term_months"`
	MonthlyPayment  float64   `json:"monthly_payment" db:"monthly_payment"`
	RemainingAmount float64   `json:"remaining_amount" db:"remaining_amount"`
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// PaymentSchedule описывает отдельный запланированный платёж по кредиту
type PaymentSchedule struct {
	ID         int64     `json:"id" db:"id"`
	CreditID   int64     `json:"credit_id" db:"credit_id"`
	DueDate    time.Time `json:"due_date" db:"due_date"`
	Amount     float64   `json:"amount" db:"amount"`
	PaidAmount float64   `json:"paid_amount" db:"paid_amount"`
	Status     string    `json:"status" db:"status"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
