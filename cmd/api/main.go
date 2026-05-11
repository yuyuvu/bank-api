package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"bank-api/internal/config"
	"bank-api/internal/handler"
	"bank-api/internal/middleware"
	"bank-api/internal/repository"
	"bank-api/internal/scheduler"
	"bank-api/internal/security"
	"bank-api/internal/service"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func main() {
	// Инициализация конфигурации
	cfg := config.Load()

	// Настройка логгера
	log := logrus.New()
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		logrus.Fatalf("Неверный уровень логирования: %v", err)
	}
	log.SetLevel(level)
	log.SetFormatter(&logrus.JSONFormatter{})

	// Подключение к базе данных
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверка соединения
	if err := db.Ping(); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}
	log.Info("Подключение к PostgreSQL установлено")

	// Применение миграций через SQL-файл
	if err := applyMigrations(db); err != nil {
		log.Fatalf("Ошибка миграций: %v", err)
	}

	// Инициализация PGP и HMAC
	pgpService, err := initPGPService(cfg, log)
	if err != nil {
		log.Fatalf("Ошибка инициализации PGP: %v", err)
	}
	hmacSecret := []byte(cfg.Encryption.HMACSecret)

	// Инициализация репозиториев
	repos := repository.NewRepositories(db)

	// Инициализация сервисов
	notificationService := service.NewNotificationService(cfg.SMTP, log)
	cbrService := service.NewCBRService(cfg.CBR.Margin, log)

	services := service.NewServices(
		repos,
		pgpService,
		hmacSecret,
		[]byte(cfg.JWT.Secret),
		cfg.JWT.TTL,
		notificationService,
		cbrService,
		log,
		db,
	)

	// Инициализация обработчиков
	handlers := handler.NewHandlers(services, log)

	// Инициализация middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret, repos.User)
	loggingMiddleware := middleware.NewLoggingMiddleware(log)
	recoveryMiddleware := middleware.NewRecoveryMiddleware(log)

	// Настройка роутера
	router := handler.SetupRouter(handlers, authMiddleware, loggingMiddleware, recoveryMiddleware)

	// Запуск фонового планировщика для обработки просроченных платежей
	sched, err := scheduler.StartScheduler(services.CreditService, log)
	if err != nil {
		log.Fatalf("Ошибка запуска фонового планировщика для обработки просроченных платежей: %v", err)
	}
	defer sched.Shutdown()

	// HTTP-сервер
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Запуск и выключение
	go func() {
		log.Infof("Сервер запущен на порте %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Выключение сервера...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка выключения: %v", err)
	}
	log.Info("Сервер остановлен")
}

// applyMigrations находит файл миграции и применяет его к базе
func applyMigrations(db *sql.DB) error {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("не удалось определить путь к main.go")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "001_init.up.sql")
	data, err := os.ReadFile(filepath.Clean(migrationPath))
	if err != nil {
		return fmt.Errorf("не удалось прочитать миграцию: %w", err)
	}
	_, err = db.Exec(string(data))
	return err
}

// initPGPService запускает PGP-сервис и локально сохраняет пару ключей, если ключи явно не заданы в конфиге
func initPGPService(cfg *config.Config, log *logrus.Logger) (*security.PGPService, error) {
	pubKey := cfg.Encryption.PGPPublicKey
	privKey := cfg.Encryption.PGPPrivateKey

	if (pubKey == "") != (privKey == "") {
		return nil, fmt.Errorf("для PGP нужно передать и публичный, и приватный ключ одновременно")
	}

	if pubKey == "" && privKey == "" {
		publicPath, privatePath, err := pgpKeyPaths()
		if err != nil {
			return nil, err
		}

		pubKey, privKey, err = readStoredPGPKeys(publicPath, privatePath)
		if err != nil {
			return nil, err
		}

		pgpService, err := security.NewPGPService(pubKey, privKey, cfg.Encryption.PGPPassphrase)
		if err != nil {
			return nil, err
		}

		if pubKey == "" && privKey == "" {
			if err := storePGPKeys(publicPath, privatePath, pgpService.PublicKeyArmored(), pgpService.PrivateKeyArmored()); err != nil {
				return nil, err
			}
			log.Infof("PGP-ключи сгенерированы и сохранены в %s и %s", publicPath, privatePath)
		}

		return pgpService, nil
	}

	return security.NewPGPService(pubKey, privKey, cfg.Encryption.PGPPassphrase)
}

// pgpKeyPaths возвращает пути, где локально хранятся PGP-ключи для карт
func pgpKeyPaths() (string, string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", "", fmt.Errorf("не удалось определить путь к main.go")
	}

	baseDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "configs", "keys")
	return filepath.Join(baseDir, "pgp_public.asc"), filepath.Join(baseDir, "pgp_private.asc"), nil
}

// readStoredPGPKeys читает локально сохранённые ключи, если они уже были созданы раньше.
func readStoredPGPKeys(publicPath, privatePath string) (string, string, error) {
	publicData, publicExists, err := readFileIfExists(publicPath)
	if err != nil {
		return "", "", err
	}
	privateData, privateExists, err := readFileIfExists(privatePath)
	if err != nil {
		return "", "", err
	}

	if !publicExists && !privateExists {
		return "", "", nil
	}
	if publicExists != privateExists {
		return "", "", fmt.Errorf("локальная пара ключей PGP повреждена: найден только один из ключей")
	}

	return publicData, privateData, nil
}

// readFileIfExists возвращает содержимое файла и факт его существования
func readFileIfExists(path string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	return strings.TrimSpace(string(data)), true, nil
}

// storePGPKeys сохраняет новую пару ключей локально, чтобы карты продолжали считываться после перезапуска
func storePGPKeys(publicPath, privatePath, publicKey, privateKey string) error {
	if err := os.MkdirAll(filepath.Dir(publicPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Clean(publicPath), []byte(publicKey), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Clean(privatePath), []byte(privateKey), 0o600); err != nil {
		return err
	}

	return nil
}
