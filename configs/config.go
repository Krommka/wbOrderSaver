package configs

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"
	"wb_l0/configs/loader"
)

type DBConfig struct {
	User           string        `validate:"required"`
	Password       string        `validate:"required"`
	Name           string        `validate:"required"`
	Host           string        `validate:"required"`
	Port           string        `validate:"required"`
	ConnectTimeout time.Duration `validate:"required"`
	Retries        int           `validate:"required"`
}

type RedisConfig struct {
	Host         string        `validate:"required"`
	DB           int           `validate:"required"`
	User         string        `validate:"required"`
	Password     string        `validate:"required"`
	MaxRetries   int           `validate:"required"`
	DialTimeout  time.Duration `validate:"required"`
	ReadTimeout  time.Duration `validate:"required"`
	WriteTimeout time.Duration `validate:"required"`
}

type Config struct {
	DB  DBConfig
	RD  RedisConfig
	Env string
}

func MustLoad(loader loader.ConfigLoader) *Config {
	env := flag.String("env", "dev", "Environment type")
	flag.Parse()

	const op = "configs.MustLoad"
	envs, err := loader.Load()
	if err != nil {
		log.Fatalf("%s: config load failed: %+v", op, err)
	}
	cfg := &Config{
		DB: DBConfig{
			User:           envs["POSTGRES_USER"],
			Password:       envs["POSTGRES_PASSWORD"],
			Name:           envs["POSTGRES_DB"],
			Host:           envs["POSTGRES_HOST"],
			Port:           envs["POSTGRES_PORT"],
			ConnectTimeout: getEnvAsDuration(envs["POSTGRES_CONNECT_TIMEOUT"], 5*time.Second),
			Retries:        getEnvAsInt("POSTGRES_RETRIES", 1),
		},
		RD: RedisConfig{
			Host:         envs["REDIS_HOST"],
			DB:           getEnvAsInt(envs["REDIS_DB"], 0),
			User:         envs["REDIS_USER"],
			Password:     envs["REDIS_PASSWORD"],
			MaxRetries:   getEnvAsInt(envs["REDIS_MAX_RETRIES"], 3),
			DialTimeout:  getEnvAsDuration(envs["REDIS_DIAL_TIMEOUT"], 5*time.Second),
			ReadTimeout:  getEnvAsDuration(envs["REDIS_READ_TIMEOUT"], 5*time.Second),
			WriteTimeout: getEnvAsDuration(envs["REDIS_WRITE_TIMEOUT"], 5*time.Second),
		},
		Env: *env,
	}

	if err := validateConfig(cfg); err != nil {
		log.Fatalf("%s: ошибка валидации конфига: %+v", op, err)
	}

	return cfg
}

func validateConfig(cfg *Config) error {
	if cfg.DB.User == "" || cfg.DB.Password == "" || cfg.DB.Name == "" ||
		cfg.DB.Host == "" || cfg.DB.Port == "" || cfg.DB.Retries <= 0 {
		return fmt.Errorf("отсутствуют необходимые поля конфигурации базы данных")
	}
	if cfg.RD.Host == "" || cfg.RD.DialTimeout <= 0 || cfg.RD.ReadTimeout <= 0 || cfg.RD.
		WriteTimeout <= 0 {
		return fmt.Errorf("отсутствуют необходимые поля конфигурации кэш хранилища")
	}
	return nil
}

func getEnvAsDuration(strValue string, defaultValue time.Duration) time.Duration {
	const op = "configs.getEnvAsDuration"
	if strValue == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(strValue)
	if err != nil {
		log.Printf("%s:недопустимое значение для %s, использовано по умолчанию: %v", op,
			strValue, defaultValue)
		return defaultValue
	}
	return value
}

func getEnvAsInt(strValue string, defaultValue int) int {
	const op = "configs.getEnvAsInt"
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(strValue)
	if err != nil {
		log.Printf("%s:недопустимое значение для %s, использовано по умолчанию: %v", op, strValue,
			defaultValue)
		return defaultValue
	}
	return value
}
