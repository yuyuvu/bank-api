package service

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/model"
	"bank-api/internal/repository"
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/sirupsen/logrus"
)

// creditService отвечает за логику оформления кредита и работу с платежами по нему
type creditService struct {
	creditRepo          repository.CreditRepo
	paymentScheduleRepo repository.PaymentScheduleRepo
	accountRepo         repository.AccountRepo
	transactionRepo     repository.TransactionRepo
	userRepo            repository.UserRepo
	cbrService          CBRService
	notificationService *NotificationService
	db                  *sql.DB
	log                 *logrus.Logger
}

// NewCreditService создаёт сервис выдачи кредитов и обработки платежей по ним
func NewCreditService(repos *repository.Repositories, cbr CBRService, db *sql.DB, log *logrus.Logger, notification *NotificationService) CreditService {
	return &creditService{
		creditRepo:          repos.Credit,
		paymentScheduleRepo: repos.PaymentSchedule,
		accountRepo:         repos.Account,
		transactionRepo:     repos.Transaction,
		userRepo:            repos.User,
		cbrService:          cbr,
		notificationService: notification,
		db:                  db,
		log:                 log,
	}
}

// Apply оформляет кредит, строит график платежей и зачисляет сумму кредита на счёт
func (s *creditService) Apply(ctx context.Context, userID int64, req *dto.CreditApplication) (*dto.CreditResponse, error) {
	// Проверка счета
	acc, err := s.accountRepo.GetByID(ctx, req.AccountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт для оформления кредита не найден")
	}
	if acc.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя оформить кредит на чужой счёт", nil)
	}

	// Получение ключевой ставки + маржа
	rate, err := s.cbrService.GetRate(ctx)
	if err != nil {
		s.log.Errorf("Не удалось получить ставку ЦБ: %v, используется 20%%", err)
		rate = 20.0 // значение по умолчанию
	}
	monthlyRate := rate / 12 / 100

	// Аннуитетный платеж
	n := float64(req.TermMonths)
	monthlyPayment := req.Amount * (monthlyRate * math.Pow(1+monthlyRate, n)) / (math.Pow(1+monthlyRate, n) - 1)
	monthlyPayment = math.Round(monthlyPayment*100) / 100
	totalScheduledAmount := math.Round(monthlyPayment*float64(req.TermMonths)*100) / 100

	credit := &model.Credit{
		AccountID:       req.AccountID,
		UserID:          userID,
		Amount:          req.Amount,
		InterestRate:    rate,
		TermMonths:      req.TermMonths,
		MonthlyPayment:  monthlyPayment,
		RemainingAmount: totalScheduledAmount,
		Status:          "active",
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := s.creditRepo.CreateTx(ctx, tx, credit); err != nil {
		return nil, err
	}

	// Генерация графика платежей
	var schedules []*model.PaymentSchedule
	dueDate := time.Now().AddDate(0, 1, 0) // первый платёж через месяц
	for i := 0; i < req.TermMonths; i++ {
		schedule := &model.PaymentSchedule{
			CreditID: credit.ID,
			DueDate:  dueDate,
			Amount:   monthlyPayment,
			Status:   "pending",
		}
		schedules = append(schedules, schedule)
		dueDate = dueDate.AddDate(0, 1, 0)
	}
	if err := s.paymentScheduleRepo.CreateBatchTx(ctx, tx, schedules); err != nil {
		return nil, err
	}

	// Зачисление суммы кредита на счёт
	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, req.AccountID, req.Amount); err != nil {
		return nil, err
	}
	txn := &model.Transaction{
		ToAccountID: &req.AccountID,
		Amount:      req.Amount,
		Type:        "loan_issuance",
		Description: "Выдача кредита",
	}
	if err := s.transactionRepo.CreateTx(ctx, tx, txn); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &dto.CreditResponse{
		ID:             credit.ID,
		MonthlyPayment: monthlyPayment,
		InterestRate:   rate,
	}, nil
}

