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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestLogger(t *testing.T) {
	buffer := new(bytes.Buffer)
	SetOutput(buffer)
	SetLevel(LevelDebug)

	Debug("debug")
	assert.True(t, strings.Contains(buffer.String(), "debug"))
	buffer.Reset()

	Info("info")
	assert.True(t, strings.Contains(buffer.String(), "info"))
	buffer.Reset()

	Warn("warn")
	assert.True(t, strings.Contains(buffer.String(), "warn"))
	buffer.Reset()

	Error("error")
	assert.True(t, strings.Contains(buffer.String(), "error"))
	buffer.Reset()

	SetLevel(LevelInfo)
	Debug("debug")
	assert.False(t, strings.Contains(buffer.String(), "debug"))
	buffer.Reset()
}

func TestFileLogger(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-logs")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	logFilePath := filepath.Join(tmpDir, "test.log")
	file, err := os.Create(logFilePath)
	assert.Nil(t, err)

	SetOutput(file)
	SetLevel(LevelInfo)

	Info("this is a test log")

	file.Close() //nolint:errcheck

	logContent, err := os.ReadFile(logFilePath)
	assert.Nil(t, err)
	assert.Contains(t, string(logContent), "this is a test log")
}

func TestDefaultLogger(t *testing.T) {
	assert.NotNil(t, DefaultLogger())
}

func TestLevel_toZapLevel(t *testing.T) {
	assert.Equal(t, zapcore.DebugLevel, LevelDebug.toZapLevel())
	assert.Equal(t, zapcore.InfoLevel, LevelInfo.toZapLevel())
	assert.Equal(t, zapcore.WarnLevel, LevelWarn.toZapLevel())
	assert.Equal(t, zapcore.ErrorLevel, LevelError.toZapLevel())
	assert.Equal(t, zapcore.FatalLevel, LevelFatal.toZapLevel())
	assert.Equal(t, zapcore.InfoLevel, Level(99).toZapLevel())
}

func TestFormatLogger(t *testing.T) {
	buffer := new(bytes.Buffer)
	SetOutput(buffer)
	SetLevel(LevelDebug)

	Debugf("debug %s", "message")
	assert.True(t, strings.Contains(buffer.String(), "debug message"))
	buffer.Reset()

	Infof("info %s", "message")
	assert.True(t, strings.Contains(buffer.String(), "info message"))
	buffer.Reset()

	Warnf("warn %s", "message")
	assert.True(t, strings.Contains(buffer.String(), "warn message"))
	buffer.Reset()

	Errorf("error %s", "message")
	assert.True(t, strings.Contains(buffer.String(), "error message"))
	buffer.Reset()
}
