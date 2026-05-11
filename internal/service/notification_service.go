package service

import (
	"bank-api/internal/config"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
)

// NotificationService отвечает за отправку email-уведомлений
type NotificationService struct {
	cfg config.SMTPConfig
	log *logrus.Logger
}

// NewNotificationService создаёт сервис отправки уведомлений по email
func NewNotificationService(cfg config.SMTPConfig, log *logrus.Logger) *NotificationService {
	return &NotificationService{cfg: cfg, log: log}
}

// sendEmail отправляет уведомление по email
func (ns *NotificationService) sendEmail(to, subject, htmlBody string) error {
	if ns.cfg.Host == "" {
		return fmt.Errorf("smtp host не настроен")
	}

	from := ns.cfg.User
	if from == "" {
		from = "noreply@bank.local"
	}

	m := mail.NewMsg()
	if err := m.From(from); err != nil {
		return fmt.Errorf("from: %w", err)
	}
	if err := m.To(to); err != nil {
		return fmt.Errorf("to: %w", err)
	}
	m.Subject(subject)
	m.SetBodyString("text/html", htmlBody)

	client, err := mail.NewClient(ns.cfg.Host,
		mail.WithPort(ns.cfg.Port),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		ns.log.Errorf("Ошибка создания SMTP клиента: %v", err)
		return err
	}
	if err := client.DialAndSend(m); err != nil {
		ns.log.Errorf("Ошибка отправки email: %v", err)
		return err
	}
	ns.log.Infof("Email отправлен на %s", to)
	return nil
}

func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

func formatDateTime(t time.Time) string {
	loc, _ := time.LoadLocation("Europe/Moscow")
	return t.In(loc).Format("2006-01-02 15:04:05")
}

// SendCardPaymentEmail отправляет письмо об оплате банковской картой
func (ns *NotificationService) SendCardPaymentEmail(to string, when time.Time, amount float64, description string, accountID int64, transactionID int64) error {
	description = strings.TrimSpace(description)
	if description == "" {
		description = "Оплата банковской картой"
	}

	return ns.sendEmail(
		to,
		"[Банк] Выполнена оплата картой, операция #"+fmt.Sprintf("%d", transactionID),
		fmt.Sprintf(`
			<h2>Оплата картой</h2>
			<p><strong>Дата:</strong> %s</p>
			<p><strong>Сумма:</strong> %s RUB</p>
			<p><strong>Описание:</strong> %s</p>
			<p><strong>Счёт списания:</strong> %d</p>
			<p><strong>ID операции:</strong> %d</p>
			<small>Это автоматическое уведомление</small>
		`, formatDateTime(when), formatMoney(amount), description, accountID, transactionID),
	)
}

// SendTransferEmail отправляет письмо об исходящем переводе
func (ns *NotificationService) SendTransferEmail(to string, when time.Time, amount float64, toUserEmail string, toAccountID int64, fromAccountID int64, transactionID int64) error {
	return ns.sendEmail(
		to,
		fmt.Sprintf("[Банк] Выполнен перевод со счёта #%d, операция #%d", fromAccountID, transactionID),
		fmt.Sprintf(`
			<h2>Перевод</h2>
			<p><strong>Дата:</strong> %s</p>
			<p><strong>Сумма:</strong> %s RUB</p>
			<p><strong>Кому:</strong> %s</p>
			<p><strong>Счёт списания:</strong> %d</p>
			<p><strong>Счёт зачисления:</strong> %d</p>
			<p><strong>ID операции:</strong> %d</p>
			<small>Это автоматическое уведомление</small>
		`, formatDateTime(when), formatMoney(amount), toUserEmail, fromAccountID, toAccountID, transactionID),
	)
}

// SendCreditPaymentEmail отправляет письмо о платеже по кредиту
func (ns *NotificationService) SendCreditPaymentEmail(to string, when time.Time, creditID int64, dueDate time.Time, baseAmount float64, penaltyAmount float64, totalAmount float64, fromAccountID int64, transactionID int64, isAuto bool) error {
	paymentType := "Ручной платёж"
	if isAuto {
		paymentType = "Автоматическое списание"
	}

	return ns.sendEmail(
		to,
		"[Банк] Выполнен платёж по кредиту, операция #"+fmt.Sprintf("%d", transactionID),
		fmt.Sprintf(`
			<h2>Платёж по кредиту</h2>
			<p><strong>Тип:</strong> %s</p>
			<p><strong>Дата списания:</strong> %s</p>
			<p><strong>Кредит:</strong> %d</p>
			<p><strong>Платёж по графику на дату:</strong> %s</p>
			<p><strong>Сумма платежа:</strong> %s RUB</p>
			<p><strong>Штраф:</strong> %s RUB</p>
			<p><strong>Итого списано:</strong> %s RUB</p>
			<p><strong>Счёт списания:</strong> %d</p>
			<p><strong>ID операции:</strong> %d</p>
			<small>Это автоматическое уведомление</small>
		`, paymentType, formatDateTime(when), creditID, dueDate.Format("2006-01-02"), formatMoney(baseAmount), formatMoney(penaltyAmount), formatMoney(totalAmount), fromAccountID, transactionID),
	)
}
