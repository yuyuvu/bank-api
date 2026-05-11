package dto

import "time"

// AuthResponse - ответ от API после успешной аутентификации
type AuthResponse struct {
	Token string `json:"token"`
}

// AccountResponse - ответ от API для запроса на считывание данных счёта
type AccountResponse struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"user_id"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	CreatedAt string  `json:"created_at,omitempty"`
}

// TransferResponse - ответ от API с идентификатором созданной операции
type TransferResponse struct {
	TransactionID int64 `json:"transaction_id"`
}

// CardResponse - ответ от API после выпуска карты; также содержится в ответе при запросе списка карт
type CardResponse struct {
	ID           int64  `json:"id"`
	AccountID    int64  `json:"account_id"`
	MaskedNumber string `json:"masked_number"` // последние 4 цифры
	Status       string `json:"status"`
}

// CardDetailResponse - ответ от API для владельца карты с расшифрованными данными карты
type CardDetailResponse struct {
	ID     int64  `json:"id"`
	Number string `json:"number"` // расшифрованный номер (только для владельца)
	Expiry string `json:"expiry"` // MM/YYYY
	CVV    string `json:"cvv"`    // расшифрованный CVV (только для владельца)
	Status string `json:"status"`
}

// CardPaymentResponse - ответ от API с результатом оплаты картой
type CardPaymentResponse struct {
	TransactionID int64   `json:"transaction_id"`
	AccountID     int64   `json:"account_id"`
	Amount        float64 `json:"amount"`
}

// TwoFactorSetupResponse - ответ от API с данными для настройки двухфакторной аутентификации
type TwoFactorSetupResponse struct {
	Secret          string `json:"secret"`
	URL             string `json:"url"`
	Issuer          string `json:"issuer"`
	AccountName     string `json:"account_name"`
	QRCodePNGBase64 string `json:"qr_code_png_base64,omitempty"`
}

// CreditResponse - ответ от API с основными параметрами нового кредита
type CreditResponse struct {
	ID             int64   `json:"id"`
	MonthlyPayment float64 `json:"monthly_payment"`
	InterestRate   float64 `json:"interest_rate"`
}

// CreditListItemResponse - элемент ответа от API для запроса на считывание списка кредитов пользователя
type CreditListItemResponse struct {
	ID              int64   `json:"id"`
	AccountID       int64   `json:"account_id"`
	Amount          float64 `json:"amount"`
	InterestRate    float64 `json:"interest_rate"`
	TermMonths      int     `json:"term_months"`
	MonthlyPayment  float64 `json:"monthly_payment"`
	RemainingAmount float64 `json:"remaining_amount"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
}

// PaymentScheduleResponse - ответ от API с одной строкой из графика платежей
type PaymentScheduleResponse struct {
	DueDate    string  `json:"due_date"`
	Amount     float64 `json:"amount"`
	PaidAmount float64 `json:"paid_amount"`
	Status     string  `json:"status"`
}

// IncomeExpenseResponse - ответ от API с итогом по доходам и расходам за месяц
type IncomeExpenseResponse struct {
	YearMonth string  `json:"year_month"`
	Income    float64 `json:"income"`
	Expense   float64 `json:"expense"`
}

// CreditLoadResponse - ответ от API с текущей долговой нагрузкой пользователя на дату ответа
type CreditLoadResponse struct {
	TotalDebt       float64 `json:"total_debt"`
	MonthlyPayments float64 `json:"monthly_payments"`
	AsOfDate        string  `json:"as_of_date"`
}

// BalancePrediction - ответ от API с прогнозом по счёту на конкретную дату
type BalancePrediction struct {
	Date    string  `json:"date"`
	Balance float64 `json:"balance"`
}

// AnalyticsSummaryResponse - ответ от API со сводной аналитикой
type AnalyticsSummaryResponse struct {
	Scope         string                 `json:"scope"`
	YearMonth     string                 `json:"year_month"`
	AccountID     *int64                 `json:"account_id,omitempty"`
	IncomeExpense *IncomeExpenseResponse `json:"income_expense"`
	CreditLoad    *CreditLoadResponse    `json:"credit_load"`
}

// CreditPaymentResponse показывает результат ручной оплаты по кредиту
type CreditPaymentResponse struct {
	TransactionID int64   `json:"transaction_id"`
	PaidAmount    float64 `json:"paid_amount"`
	PenaltyAmount float64 `json:"penalty_amount"`
	RemainingDebt float64 `json:"remaining_debt"`
}

// AccountPredictionResponse - ответ от API с прогнозом по одному счёту
type AccountPredictionResponse struct {
	AccountID      int64                `json:"account_id"`
	CurrentBalance float64              `json:"current_balance"`
	Currency       string               `json:"currency"`
	Predictions    []*BalancePrediction `json:"predictions"`
}

// AllAccountsPredictionResponse - ответ от API с прогнозом по каждому счёту и суммарно по всем счетам
type AllAccountsPredictionResponse struct {
	Currency            string                       `json:"currency"`
	CurrentTotalBalance float64                      `json:"current_total_balance"`
	TotalPredictions    []*BalancePrediction         `json:"total_predictions"`
	Accounts            []*AccountPredictionResponse `json:"accounts"`
}

// AdminUserResponse - ответ от API для запроса на вывод всех пользователей
type AdminUserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsBlocked bool      `json:"is_blocked"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
