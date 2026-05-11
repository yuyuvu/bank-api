package repository

import (
	"bank-api/internal/model"
	"context"
	"time"
)

// UserRepo отвечает за операции с пользователями
type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	List(ctx context.Context) ([]*model.User, error)
	Delete(ctx context.Context, id int64) error
}

// AccountRepo отвечает за работу со счетами
type AccountRepo interface {
	Create(ctx context.Context, acc *model.Account) error
	GetByID(ctx context.Context, id int64) (*model.Account, error)
	ListByUser(ctx context.Context, userID int64) ([]*model.Account, error)
	UpdateBalance(ctx context.Context, id int64, delta float64) error
	GetForUpdate(ctx context.Context, tx interface{}, id int64) (*model.Account, error)
	UpdateBalanceTx(ctx context.Context, tx interface{}, id int64, delta float64) error
}

// CardRepo отвечает за работу с данными карт
type CardRepo interface {
	Create(ctx context.Context, card *model.Card) error
	GetByID(ctx context.Context, id int64) (*model.Card, error)
	ListByUser(ctx context.Context, userID int64) ([]*model.Card, error)
}

// TransactionRepo отвечает за работу с данными об операциях по счетам
type TransactionRepo interface {
	Create(ctx context.Context, txn *model.Transaction) error
	CreateTx(ctx context.Context, tx interface{}, txn *model.Transaction) error
	ListByAccount(ctx context.Context, accountID int64, from, to time.Time) ([]*model.Transaction, error)
	ListByUser(ctx context.Context, userID int64, from, to time.Time) ([]*model.Transaction, error)
	GetIncomeExpenseByUser(ctx context.Context, userID int64, from, to time.Time) (float64, float64, error)
}

// CreditRepo отвечает за работу с данными о кредитах
type CreditRepo interface {
	Create(ctx context.Context, credit *model.Credit) error
	CreateTx(ctx context.Context, tx interface{}, credit *model.Credit) error
	GetByID(ctx context.Context, id int64) (*model.Credit, error)
	ListByUser(ctx context.Context, userID int64) ([]*model.Credit, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateStatusTx(ctx context.Context, tx interface{}, id int64, status string) error
	ReduceRemainingAmountTx(ctx context.Context, tx interface{}, id int64, amount float64) (float64, error)
	GetActiveByUser(ctx context.Context, userID int64) ([]*model.Credit, error)
}

// PaymentScheduleRepo отвечает за работу с графиками платежей и состояниями платежей по кредитам
type PaymentScheduleRepo interface {
	CreateBatch(ctx context.Context, schedules []*model.PaymentSchedule) error
	CreateBatchTx(ctx context.Context, tx interface{}, schedules []*model.PaymentSchedule) error
	GetByCreditID(ctx context.Context, creditID int64) ([]*model.PaymentSchedule, error)
	GetNextUnpaidByCredit(ctx context.Context, creditID int64) (*model.PaymentSchedule, error)
	GetOverdue(ctx context.Context) ([]*model.PaymentSchedule, error)
	MarkPaid(ctx context.Context, id int64, amount float64) error
	MarkPaidTx(ctx context.Context, tx interface{}, id int64, amount float64) error
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateStatusTx(ctx context.Context, tx interface{}, id int64, status string) error
}

// Repositories собирает все репозитории в одном наборе
type Repositories struct {
	User            UserRepo
	Account         AccountRepo
	Card            CardRepo
	Transaction     TransactionRepo
	Credit          CreditRepo
	PaymentSchedule PaymentScheduleRepo
}
