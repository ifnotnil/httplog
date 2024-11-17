package httplog

import "log/slog"

type HTTPLoggerOp func(*HTTPLogger)

func WithLogger(logger *slog.Logger) HTTPLoggerOp {
	return func(h *HTTPLogger) { h.logger = logger }
}

func WithLogInLevel(lvl slog.Leveler) HTTPLoggerOp {
	return func(h *HTTPLogger) { h.logInLevel = lvl }
}

func WithMode(m Mode) HTTPLoggerOp {
	return func(h *HTTPLogger) { h.mode = m }
}

func WithLogPolicy(lp LogPolicy) HTTPLoggerOp {
	return func(h *HTTPLogger) { h.logPolicy = lp }
}

func NewHTTPLogger(ops ...HTTPLoggerOp) *HTTPLogger {
	il := &HTTPLogger{
		logInLevel: slog.LevelDebug,
		pool:       NewBytesBufferPool(1024),
	}

	for _, fn := range ops {
		fn(il)
	}

	if il.logger == nil {
		il.logger = slog.Default()
	}

	il.attrConverter = HTTPSLogAttrsConverter{
		logPolicy: il.logPolicy,
	}

	return il
}

type HTTPLogger struct {
	logPolicy     LogPolicy
	attrConverter HTTPSLogAttrsConverter
	logInLevel    slog.Leveler
	logger        *slog.Logger
	pool          *BytesBufferPool
	mode          Mode
}

type Mode int

const (
	Drain Mode = iota
	Tee
)
