package service

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/model"
	"bank-api/internal/repository"
	"context"
	"database/sql"
	"strings"
	"time"
)

// accountService отвечает за логику работы со счетами и операциями по ним
type accountService struct {
	accountRepo     repository.AccountRepo
	transactionRepo repository.TransactionRepo
	db              *sql.DB
}

// NewAccountService создаёт сервис операций со счетами
func NewAccountService(repos *repository.Repositories, db *sql.DB) AccountService {
	return &accountService{
		accountRepo:     repos.Account,
		transactionRepo: repos.Transaction,
		db:              db,
	}
}

// Create открывает новый счёт только в рублях
func (s *accountService) Create(ctx context.Context, userID int64, currency string) (*dto.AccountResponse, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency != "RUB" {
		return nil, errors.NewAppError(400, "Поддерживается только валюта RUB", nil)
	}

	acc := &model.Account{
		UserID:   userID,
		Balance:  0,
		Currency: currency,
	}
	if err := s.accountRepo.Create(ctx, acc); err != nil {
		return nil, err
	}
	return &dto.AccountResponse{
		ID:        acc.ID,
		UserID:    acc.UserID,
		Balance:   acc.Balance,
		Currency:  acc.Currency,
		CreatedAt: acc.CreatedAt.Format(time.RFC3339),
	}, nil
}

// GetByID возвращает счёт только его владельцу
func (s *accountService) GetByID(ctx context.Context, userID, accountID int64) (*dto.AccountResponse, error) {
	acc, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт не найден")
	}
	if acc.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя просматривать чужой счёт", nil)
	}
	return &dto.AccountResponse{
		ID:        acc.ID,
		UserID:    acc.UserID,
		Balance:   acc.Balance,
		Currency:  acc.Currency,
		CreatedAt: acc.CreatedAt.Format(time.RFC3339),
	}, nil
}

// List возвращает все счета пользователя
func (s *accountService) List(ctx context.Context, userID int64) ([]*dto.AccountResponse, error) {
	accounts, err := s.accountRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var res []*dto.AccountResponse
	for _, a := range accounts {
		res = append(res, &dto.AccountResponse{
			ID: a.ID, UserID: a.UserID, Balance: a.Balance, Currency: a.Currency, CreatedAt: a.CreatedAt.Format(time.RFC3339),
		})
	}
	return res, nil
}

// Deposit пополняет счёт и сохраняет операцию в истории в рамках одной транзакции
func (s *accountService) Deposit(ctx context.Context, userID, accountID int64, amount float64) error {
	if amount <= 0 {
		return errors.NewAppError(400, "Сумма пополнения должна быть положительной", nil)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	acc, err := s.accountRepo.GetForUpdate(ctx, tx, accountID)
	if err != nil {
		return errors.WrapNotFound(err, "Счёт для пополнения не найден")
	}
	if acc.UserID != userID {
		return errors.NewAppError(403, "Нельзя пополнять чужой счёт", nil)
	}

	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, accountID, amount); err != nil {
		return err
	}
	txn := &model.Transaction{
		ToAccountID: &accountID,
		Amount:      amount,
		Type:        "deposit",
		Description: "Пополнение счета",
	}
	if err := s.transactionRepo.CreateTx(ctx, tx, txn); err != nil {
		return err
	}

	return tx.Commit()
}

// Withdraw списывает деньги со счёта и сохраняет операцию в истории в рамках одной транзакции
func (s *accountService) Withdraw(ctx context.Context, userID, accountID int64, amount float64) error {
	if amount <= 0 {
		return errors.NewAppError(400, "Сумма списания должна быть положительной", nil)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	acc, err := s.accountRepo.GetForUpdate(ctx, tx, accountID)
	if err != nil {
		return errors.WrapNotFound(err, "Счёт для списания не найден")
	}
	if acc.UserID != userID {
		return errors.NewAppError(403, "Нельзя списывать деньги с чужого счёта", nil)
	}

	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, accountID, -amount); err != nil {
		if errors.HasCode(err, 400) {
			return errors.NewAppError(400, "На счёте недостаточно средств для списания", err)
		}
		return err
	}
	txn := &model.Transaction{
		FromAccountID: &accountID,
		Amount:        amount,
		Type:          "withdraw",
		Description:   "Снятие средств",
	}
	if err := s.transactionRepo.CreateTx(ctx, tx, txn); err != nil {
		return err
	}

	return tx.Commit()
}