// List возвращает все кредиты пользователя
func (s *creditService) List(ctx context.Context, userID int64) ([]*dto.CreditListItemResponse, error) {
	credits, err := s.creditRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.CreditListItemResponse, 0, len(credits))
	for _, credit := range credits {
		result = append(result, &dto.CreditListItemResponse{
			ID:              credit.ID,
			AccountID:       credit.AccountID,
			Amount:          credit.Amount,
			InterestRate:    credit.InterestRate,
			TermMonths:      credit.TermMonths,
			MonthlyPayment:  credit.MonthlyPayment,
			RemainingAmount: credit.RemainingAmount,
			Status:          credit.Status,
			CreatedAt:       credit.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

// GetSchedule возвращает график платежей только владельцу кредита
func (s *creditService) GetSchedule(ctx context.Context, userID, creditID int64) ([]*dto.PaymentScheduleResponse, error) {
	credit, err := s.creditRepo.GetByID(ctx, creditID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Кредит не найден")
	}
	if credit.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя просматривать график чужого кредита", nil)
	}
	schedules, err := s.paymentScheduleRepo.GetByCreditID(ctx, creditID)
	if err != nil {
		return nil, err
	}
	var res []*dto.PaymentScheduleResponse
	for _, sched := range schedules {
		res = append(res, &dto.PaymentScheduleResponse{
			DueDate:    sched.DueDate.Format("2006-01-02"),
			Amount:     sched.Amount,
			PaidAmount: sched.PaidAmount,
			Status:     sched.Status,
		})
	}
	return res, nil
}

// PayNext позволяет вручную погасить ближайший непогашенный платёж по кредиту
func (s *creditService) PayNext(ctx context.Context, userID, creditID int64, req *dto.CreditPaymentRequest) (*dto.CreditPaymentResponse, error) {
	if err := validateTwoFactorIfEnabled(ctx, s.userRepo, userID, req.OTPCode); err != nil {
		return nil, err
	}

	credit, err := s.creditRepo.GetByID(ctx, creditID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Кредит не найден")
	}
	if credit.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя оплачивать чужой кредит", nil)
	}
	if credit.Status != "active" {
		return nil, errors.NewAppError(400, "Этот кредит уже закрыт", nil)
	}

	schedule, err := s.paymentScheduleRepo.GetNextUnpaidByCredit(ctx, creditID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "У кредита нет непогашенных платежей")
	}

	baseDue := math.Round((schedule.Amount-schedule.PaidAmount)*100) / 100
	if baseDue <= 0 {
		return nil, errors.NewAppError(400, "Ближайший платёж уже полностью погашен", nil)
	}

	penalty := 0.0
	now := time.Now()
	if schedule.DueDate.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
		penalty = math.Round(baseDue*0.1*100) / 100
	}
	totalDue := baseDue + penalty

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	account, err := s.accountRepo.GetForUpdate(ctx, tx, credit.AccountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт кредита не найден")
	}
	if account.UserID != userID {
		return nil, errors.NewAppError(403, "Кредит привязан к чужому счёту", nil)
	}

	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, credit.AccountID, -totalDue); err != nil {
		if errors.HasCode(err, 400) {
			return nil, errors.NewAppError(400, "На счёте недостаточно средств для платежа по кредиту", err)
		}
		return nil, err
	}

	if err := s.paymentScheduleRepo.MarkPaidTx(ctx, tx, schedule.ID, baseDue); err != nil {
		return nil, err
	}

	remainingDebt, err := s.creditRepo.ReduceRemainingAmountTx(ctx, tx, credit.ID, baseDue)
	if err != nil {
		return nil, err
	}

	if remainingDebt == 0 {
		if err := s.creditRepo.UpdateStatusTx(ctx, tx, credit.ID, "paid"); err != nil {
			return nil, err
		}
	}

	txn := &model.Transaction{
		FromAccountID: &credit.AccountID,
		Amount:        totalDue,
		Type:          "loan_payment_manual",
		Description:   "Ручная оплата платежа по кредиту",
	}
	if err := s.transactionRepo.CreateTx(ctx, tx, txn); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if user, err := s.userRepo.GetByID(ctx, userID); err == nil && s.notificationService != nil {
		_ = s.notificationService.SendCreditPaymentEmail(
			user.Email,
			txn.CreatedAt,
			credit.ID,
			schedule.DueDate,
			baseDue,
			penalty,
			totalDue,
			credit.AccountID,
			txn.ID,
			false,
		)
	}

	return &dto.CreditPaymentResponse{
		TransactionID: txn.ID,
		PaidAmount:    baseDue,
		PenaltyAmount: penalty,
		RemainingDebt: remainingDebt,
	}, nil
}

