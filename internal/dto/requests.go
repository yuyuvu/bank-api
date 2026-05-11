package dto

// RegisterRequest - тело запроса для регистрации
type RegisterRequest struct {
	Username string `json:"username" validate:"required,username"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
}

// LoginRequest - тело запроса для входа в систему по почте и паролю
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	OTPCode  string `json:"otp_code,omitempty" validate:"omitempty,len=6,numeric"`
}

// CreateAccountRequest - тело запроса для открытия нового счёта
type CreateAccountRequest struct {
	Currency string `json:"currency" validate:"required,eq=RUB"`
}

// AmountRequest - тело запроса для операций пополнения и списания
type AmountRequest struct {
	Amount float64 `json:"amount" validate:"required,gt=0"`
}

// IssueCardRequest - тело запроса для выпуска банковской (виртуальной) карты
type IssueCardRequest struct {
	AccountID int64 `json:"account_id" validate:"required,gt=0"`
}

// TransferRequest - тело запроса для перевода между счетами
type TransferRequest struct {
	FromAccountID int64   `json:"from_account_id" validate:"required,gt=0"`
	ToAccountID   int64   `json:"to_account_id" validate:"required,gt=0"`
	Amount        float64 `json:"amount" validate:"required,gt=0"`
	OTPCode       string  `json:"otp_code,omitempty" validate:"omitempty,len=6,numeric"`
}

// CreditApplication - тело запроса для оформления кредита
type CreditApplication struct {
	AccountID  int64   `json:"account_id" validate:"required,gt=0"`
	Amount     float64 `json:"amount" validate:"required,gt=0"`
	TermMonths int     `json:"term_months" validate:"required,min=1,max=360"`
}

// CardPaymentRequest - тело запроса для оплаты картой
type CardPaymentRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	CVV         string  `json:"cvv" validate:"required,len=3,numeric"`
	Description string  `json:"description,omitempty" validate:"omitempty,max=255"`
	OTPCode     string  `json:"otp_code,omitempty" validate:"omitempty,len=6,numeric"`
}

// Enable2FARequest - тело запроса для включения двухфакторной аутентификации
type Enable2FARequest struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

// Disable2FARequest - тело запроса для отключения двухфакторной аутентификации
type Disable2FARequest struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

// CreditPaymentRequest - тело запроса для ручной оплаты ближайшего платежа по кредиту
type CreditPaymentRequest struct {
	OTPCode string `json:"otp_code,omitempty" validate:"omitempty,len=6,numeric"`
}

// BlockRequest - тело запроса для блокировки или разблокировки пользователя
type BlockRequest struct {
	Block *bool `json:"block" validate:"required"`
}
