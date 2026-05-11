package handler

import (
	"bank-api/internal/service"

	"github.com/sirupsen/logrus"
)

// Handlers собирает HTTP-обработчики для удобной передачи в роутер
type Handlers struct {
	Auth      *AuthHandler
	Account   *AccountHandler
	Transfer  *TransferHandler
	Card      *CardHandler
	Credit    *CreditHandler
	Analytics *AnalyticsHandler
	Admin     *AdminHandler
}

// NewHandlers создаёт все обработчики и связывает их с сервисами
func NewHandlers(services *service.Services, log *logrus.Logger) *Handlers {
	return &Handlers{
		Auth:      NewAuthHandler(services.AuthService, log),
		Account:   NewAccountHandler(services.AccountService, log),
		Transfer:  NewTransferHandler(services.TransferService, log),
		Card:      NewCardHandler(services.CardService, log),
		Credit:    NewCreditHandler(services.CreditService, log),
		Analytics: NewAnalyticsHandler(services.AnalyticsService, log),
		Admin:     NewAdminHandler(services.AdminService, log),
	}
}
