package logger

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Logger interface {
	Debug(format string, args ...any)
	Access(format string, args ...any)
	Panic(format string, args ...any)
}

type SlogLogger struct {
	debug  *slog.Logger
	access *slog.Logger
	panic  *slog.Logger
}

func New(logDir string) (*SlogLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	newLogger := func(name string) (*slog.Logger, *os.File, error) {
		f, err := os.OpenFile(filepath.Join(logDir, name+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		h := slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})
		return slog.New(h), f, nil
	}

	debugL, _, err := newLogger("debug")
	if err != nil {
		return nil, err
	}
	accessL, _, err := newLogger("access")
	if err != nil {
		return nil, err
	}
	panicL, _, err := newLogger("panic")
	if err != nil {
		return nil, err
	}

	return &SlogLogger{debug: debugL, access: accessL, panic: panicL}, nil
}

func (l *SlogLogger) Debug(format string, args ...any) {
	l.debug.Debug(fmt.Sprintf(format, args...))
}

func (l *SlogLogger) Access(format string, args ...any) {
	l.access.Info(fmt.Sprintf(format, args...))
}

func (l *SlogLogger) Panic(format string, args ...any) {
	l.panic.Error(fmt.Sprintf(format, args...))
}

type NopLogger struct{}

func (n *NopLogger) Debug(format string, args ...any)  {}
func (n *NopLogger) Access(format string, args ...any) {}
func (n *NopLogger) Panic(format string, args ...any)  {}

func AccessLog(log Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(sw, r)
			log.Access("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
		})
	}
}

func Recovery(log Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Panic("panic: %v — %s %s", rec, r.Method, r.URL.Path)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
