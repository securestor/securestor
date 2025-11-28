// internal/logger/logger.go
package logger

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func NewLogger(prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "["+prefix+"] ", log.LstdFlags),
	}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.Printf("[INFO] %s %v", msg, fields)
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("[ERROR] %s: %v", msg, err)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.Printf("[DEBUG] %s %v", msg, fields)
}
