package logger

import (
	"KinopoiskTwoActors/configs"
	"io"
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func NewLogger(cfg *configs.Config) *slog.Logger {

	var logger *slog.Logger

	switch cfg.Env {
	case envLocal:
		multiWriter, err := newMultiWriter("logs/bot.log")
		logger = slog.New(
			slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
			}))
		if err != nil {
			logger.Error("Error creating log file: ", "error", err)
		}
	case envDev:
		multiWriter, err := newMultiWriter("/var/log/telegram-bot.log")
		logger = slog.New(
			slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
			}))
		if err != nil {
			logger.Error("Error creating log file: ", err)
		}
	case envProd:
		multiWriter, err := newMultiWriter("/var/log/telegram-bot.log")
		logger = slog.New(
			slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: true,
			}))
		if err != nil {
			logger.Error("Error creating log file: ", err)
		}
	}

	return logger
}

func newMultiWriter(path string) (io.Writer, error) {
	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		return os.Stdout, err
	}
	return io.MultiWriter(os.Stdout, logFile), nil
}
