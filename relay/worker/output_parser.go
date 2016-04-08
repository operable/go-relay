package worker

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/messages"
	"regexp"
	"strings"
)

type outputParser func([]string, *messages.ExecutionResponse, messages.ExecutionRequest)

var outputParsers = map[*regexp.Regexp]outputParser{
	regexp.MustCompilePOSIX("^COGCMD_DEBUG:"): writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_INFO:"):  writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_WARN:"):  writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_ERR:"):   writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_ERROR:"): writeToLog,
	regexp.MustCompilePOSIX("^COG_TEMPLATE:"): extractTemplate,
	regexp.MustCompilePOSIX("^JSON$"):         flagJSON,
}

func parseOutput(output []byte, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	retained := []string{}
	resp.IsJSON = false
	for _, line := range strings.Split(string(output), "\n") {
		matched := false
		for re, cb := range outputParsers {
			if re.MatchString(line) {
				lines := re.Split(line, 2)
				cb(lines, resp, req)
				matched = true
				break
			}
		}
		if matched == false {
			retained = append(retained, line)
		}
	}
	remaining := []byte(strings.Join(retained, "\n"))
	if resp.IsJSON == true {
		jsonBody := interface{}(nil)
		if err := json.Unmarshal(remaining, &jsonBody); err != nil {
			resp.Status = "error"
			resp.StatusMessage = "Command returned invalid JSON."
		} else {
			resp.Body, _ = json.Marshal(jsonBody)
		}
	} else {
		resp.Body = []byte(fmt.Sprintf("{\"body\":\"%s\"}", remaining))
	}
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
	logFunc("(P: %s C: %s) %s", req.PipelineID(), req.Command, strings.Trim(line[1], " "))
}

func extractTemplate(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	if len(line) < 2 {
		return
	}
	resp.Template = strings.Trim(line[1], " ")
}

func flagJSON(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	resp.IsJSON = true
}
