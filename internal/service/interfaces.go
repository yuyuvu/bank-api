package service

import (
	"bank-api/internal/dto"
	"context"
)

// AuthService отвечает за регистрацию, вход и 2FA
type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error)
	Generate2FA(ctx context.Context, userID int64) (*dto.TwoFactorSetupResponse, error)
	Enable2FA(ctx context.Context, userID int64, code string) error
	Disable2FA(ctx context.Context, userID int64, code string) error
	Verify2FA(ctx context.Context, userID int64, code string) bool
	BootstrapAdmin(ctx context.Context, userID int64) error
}

// AccountService отвечает за операции со счетами
type AccountService interface {
	Create(ctx context.Context, userID int64, currency string) (*dto.AccountResponse, error)
	GetByID(ctx context.Context, userID, accountID int64) (*dto.AccountResponse, error)
	List(ctx context.Context, userID int64) ([]*dto.AccountResponse, error)
	Deposit(ctx context.Context, userID, accountID int64, amount float64) error
	Withdraw(ctx context.Context, userID, accountID int64, amount float64) error
}

// TransferService отвечает за переводы между счетами
type TransferService interface {
	Transfer(ctx context.Context, userID int64, req *dto.TransferRequest) (*dto.TransferResponse, error)
}

// CardService отвечает за операции с банковскими картами
type CardService interface {
	IssueCard(ctx context.Context, userID, accountID int64) (*dto.CardResponse, error)
	ListCards(ctx context.Context, userID int64) ([]*dto.CardResponse, error)
	GetCard(ctx context.Context, userID, cardID int64) (*dto.CardDetailResponse, error)
	Pay(ctx context.Context, userID, cardID int64, req *dto.CardPaymentRequest) (*dto.CardPaymentResponse, error)
}

// CreditService отвечает за выдачу кредитов и работу с графиком платежей
type CreditService interface {
	Apply(ctx context.Context, userID int64, req *dto.CreditApplication) (*dto.CreditResponse, error)
	List(ctx context.Context, userID int64) ([]*dto.CreditListItemResponse, error)
	GetSchedule(ctx context.Context, userID, creditID int64) ([]*dto.PaymentScheduleResponse, error)
	PayNext(ctx context.Context, userID, creditID int64, req *dto.CreditPaymentRequest) (*dto.CreditPaymentResponse, error)
	ProcessOverduePayments(ctx context.Context) error
}

// AnalyticsService отвечает за запросы на получение аналитики по доходам и расходам и кредитам
type AnalyticsService interface {
	IncomeExpense(ctx context.Context, userID int64, yearMonth string) (*dto.IncomeExpenseResponse, error)
	IncomeExpenseByAccount(ctx context.Context, userID, accountID int64, yearMonth string) (*dto.IncomeExpenseResponse, error)
	CreditLoad(ctx context.Context, userID int64) (*dto.CreditLoadResponse, error)
	CreditLoadByAccount(ctx context.Context, userID, accountID int64) (*dto.CreditLoadResponse, error)
	PredictBalance(ctx context.Context, userID, accountID int64, days int) ([]*dto.BalancePrediction, error)
	PredictAllBalances(ctx context.Context, userID int64, days int) (*dto.AllAccountsPredictionResponse, error)
	Summary(ctx context.Context, userID int64, yearMonth string) (*dto.AnalyticsSummaryResponse, error)
	AccountSummary(ctx context.Context, userID, accountID int64, yearMonth string) (*dto.AnalyticsSummaryResponse, error)
}
