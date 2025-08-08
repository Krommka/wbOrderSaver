package logrus

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	logger.SetOutput(&lumberjack.Logger{
		Filename:   "/var/log/telegram-bot.json",
		MaxSize:    100, // MB
		MaxBackups: 3,
	})
	return logger
}
