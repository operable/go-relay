package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/operable/cogexec/messages"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type logDirection byte

const (
	logInput logDirection = iota
	logOutput
)

const (
	okStatus                 = 0
	decodeFailedStatus       = 7
	inputLoggerFailedStatus  = 8
	outputLoggerFailedStatus = 9
)

type dataLogger struct {
	log *os.File
}

var logEOL = []byte("\n")
var logDirectory = "/var/log"

func NewDatalogger(dir string, direction logDirection, ts time.Time) (*dataLogger, error) {
	logDir := "input"
	if direction == logOutput {
		logDir = "output"
	}
	path := strings.Join([]string{dir, fmt.Sprintf("%d_%s.log", ts.UnixNano(), logDir)}, "/")
	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &dataLogger{
		log: fd,
	}, nil
}

func (dl *dataLogger) Write(data []byte) (int, error) {
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

func (dl *dataLogger) Close() {
	dl.log.Close()
}

func runCommand(req messages.ExecCommandRequest, encoder *gob.Encoder) {
	command := exec.Command(req.Executable)
	command.Env = req.Env
	input := bytes.NewBuffer(req.CogEnv)
	stdout := bytes.NewBuffer([]byte{})
	stderr := bytes.NewBuffer([]byte{})
	command.Stdin = input
	command.Stdout = stdout
	command.Stderr = stderr
	start := time.Now()
	err := command.Run()
	finish := time.Now()
	if stderr.Len() == 0 && err != nil {
		stderr.WriteString(fmt.Sprintf("%s\n", err))
	}
	resp := &messages.ExecCommandResponse{
		Executable: req.Executable,
		Success:    err == nil,
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.Bytes(),
		Elapsed:    finish.Sub(start),
	}
	encoder.Encode(resp)
}

func main() {
	cmdIn := io.Reader(os.Stdin)
	cmdOut := io.Writer(os.Stdout)
	if os.Getenv("COG_EXEC_LOG") != "" {
		now := time.Now()
		inputLogger, err := NewDatalogger(logDirectory, logInput, now)
		if err != nil {
			os.Exit(inputLoggerFailedStatus)
		}
		defer inputLogger.Close()
		outputLogger, err := NewDatalogger(logDirectory, logOutput, now)
		if err != nil {
			os.Exit(outputLoggerFailedStatus)
		}
		defer outputLogger.Close()
		cmdIn = io.TeeReader(cmdIn, inputLogger)
		cmdOut = io.MultiWriter(os.Stdout, outputLogger)
	}
	for {
		decoder := gob.NewDecoder(cmdIn)
		encoder := gob.NewEncoder(cmdOut)
		req := messages.ExecCommandRequest{}
		err := decoder.Decode(&req)
		if err != nil {
			os.Exit(decodeFailedStatus)
		}
		if req.Die {
			break
		}
		runCommand(req, encoder)
		syscall.Fsync(int(os.Stdout.Fd()))
	}
	os.Exit(okStatus)
}
