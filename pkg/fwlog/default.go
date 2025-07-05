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
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger Logger

func init() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	atomicLevel := zap.NewAtomicLevelAt(zap.InfoLevel)

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config.EncoderConfig),
		zapcore.Lock(os.Stdout),
		atomicLevel,
	)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	logger = &zapLogger{
		l:      l,
		s:      l.Sugar(),
		level:  atomicLevel,
		config: &config,
	}
}

// SetOutput sets the output of default logger. By default, it is stderr.
func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}

// SetLevel sets the level of logs below which logs will not be output.
// The default log level is LevelTrace.
// Note that this method is not concurrent-safe.
func SetLevel(lv Level) {
	logger.SetLevel(lv)
}

// DefaultLogger return the default logger for kitex.
func DefaultLogger() Logger {
	return logger
}

// SetLogger sets the default logger.
// Note that this method is not concurrent-safe and must not be called
// after the use of DefaultLogger and global functions in this package.
func SetLogger(v Logger) {
	logger = v
}

// Fatal calls the default logger's Fatal method and then os.Exit(1).
func Fatal(v ...any) {
	logger.Fatal(v...)
}

// Error calls the default logger's Error method.
func Error(v ...any) {
	logger.Error(v...)
}

// Warn calls the default logger's Warn method.
func Warn(v ...any) {
	logger.Warn(v...)
}

// Info calls the default logger's Info method.
func Info(v ...any) {
	logger.Info(v...)
}

// Debug calls the default logger's Debug method.
func Debug(v ...any) {
	logger.Debug(v...)
}

// Fatalf calls the default logger's Fatalf method and then os.Exit(1).
func Fatalf(format string, v ...any) {
	logger.Fatalf(format, v...)
}

// Errorf calls the default logger's Errorf method.
func Errorf(format string, v ...any) {
	logger.Errorf(format, v...)
}

// Warnf calls the default logger's Warnf method.
func Warnf(format string, v ...any) {
	logger.Warnf(format, v...)
}

// Infof calls the default logger's Infof method.
func Infof(format string, v ...any) {
	logger.Infof(format, v...)
}

// Debugf calls the default logger's Debugf method.
func Debugf(format string, v ...any) {
	logger.Debugf(format, v...)
}

type zapLogger struct {
	l      *zap.Logger
	s      *zap.SugaredLogger
	level  zap.AtomicLevel
	config *zap.Config
}

func (l *zapLogger) SetOutput(w io.Writer) {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(l.config.EncoderConfig),
		zapcore.AddSync(w),
		l.level,
	)
	l.l = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	l.s = l.l.Sugar()
}

func (l *zapLogger) SetLevel(lv Level) {
	l.level.SetLevel(lv.toZapLevel())
}

func (l *zapLogger) Fatal(v ...any) {
	l.s.Fatal(v...)
}

func (l *zapLogger) Error(v ...any) {
	l.s.Error(v...)
}

func (l *zapLogger) Warn(v ...any) {
	l.s.Warn(v...)
}

func (l *zapLogger) Info(v ...any) {
	l.s.Info(v...)
}

func (l *zapLogger) Debug(v ...any) {
	l.s.Debug(v...)
}

func (l *zapLogger) Fatalf(format string, v ...any) {
	l.s.Fatalf(format, v...)
}

func (l *zapLogger) Errorf(format string, v ...any) {
	l.s.Errorf(format, v...)
}

func (l *zapLogger) Warnf(format string, v ...any) {
	l.s.Warnf(format, v...)
}

func (l *zapLogger) Infof(format string, v ...any) {
	l.s.Infof(format, v...)
}

func (l *zapLogger) Debugf(format string, v ...any) {
	l.s.Debugf(format, v...)
}
