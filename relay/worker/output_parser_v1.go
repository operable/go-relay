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

type outputMatcher func([]string, *messages.ExecutionResponse, messages.ExecutionRequest)

type OutputParserV1 struct {
	matchers map[*regexp.Regexp]outputMatcher
}

// NewOutputParserV1 returns an OutputParser instance which understands Relay's
// original command output protocol.
func NewOutputParserV1() OutputParser {
	retval := &OutputParserV1{}
	retval.matchers = map[*regexp.Regexp]outputMatcher{
		regexp.MustCompilePOSIX("^COGCMD_DEBUG:"):  retval.writeToLog,
		regexp.MustCompilePOSIX("^COGCMD_INFO:"):   retval.writeToLog,
		regexp.MustCompilePOSIX("^COGCMD_WARN:"):   retval.writeToLog,
		regexp.MustCompilePOSIX("^COGCMD_ERR:"):    retval.writeToLog,
		regexp.MustCompilePOSIX("^COGCMD_ERROR:"):  retval.writeToLog,
		regexp.MustCompilePOSIX("^COGCMD_ACTION:"): retval.parseAction,
		regexp.MustCompilePOSIX("^COG_TEMPLATE:"):  retval.extractTemplate,
		regexp.MustCompilePOSIX("^JSON$"):          retval.flagJSON,
	}
	return retval
}

// Parse is required by the OutputParser interface
func (op *OutputParserV1) Parse(result api.ExecResult, req messages.ExecutionRequest, err error) *messages.ExecutionResponse {
	resp := &messages.ExecutionResponse{}
	resp.Status = "ok"
	if err != nil {
		resp.Status = "error"
		resp.StatusMessage = fmt.Sprintf("%s", err)
		return resp
	}
	retained := []string{}
	if len(result.Stdout) > 0 {
		lines := strings.Split(strings.TrimSuffix(string(result.Stdout), "\n"), "\n")
		for _, line := range lines {
			matched := false
			if resp.IsJSON == false {
				for re, cb := range op.matchers {
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
			} else {
				retained = append(retained, line)
			}
		}
	}
	if len(result.Stderr) > 0 {
		resp.Status = "error"
		resp.StatusMessage = string(result.Stderr)
		return resp
	}

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
	if resp.Status == "ok" && resp.Aborted == true {
		resp.Status = "abort"
	}
	return resp
}

func (op *OutputParserV1) writeToLog(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	message := strings.Trim(line[1], " ")
	if message == "" {
		return
	}
	format := "(P: %s C: %s) %s"

	switch line[0] {
	case "DEBUG:":
		log.Debugf(format, req.PipelineID(), req.Command, message)
	case "WARN:":
		log.Warnf(format, req.PipelineID(), req.Command, message)
	case "ERR:":
		fallthrough
	case "ERROR:":
		log.Errorf(format, req.PipelineID(), req.Command, message)
	default:
		log.Infof(format, req.PipelineID(), req.Command, message)
	}
}

func (op *OutputParserV1) extractTemplate(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	resp.Template = strings.Trim(line[1], " ")
}

func (op *OutputParserV1) flagJSON(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	resp.IsJSON = true
}

func (op *OutputParserV1) parseAction(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	switch strings.Trim(line[1], " ") {
	case "abort":
		resp.Aborted = true
	default:
		break
	}
}
