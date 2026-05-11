package scheduler

import (
	"bank-api/internal/service"
	"context"

	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"
)

// StartScheduler запускает фоновый планировщик для обработки просроченных платежей каждые 12 часов
func StartScheduler(creditService service.CreditService, log *logrus.Logger) (gocron.Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	_, err = s.NewJob(
		gocron.CronJob("0 */12 * * *", true),
		gocron.NewTask(
			func() {
				ctx := context.Background()
				if err := creditService.ProcessOverduePayments(ctx); err != nil {
					log.Errorf("Ошибка фонового планировщика для обработки просроченных платежей: %v", err)
				}
			},
		),
	)
	if err != nil {
		return nil, err
	}
	s.Start()
	log.Info("Фоновый планировщик для обработки просроченных платежей запущен")
	return s, nil
}
