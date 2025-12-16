package logger

import (
	"context"
	"io"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// Package logger is a thin wrapper around logrus' standard logger.
//
// It is designed to be imported as `log`, so applications can share a single
// logging backend configured once (typically via transport-lib/pkg/bootstrap).

type Fields = log.Fields
type Entry = log.Entry
type Logger = log.Logger
type Level = log.Level
type Formatter = log.Formatter
type Hook = log.Hook
type JSONFormatter = log.JSONFormatter
type TextFormatter = log.TextFormatter

var AllLevels = log.AllLevels

const (
	PanicLevel = log.PanicLevel
	FatalLevel = log.FatalLevel
	ErrorLevel = log.ErrorLevel
	WarnLevel  = log.WarnLevel
	InfoLevel  = log.InfoLevel
	DebugLevel = log.DebugLevel
	TraceLevel = log.TraceLevel
)

func StandardLogger() *Logger { return log.StandardLogger() }
func New() *Logger            { return log.New() }
func NewEntry(l *Logger) *Entry {
	return log.NewEntry(l)
}

func AddHook(h Hook)                         { log.AddHook(h) }
func SetFormatter(f Formatter)               { log.SetFormatter(f) }
func SetLevel(level Level)                   { log.SetLevel(level) }
func ParseLevel(level string) (Level, error) { return log.ParseLevel(level) }
func SetOutput(out io.Writer)                { log.SetOutput(out) }
func SetReportCaller(report bool)            { log.SetReportCaller(report) }
func IsLevelEnabled(level Level) bool        { return log.IsLevelEnabled(level) }

func WithField(key string, value any) *Entry { return log.WithField(key, value) }
func WithFields(fields Fields) *Entry        { return log.WithFields(fields) }
func WithError(err error) *Entry             { return log.WithError(err) }
func WithContext(ctx context.Context) *Entry { return log.WithContext(ctx) }

// WithTrace binds ctx and adds "trace_id" when OpenTelemetry span context is present.
func WithTrace(ctx context.Context) *Entry {
	e := log.WithContext(ctx)
	if ctx == nil {
		return e
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		e = e.WithField("trace_id", sc.TraceID().String())
	}
	return e
}

func Trace(args ...any) { log.Trace(args...) }
func Debug(args ...any) { log.Debug(args...) }
func Info(args ...any)  { log.Info(args...) }
func Warn(args ...any)  { log.Warn(args...) }
func Error(args ...any) { log.Error(args...) }
func Fatal(args ...any) { log.Fatal(args...) }
func Panic(args ...any) { log.Panic(args...) }
func Print(args ...any) { log.Print(args...) }

func Tracef(format string, args ...any) { log.Tracef(format, args...) }
func Debugf(format string, args ...any) { log.Debugf(format, args...) }
func Infof(format string, args ...any)  { log.Infof(format, args...) }
func Warnf(format string, args ...any)  { log.Warnf(format, args...) }
func Errorf(format string, args ...any) { log.Errorf(format, args...) }
func Fatalf(format string, args ...any) { log.Fatalf(format, args...) }
func Panicf(format string, args ...any) { log.Panicf(format, args...) }
func Printf(format string, args ...any) { log.Printf(format, args...) }

func Traceln(args ...any) { log.Traceln(args...) }
func Debugln(args ...any) { log.Debugln(args...) }
func Infoln(args ...any)  { log.Infoln(args...) }
func Warnln(args ...any)  { log.Warnln(args...) }
func Errorln(args ...any) { log.Errorln(args...) }
func Fatalln(args ...any) { log.Fatalln(args...) }
func Panicln(args ...any) { log.Panicln(args...) }
func Println(args ...any) { log.Println(args...) }
