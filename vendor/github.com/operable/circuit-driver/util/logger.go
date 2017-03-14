package util

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

type LogDirection int

const (
	LogInput LogDirection = iota
	LogOutput
)

type DataLogger struct {
	log *os.File
}

var logEOL = []byte("\n")
var logDirectory = "/var/log"

func NewDataLogger(dir string, direction LogDirection, ts time.Time) (*DataLogger, error) {
	logDir := "input"
	if direction == LogOutput {
		logDir = "output"
	}
	path := strings.Join([]string{dir, fmt.Sprintf("%d_%s.log", ts.UnixNano(), logDir)}, "/")
	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &DataLogger{
		log: fd,
	}, nil
}

func (dl *DataLogger) WriteString(text string) (int, error) {
	return (dl.log.WriteString(text))
}

func (dl *DataLogger) Write(data []byte) (int, error) {
	for i, d := range data {
		if i == 0 {
			dl.log.WriteString(fmt.Sprintf("%d", d))
		} else {
			dl.log.WriteString(fmt.Sprintf(" %d", d))
		}
	}
	dl.log.Write(logEOL)
	syscall.Fsync(int(dl.log.Fd()))
	return len(data), nil
}

func (dl *DataLogger) Close() {
	dl.log.Close()
}
