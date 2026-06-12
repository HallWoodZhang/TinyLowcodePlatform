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
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type Loggers struct {
	DebugL  Logger
	AccessL Logger
	PanicL  Logger
}

type slogAdapter struct {
	logger *slog.Logger
}

func (a *slogAdapter) Debug(msg string, args ...any) { a.logger.Debug(fmt.Sprintf(msg, args...)) }
func (a *slogAdapter) Info(msg string, args ...any)  { a.logger.Info(fmt.Sprintf(msg, args...)) }
func (a *slogAdapter) Warn(msg string, args ...any)  { a.logger.Warn(fmt.Sprintf(msg, args...)) }
func (a *slogAdapter) Error(msg string, args ...any) { a.logger.Error(fmt.Sprintf(msg, args...)) }

func New(logDir string, debugLevel, accessLevel, panicLevel string) (*Loggers, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	newLogger := func(name, rawLevel string, defaultLevel slog.Level) (*slog.Logger, *os.File, error) {
		level := parseLevel(rawLevel, defaultLevel)
		f, err := os.OpenFile(filepath.Join(logDir, name+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		h := slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
		return slog.New(h), f, nil
	}

	debugL, _, err := newLogger("debug", debugLevel, slog.LevelDebug)
	if err != nil {
		return nil, err
	}
	accessL, _, err := newLogger("access", accessLevel, slog.LevelInfo)
	if err != nil {
		return nil, err
	}
	panicL, _, err := newLogger("panic", panicLevel, slog.LevelError)
	if err != nil {
		return nil, err
	}

	return &Loggers{
		DebugL:  &slogAdapter{debugL},
		AccessL: &slogAdapter{accessL},
		PanicL:  &slogAdapter{panicL},
	}, nil
}

func parseLevel(s string, fallback slog.Level) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return fallback
	}
}

type NopLogger struct{}

func (n *NopLogger) Debug(msg string, args ...any) {}
func (n *NopLogger) Info(msg string, args ...any)  {}
func (n *NopLogger) Warn(msg string, args ...any)  {}
func (n *NopLogger) Error(msg string, args ...any) {}

var _ Logger = (*NopLogger)(nil)
var _ Logger = (*slogAdapter)(nil)

func AccessLog(log Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(sw, r)
			log.Info("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
		})
	}
}

func Recovery(log Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic: %v — %s %s", rec, r.Method, r.URL.Path)
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
