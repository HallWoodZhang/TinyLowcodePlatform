package logger

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const traceKey contextKey = "traceID"

type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}

type Loggers struct {
	DebugL  Logger
	AccessL Logger
	PanicL  Logger
}

type slogAdapter struct {
	logger *slog.Logger
}

func (a *slogAdapter) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	traceID, _ := ctx.Value(traceKey).(string)
	a.logger.Log(ctx, level, fmt.Sprintf(msg, args...), "traceID", traceID)
}

func (a *slogAdapter) Debug(ctx context.Context, msg string, args ...any) {
	a.log(ctx, slog.LevelDebug, msg, args...)
}
func (a *slogAdapter) Info(ctx context.Context, msg string, args ...any) {
	a.log(ctx, slog.LevelInfo, msg, args...)
}
func (a *slogAdapter) Warn(ctx context.Context, msg string, args ...any) {
	a.log(ctx, slog.LevelWarn, msg, args...)
}
func (a *slogAdapter) Error(ctx context.Context, msg string, args ...any) {
	a.log(ctx, slog.LevelError, msg, args...)
}

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

func (n *NopLogger) Debug(ctx context.Context, msg string, args ...any) {}
func (n *NopLogger) Info(ctx context.Context, msg string, args ...any)  {}
func (n *NopLogger) Warn(ctx context.Context, msg string, args ...any)  {}
func (n *NopLogger) Error(ctx context.Context, msg string, args ...any) {}

var _ Logger = (*NopLogger)(nil)
var _ Logger = (*slogAdapter)(nil)

func TraceMiddleware(log Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := uuid.New().String()
			ctx := context.WithValue(r.Context(), traceKey, traceID)
			r = r.WithContext(ctx)
			w.Header().Set("X-Trace-ID", traceID)
			next.ServeHTTP(w, r)
		})
	}
}

func AccessLog(log Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(sw, r)
			log.Info(r.Context(), "%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
		})
	}
}

func Recovery(log Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error(r.Context(), "panic: %v — %s %s", rec, r.Method, r.URL.Path)
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
