/**
 * KP - Restic Backup Wrapper
 *
 * Leveled logger writing to stderr with printf-style formatting.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package logger

import (
	"fmt"
	"os"
	"time"
)

// level names indexed by severity
const (
	levelDebug = iota
	levelInfo
	levelWarn
	levelError
)

// minLevel is the lowest severity that will be emitted.
// Defaults to info; raised to debug via SetDebug.
var minLevel = levelInfo

// SetDebug enables or disables debug-level output.
func SetDebug(on bool) {
	if on {
		minLevel = levelDebug
	} else {
		minLevel = levelInfo
	}
}

// emit writes a single formatted log line to stderr.
func emit(level int, name string, format string, args ...any) {

	// suppress below the configured threshold
	if level < minLevel {
		return
	}

	// timestamped, leveled, printf-formatted line
	fmt.Fprintf(os.Stderr, "%s [%s] %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		name,
		fmt.Sprintf(format, args...))
}

// Debug logs a debug-level message.
func Debug(format string, args ...any) { emit(levelDebug, "DEBUG", format, args...) }

// Info logs an info-level message.
func Info(format string, args ...any) { emit(levelInfo, "INFO", format, args...) }

// Warn logs a warning-level message.
func Warn(format string, args ...any) { emit(levelWarn, "WARN", format, args...) }

// Error logs an error-level message.
func Error(format string, args ...any) { emit(levelError, "ERROR", format, args...) }
