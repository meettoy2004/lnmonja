package utils

import (
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger(config LogConfig) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if config.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create writers based on output config
	var writers []zapcore.WriteSyncer
	
	if config.Output == "stdout" || config.Output == "both" {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}
	
	if config.Output == "file" || config.Output == "both" {
		if config.Path == "" {
			config.Path = "lnmonja.log"
		}
		
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(config.Path), 0755); err != nil {
			return nil, err
		}
		
		var writer io.Writer
		if config.Rotation.Enabled {
			writer = &lumberjack.Logger{
				Filename:   config.Path,
				MaxSize:    config.Rotation.MaxSizeMB,
				MaxAge:     config.Rotation.MaxAgeDays,
				MaxBackups: config.Rotation.MaxBackups,
				Compress:   config.Rotation.Compress,
			}
		} else {
			file, err := os.OpenFile(config.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, err
			}
			writer = file
		}
		
		writers = append(writers, zapcore.AddSync(writer))
	}
	
	// Combine writers
	writeSyncer := zapcore.NewMultiWriteSyncer(writers...)
	
	core := zapcore.NewCore(encoder, writeSyncer, level)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	
	return logger, nil
}

func NewDevelopmentLogger() (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	return config.Build()
}

// SugarLogger returns a sugared logger for convenience
func SugarLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}