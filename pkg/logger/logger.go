package logger

import (
	"io"
	"log/slog"
	"os"
	"wb_l0/configs"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func NewLogger(cfg *configs.Config) *slog.Logger {

	var logger *slog.Logger

	logRotation := &lumberjack.Logger{
		Filename:   getLogPath(cfg.Env),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	switch cfg.Env {
	case envLocal:
		logger = slog.New(
			slog.NewJSONHandler(io.MultiWriter(os.Stdout, logRotation), &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
			}))
	case envDev:
		logger = slog.New(
			slog.NewJSONHandler(io.MultiWriter(os.Stdout, logRotation), &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
			}))
	case envProd:
		logger = slog.New(
			slog.NewJSONHandler(logRotation, &slog.HandlerOptions{ // Только в файл для prod
				Level:     slog.LevelInfo,
				AddSource: true,
			}))
	}

	return logger
}

func NewTestLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}))
}

func getLogPath(env string) string {
	switch env {
	case envLocal:
		return "logs/orderSaver.log"
	case envDev, envProd:
		return "/var/log/orderSaver.log"
	default:
		return "logs/orderSaver.log"
	}
}
