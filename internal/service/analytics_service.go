package service

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/model"
	"bank-api/internal/repository"
	"context"
	"time"
)

// analyticsService собирает аналитику для пользователя
type analyticsService struct {
	transactionRepo     repository.TransactionRepo
	creditRepo          repository.CreditRepo
	paymentScheduleRepo repository.PaymentScheduleRepo
	accountRepo         repository.AccountRepo
}

// NewAnalyticsService создаёт сервис аналитики для пользователя
func NewAnalyticsService(repos *repository.Repositories) AnalyticsService {
	return &analyticsService{
		transactionRepo:     repos.Transaction,
		creditRepo:          repos.Credit,
		paymentScheduleRepo: repos.PaymentSchedule,
		accountRepo:         repos.Account,
	}
}

// IncomeExpense считает доходы и расходы пользователя за указанный месяц
func (s *analyticsService) IncomeExpense(ctx context.Context, userID int64, yearMonth string) (*dto.IncomeExpenseResponse, error) {
	from, to, err := parseYearMonth(yearMonth)
	if err != nil {
		return nil, err
	}

	income, expense, err := s.transactionRepo.GetIncomeExpenseByUser(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	return &dto.IncomeExpenseResponse{YearMonth: yearMonth, Income: income, Expense: expense}, nil
}

// IncomeExpenseByAccount считает доходы и расходы по одному счёту за указанный месяц
func (s *analyticsService) IncomeExpenseByAccount(ctx context.Context, userID, accountID int64, yearMonth string) (*dto.IncomeExpenseResponse, error) {
	account, err := s.ensureAccountOwnership(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	from, to, err := parseYearMonth(yearMonth)
	if err != nil {
		return nil, err
	}

	transactions, err := s.transactionRepo.ListByAccount(ctx, account.ID, from, to)
	if err != nil {
		return nil, err
	}

	income := 0.0
	expense := 0.0
	for _, transaction := range transactions {
		if transaction.ToAccountID != nil && *transaction.ToAccountID == account.ID {
			income += transaction.Amount
		}
		if transaction.FromAccountID != nil && *transaction.FromAccountID == account.ID {
			expense += transaction.Amount
		}
	}

	return &dto.IncomeExpenseResponse{YearMonth: yearMonth, Income: income, Expense: expense}, nil
}

// CreditLoad возвращает суммарный остаток долга и ежемесячную нагрузку по активным кредитам
func (s *analyticsService) CreditLoad(ctx context.Context, userID int64) (*dto.CreditLoadResponse, error) {
	credits, err := s.creditRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return buildCreditLoadResponse(credits, 0), nil
}

// CreditLoadByAccount возвращает кредитную нагрузку по одному счёту
func (s *analyticsService) CreditLoadByAccount(ctx context.Context, userID, accountID int64) (*dto.CreditLoadResponse, error) {
	if _, err := s.ensureAccountOwnership(ctx, userID, accountID); err != nil {
		return nil, err
	}

	credits, err := s.creditRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return buildCreditLoadResponse(credits, accountID), nil
}

// PredictBalance строит прогноз баланса по счёту с учётом будущих списаний по кредитам
func (s *analyticsService) PredictBalance(ctx context.Context, userID, accountID int64, days int) ([]*dto.BalancePrediction, error) {
	days, err := normalizeForecastDays(days)
	if err != nil {
		return nil, err
	}

	account, err := s.ensureAccountOwnership(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	return s.buildPredictionForAccount(ctx, userID, account, days, time.Now())
}

// PredictAllBalances строит прогноз сразу по всем счетам пользователя
func (s *analyticsService) PredictAllBalances(ctx context.Context, userID int64, days int) (*dto.AllAccountsPredictionResponse, error) {
	days, err := normalizeForecastDays(days)
	if err != nil {
		return nil, err
	}

	accounts, err := s.accountRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	startDate := time.Now()
	result := make([]*dto.AccountPredictionResponse, 0, len(accounts))
	totalPredictions := make([]*dto.BalancePrediction, days)
	currentTotalBalance := 0.0
	for _, account := range accounts {
		prediction, err := s.buildPredictionForAccount(ctx, userID, account, days, startDate)
		if err != nil {
			return nil, err
		}
		currentTotalBalance += account.Balance
		for i := range prediction {
			if totalPredictions[i] == nil {
				totalPredictions[i] = &dto.BalancePrediction{Date: prediction[i].Date}
			}
			totalPredictions[i].Balance += prediction[i].Balance
		}

		result = append(result, &dto.AccountPredictionResponse{
			AccountID:      account.ID,
			CurrentBalance: account.Balance,
			Currency:       account.Currency,
			Predictions:    prediction,
		})
	}

	return &dto.AllAccountsPredictionResponse{
		Currency:            "RUB",
		CurrentTotalBalance: currentTotalBalance,
		TotalPredictions:    totalPredictions,
		Accounts:            result,
	}, nil
}

// Summary собирает сводную аналитику для пользователя
func (s *analyticsService) Summary(ctx context.Context, userID int64, yearMonth string) (*dto.AnalyticsSummaryResponse, error) {
	incomeExpense, err := s.IncomeExpense(ctx, userID, yearMonth)
	if err != nil {
		return nil, err
	}

	creditLoad, err := s.CreditLoad(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.AnalyticsSummaryResponse{
		Scope:         "user",
		YearMonth:     yearMonth,
		IncomeExpense: incomeExpense,
		CreditLoad:    creditLoad,
	}, nil
}

// AccountSummary собирает сводную аналитику для одного счёта
func (s *analyticsService) AccountSummary(ctx context.Context, userID, accountID int64, yearMonth string) (*dto.AnalyticsSummaryResponse, error) {
	if _, err := s.ensureAccountOwnership(ctx, userID, accountID); err != nil {
		return nil, err
	}

	incomeExpense, err := s.IncomeExpenseByAccount(ctx, userID, accountID, yearMonth)
	if err != nil {
		return nil, err
	}

	creditLoad, err := s.CreditLoadByAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	return &dto.AnalyticsSummaryResponse{
		Scope:         "account",
		YearMonth:     yearMonth,
		AccountID:     &accountID,
		IncomeExpense: incomeExpense,
		CreditLoad:    creditLoad,
	}, nil
}

// ensureAccountOwnership проверяет, что счёт существует и действительно принадлежит текущему пользователю
func (s *analyticsService) ensureAccountOwnership(ctx context.Context, userID, accountID int64) (*model.Account, error) {
	acc, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт не найден")
	}
	if acc.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя работать с чужим счётом", nil)
	}

	return acc, nil
}

// buildPredictionForAccount считает прогноз по одному счёту с учётом будущих платежей по кредитам с этого счёта
func (s *analyticsService) buildPredictionForAccount(ctx context.Context, userID int64, account *model.Account, days int, startDate time.Time) ([]*dto.BalancePrediction, error) {
	credits, err := s.creditRepo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	dailyOut := make(map[string]float64)
	for _, credit := range credits {
		if credit.AccountID != account.ID {
			continue
		}

		schedules, err := s.paymentScheduleRepo.GetByCreditID(ctx, credit.ID)
		if err != nil {
			return nil, err
		}
		for _, schedule := range schedules {
			if schedule.Status != "pending" && schedule.Status != "overdue" {
				continue
			}

			key := schedule.DueDate.Format("2006-01-02")
			dailyOut[key] += schedule.Amount
		}
	}

	predictions := make([]*dto.BalancePrediction, days)
	currentBalance := account.Balance
	currentDate := startDate
	for i := 0; i < days; i++ {
		dateStr := currentDate.Format("2006-01-02")
		if out, ok := dailyOut[dateStr]; ok {
			currentBalance -= out
		}
		predictions[i] = &dto.BalancePrediction{Date: dateStr, Balance: currentBalance}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return predictions, nil
}

// parseYearMonth разбирает месяц вида YYYY-MM во временной интервал от начала месяца до начала следующего
func parseYearMonth(yearMonth string) (time.Time, time.Time, error) {
	from, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return time.Time{}, time.Time{}, errors.NewAppError(400, "Параметр year_month должен быть в формате YYYY-MM", err)
	}

	return from, from.AddDate(0, 1, 0), nil
}

// normalizeForecastDays не даёт запросить отрицательный прогноз и жёстко ограничивает период 365 днями
func normalizeForecastDays(days int) (int, error) {
	if days <= 0 {
		return 0, errors.NewAppError(400, "Период прогноза должен быть положительным", nil)
	}
	if days > 365 {
		return 365, nil
	}

	return days, nil
}

// buildCreditLoadResponse собирает текущую долговую нагрузку либо по всем счетам пользователя, либо по одному счёту
func buildCreditLoadResponse(credits []*model.Credit, accountID int64) *dto.CreditLoadResponse {
	totalDebt := 0.0
	monthlyPayments := 0.0
	for _, credit := range credits {
		if credit.Status != "active" {
			continue
		}
		if accountID > 0 && credit.AccountID != accountID {
			continue
		}

		totalDebt += credit.RemainingAmount
		monthlyPayments += credit.MonthlyPayment
	}

	return &dto.CreditLoadResponse{
		TotalDebt:       totalDebt,
		MonthlyPayments: monthlyPayments,
		AsOfDate:        time.Now().Format("2006-01-02"),
	}
}