// ProcessOverduePayments вызывается фоновым планировщиком для обработки просроченных платежей каждые 12 часов.
func (s *creditService) ProcessOverduePayments(ctx context.Context) error {
	s.log.Info("Запуск обработки просроченных платежей")
	overdue, err := s.paymentScheduleRepo.GetOverdue(ctx)
	if err != nil {
		return err
	}
	for _, p := range overdue {
		credit, err := s.creditRepo.GetByID(ctx, p.CreditID)
		if err != nil {
			s.log.Errorf("Ошибка получения кредита %d: %v", p.CreditID, err)
			continue
		}
		if credit.Status != "active" {
			continue
		}
		// Штраф +10% от суммы платежа
		penalty := math.Round(p.Amount*0.1*100) / 100
		baseDue := math.Round((p.Amount-p.PaidAmount)*100) / 100
		if baseDue <= 0 {
			continue
		}
		totalDue := baseDue + penalty
		// Попытка списания со счёта кредита
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			s.log.Errorf("Ошибка транзакции: %v", err)
			continue
		}
		acc, err := s.accountRepo.GetForUpdate(ctx, tx, credit.AccountID)
		if err != nil {
			tx.Rollback()
			s.log.Errorf("Ошибка получения счёта: %v", err)
			continue
		}
		if acc.Balance >= totalDue {
			if err := s.accountRepo.UpdateBalanceTx(ctx, tx, credit.AccountID, -totalDue); err != nil {
				tx.Rollback()
				s.log.Errorf("Ошибка списания: %v", err)
				continue
			}
			if err := s.paymentScheduleRepo.MarkPaidTx(ctx, tx, p.ID, baseDue); err != nil {
				tx.Rollback()
				s.log.Errorf("Ошибка отметки платежа: %v", err)
				continue
			}
			remaining, err := s.creditRepo.ReduceRemainingAmountTx(ctx, tx, credit.ID, baseDue)
			if err != nil {
				tx.Rollback()
				s.log.Errorf("Ошибка обновления остатка кредита: %v", err)
				continue
			}
			if remaining == 0 {
				if err := s.creditRepo.UpdateStatusTx(ctx, tx, credit.ID, "paid"); err != nil {
					tx.Rollback()
					s.log.Errorf("Ошибка обновления статуса кредита: %v", err)
					continue
				}
			}
			txn := &model.Transaction{
				FromAccountID: &credit.AccountID,
				Amount:        totalDue,
				Type:          "loan_payment",
				Description:   "Автоматическое погашение кредита (просрочка)",
			}
			if err := s.transactionRepo.CreateTx(ctx, tx, txn); err != nil {
				tx.Rollback()
				s.log.Errorf("Ошибка записи операции: %v", err)
				continue
			}
			if err := tx.Commit(); err != nil {
				s.log.Errorf("Ошибка коммита: %v", err)
			} else {
				s.log.Infof("Платёж %d списан", p.ID)
				// Отправка уведомления
				user, _ := s.userRepo.GetByID(ctx, credit.UserID)
				if user != nil && s.notificationService != nil {
					if err := s.notificationService.SendCreditPaymentEmail(
						user.Email,
						txn.CreatedAt,
						credit.ID,
						p.DueDate,
						baseDue,
						penalty,
						totalDue,
						credit.AccountID,
						txn.ID,
						true,
					); err != nil {
						s.log.Errorf("Не удалось отправить email: %v", err)
					}
				}
			}
		} else {
			if err := s.paymentScheduleRepo.UpdateStatus(ctx, p.ID, "overdue"); err != nil {
				s.log.Errorf("Не удалось обновить статус просрочки %d: %v", p.ID, err)
			}
			tx.Rollback()
			// Недостаточно средств, оставляем статус overdue
			s.log.Warnf("Недостаточно средств для платежа %d", p.ID)
		}
	}
	return nil
}
