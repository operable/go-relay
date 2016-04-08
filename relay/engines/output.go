package engines

import (
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/messages"
	"regexp"
	"strings"
)

type outputParser func([]string, *messages.ExecutionResponse, messages.ExecutionRequest)

var outputParsers = map[*regexp.Regexp]outputParser{
	regexp.MustCompilePOSIX("^DEBUG:"):    writeToLog,
	regexp.MustCompilePOSIX("^INFO:"):     writeToLog,
	regexp.MustCompilePOSIX("^WARN:"):     writeToLog,
	regexp.MustCompilePOSIX("^ERR:"):      writeToLog,
	regexp.MustCompilePOSIX("^ERROR:"):    writeToLog,
	regexp.MustCompilePOSIX("^TEMPLATE:"): extractTemplate,
}

func writeToLog(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	if len(line) < 2 {
		return
	}
	logFunc := log.Infof
	switch line[0] {
	case "DEBUG:":
		logFunc = log.Debugf
	case "WARN:":
		logFunc = log.Warnf
	case "ERR:":
		fallthrough
	case "ERROR:":
		logFunc = log.Errorf
	default:
		break
	}
	logFunc("Pipeline %s, Command: %s: %s", req.PipelineID(), req.Command, strings.Trim(line[1], " "))
}

func extractTemplate(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	if len(line) < 2 {
		return
	}
	resp.Template = strings.Trim(line[1], " ")
}
