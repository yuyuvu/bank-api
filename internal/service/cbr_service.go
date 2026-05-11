package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beevik/etree"
	"github.com/sirupsen/logrus"
)

// CBRService отвечает за получение ключевой ставки ЦБ РФ
type CBRService interface {
	GetRate(ctx context.Context) (float64, error)
}

// cbrService отправляет запрос в SOAP API ЦБ РФ; при проблемах использует последнее успешно полученное значение
type cbrService struct {
	margin float64
	log    *logrus.Logger
	mu     sync.RWMutex
	last   float64
}

// NewCBRService создаёт сервис для работы с SOAP API Центрального банка и получения ключевой ставки ЦБ РФ
func NewCBRService(margin float64, log *logrus.Logger) CBRService {
	return &cbrService{margin: margin, log: log}
}

// GetRate запрашивает ключевую ставку у ЦБ и добавляет банковскую маржу (по умолчанию 2%)
func (s *cbrService) GetRate(ctx context.Context) (float64, error) {
	fromDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02") + "T00:00:00"
	toDate := time.Now().Format("2006-01-02") + "T00:00:00"

	soapRequest := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <KeyRate xmlns="http://web.cbr.ru/">
      <fromDate>%s</fromDate>
      <ToDate>%s</ToDate>
    </KeyRate>
  </soap:Body>
</soap:Envelope>`, fromDate, toDate)

	// s.log.Debugf("Итоговый XML: %s", soapRequest)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx",
		strings.NewReader(soapRequest))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/KeyRate")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return s.cachedOrErr(fmt.Errorf("ошибка запроса к ЦБ: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return s.cachedOrErr(fmt.Errorf("ЦБ вернул статус %d: %s", resp.StatusCode, string(body)))
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		return s.cachedOrErr(fmt.Errorf("ошибка парсинга XML: %w", err))
	}

	// Ищем <KR> внутри <KeyRate>
	elements := doc.FindElements(".//KeyRate/KR")
	if len(elements) == 0 {
		return s.cachedOrErr(fmt.Errorf("данные по ставке не найдены"))
	}

	// Первый элемент с самой последней датой (rowOrder = 0)
	first := elements[0]
	rateElem := first.FindElement("./Rate")
	if rateElem == nil {
		return s.cachedOrErr(fmt.Errorf("тег Rate отсутствует"))
	}

	rateValue := strings.TrimSpace(strings.ReplaceAll(rateElem.Text(), ",", "."))
	rate, err := strconv.ParseFloat(rateValue, 64)
	if err != nil {
		return s.cachedOrErr(fmt.Errorf("ошибка конвертации ставки: %w", err))
	}

	rate += s.margin

	s.mu.Lock()
	s.last = rate
	s.mu.Unlock()

	return rate, nil
}

// cachedOrErr возвращает последнее успешно полученное значение ставки, если API ЦБ временно недоступен
func (s *cbrService) cachedOrErr(err error) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.last > 0 {
		s.log.Warnf("Не удалось получить ставку ЦБ, используется последнее значение %.2f: %v", s.last, err)
		return s.last, nil
	}

	return 0, err
}
