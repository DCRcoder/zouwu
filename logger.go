package zouwu

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// LogLevel log level
type LogLevel int

const (
	LogUnknown LogLevel = 0
	LogDebug   LogLevel = 1
	LogInfo    LogLevel = 2
	LogWarning LogLevel = 3
	LogError   LogLevel = 4
	LogFatal   LogLevel = 5
)

// Logger define
type Logger interface {
	Debug(msg string)
	Debugf(format string, v ...interface{})

	Info(msg string)
	Infof(format string, v ...interface{})

	Warn(msg string)
	Warnf(format string, v ...interface{})

	Error(msg string)
	Errorf(format string, v ...interface{})

	SetLogLevel(l LogLevel)
}

// FLogger Fei default Logger
type FLogger struct {
	logger zerolog.Logger
}

// NewFlogger return instance
func NewFlogger(level ...LogLevel) *FLogger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	l := &FLogger{
		logger: zerolog.New(output).With().Timestamp().Logger(),
	}
	if len(level) == 0 {
		l.SetLogLevel(LogError)
	} else {
		l.SetLogLevel(level[0])
	}
	return l
}

// Debug impl Debug
func (l *FLogger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf impl Debugf
func (l *FLogger) Debugf(format string, v ...interface{}) {
	l.logger.Debug().Msgf(format, v...)
}

// Info impl info
func (l *FLogger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof impl infof
func (l *FLogger) Infof(format string, v ...interface{}) {
	l.logger.Info().Msgf(format, v...)
}

// Warn impl Warn
func (l *FLogger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf impl Warnf
func (l *FLogger) Warnf(format string, v ...interface{}) {
	l.logger.Warn().Msgf(format, v...)
}

// Error impl Error
func (l *FLogger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf impl Errorf
func (l *FLogger) Errorf(format string, v ...interface{}) {
	l.logger.Error().Msgf(format, v...)
}

// SetLogLevel set log level
func (l *FLogger) SetLogLevel(level LogLevel) {
	switch level {
	case LogDebug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case LogInfo:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case LogWarning:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case LogError:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
}
