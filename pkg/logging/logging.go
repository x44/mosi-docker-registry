package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/service"
)

const (
	DEBUG   = 0
	INFO    = 1
	WARNING = 2
	ERROR   = 3
	SILENT  = 4
)

var levelStrings = [...]string{"[DEBUG]", "[INFO]", "[WARN]", "[ERROR]"}

var loggerService service.Logger = nil
var loggerConsole *log.Logger = nil
var loggerFile *log.Logger = nil
var logFile *os.File = nil
var logFileName string = ""

var logLevelConsole = INFO
var logLevelService = INFO
var logLevelFile = INFO

var fmtDate = true
var fmtTime = true
var fmtMicros = false

func Level(level string) int {
	switch level {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING":
	case "WARN":
		return INFO
	case "ERROR":
		return ERROR
	}
	return SILENT
}

func Init(logger service.Logger, isService bool, logFn string, levelService, levelConsole, levelFile int, printDate, printTime, printMicros bool) {
	if isService && levelService < SILENT {
		loggerService = logger
	} else {
		loggerService = nil
	}

	if !isService && levelConsole < SILENT {
		if loggerConsole == nil {
			loggerConsole = log.New(os.Stdout, "", 0)
		}
	} else {
		loggerConsole = nil
	}

	if levelFile < SILENT {
		if loggerFile == nil || logFile == nil || logFileName != logFn {
			if logFile != nil {
				logFile.Close()
				logFile = nil
			}
			logFileName = logFn
			dir := filepath.Dir(logFileName)
			err := os.MkdirAll(dir, 0700)
			if err != nil {
				log.Fatal(err)
			}
			logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Fatal(err)
			}
			loggerFile = log.New(logFile, "", 0)
		}
	} else {
		if logFile != nil {
			logFile.Close()
			logFile = nil
		}
		logFileName = ""
		loggerFile = nil
	}

	logLevelService = levelService
	logLevelConsole = levelConsole
	logLevelFile = levelFile

	fmtDate = printDate
	fmtTime = printTime
	fmtMicros = printMicros
}

func Debug(prefix string, v ...interface{}) {
	print(DEBUG, prefix, v...)
}

func Info(prefix string, v ...interface{}) {
	print(INFO, prefix, v...)
}

func Warn(prefix string, v ...interface{}) {
	print(WARNING, prefix, v...)
}

func Error(prefix string, v ...interface{}) {
	print(ERROR, prefix, v...)
}

func Fatal(prefix string, v ...interface{}) {
	print(ERROR, prefix, v...)
	os.Exit(1)
}

func print(level int, prefix string, v ...any) {
	if level < logLevelConsole && level < logLevelService && level < logLevelFile {
		return
	}
	lc := loggerConsole
	ls := loggerService
	lf := loggerFile

	if lc == nil && ls == nil && lf == nil {
		return
	}

	now := time.Now()
	var buf []byte
	formatHeader(&buf, now)

	var msg string
	if len(v) == 0 {
		msg = prefix
	} else if len(v) == 1 {
		msg = fmt.Sprintf("[%s]: %v", prefix, v[0])
	} else {
		format := v[0].(string)
		msg = fmt.Sprintf(format, v[1:]...)
		msg = fmt.Sprintf("[%s]: %s", prefix, msg)
	}

	if level >= logLevelService && ls != nil {
		switch level {
		case DEBUG, INFO:
			ls.Infof("%s%-7s %s\n", buf, levelStrings[level], msg)
		case WARNING:
			ls.Warningf("%s%-7s %s\n", buf, levelStrings[level], msg)
		default:
			ls.Errorf("%s%-7s %s\n", buf, levelStrings[level], msg)
		}
	}

	if level >= logLevelConsole && lc != nil {
		lc.Printf("%s%-7s %s\n", buf, levelStrings[level], msg)
	}

	if level >= logLevelFile && loggerFile != nil {
		lf.Printf("%s%-7s %s\n", buf, levelStrings[level], msg)
	}
}

// From log.go
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

// From log.go
func formatHeader(buf *[]byte, t time.Time) {
	if fmtDate {
		year, month, day := t.Date()
		itoa(buf, year, 4)
		*buf = append(*buf, '/')
		itoa(buf, int(month), 2)
		*buf = append(*buf, '/')
		itoa(buf, day, 2)
		*buf = append(*buf, ' ')
	}
	if fmtTime {
		hour, min, sec := t.Clock()
		itoa(buf, hour, 2)
		*buf = append(*buf, ':')
		itoa(buf, min, 2)
		*buf = append(*buf, ':')
		itoa(buf, sec, 2)
		if fmtMicros {
			*buf = append(*buf, '.')
			itoa(buf, t.Nanosecond()/1e3, 6)
		}
		*buf = append(*buf, ' ')
	}
}
