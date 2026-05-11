package service

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/model"
	"bank-api/internal/repository"
	"context"
	"database/sql"
)

// transferService отвечает за логику переводов между счетами
type transferService struct {
	accountRepo     repository.AccountRepo
	transactionRepo repository.TransactionRepo
	userRepo        repository.UserRepo
	notification    *NotificationService
	db              *sql.DB
}

// NewTransferService создаёт сервис для переводов между счетами
func NewTransferService(repos *repository.Repositories, db *sql.DB, notification *NotificationService) TransferService {
	return &transferService{
		accountRepo:     repos.Account,
		transactionRepo: repos.Transaction,
		userRepo:        repos.User,
		notification:    notification,
		db:              db,
	}
}

// Transfer выполняет перевод между счетами в одной транзакции
func (s *transferService) Transfer(ctx context.Context, userID int64, req *dto.TransferRequest) (*dto.TransferResponse, error) {
	if err := validateTwoFactorIfEnabled(ctx, s.userRepo, userID, req.OTPCode); err != nil {
		return nil, err
	}

	if req.FromAccountID == req.ToAccountID {
		return nil, errors.NewAppError(400, "Нельзя переводить на тот же счёт", nil)
	}
	if req.Amount <= 0 {
		return nil, errors.NewAppError(400, "Сумма перевода должна быть положительной", nil)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	fromAcc, err := s.accountRepo.GetForUpdate(ctx, tx, req.FromAccountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт списания не найден")
	}
	if fromAcc.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя переводить деньги с чужого счёта", nil)
	}

	toAcc, err := s.accountRepo.GetForUpdate(ctx, tx, req.ToAccountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт зачисления не найден")
	}
	if toAcc.ID == 0 {
		return nil, errors.NewAppError(404, "Счёт зачисления не найден", nil)
	}

	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, req.FromAccountID, -req.Amount); err != nil {
		if errors.HasCode(err, 400) {
			return nil, errors.NewAppError(400, "На счёте списания недостаточно средств", err)
		}
		return nil, err
	}
	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, req.ToAccountID, req.Amount); err != nil {
		return nil, err
	}

	txn := &model.Transaction{
		FromAccountID: &req.FromAccountID,
		ToAccountID:   &req.ToAccountID,
		Amount:        req.Amount,
		Type:          "transfer",
		Description:   "Перевод между счетами",
	}
	if err := s.transactionRepo.CreateTx(ctx, tx, txn); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Отправка уведомления о переводе
	if s.notification != nil {
		fromUser, fromErr := s.userRepo.GetByID(ctx, userID)
		toUser, toErr := s.userRepo.GetByID(ctx, toAcc.UserID)
		if fromErr == nil && toErr == nil && fromUser != nil && toUser != nil {
			_ = s.notification.SendTransferEmail(fromUser.Email, txn.CreatedAt, req.Amount, toUser.Email, toAcc.ID, fromAcc.ID, txn.ID)
		}
	}

	return &dto.TransferResponse{TransactionID: txn.ID}, nil
}
