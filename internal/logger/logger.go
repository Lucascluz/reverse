package logger

import (
	"context"
	"fmt"
	"log"
	"os"
)

type ctxKey int

const loggerKey ctxKey = 0

type Logger struct {
	prefix string
}

func New(prefix string) *Logger {

	// Good for container logs
	log.SetOutput(os.Stdout)

	return &Logger{
		prefix: prefix,
	}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+l.prefix+" "+format, args...)
}
func (l *Logger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+l.prefix+" "+format, args...)
}

// WithRequestFields returns a derived logger with request-scoped fields in the prefix.
func (l *Logger) WithRequestFields(requestID, method, path string) *Logger {
	newPrefix := fmt.Sprintf("%s request_id=%s method=%s path=%s", l.prefix, requestID, method, path)
	return &Logger{prefix: newPrefix}
}

func NewContext(ctx context.Context, rl *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, rl)
}

func FromContext(ctx context.Context) *Logger {
	if v := ctx.Value(loggerKey); v != nil {
		if rl, ok := v.(*Logger); ok {
			return rl
		}
	}
	// fallback logger
	return New("fallback")
}
