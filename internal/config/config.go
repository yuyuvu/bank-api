package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// SMTPConfig содержит настройки почтового сервера
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type Config struct {
	Server struct {
		Port         int           `mapstructure:"port"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
	} `mapstructure:"server"`
	DatabaseURL string `mapstructure:"database_url"`
	JWT         struct {
		Secret string        `mapstructure:"secret"`
		TTL    time.Duration `mapstructure:"ttl"`
	} `mapstructure:"jwt"`
	Encryption struct {
		PGPPublicKey  string `mapstructure:"pgp_public_key"`
		PGPPrivateKey string `mapstructure:"pgp_private_key"`
		PGPPassphrase string `mapstructure:"pgp_passphrase"`
		HMACSecret    string `mapstructure:"hmac_secret"`
	} `mapstructure:"encryption"`
	SMTP SMTPConfig `mapstructure:"smtp"`
	CBR  struct {
		Margin float64 `mapstructure:"margin"`
	} `mapstructure:"cbr"`
	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
}

// Load читает конфиг из файла и переменных окружения, отдавая приоритет окружению
func Load() *Config {
	v := viper.New()
	v.SetConfigFile("configs/config.yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.port", 8080)
	v.SetDefault("jwt.secret", "MWxj8Y0TnSu4haI3pZuyOyX4rLcsEFleGMn2tV07kbm")
	v.SetDefault("jwt.ttl", "24h")
	v.SetDefault("smtp.port", 1025)

	if err := v.ReadInConfig(); err != nil {
		// В Docker переменные окружения имеют приоритет, файл может отсутствовать
	}

	v.BindEnv("database_url", "DATABASE_URL")
	v.BindEnv("jwt.secret", "JWT_SECRET")
	v.BindEnv("encryption.pgp_passphrase", "PGP_PASSPHRASE")
	v.BindEnv("encryption.hmac_secret", "HMAC_SECRET")
	v.BindEnv("smtp.host", "SMTP_HOST")
	v.BindEnv("smtp.port", "SMTP_PORT")
	v.BindEnv("smtp.user", "SMTP_USER")
	v.BindEnv("smtp.password", "SMTP_PASSWORD")
	v.BindEnv("log.level", "LOG_LEVEL")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	return &cfg
}
