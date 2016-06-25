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

type outputParser func([]string, *messages.ExecutionResponse, messages.ExecutionRequest)

var BodyDelimiter = []byte("\n\n")

var outputParsers = map[*regexp.Regexp]outputParser{
	regexp.MustCompilePOSIX("^COGCMD_DEBUG:"): writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_INFO:"):  writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_WARN:"):  writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_ERR:"):   writeToLog,
	regexp.MustCompilePOSIX("^COGCMD_ERROR:"): writeToLog,
	regexp.MustCompilePOSIX("^COG_TEMPLATE:"): extractTemplate,
	regexp.MustCompilePOSIX("^JSON$"):         flagJSON,
}

func processHeaders(headers string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	lines := strings.Split(headers, "\n")
	for _, line := range lines {
		for re, cb := range outputParsers {
			if re.MatchString(line) {
				lines := re.Split(line, 2)
				cb(lines, resp, req)
				break
			}
		}
	}
}

func processBody(body []byte, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	if resp.IsJSON == true {
		jsonBody := interface{}(nil)
		d := util.NewJSONDecoder(bytes.NewReader(body))
		if err := d.Decode(&jsonBody); err != nil {
			resp.Status = "error"
			resp.StatusMessage = fmt.Sprintf("Command returned invalid JSON: %s.", err)
			return
		}
		resp.Body = jsonBody
		return
	} else {
		resp.Body = []map[string][]string{
			map[string][]string{
				"body": []string{string(body)},
			},
		}
	}

}

func parseOutput(result api.ExecResult, err error, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	if err != nil {
		resp.Status = "error"
		resp.StatusMessage = fmt.Sprintf("%s", err)
		return
	}

	if len(result.Stderr) > 0 {
		resp.Status = "error"
		resp.StatusMessage = string(result.Stderr)
		return
	}
	if len(result.Stdout) == 0 {
		resp.Status = "ok"
		return
	}
	parts := bytes.SplitN(result.Stdout, BodyDelimiter, 2)
	processHeaders(string(parts[0]), resp, req)
	resp.Status = "ok"
	if len(parts) == 1 {
		return
	}
	processBody(parts[1], resp, req)
}

func writeToLog(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	if len(line) < 2 {
		return
	}
	format := "(P: %s C: %s) %s"
	message := strings.Trim(line[1], " ")
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

func extractTemplate(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	if len(line) < 2 {
		return
	}
	resp.Template = strings.Trim(line[1], " ")
}

func flagJSON(line []string, resp *messages.ExecutionResponse, req messages.ExecutionRequest) {
	resp.IsJSON = true
}
