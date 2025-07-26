// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Copyright 2025 The fawa Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fwlog

import (
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a logger interface that output logs.
type Logger interface {
	Debugf(format string, v ...any)
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
	Fatalf(format string, v ...any)

	Debug(v ...any)
	Info(v ...any)
	Warn(v ...any)
	Error(v ...any)
	Fatal(v ...any)

	SetLevel(Level)
	SetOutput(io.Writer)
}

// Level defines the priority of a log message.
// When a logger is configured with a level, any log message with a lower
// log level (smaller by integer comparison) will not be output.
type Level int

// The levels of logs.
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (lv Level) toZapLevel() zapcore.Level {
	switch lv {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	case LevelFatal:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func ParseLevel(levelStr string) (Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	}

	return LevelInfo, fmt.Errorf("invalid log level: '%s'", levelStr)
}

var defaultLogger Logger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}

	defaultLogger = &zapLogger{logger: logger.Sugar()}
}

type zapLogger struct {
	logger *zap.SugaredLogger
}

func (l *zapLogger) Debugf(format string, v ...any) {
	l.logger.Debugf(format, v...)
}

func (l *zapLogger) Infof(format string, v ...any) {
	l.logger.Infof(format, v...)
}

func (l *zapLogger) Warnf(format string, v ...any) {
	l.logger.Warnf(format, v...)
}

func (l *zapLogger) Errorf(format string, v ...any) {
	l.logger.Errorf(format, v...)
}

func (l *zapLogger) Fatalf(format string, v ...any) {
	l.logger.Fatalf(format, v...)
}

func (l *zapLogger) Debug(v ...any) {
	l.logger.Debug(v...)
}

func (l *zapLogger) Info(v ...any) {
	l.logger.Info(v...)
}

func (l *zapLogger) Warn(v ...any) {
	l.logger.Warn(v...)
}

func (l *zapLogger) Error(v ...any) {
	l.logger.Error(v...)
}

func (l *zapLogger) Fatal(v ...any) {
	l.logger.Fatal(v...)
}

func (l *zapLogger) SetLevel(level Level) {
	// For zap logger, we need to recreate the logger with new level
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level.toZapLevel())
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}

	l.logger = logger.Sugar()
}

func (l *zapLogger) SetOutput(w io.Writer) {
	// For zap logger, we need to recreate the logger with new output
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}

	l.logger = logger.Sugar()
}

// Global functions
func Debugf(format string, v ...any) {
	defaultLogger.Debugf(format, v...)
}

func Infof(format string, v ...any) {
	defaultLogger.Infof(format, v...)
}

func Warnf(format string, v ...any) {
	defaultLogger.Warnf(format, v...)
}

func Errorf(format string, v ...any) {
	defaultLogger.Errorf(format, v...)
}

func Fatalf(format string, v ...any) {
	defaultLogger.Fatalf(format, v...)
}

func Debug(v ...any) {
	defaultLogger.Debug(v...)
}

func Info(v ...any) {
	defaultLogger.Info(v...)
}

func Warn(v ...any) {
	defaultLogger.Warn(v...)
}

func Error(v ...any) {
	defaultLogger.Error(v...)
}

func Fatal(v ...any) {
	defaultLogger.Fatal(v...)
}

func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}
