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
	"testing"

	"github.com/stretchr/testify/assert"
)

// test package level functions without format
func normalOutput(t *testing.T, testLevel Level, want string, args ...any) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	defer SetOutput(os.Stderr)
	switch testLevel {
	case LevelTrace:
		Trace(args...)
		assert.Equal(t, buf.String(), want)
	case LevelDebug:
		Debug(args...)
		assert.Equal(t, buf.String(), want)
	case LevelInfo:
		Info(args...)
		assert.Equal(t, buf.String(), want)
	case LevelNotice:
		Notice(args...)
		assert.Equal(t, buf.String(), want)
	case LevelWarn:
		Warn(args...)
		assert.Equal(t, buf.String(), want)
	case LevelError:
		Error(args...)
		assert.Equal(t, buf.String(), want)
	case LevelFatal:
		t.Fatal("fatal method cannot be tested")
	default:
		t.Errorf("unknown level: %d", testLevel)
	}
}

// test package level functions with 'format'
func formatOutput(t *testing.T, testLevel Level, want, format string, args ...any) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	defer SetOutput(os.Stderr)
	switch testLevel {
	case LevelTrace:
		Tracef(format, args...)
		assert.Equal(t, buf.String(), want)
	case LevelDebug:
		Debugf(format, args...)
		assert.Equal(t, buf.String(), want)
	case LevelInfo:
		Infof(format, args...)
		assert.Equal(t, buf.String(), want)
	case LevelNotice:
		Noticef(format, args...)
		assert.Equal(t, buf.String(), want)
	case LevelWarn:
		Warnf(format, args...)
		assert.Equal(t, buf.String(), want)
	case LevelError:
		Errorf(format, args...)
		assert.Equal(t, buf.String(), want)
	case LevelFatal:
		t.Fatal("fatal method cannot be tested")
	default:
		t.Errorf("unknown level: %d", testLevel)
	}
}

func TestOutput(t *testing.T) {
	l := DefaultLogger().(*defaultLogger)
	oldFlags := l.stdlog.Flags()
	l.stdlog.SetFlags(0)
	defer l.stdlog.SetFlags(oldFlags)
	defer SetLevel(LevelInfo)

	tests := []struct {
		format      string
		args        []any
		testLevel   Level
		loggerLevel Level
		want        string
	}{
		{"%s", []any{"LevelNotice test"}, LevelNotice, LevelInfo, strs[LevelNotice] + "LevelNotice test\n"},
		{"%s %s", []any{"LevelInfo", "test"}, LevelInfo, LevelWarn, ""},
		{"%s%s", []any{"LevelDebug", "Test"}, LevelDebug, LevelDebug, strs[LevelDebug] + "LevelDebugTest\n"},
		{"%s", []any{"LevelTrace test"}, LevelTrace, LevelTrace, strs[LevelTrace] + "LevelTrace test\n"},
		{"%s", []any{"LevelError test"}, LevelError, LevelInfo, strs[LevelError] + "LevelError test\n"},
		{"%s", []any{"LevelWarn test"}, LevelWarn, LevelWarn, strs[LevelWarn] + "LevelWarn test\n"},
	}

	for _, tt := range tests {
		SetLevel(tt.loggerLevel)
		normalOutput(t, tt.testLevel, tt.want, tt.args...)
		formatOutput(t, tt.testLevel, tt.want, tt.format, tt.args...)
	}
}
