package helper

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
	config := zap.NewProductionConfig()
	// Agar log mudah dibaca di console saat development, kita bisa gunakan Encoder berwarna
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	var err error
	Log, err = config.Build()
	if err != nil {
		panic(err)
	}
}

const (
	LogPositionHandler    LogPosition = "handler"
	LogPositionRepo       LogPosition = "repo"
	LogPositionService    LogPosition = "service"
	LogPositionServer     LogPosition = "server"
	LogPositionMiddleware LogPosition = "middleware"
	LogPositionConfig     LogPosition = "config"
)
