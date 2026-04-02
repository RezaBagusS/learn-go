package helper

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
	// Encoder config — menentukan format JSON output
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		CallerKey:      "caller",
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // "2024-01-15T14:00:00.000Z"
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // "info", "error"
		EncodeCaller:   zapcore.ShortCallerEncoder,    // "handler/banks.go:45"
		EncodeDuration: zapcore.MillisDurationEncoder,
	}

	// Core — gabungan encoder + output + level
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg), // ← JSON format
		zapcore.NewMultiWriteSyncer( // ← output ke beberapa tempat
			zapcore.AddSync(os.Stdout), // terminal
			// zapcore.AddSync(fileWriter), // bisa tambah file nanti untuk Promtail
		),
		zap.InfoLevel, // minimum level yang ditulis
	)

	Log = zap.New(core, zap.AddCaller()) // AddCaller → tau dari file mana log dipanggil
}

const (
	LogPositionHandler    LogPosition = "handler"
	LogPositionRepo       LogPosition = "repo"
	LogPositionService    LogPosition = "service"
	LogPositionServer     LogPosition = "server"
	LogPositionMiddleware LogPosition = "middleware"
	LogPositionConfig     LogPosition = "config"
)
