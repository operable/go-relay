package worker

import (
	"bytes"
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

func parseOutput(commandOutput []byte, commandErrors []byte, err error, resp *messages.ExecutionResponse,
	req messages.ExecutionRequest) {
	retained := []string{}
	if len(commandOutput) > 0 {
		lines := strings.Split(strings.TrimSuffix(string(commandOutput), "\n"), "\n")
		for _, line := range lines {
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
	}

	if err != nil {
		resp.Status = "error"
		if len(commandErrors) > 0 {
			resp.StatusMessage = string(commandErrors)
		} else {
			resp.StatusMessage = fmt.Sprintf("%s", err)
		}
		return
	}

	if len(commandErrors) > 0 {
		resp.Status = "error"
		resp.StatusMessage = string(commandErrors)
		return
	}

	resp.Status = "ok"
	if resp.IsJSON == true {
		jsonBody := interface{}(nil)
		remaining := []byte(strings.Join(retained, "\n"))

		d := json.NewDecoder(bytes.NewReader(remaining))
		d.UseNumber()
		if err := d.Decode(&jsonBody); err != nil {
			resp.Status = "error"
			resp.StatusMessage = "Command returned invalid JSON."
		} else {
			resp.Body = jsonBody
		}
	} else {
		if len(retained) > 0 {
			resp.Body = []map[string][]string{
				map[string][]string{
					"body": retained,
				},
			}
		}
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
