package logger

import (
"log/slog"
"os"
)

// Logger is the application-wide structured logger
var Logger *slog.Logger

func init() {
// Initialize with JSON handler for structured logging
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
Level: slog.LevelInfo,
})
Logger = slog.New(handler)
}

// SetLevel changes the logging level
func SetLevel(level slog.Level) {
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
Level: level,
})
Logger = slog.New(handler)
}

// Info logs an informational message with structured fields
func Info(msg string, args ...any) {
Logger.Info(msg, args...)
}

// Error logs an error message with structured fields
func Error(msg string, args ...any) {
Logger.Error(msg, args...)
}

// Warn logs a warning message with structured fields
func Warn(msg string, args ...any) {
Logger.Warn(msg, args...)
}

// Debug logs a debug message with structured fields
func Debug(msg string, args ...any) {
Logger.Debug(msg, args...)
}
