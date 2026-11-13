package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	tlog "go.temporal.io/sdk/log"
)

// New initializes a configured zerolog.Logger instance.
func New(level string, pretty bool, serviceName string) zerolog.Logger {
	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(parsedLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	var output io.Writer = os.Stdout
	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	return zerolog.New(output).
		Level(parsedLevel).
		With().
		Timestamp().
		Str("service", serviceName).
		Caller().
		Logger()
}

// TemporalAdapter adapts zerolog.Logger to Temporal's log.Logger interface.
type TemporalAdapter struct {
	log zerolog.Logger
}

// NewTemporalAdapter returns a Temporal-compatible logger backed by zerolog.
func NewTemporalAdapter(l zerolog.Logger) tlog.Logger {
	// Skip extra wrapper frame for accurate caller reporting
	return &TemporalAdapter{
		log: l.With().CallerWithSkipFrameCount(3).Str("component", "temporal").Logger(),
	}
}

func (t *TemporalAdapter) Debug(msg string, keyvals ...interface{}) {
	t.fields(t.log.Debug(), keyvals...).Msg(msg)
}

func (t *TemporalAdapter) Info(msg string, keyvals ...interface{}) {
	t.fields(t.log.Info(), keyvals...).Msg(msg)
}

func (t *TemporalAdapter) Warn(msg string, keyvals ...interface{}) {
	t.fields(t.log.Warn(), keyvals...).Msg(msg)
}

func (t *TemporalAdapter) Error(msg string, keyvals ...interface{}) {
	t.fields(t.log.Error(), keyvals...).Msg(msg)
}

func (t *TemporalAdapter) fields(event *zerolog.Event, keyvals ...interface{}) *zerolog.Event {
	if event == nil {
		return nil
	}
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		var val interface{}
		if i+1 < len(keyvals) {
			val = keyvals[i+1]
		}
		event = event.Interface(key, val)
	}
	return event
}
