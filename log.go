package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"text/template"
	"time"

	"github.com/kermitbu/gant-log/colors"
)

var errInvalidLogLevel = errors.New("logger: invalid log level")

const (
	levelDebug = iota
	levelInfo
	levelWarn
	levelError
	levelFatal
)

var (
	sequenceNo uint64
	instance   *QLogger
	once       sync.Once
)

var debugMode = os.Getenv("IIGSDEBUG") == "1"

var logLevel = levelDebug

// QLogger logs logging records to the specified io.Writer
type QLogger struct {
	mu     sync.Mutex
	output io.Writer
}

// LogRecord represents a log record and contains the timestamp when the record
// was created, an increasing id, level and the actual formatted log line.
type LogRecord struct {
	ID       string
	Level    string
	Message  string
	Filename string
	LineNo   int
}

var (
	logRecordTemplate      *template.Template
	debugLogRecordTemplate *template.Template
)

// getQLogger initializes the logger instance with a NewColorWriter output
// and returns a singleton
func getQLogger(w io.Writer) *QLogger {
	once.Do(func() {
		var (
			err             error
			debugLogFormat  = `[IIGService] {{Now "2006/01/02 15:04:05"}} {{.Level}} ▶ {{.ID}} {{.Filename}}:{{.LineNo}} {{.Message}}{{EndLine}}`
			relaseLogFormat = `[IIGService] {{Now "2006/01/02 15:04:05"}} {{.Level}} ▶ {{.ID}} {{.Message}}{{EndLine}}`
		)

		// Initialize and parse logging templates
		funcs := template.FuncMap{
			"Now":     Now,
			"EndLine": EndLine,
		}

		if debugMode {
			logRecordTemplate, err = template.New("debugLogFormat").Funcs(funcs).Parse(debugLogFormat)
			if err != nil {
				panic(err)
			}
		} else {
			logRecordTemplate, err = template.New("relaseLogFormat").Funcs(funcs).Parse(relaseLogFormat)
			if err != nil {
				panic(err)
			}
		}

		instance = &QLogger{output: colors.NewColorWriter(w)}
	})
	return instance
}

// SetOutput sets the logger output destination
func (l *QLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = colors.NewColorWriter(w)
}

// Now returns the current local time in the specified layout
func Now(layout string) string {
	return time.Now().Format(layout)
}

// EndLine returns the a newline escape character
func EndLine() string {
	return "\n"
}

func (l *QLogger) getLevelTag(level int) string {
	switch level {
	case levelDebug:
		return "DEBUG"
	case levelInfo:
		return "INFO "
	case levelWarn:
		return "WARN "
	case levelError:
		return "ERROR"
	case levelFatal:
		return "FATAL"
	default:
		panic(errInvalidLogLevel)
	}
}

func (l *QLogger) getColorLevel(level int) string {
	switch level {
	case levelDebug:
		return colors.CyanBold(l.getLevelTag(level))
	case levelInfo:
		return colors.GreenBold(l.getLevelTag(level))
	case levelWarn:
		return colors.YellowBold(l.getLevelTag(level))
	case levelError:
		return colors.RedBold(l.getLevelTag(level))
	case levelFatal:
		return colors.MagentaBold(l.getLevelTag(level))
	default:
		panic(errInvalidLogLevel)
	}
}

// mustLog logs the message according to the specified level and arguments.
// It panics in case of an error.
func (l *QLogger) mustLog(level int, calldepth int, message string, args ...interface{}) {
	if level < logLevel {
		return
	}
	// Acquire the lock
	l.mu.Lock()
	defer l.mu.Unlock()

	var ok bool
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}

	record := LogRecord{
		Level:    l.getColorLevel(level),
		Message:  fmt.Sprintf(message, args...),
		Filename: filepath.Base(file),
		LineNo:   line,
	}

	err := logRecordTemplate.Execute(l.output, record)
	if err != nil {
		panic(err)
	}
}

var log = getQLogger(os.Stdout)

// Debug 级别最低的，一般不用，在使用前最好加上if判断
func Debug(format string, v ...interface{}) {
	if debugMode {
		log.mustLog(levelDebug, 2, format, v...)
	}
}

// Info 反馈给用户用的信息，可以作为产品的一部分
func Info(format string, v ...interface{}) {
	log.mustLog(levelInfo, 2, format, v...)
}

// Warn 检测到了一个不正常状态，做一些修复性的工作可以系统恢复到正常状态来
func Warn(format string, v ...interface{}) {
	log.mustLog(levelWarn, 2, format, v...)
}

// Error 检测到了一个不正常状态，做一些修复性的工作不确定系统是否能恢复到正常状态来
func Error(format string, v ...interface{}) {
	log.mustLog(levelError, 2, format, v...)
}

// Fatal 检测到了一个不正常状态，相当严重，并且肯定这个错误无法修复，如果系统运行下去会越来越乱
func Fatal(format string, v ...interface{}) {
	log.mustLog(levelFatal, 2, format, v...)
	os.Exit(-1)
}

func Trace(url string, code int, result string) {
	output := colors.NewColorWriter(os.Stdout)
	io.WriteString(output, "=================================\n")
	io.WriteString(output, colors.MagentaBold(fmt.Sprintf("   URL: %+v\n", url)))
	io.WriteString(output, colors.MagentaBold(fmt.Sprintf("  CODE: %+v\n", code)))
	io.WriteString(output, colors.MagentaBold(fmt.Sprintf("RESULT: %+v\n", result)))
}
