package helper

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

type LogPosition string

const (
	LogPositionHandler    LogPosition = "handler"
	LogPositionRepo       LogPosition = "repo"
	LogPositionService    LogPosition = "service"
	LogPositionServer     LogPosition = "server"
	LogPositionMiddleware LogPosition = "middleware"
	LogPositionConfig     LogPosition = "config"
)

func InitLogger() {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		CallerKey:      "caller",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
		),
		zap.InfoLevel,
	)

	Log = zap.New(core, zap.AddCaller())
}

func NewAPIPath(method, version, path string) string {
	return fmt.Sprintf("%s /api/%s%s", method, version, path)
}

func PrintLog(domain string, position LogPosition, msg string) {
	fmt.Printf("[%s][%s] %s\n", domain, position, msg)
}
