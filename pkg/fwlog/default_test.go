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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// test package level functions without format
func normalOutput(testLevel Level, args ...any) {
	switch testLevel {
	case LevelDebug:
		Debug(args...)
	case LevelInfo:
		Info(args...)
	case LevelWarn:
		Warn(args...)
	case LevelError:
		Error(args...)
	case LevelFatal:
		// fatal method cannot be tested
	}
}

// test package level functions with 'format'
func formatOutput(testLevel Level, format string, args ...any) {
	switch testLevel {
	case LevelDebug:
		Debugf(format, args...)
	case LevelInfo:
		Infof(format, args...)
	case LevelWarn:
		Warnf(format, args...)
	case LevelError:
		Errorf(format, args...)
	case LevelFatal:
		// fatal method cannot be tested
	}
}

func TestOutput(t *testing.T) {
	// a buffer to capture log output
	buf := new(bytes.Buffer)
	SetOutput(buf)
	// restore stderr after test
	defer SetOutput(os.Stderr)
	// restore default log level
	defer SetLevel(LevelInfo)

	tests := []struct {
		format      string
		args        []any
		testLevel   Level
		loggerLevel Level
		wantContain string
	}{
		{
			"%s %s",
			[]any{"LevelInfo", "test"},
			LevelInfo,
			LevelWarn,
			"",
		},
		{
			"%s%s",
			[]any{"LevelDebug", "Test"},
			LevelDebug,
			LevelDebug,
			`"msg":"LevelDebugTest"`,
		},
		{
			"%s",
			[]any{"LevelError test"},
			LevelError,
			LevelInfo,
			`"msg":"LevelError test"`,
		},
		{
			"%s",
			[]any{"LevelWarn test"},
			LevelWarn,
			LevelWarn,
			`"msg":"LevelWarn test"`,
		},
	}

	for _, tt := range tests {
		SetLevel(tt.loggerLevel)

		// test normal output
		normalOutput(tt.testLevel, tt.args...)
		if tt.wantContain != "" {
			assert.True(t, strings.Contains(buf.String(), tt.wantContain), "[normal] want %q, got %q", tt.wantContain, buf.String())
		} else {
			assert.Equal(t, "", buf.String(), "[normal] want empty, got %q", buf.String())
		}
		buf.Reset()

		// test format output
		formatOutput(tt.testLevel, tt.format, tt.args...)
		if tt.wantContain != "" {
			assert.True(t, strings.Contains(buf.String(), tt.wantContain), "[format] want %q, got %q", tt.wantContain, buf.String())
		} else {
			assert.Equal(t, "", buf.String(), "[format] want empty, got %q", buf.String())
		}
		buf.Reset()
	}
}
