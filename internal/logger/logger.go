package logger

import (
	"context"
	"fmt"
	"log"
	"os"
)

type loggerKey int

// LoggerCtxKey is the context key for storing logger instances
const LoggerCtxKey loggerKey = 0

// Logger is a simple structured logger with prefix support
type Logger struct {
	prefix string
}

// NewLogger creates a new logger with the given prefix
func NewLogger(prefix string) *Logger {
	// Good for container logs - write to stdout
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	return &Logger{prefix: prefix}
}

// Infof logs an info-level message
func (l *Logger) Infof(format string, args ...any) {
	log.Printf("[INFO] %s %s", l.prefix, fmt.Sprintf(format, args...))
}

// Errorf logs an error-level message
func (l *Logger) Errorf(format string, args ...any) {
	log.Printf("[ERROR] %s %s", l.prefix, fmt.Sprintf(format, args...))
}

// WithRequestFields returns a new logger with request-scoped fields
func (l *Logger) WithRequestFields(requestID, method, path string) *Logger {
	newPrefix := fmt.Sprintf("%s request_id=%s method=%s path=%s",
		l.prefix, requestID, method, path)
	return &Logger{prefix: newPrefix}
}

// LoggerFromContext retrieves the logger from the context
func LoggerFromContext(ctx context.Context) *Logger {
	if v := ctx.Value(LoggerCtxKey); v != nil {
		if logger, ok := v.(*Logger); ok {
			return logger
		}
	}
	// Fallback logger if none in context
	return NewLogger("fallback")
}

// LoggerToContext stores the logger in the context
func LoggerToContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, LoggerCtxKey, logger)
}