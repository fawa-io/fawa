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
	"log"
	"os"
)

var logger Logger = &defaultLogger{
	level:  LevelInfo,
	stdlog: log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
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

// Notice calls the default logger's Notice method.
func Notice(v ...any) {
	logger.Notice(v...)
}

// Info calls the default logger's Info method.
func Info(v ...any) {
	logger.Info(v...)
}

// Debug calls the default logger's Debug method.
func Debug(v ...any) {
	logger.Debug(v...)
}

// Trace calls the default logger's Trace method.
func Trace(v ...any) {
	logger.Trace(v...)
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

// Noticef calls the default logger's Noticef method.
func Noticef(format string, v ...any) {
	logger.Noticef(format, v...)
}

// Infof calls the default logger's Infof method.
func Infof(format string, v ...any) {
	logger.Infof(format, v...)
}

// Debugf calls the default logger's Debugf method.
func Debugf(format string, v ...any) {
	logger.Debugf(format, v...)
}

// Tracef calls the default logger's Tracef method.
func Tracef(format string, v ...any) {
	logger.Tracef(format, v...)
}

type defaultLogger struct {
	stdlog *log.Logger
	level  Level
}

func (dl *defaultLogger) SetOutput(w io.Writer) {
	dl.stdlog.SetOutput(w)
}

func (dl *defaultLogger) SetLevel(lv Level) {
	dl.level = lv
}

func (dl *defaultLogger) logf(lv Level, format *string, v ...any) {
	if dl.level > lv {
		return
	}
	msg := lv.toString()
	if format != nil {
		msg += fmt.Sprintf(*format, v...)
	} else {
		msg += fmt.Sprint(v...)
	}
	dl.stdlog.Output(4, msg) // nolint:errcheck
	if lv == LevelFatal {
		os.Exit(1)
	}
}

func (dl *defaultLogger) Fatal(v ...any) {
	dl.logf(LevelFatal, nil, v...)
}

func (dl *defaultLogger) Error(v ...any) {
	dl.logf(LevelError, nil, v...)
}

func (dl *defaultLogger) Warn(v ...any) {
	dl.logf(LevelWarn, nil, v...)
}

func (dl *defaultLogger) Notice(v ...any) {
	dl.logf(LevelNotice, nil, v...)
}

func (dl *defaultLogger) Info(v ...any) {
	dl.logf(LevelInfo, nil, v...)
}

func (dl *defaultLogger) Debug(v ...any) {
	dl.logf(LevelDebug, nil, v...)
}

func (dl *defaultLogger) Trace(v ...any) {
	dl.logf(LevelTrace, nil, v...)
}

func (dl *defaultLogger) Fatalf(format string, v ...any) {
	dl.logf(LevelFatal, &format, v...)
}

func (dl *defaultLogger) Errorf(format string, v ...any) {
	dl.logf(LevelError, &format, v...)
}

func (dl *defaultLogger) Warnf(format string, v ...any) {
	dl.logf(LevelWarn, &format, v...)
}

func (dl *defaultLogger) Noticef(format string, v ...any) {
	dl.logf(LevelNotice, &format, v...)
}

func (dl *defaultLogger) Infof(format string, v ...any) {
	dl.logf(LevelInfo, &format, v...)
}

func (dl *defaultLogger) Debugf(format string, v ...any) {
	dl.logf(LevelDebug, &format, v...)
}

func (dl *defaultLogger) Tracef(format string, v ...any) {
	dl.logf(LevelTrace, &format, v...)
}
