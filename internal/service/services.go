package service

import (
	"bank-api/internal/repository"
	"bank-api/internal/security"
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
)

// Services хранит все сервисы приложения в одном наборе
type Services struct {
	AuthService         AuthService
	AccountService      AccountService
	TransferService     TransferService
	CardService         CardService
	CreditService       CreditService
	AnalyticsService    AnalyticsService
	AdminService        AdminService
	NotificationService *NotificationService
}

// NewServices собирает все сервисы приложения в одном наборе
func NewServices(
	repos *repository.Repositories,
	pgp *security.PGPService,
	hmacSecret []byte,
	jwtSecret []byte,
	jwtTTL time.Duration,
	notification *NotificationService,
	cbr CBRService,
	log *logrus.Logger,
	db *sql.DB,
) *Services {
	return &Services{
		AuthService:         NewAuthService(repos, jwtSecret, jwtTTL),
		AccountService:      NewAccountService(repos, db),
		TransferService:     NewTransferService(repos, db, notification),
		CardService:         NewCardService(repos, pgp, hmacSecret, db, notification),
		CreditService:       NewCreditService(repos, cbr, db, log, notification),
		AnalyticsService:    NewAnalyticsService(repos),
		AdminService:        NewAdminService(repos),
		NotificationService: notification,
	}
}
