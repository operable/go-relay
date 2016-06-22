package worker

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/circuit-driver/api"
	"github.com/operable/go-relay/relay/messages"
	"github.com/operable/go-relay/relay/util"
	"regexp"
	"strings"
)

type outputParser func([]string, *messages.ExecutionResponse, messages.PipelineStageRequest, messages.ExecutionRequest)

var outputParsers = map[*regexp.Regexp]outputParser{
	regexp.MustCompilePOSIX("^COGCMD_DEBUG:"): writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_INFO:"):  writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_WARN:"):  writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_ERR:"):   writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_ERROR:"): writeToLog,
	regexp.MustCompilePOSIX("^COG_TEMPLATE:"): extractTemplate,
	regexp.MustCompilePOSIX("^JSON$"):         flagJSON,
}

func parseOutput(result api.ExecResult, err error, resp *messages.ExecutionResponse, stageReq messages.PipelineStageRequest,
	req messages.ExecutionRequest) {
	if err != nil {
		resp.Status = "error"
		resp.StatusMessage = fmt.Sprintf("%s", err)
		return
	}
	retained := []string{}
	if len(result.Stdout) > 0 {
		lines := strings.Split(strings.TrimSuffix(string(result.Stdout), "\n"), "\n")
		for _, line := range lines {
			matched := false
			if resp.IsJSON == false {
				for re, cb := range outputParsers {
					if re.MatchString(line) {
						lines := re.Split(line, 2)
						cb(lines, resp, stageReq, req)
						matched = true
						break
					}
				}
				if matched == false {
					retained = append(retained, line)
				}
			} else {
				retained = append(retained, line)
			}
		}
	}
	if len(result.Stderr) > 0 {
		resp.Status = "error"
		resp.StatusMessage = string(result.Stderr)
		return
	}

	resp.Status = "ok"
	if resp.IsJSON == true {
		jsonBody := interface{}(nil)
		remaining := []byte(strings.Join(retained, "\n"))

		d := util.NewJSONDecoder(bytes.NewReader(remaining))
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

func writeToLog(line []string, resp *messages.ExecutionResponse, stageReq messages.PipelineStageRequest, req messages.ExecutionRequest) {
	if len(line) < 2 {
		return
	}
	format := "(P: %s C: %s) %s"
	message := strings.Trim(line[1], " ")
	switch line[0] {
	case "DEBUG:":
		log.Debugf(format, stageReq.PipelineID(), stageReq.Command, message)
	case "WARN:":
		log.Warnf(format, stageReq.PipelineID(), stageReq.Command, message)
	case "ERR:":
		fallthrough
	case "ERROR:":
		log.Errorf(format, stageReq.PipelineID(), stageReq.Command, message)
	default:
		log.Infof(format, stageReq.PipelineID(), stageReq.Command, message)
	}
}

func extractTemplate(line []string, resp *messages.ExecutionResponse, stageReq messages.PipelineStageRequest, req messages.ExecutionRequest) {
	if len(line) < 2 {
		return
	}
	resp.Template = strings.Trim(line[1], " ")
}

func flagJSON(line []string, resp *messages.ExecutionResponse, stageReq messages.PipelineStageRequest, req messages.ExecutionRequest) {
	resp.IsJSON = true
}
