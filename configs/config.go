package configs

import (
	"flag"
	"fmt"
	"log"
	"os"
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
	Capacity     int           `validate:"required"`
	WarmUp       bool          `validate:"required"`
}

type KafkaConfig struct {
	BootstrapServers     string `validate:"required"`
	AutoCommitIntervalMs int    `validate:"required"`
	AutoOffsetReset      string `validate:"required"`
	SessionTimeoutMs     int    `validate:"required"`
	Topic                string `validate:"required"`
	ConsumerGroup        string `validate:"required"`
	ProducerNumberOfKeys int    `validate:"required"`
	FlushTimeout         int    `validate:"required"`
}

type HttpConfig struct {
	Port         string        `validate:"required"`
	ReadTimeout  time.Duration `validate:"required"`
	WriteTimeout time.Duration `validate:"required"`
	IdleTimeout  time.Duration `validate:"required"`
}

type Config struct {
	DB   DBConfig
	RD   RedisConfig
	KF   KafkaConfig
	HTTP HttpConfig
	Env  string
}

func MustLoad(loader loader.ConfigLoader) *Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		envFlag := flag.String("env", "dev", "Environment type")
		flag.Parse()
		env = *envFlag
	}

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
			Retries:        getEnvAsInt(envs["POSTGRES_RETRIES"], 1),
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
			Capacity:     getEnvAsInt(envs["REDIS_CAPACITY"], 100),
			WarmUp:       getEnvAsBool(envs["REDIS_WARMUP"], false),
		},
		KF: KafkaConfig{
			BootstrapServers:     envs["KAFKA_BOOTSTRAP_SERVERS"],
			AutoCommitIntervalMs: getEnvAsInt(envs["KAFKA_AUTO_COMMIT_INTERVAL_MS"], 1000),
			AutoOffsetReset:      envs["KAFKA_AUTO_OFFSET_RESET"],
			SessionTimeoutMs:     getEnvAsInt(envs["KAFKA_SESSION_TIMEOUT_MS"], 1000),
			Topic:                envs["KAFKA_TOPIC"],
			ConsumerGroup:        envs["KAFKA_CONSUMER_GROUP"],
			ProducerNumberOfKeys: getEnvAsInt(envs["KAFKA_PRODUCER_NUM_OF_KEYS"], 20),
			FlushTimeout:         getEnvAsInt(envs["KAFKA_FLUSH_TIMEOUT"], 5000),
		},
		HTTP: HttpConfig{
			Port:         envs["HTTP_PORT"],
			ReadTimeout:  getEnvAsDuration(envs["HTTP_READ_TIMEOUT"], 10*time.Second),
			WriteTimeout: getEnvAsDuration(envs["HTTP_WRITE_TIMEOUT"], 10*time.Second),
			IdleTimeout:  getEnvAsDuration(envs["HTTP_WRITE_TIMEOUT"], 60*time.Second),
		},
		Env: env,
	}

	if err := validateConfig(cfg); err != nil {
		log.Fatalf("%s: error validation config: %+v", op, err)
	}

	return cfg
}

func validateConfig(cfg *Config) error {
	if cfg.DB.User == "" || cfg.DB.Password == "" || cfg.DB.Name == "" ||
		cfg.DB.Host == "" || cfg.DB.Port == "" || cfg.DB.Retries <= 0 || cfg.DB.ConnectTimeout <= 0*time.Second {
		return fmt.Errorf("incorrect database config fields")
	}

	if cfg.RD.Host == "" || cfg.RD.DialTimeout <= 0*time.Second || cfg.RD.ReadTimeout <= 0*time.Second || cfg.RD.
		WriteTimeout <= 0*time.Second || cfg.RD.Capacity <= 0 || cfg.RD.MaxRetries <= 0 {
		return fmt.Errorf("incorrect cache config fields")
	}

	if cfg.KF.BootstrapServers == "" || cfg.KF.AutoCommitIntervalMs <= 0 || cfg.KF.SessionTimeoutMs <= 0 ||
		cfg.KF.Topic == "" || cfg.KF.ConsumerGroup == "" || cfg.KF.AutoOffsetReset == "" ||
		cfg.KF.FlushTimeout <= 0 || cfg.KF.ProducerNumberOfKeys <= 0 {
		return fmt.Errorf("incorrect kafka config fields")
	}

	if cfg.HTTP.Port == "" || cfg.HTTP.ReadTimeout <= 0*time.Second || cfg.HTTP.WriteTimeout <= 0*time.Second ||
		cfg.HTTP.IdleTimeout <= 0*time.Second {
		return fmt.Errorf("incorrect http config fields")
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
		log.Printf("%s:forbidden value for %s, using default: %v", op,
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
		log.Printf("%s:forbidden value for %s, using default: %v", op, strValue,
			defaultValue)
		return defaultValue
	}
	return value
}

func getEnvAsBool(strValue string, defaultValue bool) bool {
	const op = "configs.getEnvAsBool"
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(strValue)
	if err != nil {
		log.Printf("%s:forbidden value for %s, using default: %v", op, strValue, defaultValue)
		return defaultValue
	}
	return value
}
